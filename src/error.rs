use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
};

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("configuration error: {0}")]
    Config(String),
    #[error("database error: {0}")]
    Database(#[from] sqlx::Error),
    #[error("template error: {0}")]
    Template(#[from] askama::Error),
    #[error("I/O error: {0}")]
    Io(#[from] std::io::Error),
    #[error("validation error: {0}")]
    Validation(String),
    #[error("authentication failed")]
    Unauthorized,
    #[error("resource not found")]
    NotFound,
    #[error("conflict: {0}")]
    Conflict(String),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, message) = match &self {
            Self::Validation(message) => (StatusCode::BAD_REQUEST, message.as_str()),
            Self::Unauthorized => (StatusCode::UNAUTHORIZED, "로그인이 필요합니다."),
            Self::NotFound => (StatusCode::NOT_FOUND, "요청한 페이지를 찾을 수 없습니다."),
            Self::Conflict(message) => (StatusCode::CONFLICT, message.as_str()),
            _ => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "서버에서 요청을 처리하지 못했습니다.",
            ),
        };
        if status.is_server_error() {
            tracing::error!(error = ?self, "request failed");
        }
        (
            status,
            [(
                axum::http::header::CONTENT_TYPE,
                "text/plain; charset=utf-8",
            )],
            message.to_owned(),
        )
            .into_response()
    }
}
