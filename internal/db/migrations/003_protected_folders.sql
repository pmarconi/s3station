-- UI-only folder passwords. The S3 objects stay readable with bucket credentials.
CREATE TABLE IF NOT EXISTS protected_folders (
    prefix TEXT PRIMARY KEY,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
