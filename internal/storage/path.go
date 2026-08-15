package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

const (
	TrashPrefix = ".station-trash/"
	ThumbPrefix = ".station-thumbs/"
)

var (
	ErrInvalidPath = errors.New("invalid path")
	ErrReserved    = errors.New("reserved path")
)

func NormalizePrefix(p string) (string, error) {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimLeft(p, "/")
	if p == "" {
		return "", nil
	}

	parts := strings.Split(p, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." || strings.Contains(part, "..") {
			return "", ErrInvalidPath
		}
		if strings.ContainsRune(part, 0) {
			return "", ErrInvalidPath
		}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		return "", nil
	}
	return strings.Join(clean, "/") + "/", nil
}

func SanitizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.Trim(name, " ")
	if name == "" || name == "." || name == ".." {
		return "", ErrInvalidPath
	}
	if strings.ContainsAny(name, "/\x00") {
		return "", ErrInvalidPath
	}
	var b strings.Builder
	for _, r := range name {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	name = strings.TrimSpace(b.String())
	if name == "" || name == "." || name == ".." {
		return "", ErrInvalidPath
	}
	return name, nil
}

func JoinKey(prefix, name string) string {
	return prefix + name
}

func ParentPrefix(key string) string {
	key = strings.TrimRight(key, "/")
	i := strings.LastIndex(key, "/")
	if i < 0 {
		return ""
	}
	return key[:i+1]
}

func BaseName(key string) string {
	key = strings.TrimRight(key, "/")
	i := strings.LastIndex(key, "/")
	if i < 0 {
		return key
	}
	return key[i+1:]
}

func IsFolderKey(key string) bool {
	return strings.HasSuffix(key, "/")
}

func NormalizeObjectKey(key string) (string, error) {
	key = strings.TrimSpace(strings.ReplaceAll(key, "\\", "/"))
	key = strings.TrimLeft(key, "/")
	if key == "" || strings.Contains(key, "..") || strings.ContainsRune(key, 0) {
		return "", ErrInvalidPath
	}
	if strings.HasSuffix(key, "/") {
		return NormalizePrefix(key)
	}
	parent, err := NormalizePrefix(ParentPrefix(key))
	if err != nil {
		return "", err
	}
	name, err := SanitizeName(BaseName(key))
	if err != nil {
		return "", err
	}
	return parent + name, nil
}

func IsReservedName(name string) bool {
	switch name {
	case strings.TrimSuffix(TrashPrefix, "/"), strings.TrimSuffix(ThumbPrefix, "/"):
		return true
	default:
		return false
	}
}

func IsTrashKey(key string) bool {
	return strings.HasPrefix(key, TrashPrefix)
}

func IsThumbKey(key string) bool {
	return strings.HasPrefix(key, ThumbPrefix)
}

func ThumbKey(original string) string {
	return ThumbPrefix + strings.TrimPrefix(original, "/") + ".jpg"
}

const VideoThumbCount = 5

func VideoThumbKey(original string, i int) string {
	return fmt.Sprintf("%s%s.v%02d.jpg", ThumbPrefix, strings.TrimPrefix(original, "/"), i)
}

func AllThumbKeys(original string) []string {
	keys := []string{ThumbKey(original)}
	for i := 0; i < VideoThumbCount; i++ {
		keys = append(keys, VideoThumbKey(original, i))
	}
	return keys
}

func GuardUserKey(key string) error {
	if IsTrashKey(key) || IsThumbKey(key) || IsReservedName(BaseName(strings.TrimSuffix(key, "/"))) {
		return ErrReserved
	}
	return nil
}

func NewTrashKey(original string) string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s%d-%s/%s", TrashPrefix, time.Now().UnixMilli(), hex.EncodeToString(b[:]), original)
}

func ParseTrashKey(key string) (batchID, original string, ok bool) {
	rest, ok := strings.CutPrefix(key, TrashPrefix)
	if !ok || rest == "" {
		return "", "", false
	}
	batchID, original, found := strings.Cut(rest, "/")
	if !found || batchID == "" || original == "" {
		return "", "", false
	}
	return batchID, original, true
}
