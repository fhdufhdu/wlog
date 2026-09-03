use askama::Template;
use axum::{
    Json,
    extract::{Extension, Form, Multipart, Path, Query, Request, State},
    http::{Method, StatusCode, header},
    middleware::Next,
    response::{Html, IntoResponse, Redirect, Response},
};
use axum_extra::extract::cookie::SignedCookieJar;
use chrono::SecondsFormat;
use serde::{Deserialize, Serialize};
use serde_json::json;
use url::Url;
use uuid::Uuid;

use crate::{
    about::dto::AboutForm,
    auth::Session,
    error::AppError,
    image::service::sanitize_svg,
    markdown,
    post::{
        dto::{IndexQuery, PostForm},
        model::Post,
    },
    state::AppState,
    topic::dto::TopicForm,
};

#[derive(Clone)]
struct Seo {
    title: String,
    description: String,
    canonical: String,
    og_type: String,
    og_image: String,
    json_ld: serde_json::Value,
    robots: String,
}
#[derive(Clone)]
struct PostCard {
    title: String,
    slug: String,
    summary: String,
    topic: String,
    date: String,
}
#[derive(Clone, Default)]
struct PostNavigation {
    title: String,
    slug: String,
    exists: bool,
}
#[derive(Clone)]
struct AdminPost {
    title: String,
    slug: String,
    status: String,
    updated: String,
    edit_url: String,
    delete_url: String,
}
#[derive(Clone, Default)]
struct EditorPost {
    id: String,
    title: String,
    slug: String,
    description: String,
    description_manual: bool,
    topic_id: String,
    content_markdown: String,
    has_public_post: bool,
}
#[derive(Clone)]
struct TopicOption {
    id: String,
    name: String,
}
#[derive(Clone)]
struct AdminTopic {
    id: String,
    name: String,
    post_count: i64,
    temp_count: i64,
    in_use: bool,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    seo: Seo,
    site_name: String,
    posts: Vec<PostCard>,
    topics: Vec<TopicOption>,
    topic_id: String,
    topic_name: String,
}
#[derive(Template)]
#[template(path = "post.html")]
struct PostTemplate {
    seo: Seo,
    site_name: String,
    post: PostCard,
    body_html: String,
    published_iso: String,
    updated_iso: String,
    previous: PostNavigation,
    next: PostNavigation,
}
#[derive(Template)]
#[template(path = "about.html")]
struct AboutTemplate {
    seo: Seo,
    site_name: String,
    title: String,
    body_html: String,
}
#[derive(Template)]
#[template(path = "login.html")]
struct LoginTemplate {
    site_name: String,
    error: String,
}
#[derive(Template)]
#[template(path = "admin.html")]
struct AdminTemplate {
    page_title: String,
    site_name: String,
    csrf: String,
    posts: Vec<AdminPost>,
}
#[derive(Template)]
#[template(path = "topics.html")]
struct TopicsTemplate {
    page_title: String,
    site_name: String,
    csrf: String,
    topics: Vec<AdminTopic>,
}
#[derive(Template)]
#[template(path = "editor.html")]
struct EditorTemplate {
    page_title: String,
    site_name: String,
    csrf: String,
    action: String,
    post: EditorPost,
    topics: Vec<TopicOption>,
}
#[derive(Template)]
#[template(path = "about_editor.html")]
struct AboutEditorTemplate {
    page_title: String,
    site_name: String,
    csrf: String,
    title: String,
    content_markdown: String,
}

pub async fn index(
    State(state): State<AppState>,
    Query(query): Query<IndexQuery>,
) -> Result<Html<String>, AppError> {
    let topics = state.topic_service.list().await?;
    let selected_topic = query.topic.or_else(|| {
        query.category.as_deref().and_then(|name| {
            topics
                .iter()
                .find(|topic| topic.name == name.trim())
                .map(|topic| topic.id)
        })
    });
    let topic_name = selected_topic
        .and_then(|id| topics.iter().find(|topic| topic.id == id))
        .map(|topic| topic.name.clone())
        .unwrap_or_else(|| "전체".into());
    let posts = state
        .post_service
        .list_public(selected_topic)
        .await?
        .into_iter()
        .map(card)
        .collect();
    let canonical = if let Some(topic_id) = selected_topic {
        let mut url = Url::parse(&format!("{}/", state.config.public_base_url))
            .expect("PUBLIC_BASE_URL was validated at startup");
        url.query_pairs_mut()
            .append_pair("topic", &topic_id.to_string());
        url.to_string()
    } else {
        format!("{}/", state.config.public_base_url)
    };
    let json_ld = json!({"@context":"https://schema.org","@type":"Blog","name":state.config.site_name,"description":state.config.site_description,"url":canonical});
    render(IndexTemplate {
        seo: Seo {
            title: state.config.site_name.clone(),
            description: state.config.site_description.clone(),
            canonical,
            og_type: "website".into(),
            og_image: state.config.default_social_image.clone(),
            json_ld,
            robots: "index,follow".into(),
        },
        site_name: state.config.site_name.clone(),
        posts,
        topics: topics.into_iter().map(topic_option).collect(),
        topic_id: selected_topic.map(|id| id.to_string()).unwrap_or_default(),
        topic_name,
    })
}

