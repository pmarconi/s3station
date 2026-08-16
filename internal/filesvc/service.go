package filesvc

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"station/internal/cache"
	"station/internal/config"
	"station/internal/locks"
	"station/internal/models"
	"station/internal/s3store"
	"station/internal/storage"
	"station/internal/thumbs"
)

var (
	ErrExists      = errors.New("that name is already used")
	ErrInvalidMove = errors.New("cannot move a folder into itself")
	ErrNoThumb     = errors.New("thumbnails are only for images, videos, and PDFs")
	ErrNoSelection = errors.New("nothing selected")
)

type Service struct {
	s3         *s3store.Client
	cache      *cache.Store
	locks      *locks.Store
	thumbCache *thumbs.Mem
	maxSize    int64
	root       string
}

func New(s3c *s3store.Client, c *cache.Store, lockStore *locks.Store, cfg config.Config) *Service {
	root := cfg.FilesPrefix
	if root == "" {
		root = storage.FilesPrefix
	}
	return &Service{s3: s3c, cache: c, locks: lockStore, thumbCache: thumbs.NewMem(32 << 20), maxSize: cfg.MaxUploadBytes, root: root}
}

func (s *Service) List(ctx context.Context, prefix string, refresh bool) (models.Listing, error) {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return models.Listing{}, err
	}
	prefix, err = s.resolvePrefix(ctx, prefix)
	if err != nil {
		return models.Listing{}, err
	}
	if err := storage.GuardUserKey(prefix); err != nil && prefix != "" {
		return models.Listing{}, err
	}

	protected, err := s.locks.IsProtected(ctx, prefix)
	if err != nil {
		return models.Listing{}, err
	}
	if covering, locked := s.lockedBy(ctx, prefix); locked {
		return models.Listing{
			Prefix:       prefix,
			Locked:       true,
			LockedPrefix: covering,
			Protected:    protected,
		}, nil
	}

	if refresh {
		if err := s.cache.DeletePrefix(ctx, prefix); err != nil {
			return models.Listing{}, fmt.Errorf("clear cache: %w", err)
		}
	} else if listing, ok, err := s.cache.Get(ctx, prefix); err != nil {
		return models.Listing{}, err
	} else if ok {
		listing.Entries = visibleEntries(listing.Entries)
		if err := s.markLocks(ctx, listing.Entries); err != nil {
			return models.Listing{}, err
		}
		if err := s.attachURLs(ctx, listing.Entries); err != nil {
			return models.Listing{}, err
		}
		listing.Protected = protected
		return listing, nil
	}

	entries, err := s.s3.List(ctx, s.abs(prefix))
	if err != nil {
		return models.Listing{}, err
	}
	for i := range entries {
		entries[i].Key = s.rel(entries[i].Key)
	}
	entries = visibleEntries(entries)
	sortEntries(entries)
	if err := s.cache.Put(ctx, prefix, entries); err != nil {
		return models.Listing{}, fmt.Errorf("write cache: %w", err)
	}
	if err := s.markLocks(ctx, entries); err != nil {
		return models.Listing{}, err
	}
	if err := s.attachURLs(ctx, entries); err != nil {
		return models.Listing{}, err
	}
	listing, _, err := s.cache.Get(ctx, prefix)
	if err != nil {
		return models.Listing{}, err
	}
	listing.FromCache = false
	listing.Entries = entries
	listing.Protected = protected
	return listing, nil
}

func (s *Service) resolvePrefix(ctx context.Context, prefix string) (string, error) {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil || prefix == "" {
		return prefix, err
	}
	parts := strings.Split(strings.Trim(prefix, "/"), "/")
	cur := ""
	for i, part := range parts {
		entries, err := s.childEntries(ctx, cur)
		if err != nil {
			return "", err
		}
		if e, ok := storage.MatchFoldEntry(entries, part, true); ok {
			cur = e.Key
			if !strings.HasSuffix(cur, "/") {
				cur += "/"
			}
			continue
		}
		rest := make([]string, 0, len(parts)-i)
		for _, p := range parts[i:] {
			p, err = storage.FolderName(p)
			if err != nil {
				return "", err
			}
			rest = append(rest, p)
		}
		return cur + strings.Join(rest, "/") + "/", nil
	}
	return cur, nil
}

