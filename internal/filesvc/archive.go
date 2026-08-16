package filesvc

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

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
	zw := zip.NewWriter(w)

	for _, obj := range arch.Objects {
		name, ok := archiveRelPath(arch.Root, arch.Prefix, obj.Key)
		if !ok {
			continue
		}
		hdr := &zip.FileHeader{
			Name:     name,
			Method:   zip.Deflate,
			Modified: obj.LastModified,
		}
		fw, err := zw.CreateHeader(hdr)
		if err != nil {
			return fmt.Errorf("zip entry: %w", err)
		}
		body, _, err := s.s3.Open(ctx, obj.Key)
		if err != nil {
			return fmt.Errorf("read %s: %w", obj.Key, err)
		}
		_, copyErr := io.Copy(fw, body)
		_ = body.Close()
		if copyErr != nil {
			return fmt.Errorf("copy %s: %w", obj.Key, copyErr)
		}
		if err := zw.Flush(); err != nil {
			return err
		}
	}
	return zw.Close()
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