pub async fn show_post(
    State(state): State<AppState>,
    Path(slug): Path<String>,
) -> Result<Html<String>, AppError> {
    let post = state.post_service.public_by_slug(&slug).await?;
    let (previous, next) = state.post_service.adjacent(&post).await?;
    let canonical = format!("{}/posts/{}", state.config.public_base_url, post.slug);
    let published_iso = post.published_at.to_rfc3339_opts(SecondsFormat::Secs, true);
    let updated_iso = post.updated_at.to_rfc3339_opts(SecondsFormat::Secs, true);
    let json_ld = json!({"@context":"https://schema.org","@type":"BlogPosting","headline":post.title,
        "description":post.description,"datePublished":published_iso,"dateModified":updated_iso,"mainEntityOfPage":canonical,
        "articleSection":post.topic_name,"author":{"@type":"Person","name":state.config.site_name}});
    let seo = Seo {
        title: format!("{} — {}", post.title, state.config.site_name),
        description: post.description.clone(),
        canonical,
        og_type: "article".into(),
        og_image: state.config.default_social_image.clone(),
        json_ld,
        robots: "index,follow,max-image-preview:large".into(),
    };
    let body_html = markdown::render(&post.content_markdown);
    render(PostTemplate {
        seo,
        site_name: state.config.site_name.clone(),
        post: card(post),
        body_html,
        published_iso,
        updated_iso,
        previous: navigation(previous),
        next: navigation(next),
    })
}

pub async fn about(State(state): State<AppState>) -> Result<Html<String>, AppError> {
    let page = state.about_service.get().await?;
    let canonical = format!("{}/about", state.config.public_base_url);
    let excerpt = markdown::excerpt(&page.content_markdown, 80);
    let description = if excerpt.is_empty() {
        state.config.site_description.clone()
    } else {
        excerpt
    };
    let json_ld = json!({
        "@context":"https://schema.org",
        "@type":"ProfilePage",
        "name":page.title,
        "url":canonical,
        "dateModified":page.updated_at.to_rfc3339_opts(SecondsFormat::Secs, true),
        "mainEntity":{"@type":"Person","name":state.config.site_name}
    });
    render(AboutTemplate {
        seo: Seo {
            title: format!("{} — {}", page.title, state.config.site_name),
            description,
            canonical,
            og_type: "profile".into(),
            og_image: state.config.default_social_image.clone(),
            json_ld,
            robots: "index,follow,max-image-preview:large".into(),
        },
        site_name: state.config.site_name.clone(),
        title: page.title,
        body_html: markdown::render(&page.content_markdown),
    })
}

pub async fn login_page(
    State(state): State<AppState>,
    jar: SignedCookieJar,
) -> Result<Response, AppError> {
    if state.auth.session(&jar).is_ok() {
        return Ok(Redirect::to("/admin").into_response());
    }
    Ok(render(LoginTemplate {
        site_name: state.config.site_name.clone(),
        error: String::new(),
    })?
    .into_response())
}

#[derive(Deserialize)]
pub struct LoginForm {
    username: String,
    password: String,
}

pub async fn login(
    State(state): State<AppState>,
    jar: SignedCookieJar,
    Form(form): Form<LoginForm>,
) -> Result<Response, AppError> {
    if !state.auth.verify_password(&form.username, &form.password) {
        return Ok((
            StatusCode::UNAUTHORIZED,
            render(LoginTemplate {
                site_name: state.config.site_name.clone(),
                error: "아이디 또는 비밀번호를 확인해주세요.".into(),
            })?,
        )
            .into_response());
    }
    Ok((
        jar.add(state.auth.login_cookie(state.config.secure_cookie)),
        Redirect::to("/admin"),
    )
        .into_response())
}

#[derive(Deserialize)]
pub struct CsrfForm {
    csrf_token: String,
}

pub async fn logout(
    State(state): State<AppState>,
    jar: SignedCookieJar,
    Extension(session): Extension<Session>,
    Form(form): Form<CsrfForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    Ok((jar.remove(state.auth.logout_cookie()), Redirect::to("/")).into_response())
}