func (s *Service) childEntries(ctx context.Context, prefix string) ([]models.Entry, error) {
	if listing, ok, err := s.cache.Get(ctx, prefix); err != nil {
		return nil, err
	} else if ok {
		return visibleEntries(listing.Entries), nil
	}
	entries, err := s.s3.List(ctx, s.abs(prefix))
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].Key = s.rel(entries[i].Key)
	}
	return visibleEntries(entries), nil
}

func (s *Service) ListFolders(ctx context.Context, prefix string) (models.Listing, error) {
	listing, err := s.List(ctx, prefix, false)
	if err != nil {
		return listing, err
	}
	dirs := make([]models.Entry, 0, len(listing.Entries))
	for _, e := range listing.Entries {
		if e.IsDir {
			dirs = append(dirs, e)
		}
	}
	listing.Entries = dirs
	return listing, nil
}

func (s *Service) CreateFolder(ctx context.Context, prefix, name string) error {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return err
	}
	if err := storage.GuardUserKey(prefix); err != nil && prefix != "" {
		return err
	}
	if err := s.denyIfLocked(ctx, prefix); err != nil {
		return err
	}
	name, err = storage.FolderName(name)
	if err != nil {
		return err
	}
	if storage.IsReservedName(name) {
		return storage.ErrReserved
	}
	prefix, err = s.resolvePrefix(ctx, prefix)
	if err != nil {
		return err
	}
	if kids, err := s.childEntries(ctx, prefix); err != nil {
		return err
	} else if _, exists := storage.MatchFoldEntry(kids, name, true); exists {
		return ErrExists
	}
	key := storage.JoinKey(prefix, name) + "/"
	if err := s.s3.PutFolder(ctx, s.abs(key)); err != nil {
		return err
	}
	return s.cache.UpsertObject(ctx, prefix, models.Entry{
		Key:   key,
		Name:  name,
		IsDir: true,
		Kind:  models.KindFolder,
	})
}

