package thumbs

import (
	"strings"
	"testing"

	"station/internal/models"
)

func TestLocalURLsPDF(t *testing.T) {
	urls := LocalURLs("files/manual.pdf", models.KindPDF)
	if len(urls) != 1 {
		t.Fatalf("urls = %v", urls)
	}
	if !strings.HasSuffix(urls[0], "manual.pdf.webp") {
		t.Fatalf("thumb path = %q", urls[0])
	}
}
