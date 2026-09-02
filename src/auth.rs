use crate::error::AppError;
use argon2::{
    Argon2,
    password_hash::{PasswordVerifier, phc::PasswordHash},
};
use axum_extra::extract::cookie::{Cookie, SameSite, SignedCookieJar};
use std::time::{SystemTime, UNIX_EPOCH};
use uuid::Uuid;

const COOKIE_NAME: &str = "wlog_session";
#[derive(Clone)]
pub struct Auth {
    username: String,
    password_hash: String,
}
#[derive(Clone, Debug)]
pub struct Session {
    pub csrf: String,
}

impl Auth {
    pub fn new(username: String, password_hash: String) -> Result<Self, AppError> {
        PasswordHash::new(&password_hash).map_err(|_| {
            AppError::Config("ADMIN_PASSWORD_HASH가 유효한 PHC 문자열이 아닙니다".into())
        })?;
        Ok(Self {
            username,
            password_hash,
        })
    }
    pub fn verify_password(&self, username: &str, password: &str) -> bool {
        username == self.username
            && PasswordHash::new(&self.password_hash)
                .ok()
                .is_some_and(|hash| {
                    Argon2::default()
                        .verify_password(password.as_bytes(), &hash)
                        .is_ok()
                })
    }
    pub fn login_cookie(&self, secure: bool) -> Cookie<'static> {
        let payload = format!("{}|{}|{}", self.username, now() + 604_800, Uuid::new_v4());
        Cookie::build((COOKIE_NAME, payload))
            .path("/")
            .http_only(true)
            .same_site(SameSite::Strict)
            .secure(secure)
            .build()
    }
    pub fn logout_cookie(&self) -> Cookie<'static> {
        Cookie::build((COOKIE_NAME, ""))
            .path("/")
            .http_only(true)
            .same_site(SameSite::Strict)
            .build()
    }
    pub fn session(&self, jar: &SignedCookieJar) -> Result<Session, AppError> {
        let payload = jar.get(COOKIE_NAME).ok_or(AppError::Unauthorized)?;
        let mut parts = payload.value().split('|');
        let username = parts.next().ok_or(AppError::Unauthorized)?;
        let expiry: u64 = parts
            .next()
            .ok_or(AppError::Unauthorized)?
            .parse()
            .map_err(|_| AppError::Unauthorized)?;
        let csrf = parts.next().ok_or(AppError::Unauthorized)?.to_owned();
        if username != self.username || expiry < now() || parts.next().is_some() {
            return Err(AppError::Unauthorized);
        }
        Ok(Session { csrf })
    }
}
fn now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

#[cfg(test)]
mod tests {
    use super::*;

    const HASH: &str = "$argon2id$v=19$m=19456,t=2,p=1$bIQ5JeCLKlP/HKx9tLSxLQ$7myOb/wrmOBE639j16R+0MdwwjUsG2oNNtQem3OYtzs";

    #[test]
    fn verifies_credentials_and_rejects_tampered_sessions() {
        let auth = Auth::new("owner".into(), HASH.into()).unwrap();
        assert!(auth.verify_password("owner", "testtesttest12"));
        assert!(!auth.verify_password("owner", "wrong-password"));

        let key = axum_extra::extract::cookie::Key::derive_from(
            b"integration-test-secret-at-least-32-bytes",
        );
        let jar = SignedCookieJar::new(key.clone()).add(auth.login_cookie(false));
        assert!(auth.session(&jar).is_ok());
        let mut headers = axum::http::HeaderMap::new();
        headers.insert(
            axum::http::header::COOKIE,
            axum::http::HeaderValue::from_static("wlog_session=owner%7C9999999999%7Cforged"),
        );
        let unsigned = SignedCookieJar::from_headers(&headers, key);
        assert!(auth.session(&unsigned).is_err());
    }
}