func (s *Service) MoveMany(ctx context.Context, keys []string, destPrefix string) error {
	if len(keys) == 0 {
		return ErrNoSelection
	}
	for _, key := range keys {
		if err := s.Move(ctx, key, destPrefix); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Move(ctx context.Context, srcKey, destPrefix string) error {
	src, err := storage.NormalizeObjectKey(srcKey)
	if err != nil {
		return err
	}
	if err := storage.GuardUserKey(src); err != nil {
		return err
	}
	destPrefix, err = storage.NormalizePrefix(destPrefix)
	if err != nil {
		return err
	}
	destPrefix, err = s.resolvePrefix(ctx, destPrefix)
	if err != nil {
		return err
	}
	if err := storage.GuardUserKey(destPrefix); err != nil && destPrefix != "" {
		return err
	}
	if err := s.denyIfLocked(ctx, src); err != nil {
		return err
	}
	if err := s.denyIfLocked(ctx, destPrefix); err != nil {
		return err
	}
	if storage.ParentPrefix(src) == destPrefix {
		return nil
	}

	if storage.IsFolderKey(src) {
		if destPrefix == src || strings.HasPrefix(destPrefix, src) {
			return ErrInvalidMove
		}
		name, err := storage.FolderName(storage.BaseName(src))
		if err != nil {
			return err
		}
		newPrefix := destPrefix + name + "/"
		srcAbs := s.abs(src)
		destAbs := s.abs(newPrefix)
		objs, err := s.s3.ListAll(ctx, srcAbs)
		if err != nil {
			return err
		}
		oldKeys := make([]string, 0, len(objs)+1)
		for _, obj := range objs {
			oldKeys = append(oldKeys, obj.Key)
		}
		exists, err := s.s3.Exists(ctx, srcAbs)
		if err != nil {
			return err
		}
		if exists {
			oldKeys = append(oldKeys, srcAbs)
		}
		seen := map[string]struct{}{}
		for _, old := range oldKeys {
			if _, ok := seen[old]; ok {
				continue
			}
			seen[old] = struct{}{}
			rel := strings.TrimPrefix(old, srcAbs)
			dest := destAbs + rel
			if err := s.s3.Copy(ctx, old, dest); err != nil {
				return err
			}
			if !storage.IsFolderKey(old) {
				thumbs.Relocate(ctx, s.s3, old, dest)
				s.thumbCache.Delete(storage.AllThumbKeys(old)...)
			}
		}
		if err := s.s3.DeleteKeys(ctx, oldKeys); err != nil {
			return err
		}
		_ = s.locks.Relocate(ctx, src, newPrefix)
		_ = s.cache.DeletePrefix(ctx, src)
		_ = s.cache.DeletePrefix(ctx, destPrefix)
		return s.cache.RemoveObject(ctx, storage.ParentPrefix(src), src)
	}

	dest := destPrefix + storage.BaseName(src)
	dest, err = s.uniqueKey(ctx, dest)
	if err != nil {
		return err
	}
	if err := s.s3.Copy(ctx, s.abs(src), s.abs(dest)); err != nil {
		return err
	}
	if err := s.s3.DeleteKeys(ctx, []string{s.abs(src)}); err != nil {
		return err
	}
	thumbs.Relocate(ctx, s.s3, s.abs(src), s.abs(dest))
	s.thumbCache.Delete(storage.AllThumbKeys(s.abs(src))...)
	_ = s.cache.RemoveObject(ctx, storage.ParentPrefix(src), src)
	_ = s.cache.DeletePrefix(ctx, destPrefix)
	return nil
}

func (s *Service) Rename(ctx context.Context, key, newName string) error {
	src, err := storage.NormalizeObjectKey(key)
	if err != nil {
		return err
	}
	if err := storage.GuardUserKey(src); err != nil {
		return err
	}
	if storage.IsFolderKey(src) {
		newName, err = storage.FolderName(newName)
	} else {
		newName, err = storage.SanitizeName(newName)
	}
	if err != nil {
		return err
	}
	if storage.IsReservedName(newName) {
		return storage.ErrReserved
	}
	if err := s.denyIfLocked(ctx, src); err != nil {
		return err
	}
	dest := storage.ParentPrefix(src) + newName
	if storage.IsFolderKey(src) {
		dest += "/"
	}
	if dest == src {
		return nil
	}
	taken, err := s.destTaken(ctx, dest, src)
	if err != nil {
		return err
	}
	if taken {
		return ErrExists
	}
	return s.relocate(ctx, src, dest)
}

func (s *Service) destTaken(ctx context.Context, dest, ignore string) (bool, error) {
	exists, err := s.s3.Exists(ctx, s.abs(dest))
	if err != nil || exists {
		return exists, err
	}
	if storage.IsFolderKey(dest) {
		objs, err := s.s3.ListAll(ctx, s.abs(dest))
		if err != nil || len(objs) > 0 {
			return len(objs) > 0, err
		}
		kids, err := s.childEntries(ctx, storage.ParentPrefix(dest))
		if err != nil {
			return false, err
		}
		e, ok := storage.MatchFoldEntry(kids, storage.BaseName(dest), true)
		return ok && e.Key != ignore, nil
	}
	return false, nil
}

func (s *Service) relocate(ctx context.Context, src, dest string) error {
	parent := storage.ParentPrefix(src)
	if storage.IsFolderKey(src) {
		srcAbs := s.abs(src)
		destAbs := s.abs(dest)
		objs, err := s.s3.ListAll(ctx, srcAbs)
		if err != nil {
			return err
		}
		oldKeys := make([]string, 0, len(objs)+1)
		for _, obj := range objs {
			oldKeys = append(oldKeys, obj.Key)
		}
		exists, err := s.s3.Exists(ctx, srcAbs)
		if err != nil {
			return err
		}
		if exists {
			oldKeys = append(oldKeys, srcAbs)
		}
		seen := map[string]struct{}{}
		for _, old := range oldKeys {
			if _, ok := seen[old]; ok {
				continue
			}
			seen[old] = struct{}{}
			rel := strings.TrimPrefix(old, srcAbs)
			if err := s.s3.Copy(ctx, old, destAbs+rel); err != nil {
				return err
			}
			if !storage.IsFolderKey(old) {
				thumbs.Relocate(ctx, s.s3, old, destAbs+rel)
				s.thumbCache.Delete(storage.AllThumbKeys(old)...)
			}
		}
		if err := s.s3.DeleteKeys(ctx, oldKeys); err != nil {
			return err
		}
		_ = s.locks.Relocate(ctx, src, dest)
		_ = s.cache.DeletePrefix(ctx, src)
		_ = s.cache.DeletePrefix(ctx, parent)
		return s.cache.RemoveObject(ctx, parent, src)
	}

	if err := s.s3.Copy(ctx, s.abs(src), s.abs(dest)); err != nil {
		return err
	}
	if err := s.s3.DeleteKeys(ctx, []string{s.abs(src)}); err != nil {
		return err
	}
	thumbs.Relocate(ctx, s.s3, s.abs(src), s.abs(dest))
	s.thumbCache.Delete(storage.AllThumbKeys(s.abs(src))...)
	_ = s.cache.RemoveObject(ctx, parent, src)
	_ = s.cache.DeletePrefix(ctx, parent)
	return nil
}

func (s *Service) uniqueKey(ctx context.Context, key string) (string, error) {
	exists, err := s.s3.Exists(ctx, s.abs(key))
	if err != nil {
		return "", err
	}
	if !exists {
		return key, nil
	}
	ext := filepath.Ext(key)
	base := strings.TrimSuffix(key, ext)
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		taken, err := s.s3.Exists(ctx, s.abs(candidate))
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find a free name")
}

func (s *Service) TrashMany(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return ErrNoSelection
	}
	for _, key := range keys {
		if err := s.Trash(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Trash(ctx context.Context, key string) error {
	key, err := storage.NormalizeObjectKey(key)
	if err != nil {
		return err
	}
	if err := storage.GuardUserKey(key); err != nil {
		return err
	}
	if err := s.denyIfLocked(ctx, key); err != nil {
		return err
	}

	var originals []string
	if storage.IsFolderKey(key) {
		objs, err := s.s3.ListAll(ctx, s.abs(key))
		if err != nil {
			return err
		}
		for _, obj := range objs {
			originals = append(originals, obj.Key)
		}
		exists, err := s.s3.Exists(ctx, s.abs(key))
		if err != nil {
			return err
		}
		if exists {
			originals = append(originals, s.abs(key))
		}
		if len(originals) == 0 {
			return s.cache.RemoveObject(ctx, storage.ParentPrefix(key), key)
		}
	} else {
		originals = []string{s.abs(key)}
	}

	seen := map[string]struct{}{}
	batchPrefix := storage.NewTrashKey("")
	for _, src := range originals {
		if _, ok := seen[src]; ok {
			continue
		}
		seen[src] = struct{}{}
		dest := batchPrefix + s.rel(src)
		if err := s.s3.Copy(ctx, src, dest); err != nil {
			return err
		}
	}
	if err := s.s3.DeleteKeys(ctx, originals); err != nil {
		return err
	}
	thumbs.Remove(ctx, s.s3, originals)
	for _, key := range originals {
		s.thumbCache.Delete(storage.AllThumbKeys(key)...)
	}

	if storage.IsFolderKey(key) {
		if err := s.cache.DeletePrefix(ctx, key); err != nil {
			return err
		}
	}
	return s.cache.RemoveObject(ctx, storage.ParentPrefix(key), key)
}

func (s *Service) ListTrash(ctx context.Context) ([]models.TrashItem, error) {
	objs, err := s.s3.ListAll(ctx, storage.TrashPrefix)
	if err != nil {
		return nil, err
	}

	type acc struct {
		item models.TrashItem
	}
	groups := map[string]*acc{}
	order := make([]string, 0)

	for _, obj := range objs {
		batch, original, ok := storage.ParseTrashKey(obj.Key)
		if !ok {
			continue
		}
		g, exists := groups[batch]
		if !exists {
			g = &acc{item: models.TrashItem{
				BatchID:     batch,
				OriginalKey: original,
				Name:        storage.BaseName(original),
				TrashedAt:   obj.LastModified,
				Kind:        storage.KindFromName(storage.BaseName(original), storage.IsFolderKey(original)),
			}}
			groups[batch] = g
			order = append(order, batch)
		}
		g.item.TrashKeys = append(g.item.TrashKeys, obj.Key)
		g.item.Size += obj.Size
		g.item.Count++
		if obj.LastModified.After(g.item.TrashedAt) {
			g.item.TrashedAt = obj.LastModified
		}
		if storage.IsFolderKey(original) || strings.Count(original, "/") > strings.Count(g.item.OriginalKey, "/") {
			// keep the shortest original path as the item root
		}
	}

	items := make([]models.TrashItem, 0, len(order))
	for _, batch := range order {
		g := groups[batch]
		root := s.rel(commonOriginalRoot(g.item.TrashKeys))
		g.item.OriginalKey = root
		g.item.Name = storage.BaseName(root)
		g.item.IsDir = g.item.Count > 1 || storage.IsFolderKey(root)
		if g.item.IsDir {
			g.item.Kind = models.KindFolder
		} else {
			g.item.Kind = storage.KindFromName(g.item.Name, false)
			src := g.item.TrashKeys[0]
			if url, err := s.s3.PresignGet(ctx, src); err == nil {
				switch g.item.Kind {
				case models.KindImage, models.KindAudio, models.KindVideo, models.KindPDF:
					g.item.PreviewURL = url
				}
			}
			if g.item.Kind == models.KindImage || g.item.Kind == models.KindVideo || g.item.Kind == models.KindPDF {
				if urls := thumbs.LocalURLs(src, g.item.Kind); len(urls) > 0 {
					g.item.ThumbURLs = urls
					g.item.ThumbURL = urls[0]
				}
				thumbs.EnsureLater(s.s3, src, g.item.Kind)
			}
		}
		if g.item.Name == "" {
			g.item.Name = root
		}
		items = append(items, g.item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].TrashedAt.After(items[j].TrashedAt)
	})
	return items, nil
}

func (s *Service) Restore(ctx context.Context, batchID string) error {
	item, err := s.trashBatch(ctx, batchID)
	if err != nil {
		return err
	}
	parents := map[string]struct{}{}
	for _, trashKey := range item.TrashKeys {
		_, original, ok := storage.ParseTrashKey(trashKey)
		if !ok {
			continue
		}
		dest := s.abs(original)
		exists, err := s.s3.Exists(ctx, dest)
		if err != nil {
			return err
		}
		if exists {
			dest = s.abs(restoredName(original))
		}
		if err := s.s3.Copy(ctx, trashKey, dest); err != nil {
			return err
		}
		parents[storage.ParentPrefix(original)] = struct{}{}
	}
	if err := s.s3.DeleteKeys(ctx, item.TrashKeys); err != nil {
		return err
	}
	for parent := range parents {
		_ = s.cache.DeletePrefix(ctx, parent)
	}
	return nil
}

func (s *Service) PurgeTrash(ctx context.Context, batchID string) error {
	item, err := s.trashBatch(ctx, batchID)
	if err != nil {
		return err
	}
	return s.s3.DeleteKeys(ctx, item.TrashKeys)
}

func (s *Service) EmptyTrash(ctx context.Context) error {
	objs, err := s.s3.ListAll(ctx, storage.TrashPrefix)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(objs))
	for _, obj := range objs {
		keys = append(keys, obj.Key)
	}
	if len(keys) == 0 {
		return nil
	}
	return s.s3.DeleteKeys(ctx, keys)
}

func (s *Service) PurgeThumbs(ctx context.Context) (int, error) {
	objs, err := s.s3.ListAll(ctx, storage.ThumbPrefix)
	if err != nil {
		return 0, err
	}
	keys := make([]string, 0, len(objs))
	for _, obj := range objs {
		if storage.IsThumbKey(obj.Key) {
			keys = append(keys, obj.Key)
		}
	}
	if len(keys) > 0 {
		if err := s.s3.DeleteKeys(ctx, keys); err != nil {
			return 0, err
		}
	}
	s.thumbCache.Flush()
	return len(keys), nil
}

func (s *Service) ClearCache(ctx context.Context, prefix string) error {
	if prefix == "" {
		return s.cache.DeletePrefix(ctx, "")
	}
	p, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return err
	}
	return s.cache.DeletePrefix(ctx, p)
}

func (s *Service) PresignUpload(ctx context.Context, prefix, name, contentType string, size int64) (models.Presign, error) {
	if s.maxSize > 0 && size > s.maxSize {
		return models.Presign{}, fmt.Errorf("file exceeds max size of %d bytes", s.maxSize)
	}
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return models.Presign{}, err
	}
	if err := storage.GuardUserKey(prefix); err != nil && prefix != "" {
		return models.Presign{}, err
	}
	if err := s.denyIfLocked(ctx, prefix); err != nil {
		return models.Presign{}, err
	}
	name, err = storage.SanitizeUploadPath(name)
	if err != nil {
		return models.Presign{}, err
	}
	key := storage.JoinKey(prefix, name)
	contentType = storage.ContentTypeFromName(name, contentType)
	spec, err := s.s3.PresignPut(ctx, s.abs(key), contentType)
	if err != nil {
		return models.Presign{}, err
	}
	spec.Key = s.abs(key)
	return spec, nil
}

