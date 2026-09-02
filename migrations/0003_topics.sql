CREATE TABLE topics (
    id UUID PRIMARY KEY,
    name VARCHAR(40) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO topics (id, name)
VALUES
    (MD5('개발')::UUID, '개발'),
    (MD5('운영')::UUID, '운영'),
    (MD5('기록')::UUID, '기록'),
    (MD5('메모')::UUID, '메모')
ON CONFLICT (name) DO NOTHING;

INSERT INTO topics (id, name)
SELECT MD5(category)::UUID, category
FROM (
    SELECT category FROM posts WHERE category <> ''
    UNION
    SELECT category FROM temp_posts WHERE category <> ''
) AS existing_categories
ON CONFLICT (name) DO NOTHING;

ALTER TABLE posts ADD COLUMN topic_id UUID REFERENCES topics(id) ON DELETE RESTRICT;
UPDATE posts SET topic_id = topics.id FROM topics WHERE topics.name = posts.category;
ALTER TABLE posts ALTER COLUMN topic_id SET NOT NULL;

ALTER TABLE temp_posts ADD COLUMN topic_id UUID REFERENCES topics(id) ON DELETE RESTRICT;
UPDATE temp_posts SET topic_id = topics.id FROM topics WHERE topics.name = temp_posts.category;

DROP INDEX IF EXISTS posts_category_idx;
ALTER TABLE posts DROP COLUMN category;
ALTER TABLE temp_posts DROP COLUMN category;

CREATE INDEX posts_topic_idx ON posts (topic_id, published_at DESC);
CREATE INDEX temp_posts_topic_idx ON temp_posts (topic_id);
