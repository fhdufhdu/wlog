use crate::{
    about::service::AboutService, auth::Auth, config::Config, post::service::PostService,
    topic::service::TopicService,
};
use axum::extract::FromRef;
use axum_extra::extract::cookie::Key;
use sqlx::PgPool;
use std::sync::Arc;

#[derive(Clone)]
pub struct AppState {
    pub pool: PgPool,
    pub post_service: PostService,
    pub topic_service: TopicService,
    pub about_service: AboutService,
    pub auth: Auth,
    pub cookie_key: Key,
    pub config: Arc<Config>,
}

impl FromRef<AppState> for Key {
    fn from_ref(state: &AppState) -> Self {
        state.cookie_key.clone()
    }
}
