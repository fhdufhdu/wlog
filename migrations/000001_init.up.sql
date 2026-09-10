CREATE TABLE topics (
    id UUID PRIMARY KEY,
    name VARCHAR(40) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE posts (
    id UUID PRIMARY KEY,
    title VARCHAR(120) NOT NULL,
    slug VARCHAR(160) NOT NULL UNIQUE,
    description VARCHAR(200) NOT NULL,
    content_markdown TEXT NOT NULL,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE temp_posts (
    id UUID PRIMARY KEY,
    post_id UUID UNIQUE REFERENCES posts(id) ON DELETE CASCADE,
    title VARCHAR(120) NOT NULL DEFAULT '',
    slug VARCHAR(160) NOT NULL DEFAULT '',
    description VARCHAR(200) NOT NULL DEFAULT '',
    description_manual BOOLEAN NOT NULL DEFAULT FALSE,
    content_markdown TEXT NOT NULL DEFAULT '',
    topic_id UUID REFERENCES topics(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE about_page (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton = TRUE),
    title VARCHAR(120) NOT NULL DEFAULT '소개',
    content_markdown TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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

INSERT INTO topics (id, name) VALUES
    (MD5('개발')::UUID, '개발'),
    (MD5('운영')::UUID, '운영'),
    (MD5('기록')::UUID, '기록'),
    (MD5('메모')::UUID, '메모');
INSERT INTO about_page (singleton) VALUES (TRUE);

CREATE INDEX posts_publication_order_idx ON posts (published_at DESC, id DESC);
CREATE INDEX posts_topic_idx ON posts (topic_id, published_at DESC, id DESC);
CREATE INDEX temp_posts_updated_idx ON temp_posts (updated_at DESC);
CREATE INDEX temp_posts_topic_idx ON temp_posts (topic_id);
CREATE INDEX images_post_id_idx ON images (post_id);
CREATE INDEX images_orphan_cleanup_idx ON images (updated_at) WHERE post_id IS NULL;
