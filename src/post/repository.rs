use super::model::{Post, PostLink, PostListItem, PostNeighbors, TempPost, TempPostListItem};
use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Clone)]
pub struct PostRepository {
    pool: PgPool,
}

#[derive(sqlx::FromRow)]
struct PostNeighborsRow {
    previous_title: Option<String>,
    previous_slug: Option<String>,
    next_title: Option<String>,
    next_slug: Option<String>,
    topic_previous_title: Option<String>,
    topic_previous_slug: Option<String>,
    topic_next_title: Option<String>,
    topic_next_slug: Option<String>,
}

impl PostRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn list_public(
        &self,
        topic_id: Option<Uuid>,
    ) -> Result<Vec<PostListItem>, sqlx::Error> {
        sqlx::query_as::<_, PostListItem>(
            "SELECT p.id, p.title, p.slug, p.description,
                    t.name AS topic_name, p.published_at, p.updated_at
             FROM posts p JOIN topics t ON t.id = p.topic_id
             WHERE ($1::UUID IS NULL OR p.topic_id = $1)
             ORDER BY p.published_at DESC, p.id DESC LIMIT 100",
        )
        .bind(topic_id)
        .fetch_all(&self.pool)
        .await
    }

    pub async fn list_all(&self) -> Result<Vec<PostListItem>, sqlx::Error> {
        sqlx::query_as::<_, PostListItem>(
            "SELECT p.id, p.title, p.slug, p.description,
                    t.name AS topic_name, p.published_at, p.updated_at
             FROM posts p JOIN topics t ON t.id = p.topic_id
             ORDER BY p.published_at DESC, p.id DESC",
        )
        .fetch_all(&self.pool)
        .await
    }

    pub async fn list_unlinked_temp(&self) -> Result<Vec<TempPostListItem>, sqlx::Error> {
        sqlx::query_as::<_, TempPostListItem>(
            "SELECT tp.id, tp.title, tp.slug, tp.updated_at
             FROM temp_posts tp
             WHERE tp.post_id IS NULL ORDER BY tp.updated_at DESC",
        )
        .fetch_all(&self.pool)
        .await
    }

    pub async fn unrendered_posts(&self) -> Result<Vec<(Uuid, String)>, sqlx::Error> {
        sqlx::query_as("SELECT id, content_markdown FROM posts WHERE content_html = ''")
            .fetch_all(&self.pool)
            .await
    }

    pub async fn set_rendered_html(&self, id: Uuid, content_html: &str) -> Result<(), sqlx::Error> {
        sqlx::query("UPDATE posts SET content_html = $2 WHERE id = $1 AND content_html = ''")
            .bind(id)
            .bind(content_html)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    pub async fn find_public_slug(&self, slug: &str) -> Result<Option<Post>, sqlx::Error> {
        sqlx::query_as::<_, Post>(
            "SELECT p.id, p.title, p.slug, p.description, p.content_markdown, p.content_html,
                    p.topic_id, t.name AS topic_name, p.published_at, p.created_at, p.updated_at
             FROM posts p JOIN topics t ON t.id = p.topic_id WHERE p.slug = $1",
        )
        .bind(slug)
        .fetch_optional(&self.pool)
        .await
    }

    pub async fn find_id(&self, id: Uuid) -> Result<Option<Post>, sqlx::Error> {
        sqlx::query_as::<_, Post>(
            "SELECT p.id, p.title, p.slug, p.description, p.content_markdown, p.content_html,
                    p.topic_id, t.name AS topic_name, p.published_at, p.created_at, p.updated_at
             FROM posts p JOIN topics t ON t.id = p.topic_id WHERE p.id = $1",
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
    }

    pub async fn find_neighbors(
        &self,
        id: Uuid,
        published_at: DateTime<Utc>,
        topic_id: Uuid,
    ) -> Result<PostNeighbors, sqlx::Error> {
        let row = sqlx::query_as::<_, PostNeighborsRow>(
            "SELECT
                previous.title AS previous_title, previous.slug AS previous_slug,
                next.title AS next_title, next.slug AS next_slug,
                topic_previous.title AS topic_previous_title,
                topic_previous.slug AS topic_previous_slug,
                topic_next.title AS topic_next_title,
                topic_next.slug AS topic_next_slug
             FROM (VALUES ($1::TIMESTAMPTZ, $2::UUID, $3::UUID)) AS current(published_at, id, topic_id)
             LEFT JOIN LATERAL (
                SELECT title, slug FROM posts
                WHERE (published_at, id) < (current.published_at, current.id)
                ORDER BY published_at DESC, id DESC LIMIT 1
             ) previous ON TRUE
             LEFT JOIN LATERAL (
                SELECT title, slug FROM posts
                WHERE (published_at, id) > (current.published_at, current.id)
                ORDER BY published_at ASC, id ASC LIMIT 1
             ) next ON TRUE
             LEFT JOIN LATERAL (
                SELECT title, slug FROM posts
                WHERE topic_id = current.topic_id
                  AND (published_at, id) < (current.published_at, current.id)
                ORDER BY published_at DESC, id DESC LIMIT 1
             ) topic_previous ON TRUE
             LEFT JOIN LATERAL (
                SELECT title, slug FROM posts
                WHERE topic_id = current.topic_id
                  AND (published_at, id) > (current.published_at, current.id)
                ORDER BY published_at ASC, id ASC LIMIT 1
             ) topic_next ON TRUE",
        )
        .bind(published_at)
        .bind(id)
        .bind(topic_id)
        .fetch_one(&self.pool)
        .await?;
        Ok(PostNeighbors {
            previous: post_link(row.previous_title, row.previous_slug),
            next: post_link(row.next_title, row.next_slug),
            topic_previous: post_link(row.topic_previous_title, row.topic_previous_slug),
            topic_next: post_link(row.topic_next_title, row.topic_next_slug),
        })
    }

    pub async fn find_temp_id(&self, id: Uuid) -> Result<Option<TempPost>, sqlx::Error> {
        sqlx::query_as::<_, TempPost>(
            "SELECT tp.id, tp.post_id, tp.title, tp.slug, tp.description, tp.description_manual,
                    tp.content_markdown, tp.content_html, tp.topic_id, t.name AS topic_name, tp.created_at, tp.updated_at
             FROM temp_posts tp LEFT JOIN topics t ON t.id = tp.topic_id WHERE tp.id = $1",
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
    }

    pub async fn create_empty_temp(&self) -> Result<TempPost, sqlx::Error> {
        sqlx::query_as::<_, TempPost>(
            "INSERT INTO temp_posts (id) VALUES ($1)
             RETURNING id, post_id, title, slug, description, description_manual,
                       content_markdown, content_html, topic_id, NULL::VARCHAR AS topic_name, created_at, updated_at",
        )
        .bind(Uuid::new_v4())
        .fetch_one(&self.pool)
        .await
    }

    pub async fn temp_for_post(&self, post_id: Uuid) -> Result<Option<TempPost>, sqlx::Error> {
        sqlx::query_as::<_, TempPost>(
            "WITH upserted AS (
                INSERT INTO temp_posts (id, post_id, title, slug, description, description_manual, content_markdown, content_html, topic_id)
                SELECT $1, id, title, slug, description, TRUE, content_markdown, content_html, topic_id FROM posts WHERE id = $2
                ON CONFLICT (post_id) DO UPDATE SET post_id = EXCLUDED.post_id
                RETURNING id, post_id, title, slug, description, description_manual, content_markdown, content_html, topic_id, created_at, updated_at
             )
             SELECT u.id, u.post_id, u.title, u.slug, u.description, u.description_manual,
                    u.content_markdown, u.content_html, u.topic_id, t.name AS topic_name, u.created_at, u.updated_at
             FROM upserted u LEFT JOIN topics t ON t.id = u.topic_id",
        )
        .bind(Uuid::new_v4())
        .bind(post_id)
        .fetch_optional(&self.pool)
        .await
    }

    pub async fn update_temp(&self, temp: &TempPost) -> Result<TempPost, sqlx::Error> {
        sqlx::query_as::<_, TempPost>(
            "WITH updated AS (
                UPDATE temp_posts SET title=$2, slug=$3, description=$4, description_manual=$5,
                    content_markdown=$6, content_html=$7, topic_id=$8, updated_at=$9 WHERE id=$1
                RETURNING id, post_id, title, slug, description, description_manual, content_markdown, content_html, topic_id, created_at, updated_at
             )
             SELECT u.id, u.post_id, u.title, u.slug, u.description, u.description_manual,
                    u.content_markdown, u.content_html, u.topic_id, t.name AS topic_name, u.created_at, u.updated_at
             FROM updated u LEFT JOIN topics t ON t.id = u.topic_id",
        )
        .bind(temp.id)
        .bind(&temp.title)
        .bind(&temp.slug)
        .bind(&temp.description)
        .bind(temp.description_manual)
        .bind(&temp.content_markdown)
        .bind(&temp.content_html)
        .bind(temp.topic_id)
        .bind(temp.updated_at)
        .fetch_one(&self.pool)
        .await
    }

    pub async fn publish_temp(
        &self,
        temp: &TempPost,
        image_names: &[String],
    ) -> Result<Post, sqlx::Error> {
        let mut transaction = self.pool.begin().await?;
        let now = Utc::now();
        let post = if let Some(post_id) = temp.post_id {
            sqlx::query_as::<_, Post>(
                "WITH updated AS (
                    UPDATE posts SET title=$2, slug=$3, description=$4, content_markdown=$5,
                        content_html=$6, topic_id=$7, updated_at=$8 WHERE id=$1
                    RETURNING id, title, slug, description, content_markdown, content_html, topic_id, published_at, created_at, updated_at
                 )
                 SELECT u.id, u.title, u.slug, u.description, u.content_markdown, u.content_html,
                        u.topic_id, t.name AS topic_name, u.published_at, u.created_at, u.updated_at
                 FROM updated u JOIN topics t ON t.id = u.topic_id",
            )
            .bind(post_id)
            .bind(&temp.title)
            .bind(&temp.slug)
            .bind(&temp.description)
            .bind(&temp.content_markdown)
            .bind(&temp.content_html)
            .bind(temp.topic_id)
            .bind(now)
            .fetch_one(&mut *transaction)
            .await?
        } else {
            let post_id = Uuid::new_v4();
            sqlx::query_as::<_, Post>(
                "WITH inserted AS (
                    INSERT INTO posts (id, title, slug, description, content_markdown, content_html, topic_id, published_at, created_at, updated_at)
                    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8,$8)
                    RETURNING id, title, slug, description, content_markdown, content_html, topic_id, published_at, created_at, updated_at
                 )
                 SELECT i.id, i.title, i.slug, i.description, i.content_markdown, i.content_html,
                        i.topic_id, t.name AS topic_name, i.published_at, i.created_at, i.updated_at
                 FROM inserted i JOIN topics t ON t.id = i.topic_id",
            )
            .bind(post_id)
            .bind(&temp.title)
            .bind(&temp.slug)
            .bind(&temp.description)
            .bind(&temp.content_markdown)
            .bind(&temp.content_html)
            .bind(temp.topic_id)
            .bind(now)
            .fetch_one(&mut *transaction)
            .await?
        };
        sqlx::query(
            "UPDATE images SET post_id = NULL, updated_at = $3
             WHERE post_id = $1 AND NOT (storage_name = ANY($2::TEXT[]))",
        )
        .bind(post.id)
        .bind(image_names)
        .bind(now)
        .execute(&mut *transaction)
        .await?;
        sqlx::query(
            "UPDATE images SET post_id = $1, updated_at = $3
             WHERE post_id IS NULL AND storage_name = ANY($2::TEXT[])",
        )
        .bind(post.id)
        .bind(image_names)
        .bind(now)
        .execute(&mut *transaction)
        .await?;
        sqlx::query("DELETE FROM temp_posts WHERE id = $1")
            .bind(temp.id)
            .execute(&mut *transaction)
            .await?;
        transaction.commit().await?;
        Ok(post)
    }

    pub async fn delete(&self, id: Uuid) -> Result<bool, sqlx::Error> {
        let mut transaction = self.pool.begin().await?;
        let now = Utc::now();
        sqlx::query("UPDATE images SET post_id = NULL, updated_at = $2 WHERE post_id = $1")
            .bind(id)
            .bind(now)
            .execute(&mut *transaction)
            .await?;
        let deleted = sqlx::query("DELETE FROM posts WHERE id = $1")
            .bind(id)
            .execute(&mut *transaction)
            .await?
            .rows_affected()
            == 1;
        transaction.commit().await?;
        Ok(deleted)
    }

    pub async fn delete_temp(&self, id: Uuid) -> Result<bool, sqlx::Error> {
        Ok(sqlx::query("DELETE FROM temp_posts WHERE id = $1")
            .bind(id)
            .execute(&self.pool)
            .await?
            .rows_affected()
            == 1)
    }
}

fn post_link(title: Option<String>, slug: Option<String>) -> Option<PostLink> {
    title
        .zip(slug)
        .map(|(title, slug)| PostLink { title, slug })
}
