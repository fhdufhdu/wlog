DROP INDEX IF EXISTS posts_publication_order_idx;
DROP INDEX IF EXISTS posts_topic_idx;

CREATE INDEX posts_publication_order_idx
ON posts (published_at DESC, id DESC);

CREATE INDEX posts_topic_idx
ON posts (topic_id, published_at DESC, id DESC);
