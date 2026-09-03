use super::{
    dto::PostForm,
    model::{Post, TempPost},
    repository::PostRepository,
};
use crate::{error::AppError, markdown};
use chrono::Utc;
use uuid::Uuid;

#[derive(Clone)]
pub struct PostService {
    repository: PostRepository,
}

impl PostService {
    pub fn new(repository: PostRepository) -> Self {
        Self { repository }
    }
    pub async fn list_public(&self, topic_id: Option<Uuid>) -> Result<Vec<Post>, AppError> {
        Ok(self.repository.list_public(topic_id).await?)
    }
    pub async fn list_all(&self) -> Result<Vec<Post>, AppError> {
        Ok(self.repository.list_all().await?)
    }
    pub async fn list_unlinked_temp(&self) -> Result<Vec<TempPost>, AppError> {
        Ok(self.repository.list_unlinked_temp().await?)
    }
    pub async fn public_by_slug(&self, slug: &str) -> Result<Post, AppError> {
        self.repository
            .find_public_slug(slug)
            .await?
            .ok_or(AppError::NotFound)
    }
    pub async fn by_id(&self, id: Uuid) -> Result<Post, AppError> {
        self.repository.find_id(id).await?.ok_or(AppError::NotFound)
    }
    pub async fn temp_by_id(&self, id: Uuid) -> Result<TempPost, AppError> {
        self.repository
            .find_temp_id(id)
            .await?
            .ok_or(AppError::NotFound)
    }
    pub async fn new_temp(&self) -> Result<TempPost, AppError> {
        Ok(self.repository.create_empty_temp().await?)
    }
    pub async fn temp_for_post(&self, post_id: Uuid) -> Result<TempPost, AppError> {
        self.repository
            .temp_for_post(post_id)
            .await?
            .ok_or(AppError::NotFound)
    }
    pub async fn save_temp(&self, id: Uuid, form: PostForm) -> Result<TempPost, AppError> {
        let clean = validate(form, false)?;
        let previous = self.temp_by_id(id).await?;
        let temp = TempPost {
            id,
            post_id: previous.post_id,
            title: clean.title,
            slug: clean.slug,
            description: clean.description,
            description_manual: clean.description_manual,
            content_markdown: clean.content_markdown,
            topic_id: clean.topic_id,
            topic_name: previous.topic_name,
            created_at: previous.created_at,
            updated_at: Utc::now(),
        };
        self.repository.update_temp(&temp).await.map_err(db_error)
    }
    pub async fn publish_temp(&self, id: Uuid, form: PostForm) -> Result<Post, AppError> {
        let clean = validate(form, true)?;
        let previous = self.temp_by_id(id).await?;
        let temp = TempPost {
            id,
            post_id: previous.post_id,
            title: clean.title,
            slug: clean.slug,
            description: clean.description,
            description_manual: clean.description_manual,
            content_markdown: clean.content_markdown,
            topic_id: clean.topic_id,
            topic_name: previous.topic_name,
            created_at: previous.created_at,
            updated_at: Utc::now(),
        };
        let saved = self.repository.update_temp(&temp).await.map_err(db_error)?;
        let image_names: Vec<String> = markdown::upload_names(&saved.content_markdown)
            .into_iter()
            .collect();
        self.repository
            .publish_temp(&saved, &image_names)
            .await
            .map_err(db_error)
    }
    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        if self.repository.delete(id).await? {
            Ok(())
        } else {
            Err(AppError::NotFound)
        }
    }
    pub async fn delete_temp(&self, id: Uuid) -> Result<(), AppError> {
        if self.repository.delete_temp(id).await? {
            Ok(())
        } else {
            Err(AppError::NotFound)
        }
    }
}

struct CleanPost {
    title: String,
    slug: String,
    description: String,
    description_manual: bool,
    topic_id: Option<Uuid>,
    content_markdown: String,
}

fn validate(form: PostForm, publishing: bool) -> Result<CleanPost, AppError> {
    let title = form.title.trim().to_owned();
    let topic_id = if form.topic_id.trim().is_empty() {
        None
    } else {
        Some(
            Uuid::parse_str(form.topic_id.trim())
                .map_err(|_| AppError::Validation("올바른 주제를 선택해주세요.".into()))?,
        )
    };
    let content_markdown = form.content_markdown.trim().to_owned();
    let slug = slug::slugify(form.slug.trim());
    let description_manual = form.description_manual;
    let description = if !description_manual {
        markdown::excerpt(&content_markdown, 80)
    } else {
        form.description.trim().to_owned()
    };
    if title.chars().count() > 120 || (publishing && title.is_empty()) {
        return Err(AppError::Validation(
            "제목은 1–120자로 입력해주세요.".into(),
        ));
    }
    if slug.len() > 160 || (publishing && slug.is_empty()) {
        return Err(AppError::Validation(
            "주소는 영문·숫자 중심의 1–160자로 입력해주세요.".into(),
        ));
    }
    if description.chars().count() > 200 || (publishing && description.is_empty()) {
        return Err(AppError::Validation(
            "요약은 1–200자로 입력해주세요.".into(),
        ));
    }
    if publishing && topic_id.is_none() {
        return Err(AppError::Validation("주제를 선택해주세요.".into()));
    }
    if publishing && content_markdown.is_empty() {
        return Err(AppError::Validation("본문을 입력해주세요.".into()));
    }
    Ok(CleanPost {
        title,
        slug,
        description,
        description_manual,
        topic_id,
        content_markdown,
    })
}

fn db_error(error: sqlx::Error) -> AppError {
    match error.as_database_error().and_then(|e| e.code()) {
        Some(code) if code.as_ref() == "23505" => {
            AppError::Conflict("이미 사용 중인 글 주소입니다.".into())
        }
        Some(code) if code.as_ref() == "23503" => {
            AppError::Validation("선택한 주제를 찾을 수 없습니다.".into())
        }
        _ => AppError::Database(error),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn form(description: &str, description_manual: bool) -> PostForm {
        PostForm {
            title: "제목".into(),
            slug: "post-slug".into(),
            description: description.into(),
            description_manual,
            topic_id: Uuid::new_v4().to_string(),
            content_markdown: "**자동으로 추출할 본문입니다.**".into(),
            csrf_token: "test".into(),
        }
    }

    #[test]
    fn derives_description_unless_author_edited_it() {
        let automatic = validate(form("오래된 자동 설명", false), true).unwrap();
        assert_eq!(automatic.description, "자동으로 추출할 본문입니다.");

        let manual = validate(form("직접 작성한 설명", true), true).unwrap();
        assert_eq!(manual.description, "직접 작성한 설명");
    }

    #[test]
    fn preserves_an_intentionally_cleared_manual_summary() {
        let draft = validate(form("", true), false).unwrap();
        assert!(draft.description.is_empty());
        assert!(validate(form("", true), true).is_err());
    }

    #[test]
    fn allows_incomplete_temp_but_not_incomplete_publication() {
        let empty = PostForm {
            title: String::new(),
            slug: String::new(),
            description: String::new(),
            description_manual: false,
            topic_id: String::new(),
            content_markdown: String::new(),
            csrf_token: "test".into(),
        };
        assert!(validate(empty.clone(), false).is_ok());
        assert!(validate(empty, true).is_err());
    }
}
