pub mod about;
pub mod auth;
pub mod config;
pub mod error;
pub mod handlers;
pub mod image;
pub mod markdown;
pub mod post;
pub mod state;
pub mod topic;

use axum::{
    Router,
    extract::DefaultBodyLimit,
    http::{HeaderName, HeaderValue, header},
    middleware,
    routing::{get, post},
};
use state::AppState;
use tower_http::{
    compression::CompressionLayer,
    request_id::{MakeRequestUuid, PropagateRequestIdLayer, SetRequestIdLayer},
    services::{ServeDir, ServeFile},
    trace::TraceLayer,
};

pub fn build_router(state: AppState) -> Router {
    let upload_limit = state.config.max_upload_bytes + 65_536;
    let uploads = state.config.upload_dir.clone();
    let request_id = HeaderName::from_static("x-request-id");
    let admin = Router::new()
        .route("/admin/logout", post(handlers::logout))
        .route("/admin", get(handlers::admin_index))
        .route(
            "/admin/topics",
            get(handlers::topics_page).post(handlers::create_topic),
        )
        .route("/admin/topics/{id}", post(handlers::update_topic))
        .route("/admin/topics/{id}/delete", post(handlers::delete_topic))
        .route(
            "/admin/about",
            get(handlers::about_editor).post(handlers::save_about),
        )
        .route("/admin/posts/new", get(handlers::new_post_page))
        .route("/admin/posts/{id}/edit", get(handlers::edit_post_page))
        .route("/admin/posts/{id}/delete", post(handlers::delete_post))
        .route("/admin/temp-posts/{id}/edit", get(handlers::edit_temp_page))
        .route(
            "/admin/temp-posts/{id}/save",
            post(handlers::save_temp_post),
        )
        .route(
            "/admin/temp-posts/{id}/autosave",
            post(handlers::autosave_temp_post),
        )
        .route(
            "/admin/temp-posts/{id}/publish",
            post(handlers::publish_temp_post),
        )
        .route(
            "/admin/temp-posts/{id}/delete",
            post(handlers::delete_temp_post),
        )
        .route("/admin/preview", post(handlers::preview_markdown))
        .route(
            "/admin/uploads",
            post(handlers::upload_image).layer(DefaultBodyLimit::max(upload_limit)),
        )
        .layer(middleware::from_fn_with_state(
            state.clone(),
            handlers::require_admin,
        ));

    Router::new()
        .route("/", get(handlers::index))
        .route("/about", get(handlers::about))
        .route("/posts/{slug}", get(handlers::show_post))
        .route(
            "/admin/login",
            get(handlers::login_page).post(handlers::login),
        )
        .merge(admin)
        .route("/sitemap.xml", get(handlers::sitemap))
        .route("/robots.txt", get(handlers::robots))
        .route("/health/live", get(handlers::health_live))
        .route("/health/ready", get(handlers::health_ready))
        .route_service("/styles.css", ServeFile::new("styles.css"))
        .route_service("/admin.js", ServeFile::new("admin.js"))
        .route_service("/theme.js", ServeFile::new("theme.js"))
        .route_service("/mermaid.js", ServeFile::new("mermaid.js"))
        .route_service("/math.js", ServeFile::new("math.js"))
        .nest_service("/assets", ServeDir::new("assets"))
        .nest_service(
            "/uploads",
            ServeDir::new(uploads).append_index_html_on_directories(false),
        )
        .layer(PropagateRequestIdLayer::new(request_id.clone()))
        .layer(CompressionLayer::new())
        .layer(TraceLayer::new_for_http())
        .layer(SetRequestIdLayer::new(request_id, MakeRequestUuid))
        .layer(
            tower_http::set_header::SetResponseHeaderLayer::if_not_present(
                header::X_CONTENT_TYPE_OPTIONS,
                HeaderValue::from_static("nosniff"),
            ),
        )
        .with_state(state)
}
