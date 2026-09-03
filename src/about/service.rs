use crate::{error::AppError, markdown};

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

    pub async fn backfill_rendered_html(&self) -> Result<bool, AppError> {
        let page = self.repository.get().await?;
        if !page.content_html.is_empty() {
            return Ok(false);
        }
        let content_html = markdown::render(&page.content_markdown);
        self.repository
            .set_rendered_html_if_empty(&content_html)
            .await?;
        Ok(true)
    }

    pub async fn update(
        &self,
        title: &str,
        content_markdown: &str,
        content_html: &str,
    ) -> Result<AboutPage, AppError> {
        let title = validate_title(title)?;
        let content_html = if content_html.trim().is_empty() {
            markdown::render(content_markdown)
        } else {
            markdown::sanitize_html(content_html)
        };
        Ok(self
            .repository
            .update(&title, content_markdown, &content_html)
            .await?)
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
