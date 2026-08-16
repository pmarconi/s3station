package thumbs

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"

	"station/internal/models"
	"station/internal/s3store"
	"station/internal/storage"
)

const maxEdge = 256

func LocalURLs(key string, kind models.Kind) []string {
	keys := thumbKeys(key, kind)
	if len(keys) == 0 {
		return nil
	}
	urls := make([]string, len(keys))
	for i, thumbKey := range keys {
		urls[i] = storage.PublicThumbPath(thumbKey)
	}
	return urls
}

func EnsureLater(s3c *s3store.Client, key string, kind models.Kind) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_, _ = Ensure(ctx, s3c, key, kind)
	}()
}

func thumbKeys(key string, kind models.Kind) []string {
	switch kind {
	case models.KindImage:
		return []string{storage.ThumbKey(key)}
	case models.KindVideo:
		keys := make([]string, storage.VideoThumbCount)
		for i := range keys {
			keys[i] = storage.VideoThumbKey(key, i)
		}
		return keys
	default:
		return nil
	}
}

func Refresh(ctx context.Context, s3c *s3store.Client, key string, kind models.Kind) ([]string, error) {
	Remove(ctx, s3c, []string{key})
	return Ensure(ctx, s3c, key, kind)
}

var genLocks sync.Map

func Ensure(ctx context.Context, s3c *s3store.Client, key string, kind models.Kind) ([]string, error) {
	v, _ := genLocks.LoadOrStore(key+"\x00"+string(kind), &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	switch kind {
	case models.KindImage:
		url, err := ensureImage(ctx, s3c, key)
		if err != nil {
			return nil, err
		}
		return []string{url}, nil
	case models.KindVideo:
		return ensureVideo(ctx, s3c, key)
	default:
		return nil, nil
	}
}

func Relocate(ctx context.Context, s3c *s3store.Client, src, dest string) {
	for _, oldKey := range storage.AllThumbKeys(src) {
		exists, err := s3c.Exists(ctx, oldKey)
		if err != nil || !exists {
			continue
		}
		newKey := relocateThumbKey(oldKey, src, dest)
		if newKey == "" || newKey == oldKey {
			continue
		}
		if err := s3c.Copy(ctx, oldKey, newKey); err != nil {
			continue
		}
		_ = s3c.DeleteKeys(ctx, []string{oldKey})
	}
}

func Remove(ctx context.Context, s3c *s3store.Client, originalKeys []string) {
	keys := make([]string, 0)
	for _, key := range originalKeys {
		if key == "" || storage.IsFolderKey(key) {
			continue
		}
		keys = append(keys, storage.AllThumbKeys(key)...)
	}
	if len(keys) > 0 {
		_ = s3c.DeleteKeys(ctx, keys)
	}
}

func relocateThumbKey(oldKey, src, dest string) string {
	if oldKey == storage.ThumbKey(src) {
		return storage.ThumbKey(dest)
	}
	for i := 0; i < storage.VideoThumbCount; i++ {
		if oldKey == storage.VideoThumbKey(src, i) {
			return storage.VideoThumbKey(dest, i)
		}
	}
	return ""
}

func ensureImage(ctx context.Context, s3c *s3store.Client, key string) (string, error) {
	thumbKey := storage.ThumbKey(key)
	exists, err := s3c.Exists(ctx, thumbKey)
	if err != nil {
		return "", err
	}
	if !exists {
		if err := generateImage(ctx, s3c, key, thumbKey); err != nil {
			return "", err
		}
	}
	return s3c.PresignGet(ctx, thumbKey)
}

func generateImage(ctx context.Context, s3c *s3store.Client, src, dest string) error {
	raw, _, err := s3c.Get(ctx, src)
	if err != nil {
		return fmt.Errorf("download source: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}
	thumb := imaging.Fit(img, maxEdge, maxEdge, imaging.Lanczos)
	raw, err = encodeWebP(thumb)
	if err != nil {
		return err
	}
	return s3c.Put(ctx, dest, raw, "image/webp")
}