func (s *Service) CompleteUploads(ctx context.Context, prefix string, keys, folders []string) error {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return err
	}
	if err := storage.GuardUserKey(prefix); err != nil && prefix != "" {
		return err
	}
	if err := s.denyIfLocked(ctx, prefix); err != nil {
		return err
	}
	for _, rel := range folders {
		rel, err := storage.SanitizeFolderRelPath(rel)
		if err != nil {
			return err
		}
		folderKey := storage.JoinKey(prefix, rel) + "/"
		if err := s.s3.PutFolder(ctx, s.abs(folderKey)); err != nil {
			return err
		}
		if err := s.rememberFolderTree(ctx, folderKey); err != nil {
			return err
		}
	}
	for _, raw := range keys {
		key, err := storage.NormalizeObjectKey(raw)
		if err != nil {
			return err
		}
		logical := s.rel(key)
		if err := storage.GuardUserKey(logical); err != nil {
			return err
		}
		if err := s.denyIfLocked(ctx, logical); err != nil {
			return err
		}
		entry, err := s.s3.Stat(ctx, s.abs(logical))
		if err != nil {
			return fmt.Errorf("head %s: %w", logical, err)
		}
		entry.Key = logical
		if err := s.cache.UpsertObject(ctx, storage.ParentPrefix(logical), entry); err != nil {
			return err
		}
		if err := s.rememberFolderTree(ctx, storage.ParentPrefix(logical)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) rememberFolderTree(ctx context.Context, folderKey string) error {
	if folderKey == "" {
		return nil
	}
	parent, err := storage.NormalizePrefix(folderKey)
	if err != nil {
		return err
	}
	for parent != "" {
		grand := storage.ParentPrefix(parent)
		entry := models.Entry{
			Key:   parent,
			Name:  storage.BaseName(parent),
			IsDir: true,
			Kind:  models.KindFolder,
		}
		if err := s.cache.UpsertObject(ctx, grand, entry); err != nil {
			return err
		}
		parent = grand
	}
	return nil
}

func (s *Service) trashBatch(ctx context.Context, batchID string) (models.TrashItem, error) {
	items, err := s.ListTrash(ctx)
	if err != nil {
		return models.TrashItem{}, err
	}
	for _, item := range items {
		if item.BatchID == batchID {
			return item, nil
		}
	}
	return models.TrashItem{}, fmt.Errorf("trash item not found")
}

func (s *Service) ProtectFolder(ctx context.Context, prefix, password string) error {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return err
	}
	if prefix == "" {
		return locks.ErrRoot
	}
	if err := storage.GuardUserKey(prefix); err != nil {
		return err
	}
	if err := s.denyIfLocked(ctx, storage.ParentPrefix(prefix)); err != nil {
		return err
	}
	return s.locks.Protect(ctx, prefix, password)
}

