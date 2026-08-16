package locks

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"station/internal/storage"
)

var (
	ErrLocked         = errors.New("folder is locked")
	ErrWrongPassword  = errors.New("wrong folder password")
	ErrAlreadyProtected = errors.New("folder is already protected")
	ErrNotProtected   = errors.New("folder is not protected")
	ErrRoot           = errors.New("the depot root cannot be password protected")
	ErrPasswordShort  = errors.New("folder password must be at least 4 characters")
)

type ctxKey struct{}

func WithUnlocked(ctx context.Context, prefixes []string) context.Context {
	return context.WithValue(ctx, ctxKey{}, prefixes)
}

func Unlocked(ctx context.Context) []string {
	v, _ := ctx.Value(ctxKey{}).([]string)
	return v
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Protect(ctx context.Context, prefix, password string) error {
	prefix, err := normalizeFolder(prefix)
	if err != nil {
		return err
	}
	if prefix == "" {
		return ErrRoot
	}
	if len([]rune(strings.TrimSpace(password))) < 4 {
		return ErrPasswordShort
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO protected_folders (prefix, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (prefix) DO NOTHING
	`, prefix, string(hash))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyProtected
	}
	return nil
}

func (s *Store) Unprotect(ctx context.Context, prefix, password string) error {
	if err := s.Check(ctx, prefix, password); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM protected_folders WHERE prefix = $1`, prefix)
	return err
}

func (s *Store) Check(ctx context.Context, prefix, password string) error {
	prefix, err := normalizeFolder(prefix)
	if err != nil {
		return err
	}
	var hash string
	err = s.pool.QueryRow(ctx, `SELECT password_hash FROM protected_folders WHERE prefix = $1`, prefix).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotProtected
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return ErrWrongPassword
	}
	return nil
}

func (s *Store) IsProtected(ctx context.Context, prefix string) (bool, error) {
	prefix, err := normalizeFolder(prefix)
	if err != nil {
		return false, err
	}
	if prefix == "" {
		return false, nil
	}
	var exists bool
	err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM protected_folders WHERE prefix = $1)`, prefix).Scan(&exists)
	return exists, err
}

// Covering returns protected prefixes that enclose prefix, shortest first
// (outermost folder first).
func (s *Store) Covering(ctx context.Context, prefix string) ([]string, error) {
	prefix, err := normalizeFolder(prefix)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT prefix
		FROM protected_folders
		WHERE $1 = prefix OR $1 LIKE prefix || '%'
		ORDER BY length(prefix) ASC
	`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Nested returns protected prefixes strictly inside prefix.
func (s *Store) Nested(ctx context.Context, prefix string) ([]string, error) {
	prefix, err := normalizeFolder(prefix)
	if err != nil {
		return nil, err
	}
	var rows pgx.Rows
	if prefix == "" {
		rows, err = s.pool.Query(ctx, `SELECT prefix FROM protected_folders`)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT prefix
			FROM protected_folders
			WHERE prefix LIKE $1 || '%' AND prefix <> $1
		`, prefix)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Relocate(ctx context.Context, oldPrefix, newPrefix string) error {
	oldPrefix, err := normalizeFolder(oldPrefix)
	if err != nil {
		return err
	}
	newPrefix, err = normalizeFolder(newPrefix)
	if err != nil {
		return err
	}
	if oldPrefix == "" || newPrefix == "" || oldPrefix == newPrefix {
		return nil
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE protected_folders
		SET prefix = $2 || substr(prefix, length($1) + 1)
		WHERE prefix = $1 OR prefix LIKE $1 || '%'
	`, oldPrefix, newPrefix)
	return err
}

func (s *Store) RemoveTree(ctx context.Context, prefix string) error {
	prefix, err := normalizeFolder(prefix)
	if err != nil {
		return err
	}
	if prefix == "" {
		return nil
	}
	_, err = s.pool.Exec(ctx, `
		DELETE FROM protected_folders
		WHERE prefix = $1 OR prefix LIKE $1 || '%'
	`, prefix)
	return err
}

func normalizeFolder(prefix string) (string, error) {
	if prefix == "" {
		return "", nil
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return storage.NormalizePrefix(prefix)
}
