package thumbs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"

	"station/internal/s3store"
	"station/internal/storage"
)

var ErrNoPDFRenderer = errors.New("no pdf renderer")

func pdfTool() string {
	for _, bin := range []string{"pdftoppm", "mutool", "magick", "convert"} {
		if _, err := exec.LookPath(bin); err == nil {
			return bin
		}
	}
	return ""
}

func ensurePDF(ctx context.Context, s3c *s3store.Client, key string) (string, error) {
	if pdfTool() == "" {
		return "", ErrNoPDFRenderer
	}
	thumbKey := storage.ThumbKey(key)
	exists, err := s3c.Exists(ctx, thumbKey)
	if err != nil {
		return "", err
	}
	if !exists {
		if err := generatePDF(ctx, s3c, key, thumbKey); err != nil {
			return "", err
		}
	}
	return s3c.PresignGet(ctx, thumbKey)
}

func generatePDF(ctx context.Context, s3c *s3store.Client, src, dest string) error {
	dir, err := os.MkdirTemp("", "station-pdfthumb-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	srcPath := filepath.Join(dir, "source.pdf")
	if err := downloadToFile(ctx, s3c, src, srcPath); err != nil {
		return fmt.Errorf("download source: %w", err)
	}
	pngPath := filepath.Join(dir, "page.png")
	if err := renderPDFPage(ctx, srcPath, pngPath); err != nil {
		return err
	}
	raw, err := os.ReadFile(pngPath)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("decode page: %w", err)
	}
	thumb := imaging.Fit(img, maxEdge, maxEdge, imaging.Lanczos)
	webp, err := encodeWebP(thumb)
	if err != nil {
		return err
	}
	return s3c.Put(ctx, dest, webp, "image/webp")
}

func downloadToFile(ctx context.Context, s3c *s3store.Client, key, dest string) error {
	body, _, err := s3c.Open(ctx, key)
	if err != nil {
		return err
	}
	defer body.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func renderPDFPage(ctx context.Context, src, dest string) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	tool := pdfTool()
	var cmd *exec.Cmd
	switch tool {
	case "pdftoppm":
		prefix := strings.TrimSuffix(dest, ".png")
		cmd = exec.CommandContext(ctx, "pdftoppm", "-png", "-f", "1", "-l", "1", "-singlefile", "-scale-to", fmt.Sprintf("%d", maxEdge), src, prefix)
	case "mutool":
		cmd = exec.CommandContext(ctx, "mutool", "draw", "-o", dest, "-F", "png", "-w", fmt.Sprintf("%d", maxEdge), src, "1")
	case "magick", "convert":
		cmd = exec.CommandContext(ctx, tool, "-density", "120", src+"[0]", "-background", "white", "-alpha", "remove", "-resize", fmt.Sprintf("%dx%d", maxEdge, maxEdge), dest)
	default:
		return ErrNoPDFRenderer
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s: %s", tool, msg)
	}
	return nil
}