pub async fn admin_index(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
) -> Result<Response, AppError> {
    let mut posts: Vec<AdminPost> = state
        .post_service
        .list_unlinked_temp()
        .await?
        .into_iter()
        .map(|post| AdminPost {
            title: display_title(&post.title),
            slug: post.slug,
            status: "임시저장".into(),
            updated: post.updated_at.format("%Y. %-m. %-d.").to_string(),
            edit_url: format!("/admin/temp-posts/{}/edit", post.id),
            delete_url: format!("/admin/temp-posts/{}/delete", post.id),
        })
        .collect();
    posts.extend(
        state
            .post_service
            .list_all()
            .await?
            .into_iter()
            .map(|post| AdminPost {
                title: display_title(&post.title),
                slug: post.slug,
                status: "공개".into(),
                updated: post.updated_at.format("%Y. %-m. %-d.").to_string(),
                edit_url: format!("/admin/posts/{}/edit", post.id),
                delete_url: format!("/admin/posts/{}/delete", post.id),
            }),
    );
    Ok(render(AdminTemplate {
        page_title: "글 관리".into(),
        site_name: state.config.site_name.clone(),
        csrf: session.csrf,
        posts,
    })?
    .into_response())
}

pub async fn topics_page(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
) -> Result<Response, AppError> {
    let topics = state
        .topic_service
        .list_with_counts()
        .await?
        .into_iter()
        .map(|topic| AdminTopic {
            id: topic.id.to_string(),
            name: topic.name,
            post_count: topic.post_count,
            temp_count: topic.temp_count,
            in_use: topic.post_count > 0 || topic.temp_count > 0,
        })
        .collect();
    Ok(render(TopicsTemplate {
        page_title: "주제 관리".into(),
        site_name: state.config.site_name.clone(),
        csrf: session.csrf,
        topics,
    })?
    .into_response())
}

pub async fn about_editor(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
) -> Result<Response, AppError> {
    let page = state.about_service.get().await?;
    Ok(render(AboutEditorTemplate {
        page_title: "소개 작성".into(),
        site_name: state.config.site_name.clone(),
        csrf: session.csrf,
        title: page.title,
        content_markdown: page.content_markdown,
    })?
    .into_response())
}

pub async fn save_about(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Form(form): Form<AboutForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    state
        .about_service
        .update(&form.title, &form.content_markdown)
        .await?;
    Ok(Redirect::to("/about").into_response())
}

pub async fn create_topic(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Form(form): Form<TopicForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    state.topic_service.create(&form.name).await?;
    Ok(Redirect::to("/admin/topics").into_response())
}

pub async fn update_topic(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
    Form(form): Form<TopicForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    state.topic_service.update(id, &form.name).await?;
    Ok(Redirect::to("/admin/topics").into_response())
}

pub async fn delete_topic(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
    Form(form): Form<CsrfForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    state.topic_service.delete(id).await?;
    Ok(Redirect::to("/admin/topics").into_response())
}

pub async fn new_post_page(
    State(state): State<AppState>,
    Extension(_session): Extension<Session>,
) -> Result<Response, AppError> {
    let temp = state.post_service.new_temp().await?;
    Ok(Redirect::to(&format!("/admin/temp-posts/{}/edit", temp.id)).into_response())
}

pub async fn edit_post_page(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    let temp = state.post_service.temp_for_post(id).await?;
    render_editor(&state, session, temp).await
}

pub async fn edit_temp_page(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    let temp = state.post_service.temp_by_id(id).await?;
    render_editor(&state, session, temp).await
}

pub async fn save_temp_post(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
    Form(form): Form<PostForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    state.post_service.save_temp(id, form).await?;
    Ok(Redirect::to(&format!("/admin/temp-posts/{id}/edit")).into_response())
}

#[derive(Serialize)]
pub struct AutosaveResponse {
    saved_at: String,
}

pub async fn autosave_temp_post(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
    Form(form): Form<PostForm>,
) -> Result<Json<AutosaveResponse>, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    let temp = state.post_service.save_temp(id, form).await?;
    Ok(Json(AutosaveResponse {
        saved_at: temp.updated_at.to_rfc3339_opts(SecondsFormat::Secs, true),
    }))
}

pub async fn publish_temp_post(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
    Form(form): Form<PostForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    state.post_service.publish_temp(id, form).await?;
    Ok(Redirect::to("/admin").into_response())
}

pub async fn delete_post(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
    Form(form): Form<CsrfForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    state.post_service.delete(id).await?;
    Ok(Redirect::to("/admin").into_response())
}

