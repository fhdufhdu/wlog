use std::sync::Arc;

use axum_extra::extract::cookie::Key;
use sqlx::postgres::PgPoolOptions;
use tokio::net::TcpListener;
use tracing_subscriber::{EnvFilter, layer::SubscriberExt, util::SubscriberInitExt};
use wlog::{
    about::{repository::AboutRepository, service::AboutService},
    auth::Auth,
    build_router,
    config::Config,
    post::{repository::PostRepository, service::PostService},
    state::AppState,
    topic::{repository::TopicRepository, service::TopicService},
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    if let Err(error) = dotenvy::dotenv() {
        eprintln!(
            ".env 로드 실패: {error}; cwd={}",
            std::env::current_dir()?.display()
        );
    }
    tracing_subscriber::registry()
        .with(
            EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "wlog=info,tower_http=info".into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    let config = Arc::new(Config::from_env()?);
    tokio::fs::create_dir_all(&config.upload_dir).await?;
    let pool = PgPoolOptions::new()
        .max_connections(10)
        .connect(&config.database_url)
        .await?;
    sqlx::migrate!().run(&pool).await?;
    let post_service = PostService::new(PostRepository::new(pool.clone()));
    let topic_service = TopicService::new(TopicRepository::new(pool.clone()));
    let about_service = AboutService::new(AboutRepository::new(pool.clone()));
    let auth = Auth::new(
        config.admin_username.clone(),
        config.admin_password_hash.clone(),
    )?;

    let cookie_key = Key::derive_from(config.session_secret.as_bytes());
    let state = AppState {
        pool,
        post_service,
        topic_service,
        about_service,
        auth,
        cookie_key,
        config: config.clone(),
    };
    let listener = TcpListener::bind(config.bind_addr).await?;
    tracing::info!(address = %listener.local_addr()?, "wlog started");
    axum::serve(listener, build_router(state))
        .with_graceful_shutdown(shutdown_signal())
        .await?;
    Ok(())
}

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler")
    };
    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };
    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();
    tokio::select! { _ = ctrl_c => {}, _ = terminate => {} }
}
