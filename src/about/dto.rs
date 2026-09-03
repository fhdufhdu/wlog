use serde::Deserialize;

#[derive(Deserialize)]
pub struct AboutForm {
    pub title: String,
    pub content_markdown: String,
    pub csrf_token: String,
}
