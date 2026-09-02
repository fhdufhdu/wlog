use std::{collections::HashSet, io::ErrorKind, path::PathBuf};

use chrono::{Duration, Utc};
use uuid::Uuid;

use crate::{error::AppError, markdown};

use super::repository::ImageRepository;

#[derive(Clone)]
pub struct ImageService {
    repository: ImageRepository,
    upload_dir: PathBuf,
    orphan_grace: Duration,
}

impl ImageService {
    pub fn new(repository: ImageRepository, upload_dir: PathBuf, orphan_grace_hours: i64) -> Self {
        Self {
            repository,
            upload_dir,
            orphan_grace: Duration::hours(orphan_grace_hours),
        }
    }

    pub async fn register(
        &self,
        id: Uuid,
        storage_name: &str,
        original_name: &str,
        mime_type: &str,
        byte_size: usize,
    ) -> Result<(), AppError> {
        let original_name: String = original_name.chars().take(255).collect();
        let original_name = if original_name.trim().is_empty() {
            storage_name
        } else {
            original_name.trim()
        };
        let byte_size = i64::try_from(byte_size)
            .map_err(|_| AppError::Validation("이미지 크기를 처리할 수 없습니다.".into()))?;
        self.repository
            .create(id, storage_name, original_name, mime_type, byte_size)
            .await?;
        Ok(())
    }

    pub async fn cleanup_orphans(&self) -> Result<usize, AppError> {
        let before = Utc::now() - self.orphan_grace;
        let candidates = self.repository.orphan_candidates(before).await?;
        if candidates.is_empty() {
            return Ok(0);
        }

        let referenced: HashSet<String> = self
            .repository
            .all_markdown()
            .await?
            .iter()
            .flat_map(|content| markdown::upload_names(content))
            .collect();
        let mut removed = 0;
        for image in candidates {
            if referenced.contains(&image.storage_name) {
                continue;
            }
            match tokio::fs::remove_file(self.upload_dir.join(&image.storage_name)).await {
                Ok(()) => {}
                Err(error) if error.kind() == ErrorKind::NotFound => {}
                Err(error) => {
                    tracing::warn!(file = %image.storage_name, error = ?error, "orphan image file removal failed");
                    continue;
                }
            }
            if self.repository.delete_if_orphan(image.id).await? {
                removed += 1;
            }
        }
        Ok(removed)
    }
}