func (s *Service) UnprotectFolder(ctx context.Context, prefix, password string) error {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return err
	}
	return s.locks.Unprotect(ctx, prefix, password)
}

func (s *Service) UnlockFolder(ctx context.Context, prefix, password string) (string, error) {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return "", err
	}
	covering, locked := s.lockedBy(ctx, prefix)
	if !locked {
		return prefix, nil
	}
	if err := s.locks.Check(ctx, covering, password); err != nil {
		return "", err
	}
	return covering, nil
}

func (s *Service) denyIfLocked(ctx context.Context, key string) error {
	if _, locked := s.lockedBy(ctx, key); locked {
		return locks.ErrLocked
	}
	return nil
}

func (s *Service) lockedBy(ctx context.Context, key string) (string, bool) {
	prefix := key
	if key != "" && !storage.IsFolderKey(key) {
		prefix = storage.ParentPrefix(key)
	}
	covers, err := s.locks.Covering(ctx, prefix)
	if err != nil || len(covers) == 0 {
		return "", false
	}
	open := unlockedSet(locks.Unlocked(ctx))
	for _, p := range covers {
		if !open[p] {
			return p, true
		}
	}
	return "", false
}

func (s *Service) markLocks(ctx context.Context, entries []models.Entry) error {
	open := unlockedSet(locks.Unlocked(ctx))
	for i := range entries {
		if !entries[i].IsDir {
			continue
		}
		ok, err := s.locks.IsProtected(ctx, entries[i].Key)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		entries[i].Protected = true
		if !open[entries[i].Key] {
			entries[i].Locked = true
		}
	}
	return nil
}

