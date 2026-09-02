use super::model::{Topic, TopicSummary};
use chrono::Utc;
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Clone)]
pub struct TopicRepository {
    pool: PgPool,
}

impl TopicRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn list(&self) -> Result<Vec<Topic>, sqlx::Error> {
        sqlx::query_as::<_, Topic>(
            "SELECT id, name, created_at, updated_at FROM topics ORDER BY name",
        )
        .fetch_all(&self.pool)
        .await
    }

    pub async fn list_with_counts(&self) -> Result<Vec<TopicSummary>, sqlx::Error> {
        sqlx::query_as::<_, TopicSummary>(
            "SELECT t.id, t.name,
                (SELECT COUNT(*) FROM posts p WHERE p.topic_id = t.id) AS post_count,
                (SELECT COUNT(*) FROM temp_posts tp WHERE tp.topic_id = t.id) AS temp_count
             FROM topics t ORDER BY t.name",
        )
        .fetch_all(&self.pool)
        .await
    }

    pub async fn create(&self, name: &str) -> Result<Topic, sqlx::Error> {
        sqlx::query_as::<_, Topic>(
            "INSERT INTO topics (id, name) VALUES ($1, $2)
             RETURNING id, name, created_at, updated_at",
        )
        .bind(Uuid::new_v4())
        .bind(name)
        .fetch_one(&self.pool)
        .await
    }

    pub async fn update(&self, id: Uuid, name: &str) -> Result<Option<Topic>, sqlx::Error> {
        sqlx::query_as::<_, Topic>(
            "UPDATE topics SET name = $2, updated_at = $3 WHERE id = $1
             RETURNING id, name, created_at, updated_at",
        )
        .bind(id)
        .bind(name)
        .bind(Utc::now())
        .fetch_optional(&self.pool)
        .await
    }

    pub async fn delete(&self, id: Uuid) -> Result<bool, sqlx::Error> {
        Ok(sqlx::query("DELETE FROM topics WHERE id = $1")
            .bind(id)
            .execute(&self.pool)
            .await?
            .rows_affected()
            == 1)
    }
}
