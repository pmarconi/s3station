package storage

import (
	"path/filepath"
	"strings"

	"station/internal/models"
)

func ResolveKind(name, contentType string, isDir bool) models.Kind {
	if isDir {
		return models.KindFolder
	}
	if k := KindFromContentType(contentType); k != models.KindFile {
		return k
	}
	return KindFromName(name, false)
}

func KindFromContentType(contentType string) models.Kind {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch {
	case strings.HasPrefix(ct, "image/"):
		return models.KindImage
	case strings.HasPrefix(ct, "audio/"):
		return models.KindAudio
	case strings.HasPrefix(ct, "video/"):
		return models.KindVideo
	case ct == "application/pdf":
		return models.KindPDF
	case strings.HasPrefix(ct, "text/"):
		return models.KindText
	case ct == "application/zip" || ct == "application/gzip" || ct == "application/x-tar":
		return models.KindArchive
	default:
		return models.KindFile
	}
}

func KindFromName(name string, isDir bool) models.Kind {
	if isDir {
		return models.KindFolder
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".svg", ".heic", ".bmp":
		return models.KindImage
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a", ".aiff":
		return models.KindAudio
	case ".mp4", ".webm", ".mov", ".mkv", ".m4v":
		return models.KindVideo
	case ".pdf":
		return models.KindPDF
	case ".txt", ".md", ".csv", ".json", ".xml", ".log":
		return models.KindText
	case ".zip", ".tar", ".gz", ".tgz", ".7z", ".rar":
		return models.KindArchive
	default:
		return models.KindFile
	}
}

func ContentTypeFromName(name, fallback string) string {
	if fallback != "" && fallback != "application/octet-stream" && fallback != "binary/octet-stream" {
		return fallback
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".m4a":
		return "audio/mp4"
	case ".ogg":
		return "audio/ogg"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	default:
		if fallback != "" {
			return fallback
		}
		return "application/octet-stream"
	}
}