func unlockedSet(prefixes []string) map[string]bool {
	out := make(map[string]bool, len(prefixes))
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		out[p] = true
	}
	return out
}

func (s *Service) GenerateThumb(ctx context.Context, key string) error {
	key, err := storage.NormalizeObjectKey(key)
	if err != nil {
		return err
	}
	if err := storage.GuardUserKey(key); err != nil {
		return err
	}
	if storage.IsFolderKey(key) {
		return ErrNoThumb
	}
	if err := s.denyIfLocked(ctx, key); err != nil {
		return err
	}
	entry := models.Entry{Key: key, Name: storage.BaseName(key)}
	if err := s.enrichKind(ctx, &entry); err != nil {
		return err
	}
	if entry.Kind != models.KindImage && entry.Kind != models.KindVideo && entry.Kind != models.KindPDF {
		return ErrNoThumb
	}
	abs := s.abs(key)
	s.thumbCache.Delete(storage.AllThumbKeys(abs)...)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	_, err = thumbs.Refresh(ctx, s.s3, abs, entry.Kind)
	return err
}

func (s *Service) ServeThumb(ctx context.Context, thumbKey string) ([]byte, string, string, error) {
	if !storage.IsThumbKey(thumbKey) {
		return nil, "", "", storage.ErrInvalidPath
	}
	source, ok := storage.SourceKeyFromThumb(thumbKey)
	if !ok {
		return nil, "", "", storage.ErrInvalidPath
	}
	if err := s.denyIfLocked(ctx, s.rel(source)); err != nil {
		return nil, "", "", err
	}
	if item, hit := s.thumbCache.Get(thumbKey); hit {
		return item.Data, item.Type, item.ETag, nil
	}
	raw, ctype, err := s.s3.Get(ctx, thumbKey)
	if err != nil || len(raw) == 0 {
		kind := models.KindImage
		if storage.IsVideoThumb(thumbKey) {
			kind = models.KindVideo
		}
		if _, genErr := thumbs.Ensure(ctx, s.s3, source, kind); genErr != nil {
			if err != nil {
				return nil, "", "", err
			}
			return nil, "", "", genErr
		}
		raw, ctype, err = s.s3.Get(ctx, thumbKey)
		if err != nil {
			return nil, "", "", err
		}
	}
	if ctype == "" {
		ctype = "image/webp"
	}
	sum := sha256.Sum256(raw)
	etag := fmt.Sprintf(`"%x"`, sum[:8])
	s.thumbCache.Put(thumbKey, raw, ctype, etag)
	return raw, ctype, etag, nil
}

