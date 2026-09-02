use chrono::{DateTime, Utc};

#[derive(Clone, sqlx::FromRow)]
pub struct AboutPage {
    pub title: String,
    pub content_markdown: String,
    pub updated_at: DateTime<Utc>,
}
