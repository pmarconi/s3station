package views

import (
	"encoding/json"
	"fmt"
	"strings"

	"station/internal/filesvc"
	"station/internal/models"
)

func jsString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

func InitialSignals(prefix string) string {
	b, err := json.Marshal(map[string]any{
		"prefix":           prefix,
		"newFolderName":    "",
		"targetKey":        "",
		"targetName":       "",
		"targetBatch":      "",
		"view":             "grid",
		"query":            "",
		"refresh":          false,
		"_busy":            false,
		"_flash":           "",
		"_flashKind":       "ok",
		"_showNewFolder":   false,
		"_showDelete":      false,
		"_showPreview":     false,
		"_showEmptyTrash":  false,
		"_previewURL":      "",
		"_previewKind":     "",
		"_previewName":     "",
	})
	if err != nil {
		return "{}"
	}
	return string(b)
}

func NavigateExpr(prefix string) string {
	return fmt.Sprintf("$prefix = %s; $refresh = false; @get('/files')", jsString(prefix))
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
