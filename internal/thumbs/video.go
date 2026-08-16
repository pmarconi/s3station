package thumbs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"station/internal/s3store"
	"station/internal/storage"
)

var ErrNoFFmpeg = errors.New("ffmpeg is not installed")

func ensureVideo(ctx context.Context, s3c *s3store.Client, key string) ([]string, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, ErrNoFFmpeg
	}

	urls := make([]string, 0, storage.VideoThumbCount)
	missing := false
	for i := 0; i < storage.VideoThumbCount; i++ {
		thumbKey := storage.VideoThumbKey(key, i)
		exists, err := s3c.Exists(ctx, thumbKey)
		if err != nil {
			return nil, err
		}
		if !exists {
			missing = true
			break
		}
	}
	if missing {
		if err := generateVideo(ctx, s3c, key); err != nil {
			return nil, err
		}
	}
	for i := 0; i < storage.VideoThumbCount; i++ {
		url, err := s3c.PresignGet(ctx, storage.VideoThumbKey(key, i))
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func generateVideo(ctx context.Context, s3c *s3store.Client, key string) error {
	raw, _, err := s3c.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("download source: %w", err)
	}
	dir, err := os.MkdirTemp("", "station-vthumbs-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	ext := filepath.Ext(key)
	if ext == "" {
		ext = ".mp4"
	}
	srcPath := filepath.Join(dir, "source"+ext)
	if err := os.WriteFile(srcPath, raw, 0o600); err != nil {
		return err
	}

	duration := probeDuration(ctx, srcPath)
	stamps := frameTimes(duration, storage.VideoThumbCount)
	for i, stamp := range stamps {
		dest := filepath.Join(dir, fmt.Sprintf("%02d.png", i))
		if err := extractFrame(ctx, srcPath, stamp, dest); err != nil {
			return fmt.Errorf("frame %d: %w", i, err)
		}
		raw, err := os.ReadFile(dest)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			return fmt.Errorf("frame %d was empty", i)
		}
		frame, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return fmt.Errorf("frame %d decode: %w", i, err)
		}
		webp, err := encodeWebP(frame)
		if err != nil {
			return fmt.Errorf("frame %d webp: %w", i, err)
		}
		if err := s3c.Put(ctx, storage.VideoThumbKey(key, i), webp, "image/webp"); err != nil {
			return err
		}
	}
	return nil
}

func probeDuration(ctx context.Context, srcURL string) float64 {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		srcURL,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || sec <= 0 {
		return 0
	}
	return sec
}

func frameTimes(duration float64, n int) []float64 {
	out := make([]float64, n)
	if n <= 0 {
		return out
	}
	if duration <= 0 {
		for i := 0; i < n; i++ {
			out[i] = 0.2 + rand.Float64()*8
		}
		return out
	}

	margin := math.Max(0.35, duration*0.08)
	usable := duration - 2*margin
	if usable < 0.25 {
		margin = duration * 0.05
		usable = duration - 2*margin
	}
	if usable <= 0 {
		for i := 0; i < n; i++ {
			out[i] = duration / 2
		}
		return out
	}

	// One random pick in each slice of the video, then shuffle so the
	// card poster (first frame) is not always from the opening seconds.
	bucket := usable / float64(n)
	for i := 0; i < n; i++ {
		out[i] = margin + float64(i)*bucket + rand.Float64()*bucket
	}
	rand.Shuffle(n, func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

func extractFrame(ctx context.Context, srcURL string, seconds float64, dest string) error {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-ss", strconv.FormatFloat(seconds, 'f', 3, 64),
		"-i", srcURL,
		"-frames:v", "1",
		"-vf", fmt.Sprintf("scale='min(%d,iw)':-2", maxEdge),
		"-q:v", "4",
		"-y",
		dest,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
