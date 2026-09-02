CREATE TABLE posts (
    id UUID PRIMARY KEY,
    title VARCHAR(120) NOT NULL,
    slug VARCHAR(160) NOT NULL UNIQUE,
    description VARCHAR(200) NOT NULL,
    content_markdown TEXT NOT NULL,
    category VARCHAR(40) NOT NULL,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT published_has_date CHECK (NOT published OR published_at IS NOT NULL)
);
CREATE INDEX posts_publication_order_idx ON posts (published_at DESC) WHERE published = TRUE;
CREATE INDEX posts_category_idx ON posts (category) WHERE published = TRUE;
