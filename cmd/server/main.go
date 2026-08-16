package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"station/internal/cache"
	"station/internal/config"
	"station/internal/db"
	"station/internal/filesvc"
	"station/internal/locks"
	"station/internal/s3store"
	"station/internal/session"
	"station/internal/web"
)

func main() {
	_ = godotenv.Load()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	sessions, err := session.Connect(ctx, cfg.RedisURL, cfg.CookieSecure)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}

	s3c, err := s3store.New(ctx, cfg)
	if err != nil {
		log.Error("s3", "err", err)
		os.Exit(1)
	}

	h := web.New(cfg, sessions, filesvc.New(s3c, cache.New(pool), locks.New(pool), cfg), log)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr, "bucket", cfg.S3Bucket)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
