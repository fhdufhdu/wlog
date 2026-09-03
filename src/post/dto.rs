use serde::Deserialize;
use uuid::Uuid;

#[derive(Clone, Debug, Deserialize)]
pub struct PostForm {
    pub title: String,
    pub slug: String,
    pub description: String,
    #[serde(default)]
    pub description_manual: bool,
    pub topic_id: String,
    pub content_markdown: String,
    pub csrf_token: String,
}

#[derive(Debug, Deserialize, Default)]
pub struct IndexQuery {
    pub topic: Option<Uuid>,
    pub category: Option<String>,
}
