package views

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"station/internal/filesvc"
	"station/internal/models"
	"station/internal/storage"
)

func jsString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

func InitialSignals(listing models.Listing) string {
	b, err := json.Marshal(map[string]any{
		"prefix":           listing.Prefix,
		"newFolderName":    "",
		"targetKey":        "",
		"targetName":       "",
		"targetBatch":      "",
		"folderPassword":   "",
		"renameTo":         "",
		"view":             "grid",
		"query":            "",
		"refresh":          false,
		"nav":              false,
		"_busy":            false,
		"_flash":           "",
		"_flashKind":       "ok",
		"_showNewFolder":   false,
		"_showRename":      false,
		"_showDelete":      false,
		"_showPreview":     false,
		"_showEmptyTrash":  false,
		"_showPurgeThumbs": false,
		"_showUnlock":      false,
		"_showProtect":     false,
		"_showUnprotect":   false,
		"_locked":          listing.Locked,
		"_protected":       listing.Protected,
		"_previewURL":      "",
		"_previewKind":     "",
		"_previewName":     "",
	})
	if err != nil {
		return "{}"
	}
	return string(b)
}

func AvatarInitials(user string) string {
	user = strings.TrimSpace(user)
	if user == "" {
		return "?"
	}
	parts := strings.FieldsFunc(user, func(r rune) bool {
		return r == ' ' || r == '.' || r == '_' || r == '-' || r == '@'
	})
	if len(parts) == 0 {
		return strings.ToUpper(string([]rune(user)[0]))
	}
	if len(parts) == 1 {
		r := []rune(strings.ToUpper(parts[0]))
		if len(r) > 1 {
			return string(r[:2])
		}
		return string(r)
	}
	return strings.ToUpper(string([]rune(parts[0])[0]) + string([]rune(parts[1])[0]))
}

func FolderHref(prefix string) string {
	return storage.PublicFolderPath(prefix)
}

func NavigateExpr(prefix string) string {
	return fmt.Sprintf("if(evt.metaKey||evt.ctrlKey||evt.shiftKey||evt.altKey) return; evt.preventDefault(); $_busy = true; $prefix = %s; $refresh = false; $nav = true; @get('/files')", jsString(prefix))
}

func UnlockExpr(e models.Entry) string {
	return fmt.Sprintf("evt.stopPropagation(); $targetKey = %s; $targetName = %s; $folderPassword = ''; $_showUnlock = true", jsString(e.Key), jsString(e.Name))
}

func FolderClickExpr(e models.Entry) string {
	if e.Locked {
		return UnlockExpr(e)
	}
	return NavigateExpr(e.Key)
}

func FilterExpr(name string) string {
	return fmt.Sprintf("$query === '' || %s.toLowerCase().includes($query.toLowerCase())", jsString(name))
}

func GallerySrc(e models.Entry) string {
	if e.PreviewURL != "" {
		return e.PreviewURL
	}
	return e.DownloadURL
}

func PreviewExpr(e models.Entry) string {
	kind := string(e.Kind)
	url := e.PreviewURL
	if url == "" {
		url = e.DownloadURL
	}
	return fmt.Sprintf("$_previewURL = %s; $_previewKind = %s; $_previewName = %s; $_showPreview = true", jsString(url), jsString(kind), jsString(e.Name))
}

func ArchiveHref(prefix string) string {
	return "/folders/archive?prefix=" + url.QueryEscape(prefix)
}

func RenameExpr(e models.Entry) string {
	return fmt.Sprintf("evt.stopPropagation(); $targetKey = %s; $targetName = %s; $renameTo = %s; $_showRename = true", jsString(e.Key), jsString(e.Name), jsString(e.Name))
}

func TrashExpr(e models.Entry) string {
	return fmt.Sprintf("evt.stopPropagation(); $targetKey = %s; $targetName = %s; $_showDelete = true", jsString(e.Key), jsString(e.Name))
}

func PurgeExpr(item models.TrashItem) string {
	return fmt.Sprintf("$targetBatch = %s; $targetName = %s; $_showDelete = true", jsString(item.BatchID), jsString(item.Name))
}

func RestoreExpr(item models.TrashItem) string {
	return fmt.Sprintf("$targetBatch = %s; @post('/trash/restore')", jsString(item.BatchID))
}

func CacheLabel(listing models.Listing) string {
	if listing.FromCache {
		if rel := filesvc.RelTime(listing.FetchedAt); rel != "" {
			return "Postgres cache · " + rel
		}
		return "Postgres cache"
	}
	return "Read from the bucket"
}

func ItemMeta(e models.Entry) string {
	if e.IsDir {
		if e.Locked {
			return "Locked folder"
		}
		if e.Protected {
			return "Protected folder"
		}
		return "Folder"
	}
	parts := []string{filesvc.FormatBytes(e.Size)}
	if !e.LastModified.IsZero() {
		parts = append(parts, filesvc.RelTime(e.LastModified))
	}
	return strings.Join(parts, " · ")
}

func ItemCount(entries []models.Entry) string {
	n := len(entries)
	if n == 1 {
		return "1 item"
	}
	return fmt.Sprintf("%d items", n)
}

func TotalSize(entries []models.Entry) int64 {
	var n int64
	for _, e := range entries {
		n += e.Size
	}
	return n
}

func TrashAsEntry(item models.TrashItem) models.Entry {
	return models.Entry{
		Name:         item.Name,
		IsDir:        item.IsDir,
		Kind:         item.Kind,
		Size:         item.Size,
		LastModified: item.TrashedAt,
		PreviewURL:   item.PreviewURL,
		ThumbURL:     item.ThumbURL,
		ThumbURLs:    item.ThumbURLs,
	}
}

func TrashMeta(item models.TrashItem) string {
	parts := []string{filesvc.FormatBytes(item.Size)}
	if item.Count > 1 {
		parts = append(parts, fmt.Sprintf("%d objects", item.Count))
	}
	if item.OriginalKey != "" {
		parts = append(parts, item.OriginalKey)
	}
	if !item.TrashedAt.IsZero() {
		parts = append(parts, filesvc.RelTime(item.TrashedAt))
	}
	return strings.Join(parts, " · ")
}
