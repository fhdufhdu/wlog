use serde::Deserialize;

#[derive(Debug, Deserialize)]
pub struct TopicForm {
    pub name: String,
    pub csrf_token: String,
}