func (s *Service) attachURLs(ctx context.Context, entries []models.Entry) error {
	for i := range entries {
		if entries[i].IsDir {
			continue
		}
		if err := s.enrichKind(ctx, &entries[i]); err != nil {
			return err
		}
		s3key := s.abs(entries[i].Key)
		url, err := s.s3.PresignGet(ctx, s3key)
		if err != nil {
			return err
		}
		entries[i].DownloadURL = url
		if save, err := s.s3.PresignDownload(ctx, s3key, entries[i].Name); err == nil {
			entries[i].SaveURL = save
		}
		switch entries[i].Kind {
		case models.KindImage, models.KindAudio, models.KindVideo, models.KindPDF:
			entries[i].PreviewURL = url
		}
		if entries[i].Kind == models.KindImage || entries[i].Kind == models.KindVideo || entries[i].Kind == models.KindPDF {
			if urls := thumbs.LocalURLs(s3key, entries[i].Kind); len(urls) > 0 {
				entries[i].ThumbURLs = urls
				entries[i].ThumbURL = urls[0]
			}
			thumbs.EnsureLater(s.s3, s3key, entries[i].Kind)
		}
	}
	return nil
}

func (s *Service) enrichKind(ctx context.Context, e *models.Entry) error {
	e.Kind = storage.ResolveKind(e.Name, e.ContentType, e.IsDir)
	needHead := e.ContentType == "" || e.ContentType == "application/octet-stream" || e.Kind == models.KindFile || (!e.IsDir && e.Size <= 0)
	if !needHead {
		return nil
	}
	headed, err := s.s3.Stat(ctx, s.abs(e.Key))
	if err != nil {
		return nil
	}
	applyHead(e, headed)
	_ = s.cache.UpsertObject(ctx, storage.ParentPrefix(e.Key), *e)
	return nil
}

