package filesvc

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"station/internal/cache"
	"station/internal/config"
	"station/internal/models"
	"station/internal/s3store"
	"station/internal/storage"
	"station/internal/thumbs"
)

type Service struct {
	s3      *s3store.Client
	cache   *cache.Store
	maxSize int64
}

func New(s3c *s3store.Client, c *cache.Store, cfg config.Config) *Service {
	return &Service{s3: s3c, cache: c, maxSize: cfg.MaxUploadBytes}
}

func (s *Service) List(ctx context.Context, prefix string, refresh bool) (models.Listing, error) {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return models.Listing{}, err
	}
	if err := storage.GuardUserKey(prefix); err != nil && prefix != "" {
		return models.Listing{}, err
	}

	if refresh {
		if err := s.cache.DeletePrefix(ctx, prefix); err != nil {
			return models.Listing{}, fmt.Errorf("clear cache: %w", err)
		}
	} else if listing, ok, err := s.cache.Get(ctx, prefix); err != nil {
		return models.Listing{}, err
	} else if ok {
		listing.Entries = visibleEntries(listing.Entries)
		if err := s.attachURLs(ctx, listing.Entries); err != nil {
			return models.Listing{}, err
		}
		return listing, nil
	}

	entries, err := s.s3.List(ctx, prefix)
	if err != nil {
		return models.Listing{}, err
	}
	entries = visibleEntries(entries)
	sortEntries(entries)
	if err := s.cache.Put(ctx, prefix, entries); err != nil {
		return models.Listing{}, fmt.Errorf("write cache: %w", err)
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
	name, err = storage.SanitizeName(name)
	if err != nil {
		return err
	}
	if storage.IsReservedName(name) {
		return storage.ErrReserved
	}
	key := storage.JoinKey(prefix, name) + "/"
	if err := s.s3.PutFolder(ctx, key); err != nil {
		return err
	}
	return s.cache.UpsertObject(ctx, prefix, models.Entry{
		Key:   key,
		Name:  name,
		IsDir: true,
		Kind:  models.KindFolder,
	})
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
	if err := storage.GuardUserKey(destPrefix); err != nil && destPrefix != "" {
		return err
	}
	if storage.ParentPrefix(src) == destPrefix {
		return nil
	}

	if storage.IsFolderKey(src) {
		if destPrefix == src || strings.HasPrefix(destPrefix, src) {
			return fmt.Errorf("cannot move a folder into itself")
		}
		name := storage.BaseName(src)
		newPrefix := destPrefix + name + "/"
		objs, err := s.s3.ListAll(ctx, src)
		if err != nil {
			return err
		}
		oldKeys := make([]string, 0, len(objs)+1)
		for _, obj := range objs {
			oldKeys = append(oldKeys, obj.Key)
		}
		exists, err := s.s3.Exists(ctx, src)
		if err != nil {
			return err
		}
		if exists {
			oldKeys = append(oldKeys, src)
		}
		seen := map[string]struct{}{}
		for _, old := range oldKeys {
			if _, ok := seen[old]; ok {
				continue
			}
			seen[old] = struct{}{}
			rel := strings.TrimPrefix(old, src)
			dest := newPrefix + rel
			if err := s.s3.Copy(ctx, old, dest); err != nil {
				return err
			}
			if !storage.IsFolderKey(old) {
				thumbs.Relocate(ctx, s.s3, old, dest)
			}
		}
		if err := s.s3.DeleteKeys(ctx, oldKeys); err != nil {
			return err
		}
		_ = s.cache.DeletePrefix(ctx, src)
		_ = s.cache.DeletePrefix(ctx, destPrefix)
		return s.cache.RemoveObject(ctx, storage.ParentPrefix(src), src)
	}

	dest := destPrefix + storage.BaseName(src)
	dest, err = s.uniqueKey(ctx, dest)
	if err != nil {
		return err
	}
	if err := s.s3.Copy(ctx, src, dest); err != nil {
		return err
	}
	if err := s.s3.DeleteKeys(ctx, []string{src}); err != nil {
		return err
	}
	thumbs.Relocate(ctx, s.s3, src, dest)
	_ = s.cache.RemoveObject(ctx, storage.ParentPrefix(src), src)
	_ = s.cache.DeletePrefix(ctx, destPrefix)
	return nil
}

func (s *Service) uniqueKey(ctx context.Context, key string) (string, error) {
	exists, err := s.s3.Exists(ctx, key)
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
		taken, err := s.s3.Exists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find a free name")
}

