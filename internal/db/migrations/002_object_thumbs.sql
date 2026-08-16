-- Multiple thumbs per object. A video can have several frames; an image
-- (or frame) can later have more than one size (card / grid / preview).
CREATE TABLE IF NOT EXISTS object_thumbs (
    source_key TEXT NOT NULL,
    thumb_key TEXT NOT NULL,
    variant TEXT NOT NULL DEFAULT 'card',
    position INT NOT NULL DEFAULT 0,
    width INT,
    height INT,
    content_type TEXT NOT NULL DEFAULT 'image/jpeg',
    etag TEXT NOT NULL DEFAULT '',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source_key, variant, position)
);

CREATE INDEX IF NOT EXISTS idx_object_thumbs_source ON object_thumbs (source_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_object_thumbs_key ON object_thumbs (thumb_key);
