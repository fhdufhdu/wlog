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
            "SELECT title, content_markdown, content_html, updated_at
             FROM about_page WHERE singleton = TRUE",
        )
        .fetch_one(&self.pool)
        .await
    }

    pub async fn update(
        &self,
        title: &str,
        content_markdown: &str,
        content_html: &str,
    ) -> Result<AboutPage, sqlx::Error> {
        sqlx::query_as::<_, AboutPage>(
            "UPDATE about_page
             SET title = $1, content_markdown = $2, content_html = $3, updated_at = $4
             WHERE singleton = TRUE
             RETURNING title, content_markdown, content_html, updated_at",
        )
        .bind(title)
        .bind(content_markdown)
        .bind(content_html)
        .bind(Utc::now())
        .fetch_one(&self.pool)
        .await
    }

    pub async fn set_rendered_html_if_empty(&self, content_html: &str) -> Result<(), sqlx::Error> {
        sqlx::query(
            "UPDATE about_page SET content_html = $1
             WHERE singleton = TRUE AND content_html = ''",
        )
        .bind(content_html)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
