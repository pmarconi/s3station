package cache

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"station/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Get(ctx context.Context, prefix string) (models.Listing, bool, error) {
	var fetchedAt time.Time
	err := s.pool.QueryRow(ctx, `SELECT fetched_at FROM folder_listings WHERE prefix = $1`, prefix).Scan(&fetchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Listing{}, false, nil
	}
	if err != nil {
		return models.Listing{}, false, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT key, name, is_dir, size, etag, content_type, last_modified
		FROM cached_objects
		WHERE prefix = $1
		ORDER BY is_dir DESC, name ASC
	`, prefix)
	if err != nil {
		return models.Listing{}, false, err
	}
	defer rows.Close()

	entries := make([]models.Entry, 0)
	for rows.Next() {
		var e models.Entry
		var lastMod *time.Time
		if err := rows.Scan(&e.Key, &e.Name, &e.IsDir, &e.Size, &e.ETag, &e.ContentType, &lastMod); err != nil {
			return models.Listing{}, false, err
		}
		if lastMod != nil {
			e.LastModified = *lastMod
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return models.Listing{}, false, err
	}

	return models.Listing{
		Prefix:    prefix,
		Entries:   entries,
		FromCache: true,
		FetchedAt: fetchedAt,
	}, true, nil
}

func (s *Store) Put(ctx context.Context, prefix string, entries []models.Entry) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM folder_listings WHERE prefix = $1`, prefix); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO folder_listings (prefix, fetched_at) VALUES ($1, NOW())`, prefix); err != nil {
		return err
	}
	for _, e := range entries {
		var lastMod any
		if !e.LastModified.IsZero() {
			lastMod = e.LastModified
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO cached_objects (prefix, key, name, is_dir, size, etag, content_type, last_modified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, prefix, e.Key, e.Name, e.IsDir, e.Size, e.ETag, e.ContentType, lastMod); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) DeletePrefix(ctx context.Context, prefix string) error {
	if prefix == "" {
		_, err := s.pool.Exec(ctx, `TRUNCATE cached_objects, folder_listings`)
		return err
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM folder_listings
		WHERE prefix = $1 OR prefix LIKE $2
	`, prefix, prefix+"%")
	return err
}

func (s *Store) UpsertObject(ctx context.Context, prefix string, entry models.Entry) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM folder_listings WHERE prefix = $1)`, prefix).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return nil
	}
	var lastMod any
	if !entry.LastModified.IsZero() {
		lastMod = entry.LastModified
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cached_objects (prefix, key, name, is_dir, size, etag, content_type, last_modified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (prefix, key) DO UPDATE SET
			name = EXCLUDED.name,
			is_dir = EXCLUDED.is_dir,
			size = CASE
				WHEN EXCLUDED.size > 0 THEN EXCLUDED.size
				WHEN EXCLUDED.is_dir THEN 0
				ELSE cached_objects.size
			END,
			etag = COALESCE(NULLIF(EXCLUDED.etag, ''), cached_objects.etag),
			content_type = COALESCE(NULLIF(EXCLUDED.content_type, ''), cached_objects.content_type),
			last_modified = COALESCE(EXCLUDED.last_modified, cached_objects.last_modified)
	`, prefix, entry.Key, entry.Name, entry.IsDir, entry.Size, entry.ETag, entry.ContentType, lastMod)
	return err
}

func (s *Store) RemoveObject(ctx context.Context, prefix, key string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM cached_objects WHERE prefix = $1 AND key = $2`, prefix, key)
	return err
}
