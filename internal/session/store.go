package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	cookieName = "station_session"
	keyPrefix  = "session:"
	loginKey   = "login:"
	ttl        = 7 * 24 * time.Hour
)

type Store struct {
	rdb    *redis.Client
	secure bool
}

func Connect(ctx context.Context, redisURL string, secure bool) (*Store, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opt)

	deadline := time.Now().Add(45 * time.Second)
	for {
		err = rdb.Ping(ctx).Err()
		if err == nil {
			return &Store{rdb: rdb, secure: secure}, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("connect redis: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (s *Store) Create(ctx context.Context, username string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := s.rdb.Set(ctx, keyPrefix+token, username, ttl).Err(); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) Username(ctx context.Context, token string) (string, bool, error) {
	if token == "" {
		return "", false, nil
	}
	user, err := s.rdb.Get(ctx, keyPrefix+token).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return user, true, nil
}

func (s *Store) Delete(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.rdb.Del(ctx, keyPrefix+token).Err()
}

func (s *Store) AllowLogin(ctx context.Context, ip string) (bool, error) {
	key := loginKey + ip
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		_ = s.rdb.Expire(ctx, key, 15*time.Minute).Err()
	}
	return n <= 20, nil
}

func (s *Store) SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
}

func (s *Store) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
}

func TokenFromRequest(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(c.Value)
}

func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	ip := r.RemoteAddr
	if i := strings.LastIndex(ip, ":"); i >= 0 {
		return ip[:i]
	}
	return ip
}
