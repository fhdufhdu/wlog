use chrono::Utc;
use sqlx::PgPool;

use super::model::AboutPage;

#[derive(Clone)]
pub struct AboutRepository {
    pool: PgPool,
}

impl AboutRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn get(&self) -> Result<AboutPage, sqlx::Error> {
        sqlx::query_as::<_, AboutPage>(
            "SELECT title, content_markdown, updated_at
             FROM about_page WHERE singleton = TRUE",
        )
        .fetch_one(&self.pool)
        .await
    }

    pub async fn update(
        &self,
        title: &str,
        content_markdown: &str,
    ) -> Result<AboutPage, sqlx::Error> {
        sqlx::query_as::<_, AboutPage>(
            "UPDATE about_page
             SET title = $1, content_markdown = $2, updated_at = $3
             WHERE singleton = TRUE
             RETURNING title, content_markdown, updated_at",
        )
        .bind(title)
        .bind(content_markdown)
        .bind(Utc::now())
        .fetch_one(&self.pool)
        .await
    }
}
