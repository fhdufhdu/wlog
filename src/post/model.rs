use chrono::{DateTime, Utc};
use uuid::Uuid;

#[derive(Clone, Debug, sqlx::FromRow)]
pub struct Post {
    pub id: Uuid,
    pub title: String,
    pub slug: String,
    pub description: String,
    pub content_markdown: String,
    pub topic_id: Uuid,
    pub topic_name: String,
    pub published_at: DateTime<Utc>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Clone, Debug, sqlx::FromRow)]
pub struct PostLink {
    pub title: String,
    pub slug: String,
}

#[derive(Clone, Debug, sqlx::FromRow)]
pub struct TempPost {
    pub id: Uuid,
    pub post_id: Option<Uuid>,
    pub title: String,
    pub slug: String,
    pub description: String,
    pub description_manual: bool,
    pub content_markdown: String,
    pub topic_id: Option<Uuid>,
    pub topic_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}
