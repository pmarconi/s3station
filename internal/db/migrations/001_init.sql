CREATE TABLE IF NOT EXISTS folder_listings (
    prefix TEXT PRIMARY KEY,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cached_objects (
    prefix TEXT NOT NULL REFERENCES folder_listings(prefix) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    is_dir BOOLEAN NOT NULL DEFAULT FALSE,
    size BIGINT NOT NULL DEFAULT 0,
    etag TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    last_modified TIMESTAMPTZ,
    PRIMARY KEY (prefix, key)
);

CREATE INDEX IF NOT EXISTS idx_cached_objects_prefix ON cached_objects (prefix);
