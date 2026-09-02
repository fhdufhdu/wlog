CREATE TABLE images (
    id UUID PRIMARY KEY,
    post_id UUID REFERENCES posts(id) ON DELETE SET NULL,
    storage_name VARCHAR(80) NOT NULL UNIQUE,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(50) NOT NULL,
    byte_size BIGINT NOT NULL CHECK (byte_size > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX images_post_id_idx ON images (post_id);
CREATE INDEX images_orphan_cleanup_idx ON images (updated_at) WHERE post_id IS NULL;
