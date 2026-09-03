use std::{collections::HashSet, io::Cursor, io::ErrorKind, path::PathBuf};

use chrono::{Duration, Utc};
use image::{
    DynamicImage, ImageDecoder, ImageFormat, ImageReader, imageops::FilterType,
    metadata::Orientation,
};
use uuid::Uuid;

use crate::{error::AppError, markdown};

use super::repository::ImageRepository;

pub fn sanitize_svg(svg: &[u8]) -> Result<Vec<u8>, AppError> {
    let mut input = svg;
    let mut sanitized = Vec::with_capacity(svg.len());
    svg_hush::Filter::new()
        .filter(&mut input, &mut sanitized)
        .map_err(|_| AppError::Validation("올바르고 안전한 SVG 파일인지 확인해주세요.".into()))?;

    if sanitized.is_empty() {
        return Err(AppError::Validation(
            "내용이 없는 SVG 파일은 업로드할 수 없습니다.".into(),
        ));
    }
    Ok(sanitized)
}

pub struct NormalizedRaster {
    pub bytes: Vec<u8>,
    pub mime_type: &'static str,
    pub extension: &'static str,
}

pub fn normalize_raster(
    bytes: Vec<u8>,
    mime_type: &str,
    max_dimension: u32,
    max_pixels: u64,
    webp_quality: f32,
) -> Result<NormalizedRaster, AppError> {
    let format = match mime_type {
        "image/jpeg" => ImageFormat::Jpeg,
        "image/png" => ImageFormat::Png,
        "image/gif" => ImageFormat::Gif,
        "image/webp" => ImageFormat::WebP,
        _ => {
            return Err(AppError::Validation(
                "JPEG, PNG, GIF, WebP, SVG만 업로드할 수 있습니다.".into(),
            ));
        }
    };

    let reader = ImageReader::with_format(Cursor::new(bytes.as_slice()), format);
    let mut decoder = reader.into_decoder().map_err(|_| invalid_raster_error())?;
    let (width, height) = decoder.dimensions();
    validate_dimensions(width, height, max_pixels)?;

    if format == ImageFormat::Gif {
        if width > max_dimension || height > max_dimension {
            return Err(AppError::Validation(format!(
                "움직이는 GIF는 자동 축소하지 않습니다. 가로와 세로를 각각 {max_dimension}px 이하로 줄여주세요."
            )));
        }
        drop(decoder);
        return Ok(NormalizedRaster {
            bytes,
            mime_type: "image/gif",
            extension: "gif",
        });
    }

    let orientation = decoder.orientation().unwrap_or(Orientation::NoTransforms);
    let mut image = DynamicImage::from_decoder(decoder).map_err(|_| invalid_raster_error())?;
    image.apply_orientation(orientation);
    if image.width() > max_dimension || image.height() > max_dimension {
        image = image.resize(max_dimension, max_dimension, FilterType::Lanczos3);
    }

    let rgba = image.to_rgba8();
    let encoded = webp::Encoder::from_rgba(rgba.as_raw(), rgba.width(), rgba.height())
        .encode_simple(false, webp_quality)
        .map_err(|_| AppError::Validation("이미지를 WebP로 변환하지 못했습니다.".into()))?;
    Ok(NormalizedRaster {
        bytes: encoded.to_vec(),
        mime_type: "image/webp",
        extension: "webp",
    })
}

fn validate_dimensions(width: u32, height: u32, max_pixels: u64) -> Result<(), AppError> {
    let pixels = u64::from(width) * u64::from(height);
    if width == 0 || height == 0 || pixels > max_pixels {
        return Err(AppError::Validation(format!(
            "이미지 해상도가 너무 큽니다. 전체 픽셀 수는 {max_pixels} 이하여야 합니다."
        )));
    }
    Ok(())
}

fn invalid_raster_error() -> AppError {
    AppError::Validation("손상되었거나 지원하지 않는 이미지 파일입니다.".into())
}

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

#[cfg(test)]
mod tests {
    use std::io::Cursor;

    use image::{DynamicImage, GenericImageView, ImageFormat, Rgba, RgbaImage};

    use super::{normalize_raster, sanitize_svg};

    fn png_fixture(width: u32, height: u32) -> Vec<u8> {
        let image = RgbaImage::from_pixel(width, height, Rgba([18, 118, 150, 255]));
        let mut bytes = Cursor::new(Vec::new());
        DynamicImage::ImageRgba8(image)
            .write_to(&mut bytes, ImageFormat::Png)
            .expect("fixture should encode");
        bytes.into_inner()
    }

    fn gif_fixture(width: u32, height: u32) -> Vec<u8> {
        let image = RgbaImage::from_pixel(width, height, Rgba([18, 118, 150, 255]));
        let mut bytes = Cursor::new(Vec::new());
        DynamicImage::ImageRgba8(image)
            .write_to(&mut bytes, ImageFormat::Gif)
            .expect("fixture should encode");
        bytes.into_inner()
    }

    #[test]
    fn sanitizer_removes_active_svg_content() {
        let svg = br#"<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)">
            <script>alert(1)</script>
            <a href="https://example.com/evil"><rect width="10" height="10" /></a>
        </svg>"#;

        let sanitized = sanitize_svg(svg).expect("valid SVG should be sanitized");
        let sanitized = String::from_utf8(sanitized).expect("sanitized SVG should be UTF-8");

        assert!(sanitized.contains("<svg"));
        assert!(sanitized.contains("rect"));
        assert!(!sanitized.contains("script"));
        assert!(!sanitized.contains("onload"));
        assert!(!sanitized.contains("example.com"));
    }

    #[test]
    fn sanitizer_rejects_malformed_svg() {
        assert!(sanitize_svg(b"<svg><path></svg>").is_err());
    }

    #[test]
    fn raster_is_resized_and_normalized_to_webp() {
        let normalized = normalize_raster(png_fixture(400, 200), "image/png", 100, 1_000_000, 82.0)
            .expect("PNG should normalize");

        assert_eq!(normalized.mime_type, "image/webp");
        assert_eq!(normalized.extension, "webp");
        assert_eq!(&normalized.bytes[0..4], b"RIFF");
        assert_eq!(&normalized.bytes[8..12], b"WEBP");
        let decoded = image::load_from_memory_with_format(&normalized.bytes, ImageFormat::WebP)
            .expect("normalized WebP should decode");
        assert_eq!(decoded.dimensions(), (100, 50));
    }

    #[test]
    fn raster_over_pixel_limit_is_rejected_before_decode() {
        let result = normalize_raster(png_fixture(20, 20), "image/png", 100, 399, 82.0);

        assert!(result.is_err());
    }

    #[test]
    fn gif_is_preserved_when_it_fits_limits() {
        let original = gif_fixture(20, 10);
        let normalized = normalize_raster(original.clone(), "image/gif", 100, 1_000, 82.0)
            .expect("GIF should pass inspection");

        assert_eq!(normalized.bytes, original);
        assert_eq!(normalized.mime_type, "image/gif");
        assert_eq!(normalized.extension, "gif");
    }

    #[test]
    fn oversized_gif_is_rejected_instead_of_losing_animation() {
        let result = normalize_raster(gif_fixture(101, 50), "image/gif", 100, 10_000, 82.0);

        assert!(result.is_err());
    }
}
