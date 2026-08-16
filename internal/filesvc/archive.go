package filesvc

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"station/internal/locks"
	"station/internal/s3store"
	"station/internal/storage"
)

type FolderArchive struct {
	Filename string
	Prefix   string
	Root     string
	Objects  []s3store.Object
}

type ZipItem struct {
	Key      string
	Name     string
	Modified time.Time
}

type SelectionArchive struct {
	Filename string
	Items    []ZipItem
}

func (s *Service) PrepareFolderArchive(ctx context.Context, prefix string) (FolderArchive, error) {
	prefix, err := storage.NormalizePrefix(prefix)
	if err != nil {
		return FolderArchive{}, err
	}
	if err := storage.GuardUserKey(prefix); err != nil && prefix != "" {
		return FolderArchive{}, err
	}
	if err := s.denyIfLocked(ctx, prefix); err != nil {
		return FolderArchive{}, err
	}

	s3prefix := s.abs(prefix)
	objs, err := s.s3.ListAll(ctx, s3prefix)
	if err != nil {
		return FolderArchive{}, err
	}
	blocked, err := s.lockedNested(ctx, prefix)
	if err != nil {
		return FolderArchive{}, err
	}

	root := storage.BaseName(prefix)
	if root == "" {
		root = "depot"
	}
	kept := make([]s3store.Object, 0, len(objs))
	for _, obj := range objs {
		if _, ok := archiveRelPath(root, s3prefix, obj.Key); !ok {
			continue
		}
		if underLocked(s.rel(obj.Key), blocked) {
			continue
		}
		kept = append(kept, obj)
	}
	return FolderArchive{
		Filename: root + ".zip",
		Prefix:   s3prefix,
		Root:     root,
		Objects:  kept,
	}, nil
}

func (s *Service) WriteFolderArchive(ctx context.Context, arch FolderArchive, w io.Writer) error {
	items := make([]ZipItem, 0, len(arch.Objects))
	for _, obj := range arch.Objects {
		name, ok := archiveRelPath(arch.Root, arch.Prefix, obj.Key)
		if !ok {
			continue
		}
		items = append(items, ZipItem{Key: obj.Key, Name: name, Modified: obj.LastModified})
	}
	return s.writeZip(ctx, items, w)
}

func (s *Service) PrepareSelectionArchive(ctx context.Context, keys []string) (SelectionArchive, error) {
	seen := map[string]struct{}{}
	clean := make([]string, 0, len(keys))
	for _, key := range keys {
		key, err := storage.NormalizeObjectKey(key)
		if err != nil {
			return SelectionArchive{}, err
		}
		if err := storage.GuardUserKey(key); err != nil {
			return SelectionArchive{}, err
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, key)
	}
	if len(clean) == 0 {
		return SelectionArchive{}, ErrNoSelection
	}

	used := map[string]int{}
	items := make([]ZipItem, 0)
	for _, key := range clean {
		if err := s.denyIfLocked(ctx, key); err != nil {
			return SelectionArchive{}, err
		}
		if storage.IsFolderKey(key) {
			folderItems, err := s.folderZipItems(ctx, key, used)
			if err != nil {
				return SelectionArchive{}, err
			}
			items = append(items, folderItems...)
			continue
		}
		ent, err := s.s3.Head(ctx, s.abs(key))
		if err != nil {
			return SelectionArchive{}, err
		}
		items = append(items, ZipItem{
			Key:      s.abs(key),
			Name:     uniqueZipName(used, storage.BaseName(key)),
			Modified: ent.LastModified,
		})
	}
	if len(items) == 0 {
		return SelectionArchive{}, ErrNoSelection
	}
	filename := "download.zip"
	if len(clean) == 1 {
		filename = storage.BaseName(clean[0]) + ".zip"
	}
	return SelectionArchive{Filename: filename, Items: items}, nil
}

func (s *Service) folderZipItems(ctx context.Context, prefix string, used map[string]int) ([]ZipItem, error) {
	s3prefix := s.abs(prefix)
	objs, err := s.s3.ListAll(ctx, s3prefix)
	if err != nil {
		return nil, err
	}
	blocked, err := s.lockedNested(ctx, prefix)
	if err != nil {
		return nil, err
	}
	root := storage.BaseName(prefix)
	if root == "" {
		root = "folder"
	}
	root = uniqueZipName(used, root)
	items := make([]ZipItem, 0, len(objs))
	for _, obj := range objs {
		name, ok := archiveRelPath(root, s3prefix, obj.Key)
		if !ok || underLocked(s.rel(obj.Key), blocked) {
			continue
		}
		items = append(items, ZipItem{Key: obj.Key, Name: name, Modified: obj.LastModified})
	}
	return items, nil
}

func (s *Service) WriteSelectionArchive(ctx context.Context, arch SelectionArchive, w io.Writer) error {
	return s.writeZip(ctx, arch.Items, w)
}

func (s *Service) writeZip(ctx context.Context, items []ZipItem, w io.Writer) error {
	zw := zip.NewWriter(w)
	for _, item := range items {
		if item.Name == "" || strings.Contains(item.Name, "..") {
			continue
		}
		hdr := &zip.FileHeader{
			Name:     item.Name,
			Method:   zip.Deflate,
			Modified: item.Modified,
		}
		fw, err := zw.CreateHeader(hdr)
		if err != nil {
			return fmt.Errorf("zip entry: %w", err)
		}
		body, _, err := s.s3.Open(ctx, item.Key)
		if err != nil {
			return fmt.Errorf("read %s: %w", item.Key, err)
		}
		_, copyErr := io.Copy(fw, body)
		_ = body.Close()
		if copyErr != nil {
			return fmt.Errorf("copy %s: %w", item.Key, copyErr)
		}
		if err := zw.Flush(); err != nil {
			return err
		}
	}
	return zw.Close()
}

func uniqueZipName(used map[string]int, name string) string {
	n := used[name]
	used[name]++
	if n == 0 {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s (%d)%s", base, n+1, ext)
}

func (s *Service) lockedNested(ctx context.Context, prefix string) ([]string, error) {
	nested, err := s.locks.Nested(ctx, prefix)
	if err != nil {
		return nil, err
	}
	open := unlockedSet(locks.Unlocked(ctx))
	out := make([]string, 0, len(nested))
	for _, p := range nested {
		if !open[p] {
			out = append(out, p)
		}
	}
	return out, nil
}

func underLocked(key string, locked []string) bool {
	for _, p := range locked {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

func archiveRelPath(root, prefix, key string) (string, bool) {
	if storage.IsFolderKey(key) || storage.IsTrashKey(key) || storage.IsThumbKey(key) {
		return "", false
	}
	rel := strings.TrimPrefix(key, prefix)
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		return "", false
	}
	if prefix != "" && rel == key {
		return "", false
	}
	return root + "/" + rel, true
}

func AttachmentDisposition(filename string) string {
	filename = strings.ReplaceAll(filename, `"`, `'`)
	filename = strings.ReplaceAll(filename, "\n", "")
	filename = strings.ReplaceAll(filename, "\r", "")
	if filename == "" {
		filename = "download.zip"
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, url.PathEscape(filename))
}
