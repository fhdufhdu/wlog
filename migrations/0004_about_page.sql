CREATE TABLE about_page (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton = TRUE),
    title VARCHAR(120) NOT NULL DEFAULT '소개',
    content_markdown TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO about_page (singleton) VALUES (TRUE);
