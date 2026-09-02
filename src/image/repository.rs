use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

use super::model::OrphanImage;

#[derive(Clone)]
pub struct ImageRepository {
    pool: PgPool,
}

impl ImageRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn create(
        &self,
        id: Uuid,
        storage_name: &str,
        original_name: &str,
        mime_type: &str,
        byte_size: i64,
    ) -> Result<(), sqlx::Error> {
        sqlx::query(
            "INSERT INTO images (id, storage_name, original_name, mime_type, byte_size)
             VALUES ($1, $2, $3, $4, $5)",
        )
        .bind(id)
        .bind(storage_name)
        .bind(original_name)
        .bind(mime_type)
        .bind(byte_size)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn orphan_candidates(
        &self,
        before: DateTime<Utc>,
    ) -> Result<Vec<OrphanImage>, sqlx::Error> {
        sqlx::query_as::<_, OrphanImage>(
            "SELECT id, storage_name FROM images
             WHERE post_id IS NULL AND updated_at < $1",
        )
        .bind(before)
        .fetch_all(&self.pool)
        .await
    }

    pub async fn all_markdown(&self) -> Result<Vec<String>, sqlx::Error> {
        sqlx::query_scalar::<_, String>(
            "SELECT content_markdown FROM posts
             UNION ALL SELECT content_markdown FROM temp_posts
             UNION ALL SELECT content_markdown FROM about_page",
        )
        .fetch_all(&self.pool)
        .await
    }

    pub async fn delete_if_orphan(&self, id: Uuid) -> Result<bool, sqlx::Error> {
        Ok(
            sqlx::query("DELETE FROM images WHERE id = $1 AND post_id IS NULL")
                .bind(id)
                .execute(&self.pool)
                .await?
                .rows_affected()
                == 1,
        )
    }
}