func (s *Service) Trash(ctx context.Context, key string) error {
	key, err := storage.NormalizeObjectKey(key)
	if err != nil {
		return err
	}
	if err := storage.GuardUserKey(key); err != nil {
		return err
	}

	var originals []string
	if storage.IsFolderKey(key) {
		objs, err := s.s3.ListAll(ctx, key)
		if err != nil {
			return err
		}
		for _, obj := range objs {
			originals = append(originals, obj.Key)
		}
		exists, err := s.s3.Exists(ctx, key)
		if err != nil {
			return err
		}
		if exists {
			originals = append(originals, key)
		}
		if len(originals) == 0 {
			return s.cache.RemoveObject(ctx, storage.ParentPrefix(key), key)
		}
	} else {
		originals = []string{key}
	}

	seen := map[string]struct{}{}
	batchPrefix := storage.NewTrashKey("")
	for _, src := range originals {
		if _, ok := seen[src]; ok {
			continue
		}
		seen[src] = struct{}{}
		dest := batchPrefix + src
		if err := s.s3.Copy(ctx, src, dest); err != nil {
			return err
		}
	}
	if err := s.s3.DeleteKeys(ctx, originals); err != nil {
		return err
	}
	thumbs.Remove(ctx, s.s3, originals)

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
		root := commonOriginalRoot(g.item.TrashKeys)
		g.item.OriginalKey = root
		g.item.Name = storage.BaseName(root)
		g.item.IsDir = g.item.Count > 1 || storage.IsFolderKey(root)
		if g.item.IsDir {
			g.item.Kind = models.KindFolder
		} else {
			g.item.Kind = storage.KindFromName(g.item.Name, false)
			if url, err := s.s3.PresignGet(ctx, g.item.TrashKeys[0]); err == nil {
				switch g.item.Kind {
				case models.KindImage, models.KindAudio, models.KindVideo, models.KindPDF:
					g.item.PreviewURL = url
				}
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
		dest := original
		exists, err := s.s3.Exists(ctx, dest)
		if err != nil {
			return err
		}
		if exists {
			dest = restoredName(dest)
		}
		if err := s.s3.Copy(ctx, trashKey, dest); err != nil {
			return err
		}
		parents[storage.ParentPrefix(dest)] = struct{}{}
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
	name, err = storage.SanitizeName(name)
	if err != nil {
		return models.Presign{}, err
	}
	if storage.IsReservedName(name) {
		return models.Presign{}, storage.ErrReserved
	}
	key := storage.JoinKey(prefix, name)
	contentType = storage.ContentTypeFromName(name, contentType)
	return s.s3.PresignPut(ctx, key, contentType)
}

func (s *Service) CompleteUploads(ctx context.Context, keys []string) error {
	for _, raw := range keys {
		key, err := storage.NormalizeObjectKey(raw)
		if err != nil {
			return err
		}
		if err := storage.GuardUserKey(key); err != nil {
			return err
		}
		entry, err := s.s3.Head(ctx, key)
		if err != nil {
			return fmt.Errorf("head %s: %w", key, err)
		}
		if err := s.cache.UpsertObject(ctx, storage.ParentPrefix(key), entry); err != nil {
			return err
		}
		if entry.Kind == models.KindImage || entry.Kind == models.KindVideo {
			_, _ = thumbs.Ensure(ctx, s.s3, key, entry.Kind)
		}
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

func (s *Service) attachURLs(ctx context.Context, entries []models.Entry) error {
	for i := range entries {
		if entries[i].IsDir {
			continue
		}
		if err := s.enrichKind(ctx, &entries[i]); err != nil {
			return err
		}
		url, err := s.s3.PresignGet(ctx, entries[i].Key)
		if err != nil {
			return err
		}
		entries[i].DownloadURL = url
		switch entries[i].Kind {
		case models.KindImage, models.KindAudio, models.KindVideo, models.KindPDF:
			entries[i].PreviewURL = url
		}
		if entries[i].Kind == models.KindImage || entries[i].Kind == models.KindVideo {
			if thumbURLs, ok := thumbs.Existing(ctx, s.s3, entries[i].Key, entries[i].Kind); ok {
				entries[i].ThumbURLs = thumbURLs
				entries[i].ThumbURL = thumbURLs[0]
			} else {
				thumbs.EnsureLater(s.s3, entries[i].Key, entries[i].Kind)
				if entries[i].Kind == models.KindImage {
					entries[i].ThumbURL = url
				}
			}
		}
	}
	return nil
}

func (s *Service) enrichKind(ctx context.Context, e *models.Entry) error {
	e.Kind = storage.ResolveKind(e.Name, e.ContentType, e.IsDir)
	if e.Kind != models.KindFile && e.ContentType != "" && e.ContentType != "application/octet-stream" {
		return nil
	}
	headed, err := s.s3.Head(ctx, e.Key)
	if err != nil {
		return nil
	}
	e.ContentType = headed.ContentType
	e.Kind = headed.Kind
	_ = s.cache.UpsertObject(ctx, storage.ParentPrefix(e.Key), *e)
	return nil
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
