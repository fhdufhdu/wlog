use super::{
    model::{Topic, TopicSummary},
    repository::TopicRepository,
};
use crate::error::AppError;
use uuid::Uuid;

#[derive(Clone)]
pub struct TopicService {
    repository: TopicRepository,
}

impl TopicService {
    pub fn new(repository: TopicRepository) -> Self {
        Self { repository }
    }

    pub async fn list(&self) -> Result<Vec<Topic>, AppError> {
        Ok(self.repository.list().await?)
    }

    pub async fn list_with_counts(&self) -> Result<Vec<TopicSummary>, AppError> {
        Ok(self.repository.list_with_counts().await?)
    }

    pub async fn create(&self, name: &str) -> Result<Topic, AppError> {
        let name = validate_name(name)?;
        self.repository.create(&name).await.map_err(topic_db_error)
    }

    pub async fn update(&self, id: Uuid, name: &str) -> Result<Topic, AppError> {
        let name = validate_name(name)?;
        self.repository
            .update(id, &name)
            .await
            .map_err(topic_db_error)?
            .ok_or(AppError::NotFound)
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        match self.repository.delete(id).await {
            Ok(true) => Ok(()),
            Ok(false) => Err(AppError::NotFound),
            Err(error) => Err(topic_db_error(error)),
        }
    }
}

fn validate_name(name: &str) -> Result<String, AppError> {
    let name = name.trim();
    if name.is_empty() || name.chars().count() > 40 {
        return Err(AppError::Validation(
            "주제 이름은 1–40자로 입력해주세요.".into(),
        ));
    }
    Ok(name.to_owned())
}

fn topic_db_error(error: sqlx::Error) -> AppError {
    match error.as_database_error().and_then(|e| e.code()) {
        Some(code) if code.as_ref() == "23505" => {
            AppError::Conflict("이미 사용 중인 주제 이름입니다.".into())
        }
        Some(code) if code.as_ref() == "23503" => {
            AppError::Conflict("글이나 임시글에서 사용 중인 주제는 제거할 수 없습니다.".into())
        }
        _ => AppError::Database(error),
    }
}

#[cfg(test)]
mod tests {
    use super::validate_name;

    #[test]
    fn validates_topic_name() {
        assert_eq!(validate_name("  Rust  ").unwrap(), "Rust");
        assert!(validate_name("").is_err());
        assert!(validate_name(&"가".repeat(41)).is_err());
    }
}
