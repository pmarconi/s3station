package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	FilesPrefix  = "files/"
	BrowserRoute = "files"
	TrashPrefix  = ".station-trash/"
	ThumbPrefix  = ".station-thumbs/"
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

func SanitizeRelPath(name string) (string, error) {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Trim(name, "/")
	if name == "" {
		return "", ErrInvalidPath
	}
	parts := strings.Split(name, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part, err := SanitizeName(part)
		if err != nil {
			return "", err
		}
		if IsReservedName(part) {
			return "", ErrReserved
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, "/"), nil
}

func JoinKey(prefix, name string) string {
	return prefix + name
}

func InvalidMoveDest(src, destPrefix string) bool {
	if key, err := NormalizeObjectKey(src); err == nil {
		src = key
	}
	dest, err := NormalizePrefix(destPrefix)
	if err != nil {
		return true
	}
	if ParentPrefix(src) == dest {
		return true
	}
	if IsFolderKey(src) && (dest == src || strings.HasPrefix(dest, src)) {
		return true
	}
	return false
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

func IsRouteName(name string) bool {
	switch strings.ToLower(name) {
	case "trash", "settings", "login", "logout", "healthz", "static", "files", "thumbs":
		return true
	default:
		return false
	}
}

func PublicFolderPath(prefix string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return "/" + BrowserRoute + "/"
	}
	parts := strings.Split(prefix, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return "/" + BrowserRoute + "/" + strings.Join(parts, "/") + "/"
}

func PrefixFromRequestPath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	path = strings.Trim(path, "/")
	if path == "" || path != BrowserRoute && !strings.HasPrefix(path, BrowserRoute+"/") {
		return "", false
	}
	if path == BrowserRoute {
		return "", true
	}
	rest := strings.TrimPrefix(path, BrowserRoute+"/")
	if rest == "" {
		return "", true
	}
	parts := strings.Split(rest, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part, err := url.PathUnescape(part)
		if err != nil || part == "" || part == "." || part == ".." || strings.Contains(part, "..") {
			return "", false
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, "/") + "/", true
}

func IsTrashKey(key string) bool {
	return strings.HasPrefix(key, TrashPrefix)
}

func IsThumbKey(key string) bool {
	return strings.HasPrefix(key, ThumbPrefix)
}

func ThumbKey(original string) string {
	return ThumbPrefix + strings.TrimPrefix(original, "/") + ".webp"
}

const VideoThumbCount = 5

func VideoThumbKey(original string, i int) string {
	return fmt.Sprintf("%s%s.v%02d.webp", ThumbPrefix, strings.TrimPrefix(original, "/"), i)
}

func AllThumbKeys(original string) []string {
	base := ThumbPrefix + strings.TrimPrefix(original, "/")
	keys := []string{base + ".webp", base + ".jpg"}
	for i := 0; i < VideoThumbCount; i++ {
		keys = append(keys, fmt.Sprintf("%s.v%02d.webp", base, i), fmt.Sprintf("%s.v%02d.jpg", base, i))
	}
	return keys
}

var videoThumbSuffix = regexp.MustCompile(`\.v\d{2}\.(webp|jpg)$`)

func PublicThumbPath(thumbKey string) string {
	rest := strings.TrimPrefix(thumbKey, ThumbPrefix)
	parts := strings.Split(rest, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return "/thumbs/" + strings.Join(parts, "/")
}

func ThumbKeyFromPublic(rest string) (string, error) {
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "/"))
	if rest == "" || strings.Contains(rest, "..") || strings.ContainsRune(rest, 0) {
		return "", ErrInvalidPath
	}
	parts := strings.Split(rest, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part, err := url.PathUnescape(part)
		if err != nil || part == "" || part == "." || part == ".." || strings.Contains(part, "..") {
			return "", ErrInvalidPath
		}
		clean = append(clean, part)
	}
	key := ThumbPrefix + strings.Join(clean, "/")
	if !IsThumbKey(key) || (!strings.HasSuffix(key, ".webp") && !strings.HasSuffix(key, ".jpg")) {
		return "", ErrInvalidPath
	}
	return key, nil
}

func IsVideoThumb(thumbKey string) bool {
	return videoThumbSuffix.MatchString(thumbKey)
}

func SourceKeyFromThumb(thumbKey string) (string, bool) {
	rest, ok := strings.CutPrefix(thumbKey, ThumbPrefix)
	if !ok || rest == "" {
		return "", false
	}
	if loc := videoThumbSuffix.FindStringIndex(rest); loc != nil {
		return rest[:loc[0]], true
	}
	if strings.HasSuffix(rest, ".webp") {
		return strings.TrimSuffix(rest, ".webp"), true
	}
	if strings.HasSuffix(rest, ".jpg") {
		return strings.TrimSuffix(rest, ".jpg"), true
	}
	return "", false
}

func GuardUserKey(key string) error {
	if IsTrashKey(key) || IsThumbKey(key) || IsReservedName(BaseName(strings.TrimSuffix(key, "/"))) {
		return ErrReserved
	}
	return nil
}

func AbsUserKey(root, key string) string {
	if root == "" {
		root = FilesPrefix
	}
	if key == "" {
		return root
	}
	if IsTrashKey(key) || IsThumbKey(key) {
		return key
	}
	key = strings.TrimLeft(key, "/")
	if strings.HasPrefix(key, root) {
		return key
	}
	return root + key
}

func RelUserKey(root, key string) string {
	if root == "" {
		root = FilesPrefix
	}
	if IsTrashKey(key) || IsThumbKey(key) {
		return key
	}
	return strings.TrimPrefix(key, root)
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
