use crate::error::AppError;

use super::{model::AboutPage, repository::AboutRepository};

#[derive(Clone)]
pub struct AboutService {
    repository: AboutRepository,
}

impl AboutService {
    pub fn new(repository: AboutRepository) -> Self {
        Self { repository }
    }

    pub async fn get(&self) -> Result<AboutPage, AppError> {
        Ok(self.repository.get().await?)
    }

    pub async fn update(&self, title: &str, content_markdown: &str) -> Result<AboutPage, AppError> {
        let title = validate_title(title)?;
        Ok(self.repository.update(&title, content_markdown).await?)
    }
}

fn validate_title(title: &str) -> Result<String, AppError> {
    let title = title.trim();
    if title.is_empty() || title.chars().count() > 120 {
        return Err(AppError::Validation(
            "소개 제목은 1–120자로 입력해주세요.".into(),
        ));
    }
    Ok(title.to_owned())
}

#[cfg(test)]
mod tests {
    use super::validate_title;

    #[test]
    fn validates_about_title() {
        assert_eq!(validate_title("  소개  ").unwrap(), "소개");
        assert!(validate_title("").is_err());
        assert!(validate_title(&"가".repeat(121)).is_err());
    }
}
