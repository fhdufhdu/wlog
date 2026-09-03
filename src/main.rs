use std::{sync::Arc, time::Duration};

use axum_extra::extract::cookie::Key;
use sqlx::postgres::PgPoolOptions;
use tokio::net::TcpListener;
use tracing_subscriber::{EnvFilter, layer::SubscriberExt, util::SubscriberInitExt};
use wlog::{
    about::{repository::AboutRepository, service::AboutService},
    auth::Auth,
    build_router,
    config::Config,
    image::{repository::ImageRepository, service::ImageService},
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

    wlog::markdown::warm_up();
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
    let image_service = ImageService::new(
        ImageRepository::new(pool.clone()),
        config.upload_dir.clone(),
        config.image_orphan_grace_hours,
    );
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
        image_service: image_service.clone(),
        auth,
        cookie_key,
        config: config.clone(),
    };
    spawn_image_cleanup(image_service);
    let listener = TcpListener::bind(config.bind_addr).await?;
    tracing::info!(address = %listener.local_addr()?, "wlog started");
    axum::serve(listener, build_router(state))
        .with_graceful_shutdown(shutdown_signal())
        .await?;
    Ok(())
}

fn spawn_image_cleanup(image_service: ImageService) {
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(Duration::from_secs(6 * 60 * 60));
        loop {
            interval.tick().await;
            match image_service.cleanup_orphans().await {
                Ok(removed) if removed > 0 => {
                    tracing::info!(removed, "unused images cleaned up");
                }
                Ok(_) => {}
                Err(error) => tracing::error!(error = ?error, "unused image cleanup failed"),
            }
        }
    });
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
