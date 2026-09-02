CREATE TABLE temp_posts (
    id UUID PRIMARY KEY,
    post_id UUID UNIQUE REFERENCES posts(id) ON DELETE CASCADE,
    title VARCHAR(120) NOT NULL DEFAULT '',
    slug VARCHAR(160) NOT NULL DEFAULT '',
    description VARCHAR(200) NOT NULL DEFAULT '',
    description_manual BOOLEAN NOT NULL DEFAULT FALSE,
    content_markdown TEXT NOT NULL DEFAULT '',
    category VARCHAR(40) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO temp_posts (
    id, title, slug, description, description_manual, content_markdown, category, created_at, updated_at
)
SELECT id, title, slug, description, TRUE, content_markdown, category, created_at, updated_at
FROM posts
WHERE published = FALSE;

DELETE FROM posts WHERE published = FALSE;
ALTER TABLE posts DROP CONSTRAINT published_has_date;
ALTER TABLE posts DROP COLUMN published;
ALTER TABLE posts ALTER COLUMN published_at SET NOT NULL;

CREATE INDEX posts_publication_order_idx ON posts (published_at DESC);
CREATE INDEX posts_category_idx ON posts (category);
CREATE INDEX temp_posts_updated_idx ON temp_posts (updated_at DESC);
