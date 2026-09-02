use crate::error::AppError;
use std::{env, net::SocketAddr, path::PathBuf};
use url::Url;

#[derive(Clone, Debug)]
pub struct Config {
    pub bind_addr: SocketAddr,
    pub database_url: String,
    pub public_base_url: String,
    pub site_name: String,
    pub site_description: String,
    pub default_social_image: String,
    pub admin_username: String,
    pub admin_password_hash: String,
    pub session_secret: String,
    pub secure_cookie: bool,
    pub upload_dir: PathBuf,
    pub max_upload_bytes: usize,
    pub image_orphan_grace_hours: i64,
}

impl Config {
    pub fn from_env() -> Result<Self, AppError> {
        let bind_addr = value("BIND_ADDR", "127.0.0.1:3000")
            .parse()
            .map_err(|e| AppError::Config(format!("BIND_ADDR: {e}")))?;
        let public_base_url = env::var("PUBLIC_BASE_URL")
            .map_err(|_| AppError::Config("PUBLIC_BASE_URL이 필요합니다".into()))?;
        let url = Url::parse(&public_base_url)
            .map_err(|e| AppError::Config(format!("PUBLIC_BASE_URL: {e}")))?;
        let session_secret = env::var("SESSION_SECRET")
            .map_err(|_| AppError::Config("SESSION_SECRET이 필요합니다".into()))?;
        if session_secret.len() < 32 {
            return Err(AppError::Config(
                "SESSION_SECRET은 32바이트 이상이어야 합니다".into(),
            ));
        }
        let max_upload_bytes = value("MAX_UPLOAD_BYTES", "5242880")
            .parse()
            .map_err(|e| AppError::Config(format!("MAX_UPLOAD_BYTES: {e}")))?;
        let image_orphan_grace_hours = value("IMAGE_ORPHAN_GRACE_HOURS", "24")
            .parse()
            .map_err(|e| AppError::Config(format!("IMAGE_ORPHAN_GRACE_HOURS: {e}")))?;
        if image_orphan_grace_hours < 1 {
            return Err(AppError::Config(
                "IMAGE_ORPHAN_GRACE_HOURS는 1 이상이어야 합니다".into(),
            ));
        }
        Ok(Self {
            bind_addr,
            database_url: env::var("DATABASE_URL")
                .map_err(|_| AppError::Config("DATABASE_URL이 필요합니다".into()))?,
            public_base_url: public_base_url.trim_end_matches('/').to_owned(),
            site_name: value("SITE_NAME", "Wlog"),
            site_description: value(
                "SITE_DESCRIPTION",
                "개발하고 운영하며 알게 된 것을 기록합니다.",
            ),
            default_social_image: env::var("DEFAULT_SOCIAL_IMAGE").unwrap_or_default(),
            admin_username: env::var("ADMIN_USERNAME")
                .map_err(|_| AppError::Config("ADMIN_USERNAME이 필요합니다".into()))?,
            admin_password_hash: env::var("ADMIN_PASSWORD_HASH")
                .map_err(|_| AppError::Config("ADMIN_PASSWORD_HASH가 필요합니다".into()))?,
            secure_cookie: value(
                "SECURE_COOKIE",
                if url.scheme() == "https" {
                    "true"
                } else {
                    "false"
                },
            )
            .parse()
            .map_err(|e| AppError::Config(format!("SECURE_COOKIE: {e}")))?,
            session_secret,
            upload_dir: PathBuf::from(value("UPLOAD_DIR", "uploads")),
            max_upload_bytes,
            image_orphan_grace_hours,
        })
    }
}
fn value(key: &str, default: &str) -> String {
    env::var(key).unwrap_or_else(|_| default.to_owned())
}
