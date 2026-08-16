package filesvc

import (
	"testing"
	"time"

	"station/internal/models"
)

func TestApplyHeadKeepsSize(t *testing.T) {
	e := models.Entry{Key: "clip.mp4", Name: "clip.mp4"}
	headed := models.Entry{
		Size:         12 << 20,
		ContentType:  "video/mp4",
		ETag:         "abc",
		LastModified: time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC),
		Kind:         models.KindVideo,
	}
	applyHead(&e, headed)
	if e.Size != headed.Size {
		t.Fatalf("size = %d", e.Size)
	}
	if e.Kind != models.KindVideo {
		t.Fatalf("kind = %s", e.Kind)
	}
	if e.ContentType != "video/mp4" {
		t.Fatalf("type = %s", e.ContentType)
	}
}