func applyHead(e *models.Entry, headed models.Entry) {
	if headed.ContentType != "" {
		e.ContentType = headed.ContentType
	}
	if headed.Size > 0 {
		e.Size = headed.Size
	}
	if headed.ETag != "" {
		e.ETag = headed.ETag
	}
	if !headed.LastModified.IsZero() {
		e.LastModified = headed.LastModified
	}
	e.Kind = storage.ResolveKind(e.Name, e.ContentType, e.IsDir)
}

func Breadcrumbs(prefix string) []models.Crumb {
	crumbs := []models.Crumb{{Name: "Depot", Prefix: ""}}
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return crumbs
	}
	parts := strings.Split(prefix, "/")
	cur := ""
	for _, part := range parts {
		cur += part + "/"
		crumbs = append(crumbs, models.Crumb{Name: part, Prefix: cur})
	}
	return crumbs
}

func visibleEntries(entries []models.Entry) []models.Entry {
	out := make([]models.Entry, 0, len(entries))
	for _, e := range entries {
		if storage.IsTrashKey(e.Key) || storage.IsThumbKey(e.Key) || storage.IsReservedName(e.Name) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func commonOriginalRoot(trashKeys []string) string {
	originals := make([]string, 0, len(trashKeys))
	for _, key := range trashKeys {
		_, original, ok := storage.ParseTrashKey(key)
		if ok {
			originals = append(originals, original)
		}
	}
	if len(originals) == 0 {
		return ""
	}
	if len(originals) == 1 {
		return originals[0]
	}
	prefix := originals[0]
	if !strings.HasSuffix(prefix, "/") {
		prefix = storage.ParentPrefix(prefix)
	}
	for _, o := range originals[1:] {
		for !strings.HasPrefix(o, prefix) && prefix != "" {
			prefix = storage.ParentPrefix(strings.TrimSuffix(prefix, "/"))
			if prefix != "" && !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}
		}
	}
	if prefix == "" {
		return originals[0]
	}
	return prefix
}

func restoredName(key string) string {
	ext := filepath.Ext(key)
	base := strings.TrimSuffix(key, ext)
	return fmt.Sprintf("%s (restored)%s", base, ext)
}

func sortEntries(entries []models.Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

func RelTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2 Jan 2006")
	}
}

func FormatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	val := float64(n)
	for _, u := range units {
		val /= 1024
		if val < 1024 {
			return fmt.Sprintf("%.1f %s", val, u)
		}
	}
	return fmt.Sprintf("%.1f PB", val/1024)
}