pub async fn delete_temp_post(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    Path(id): Path<Uuid>,
    Form(form): Form<CsrfForm>,
) -> Result<Response, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    state.post_service.delete_temp(id).await?;
    Ok(Redirect::to("/admin").into_response())
}

#[derive(Deserialize)]
pub struct PreviewForm {
    csrf_token: String,
    content_markdown: String,
}

pub async fn preview_markdown(
    Extension(session): Extension<Session>,
    Form(form): Form<PreviewForm>,
) -> Result<Html<String>, AppError> {
    verify_csrf(&session.csrf, &form.csrf_token)?;
    Ok(Html(markdown::render(&form.content_markdown)))
}

#[derive(Serialize)]
pub struct UploadResponse {
    url: String,
    markdown: String,
}

pub async fn upload_image(
    State(state): State<AppState>,
    Extension(session): Extension<Session>,
    mut multipart: Multipart,
) -> Result<Json<UploadResponse>, AppError> {
    let mut csrf = None;
    let mut image = None;
    while let Some(field) = multipart
        .next_field()
        .await
        .map_err(|_| AppError::Validation("업로드 요청을 읽지 못했습니다.".into()))?
    {
        match field.name() {
            Some("csrf_token") => {
                csrf = Some(
                    field
                        .text()
                        .await
                        .map_err(|_| AppError::Validation("보안 토큰을 읽지 못했습니다.".into()))?,
                )
            }
            Some("image") => {
                let original_name = field.file_name().unwrap_or("image").to_owned();
                let declared_content_type = field.content_type().map(str::to_owned);
                let bytes = field
                    .bytes()
                    .await
                    .map_err(|_| AppError::Validation("이미지를 읽지 못했습니다.".into()))?;
                image = Some((original_name, declared_content_type, bytes));
            }
            _ => {}
        }
    }
    verify_csrf(&session.csrf, csrf.as_deref().unwrap_or_default())?;
    let (original_name, declared_content_type, image) =
        image.ok_or_else(|| AppError::Validation("이미지를 선택해주세요.".into()))?;
    if image.is_empty() || image.len() > state.config.max_upload_bytes {
        return Err(AppError::Validation("이미지는 5MB 이하여야 합니다.".into()));
    }
    let is_svg = declared_content_type.as_deref() == Some("image/svg+xml")
        || original_name
            .rsplit_once('.')
            .is_some_and(|(_, extension)| extension.eq_ignore_ascii_case("svg"));
    let (image, mime_type, extension) = if is_svg {
        let image = sanitize_svg(&image)?;
        if image.len() > state.config.max_upload_bytes {
            return Err(AppError::Validation("이미지는 5MB 이하여야 합니다.".into()));
        }
        (image, "image/svg+xml", "svg")
    } else {
        let kind = infer::get(&image)
            .ok_or_else(|| AppError::Validation("이미지 파일을 확인할 수 없습니다.".into()))?;
        let mime_type = kind.mime_type();
        let extension = match mime_type {
            "image/jpeg" => "jpg",
            "image/png" => "png",
            "image/gif" => "gif",
            "image/webp" => "webp",
            _ => {
                return Err(AppError::Validation(
                    "JPEG, PNG, GIF, WebP, SVG만 업로드할 수 있습니다.".into(),
                ));
            }
        };
        (image.to_vec(), mime_type, extension)
    };
    let id = Uuid::new_v4();
    let filename = format!("{id}.{extension}");
    let path = state.config.upload_dir.join(&filename);
    tokio::fs::write(&path, &image).await?;
    if let Err(error) = state
        .image_service
        .register(id, &filename, &original_name, mime_type, image.len())
        .await
    {
        if let Err(remove_error) = tokio::fs::remove_file(&path).await {
            tracing::warn!(file = %filename, error = ?remove_error, "unregistered image cleanup failed");
        }
        return Err(error);
    }
    let url = format!("/uploads/{filename}");
    Ok(Json(UploadResponse {
        markdown: format!("![이미지]({url})"),
        url,
    }))
}

pub async fn sitemap(State(state): State<AppState>) -> Result<Response, AppError> {
    let posts = state.post_service.list_public(None).await?;
    let mut xml = format!(
        "<?xml version=\"1.0\" encoding=\"UTF-8\"?><urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\"><url><loc>{}/</loc></url><url><loc>{}/about</loc></url>",
        state.config.public_base_url, state.config.public_base_url
    );
    for post in posts {
        xml.push_str(&format!(
            "<url><loc>{}/posts/{}</loc><lastmod>{}</lastmod></url>",
            state.config.public_base_url,
            post.slug,
            post.updated_at.format("%Y-%m-%d")
        ));
    }
    xml.push_str("</urlset>");
    Ok((
        [(header::CONTENT_TYPE, "application/xml; charset=utf-8")],
        xml,
    )
        .into_response())
}

