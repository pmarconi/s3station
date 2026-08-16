package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr     string
	Username     string
	Password     string
	CookieSecure bool

	DatabaseURL string
	RedisURL    string

	AWSAccessKey string
	AWSSecretKey string
	AWSRegion    string
	S3Bucket     string
	S3Endpoint   string
	S3PublicURL  string
	S3PathStyle  bool
	FilesPrefix  string

	MaxUploadBytes int64
	PresignTTL     time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:       env("HTTP_ADDR", ":8080"),
		Username:       os.Getenv("AUTH_USERNAME"),
		Password:       os.Getenv("AUTH_PASSWORD"),
		CookieSecure:   envBool("COOKIE_SECURE", false),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		AWSAccessKey:   os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretKey:   os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSRegion:      env("AWS_REGION", "us-east-1"),
		S3Bucket:       env("S3_BUCKET", "station"),
		S3Endpoint:     os.Getenv("S3_ENDPOINT"),
		S3PublicURL:    firstEnv("S3_PUBLIC_ENDPOINT", "S3_PUBLIC_URL"),
		S3PathStyle:    envBool("S3_USE_PATH_STYLE", false),
		FilesPrefix:    normalizeFilesPrefix(env("FILES_PREFIX", "files")),
		MaxUploadBytes: envInt64("MAX_UPLOAD_BYTES", 5*1024*1024*1024),
		PresignTTL:     envDuration("PRESIGN_TTL", 15*time.Minute),
	}

	var missing []string
	if cfg.Username == "" {
		missing = append(missing, "AUTH_USERNAME")
	}
	if cfg.Password == "" {
		missing = append(missing, "AUTH_PASSWORD")
	}
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.RedisURL == "" {
		missing = append(missing, "REDIS_URL")
	}
	if cfg.S3Bucket == "" {
		missing = append(missing, "S3_BUCKET")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func normalizeFilesPrefix(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	p = strings.Trim(p, "/")
	if p == "" {
		p = "files"
	}
	return p + "/"
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
