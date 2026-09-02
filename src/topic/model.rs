use chrono::{DateTime, Utc};
use uuid::Uuid;

#[derive(Clone, Debug, sqlx::FromRow)]
pub struct Topic {
    pub id: Uuid,
    pub name: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Clone, Debug, sqlx::FromRow)]
pub struct TopicSummary {
    pub id: Uuid,
    pub name: String,
    pub post_count: i64,
    pub temp_count: i64,
}