pub async fn robots(State(state): State<AppState>) -> Response {
    (
        [(header::CONTENT_TYPE, "text/plain; charset=utf-8")],
        format!(
            "User-agent: *\nAllow: /\nDisallow: /admin\nSitemap: {}/sitemap.xml\n",
            state.config.public_base_url
        ),
    )
        .into_response()
}
pub async fn health_live() -> &'static str {
    "ok"
}
pub async fn health_ready(State(state): State<AppState>) -> Result<&'static str, AppError> {
    sqlx::query("SELECT 1").execute(&state.pool).await?;
    Ok("ok")
}

pub async fn require_admin(
    State(state): State<AppState>,
    jar: SignedCookieJar,
    mut request: Request,
    next: Next,
) -> Response {
    match state.auth.session(&jar) {
        Ok(session) => {
            request.extensions_mut().insert(session);
            next.run(request).await
        }
        Err(_) if matches!(*request.method(), Method::GET | Method::HEAD) => {
            Redirect::to("/admin/login").into_response()
        }
        Err(error) => error.into_response(),
    }
}

fn render<T: Template>(template: T) -> Result<Html<String>, AppError> {
    Ok(Html(template.render()?))
}
fn verify_csrf(expected: &str, received: &str) -> Result<(), AppError> {
    if expected.is_empty() || expected != received {
        Err(AppError::Unauthorized)
    } else {
        Ok(())
    }
}
fn card(post: Post) -> PostCard {
    PostCard {
        title: post.title,
        slug: post.slug,
        summary: post.description,
        topic: post.topic_name,
        date: post.published_at.format("%Y. %-m. %-d.").to_string(),
    }
}
fn navigation(post: Option<crate::post::model::PostLink>) -> PostNavigation {
    post.map_or_else(PostNavigation::default, |post| PostNavigation {
        title: post.title,
        slug: post.slug,
        exists: true,
    })
}
fn editor(post: crate::post::model::TempPost) -> EditorPost {
    EditorPost {
        id: post.id.to_string(),
        title: post.title,
        slug: post.slug,
        description: post.description,
        description_manual: post.description_manual,
        topic_id: post.topic_id.map(|id| id.to_string()).unwrap_or_default(),
        content_markdown: post.content_markdown,
        has_public_post: post.post_id.is_some(),
    }
}

async fn render_editor(
    state: &AppState,
    session: Session,
    post: crate::post::model::TempPost,
) -> Result<Response, AppError> {
    let has_public_post = post.post_id.is_some();
    let topics = state
        .topic_service
        .list()
        .await?
        .into_iter()
        .map(topic_option)
        .collect();
    Ok(render(EditorTemplate {
        page_title: if has_public_post {
            "글 수정"
        } else {
            "새 글"
        }
        .into(),
        site_name: state.config.site_name.clone(),
        csrf: session.csrf,
        action: format!("/admin/temp-posts/{}/publish", post.id),
        post: editor(post),
        topics,
    })?
    .into_response())
}

fn topic_option(topic: crate::topic::model::Topic) -> TopicOption {
    TopicOption {
        id: topic.id.to_string(),
        name: topic.name,
    }
}

fn display_title(title: &str) -> String {
    if title.trim().is_empty() {
        "제목 없는 글".into()
    } else {
        title.to_owned()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn maps_optional_post_navigation() {
        let empty = navigation(None);
        assert!(!empty.exists);

        let linked = navigation(Some(crate::post::model::PostLink {
            title: "다음 글".into(),
            slug: "next-post".into(),
        }));
        assert!(linked.exists);
        assert_eq!(linked.title, "다음 글");
        assert_eq!(linked.slug, "next-post");
    }

    #[test]
    fn json_ld_filter_cannot_close_script_element() {
        let html = IndexTemplate {
            seo: Seo {
                title: "Wlog".into(),
                description: "test".into(),
                canonical: "https://example.com/".into(),
                og_type: "website".into(),
                og_image: String::new(),
                json_ld: json!({"headline":"</script><script>alert(1)</script>"}),
                robots: "noindex".into(),
            },
            site_name: "Wlog".into(),
            posts: Vec::new(),
            topics: Vec::new(),
            topic_id: String::new(),
            topic_name: "전체".into(),
        }
        .render()
        .unwrap();

        assert!(!html.contains("</script><script>"));
        assert!(html.contains("\\u003c/script\\u003e"));
    }
}
