package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fhdufhdu/wlog/internal/about"
	"github.com/fhdufhdu/wlog/internal/app"
	"github.com/fhdufhdu/wlog/internal/apperr"
	"github.com/fhdufhdu/wlog/internal/auth"
	"github.com/fhdufhdu/wlog/internal/config"
	images "github.com/fhdufhdu/wlog/internal/image"
	markdownutil "github.com/fhdufhdu/wlog/internal/markdown"
	"github.com/fhdufhdu/wlog/internal/post"
	"github.com/fhdufhdu/wlog/internal/topic"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Controller struct {
	config    config.Config
	pool      *pgxpool.Pool
	auth      *auth.Auth
	posts     *post.Service
	topics    *topic.Service
	about     *about.Service
	images    *images.Service
	templates *template.Template
}
type sessionKey struct{}

func NewController(cfg config.Config, pool *pgxpool.Pool, authService *auth.Auth, posts *post.Service, topics *topic.Service, aboutService *about.Service, imageService *images.Service) (*Controller, error) {
	funcs := template.FuncMap{"safeHTML": func(v string) template.HTML { return template.HTML(v) }, "safeJS": func(v string) template.JS { return template.JS(v) }, "runeCount": utf8.RuneCountInString}
	t, err := template.New("").Funcs(funcs).ParseGlob("templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Controller{cfg, pool, authService, posts, topics, aboutService, imageService, t}, nil
}

func (c *Controller) Route() *app.AppMux {
	public := app.NewAppMux()
	public.HandleFunc("GET /{$}", c.index)
	public.HandleFunc("GET /about", c.aboutPage)
	public.HandleFunc("GET /posts/{slug}", c.showPost)
	public.HandleFunc("GET /admin/login", c.loginPage)
	public.HandleFunc("POST /admin/login", c.login)
	public.HandleFunc("GET /sitemap.xml", c.sitemap)
	public.HandleFunc("GET /robots.txt", c.robots)
	public.HandleFunc("GET /health/live", c.live)
	public.HandleFunc("GET /health/ready", c.ready)

	admin := app.NewAppMux()
	admin.HandleFunc("POST /admin/logout", c.logout)
	admin.HandleFunc("GET /admin", c.adminIndex)
	admin.HandleFunc("GET /admin/topics", c.topicsPage)
	admin.HandleFunc("POST /admin/topics", c.createTopic)
	admin.HandleFunc("POST /admin/topics/{id}", c.updateTopic)
	admin.HandleFunc("POST /admin/topics/{id}/delete", c.deleteTopic)
	admin.HandleFunc("GET /admin/about", c.aboutEditor)
	admin.HandleFunc("POST /admin/about", c.saveAbout)
	admin.HandleFunc("GET /admin/posts/new", c.newPost)
	admin.HandleFunc("GET /admin/posts/{id}/edit", c.editPost)
	admin.HandleFunc("POST /admin/posts/{id}/delete", c.deletePost)
	admin.HandleFunc("GET /admin/temp-posts/{id}/edit", c.editTemp)
	admin.HandleFunc("POST /admin/temp-posts/{id}/save", c.saveTemp)
	admin.HandleFunc("POST /admin/temp-posts/{id}/autosave", c.autosave)
	admin.HandleFunc("POST /admin/temp-posts/{id}/publish", c.publish)
	admin.HandleFunc("POST /admin/temp-posts/{id}/delete", c.deleteTemp)
	admin.HandleFunc("POST /admin/markdown-preview", c.markdownPreview)
	admin.HandleFunc("POST /admin/uploads", c.upload)
	admin.RegisterAppMiddlewares(c.requireAdmin)
	return app.NewAppMux().Nest(public).Nest(admin)
}

func (c *Controller) StaticRoutes() *app.AppMux {
	m := app.NewAppMux()
	for _, name := range []string{"styles.css", "admin.js", "theme.js", "mermaid.js", "math.js"} {
		filename := name
		m.Handle("GET /"+name, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cache := "public, max-age=3600"
			if filename == "admin.js" || filename == "styles.css" {
				cache = "no-cache"
			}
			w.Header().Set("Cache-Control", cache)
			http.ServeFile(w, r, filename)
		}))
	}
	m.Handle("GET /assets/{path...}", cacheHandler(http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))), "public, max-age=604800", false))
	m.Handle("GET /uploads/{path...}", cacheHandler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(c.config.UploadDir))), "public, max-age=31536000, immutable", true))
	return m
}

func cacheHandler(next http.Handler, cache string, sandbox bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cache)
		if sandbox {
			w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'")
		}
		next.ServeHTTP(w, r)
	})
}
func (c *Controller) render(w http.ResponseWriter, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return c.templates.ExecuteTemplate(w, name, data)
}
func redirect(w http.ResponseWriter, r *http.Request, target string) {
	http.Redirect(w, r, target, http.StatusSeeOther)
}
func seoJSON(value any) string { data, _ := json.Marshal(value); return string(data) }
func kst(t time.Time, layout string) string {
	return t.In(time.FixedZone("KST", 9*3600)).Format(layout)
}
func parseUUID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apperr.New(apperr.NotFound, "")
	}
	return id, nil
}
func parseForm(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return apperr.New(apperr.Validation, "요청을 읽지 못했습니다.")
	}
	return nil
}
func (c *Controller) seo(title, description, canonical, kind, robots string, jsonLD any) map[string]any {
	return map[string]any{"Title": title, "Description": description, "Canonical": canonical, "OGType": kind, "OGImage": c.config.DefaultSocialImage, "Robots": robots, "JSONLD": seoJSON(jsonLD)}
}

func (c *Controller) index(w http.ResponseWriter, r *http.Request) error {
	topics, err := c.topics.List(r.Context())
	if err != nil {
		return err
	}
	var selected *uuid.UUID
	if raw := r.URL.Query().Get("topic"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			selected = &id
		}
	} else if category := strings.TrimSpace(r.URL.Query().Get("category")); category != "" {
		for _, item := range topics {
			if item.Name == category {
				id := item.ID
				selected = &id
				break
			}
		}
	}
	topicName := "전체"
	if selected != nil {
		for _, item := range topics {
			if item.ID == *selected {
				topicName = item.Name
				break
			}
		}
	}
	posts, err := c.posts.ListPublic(r.Context(), selected)
	if err != nil {
		return err
	}
	canonical := c.config.PublicBaseURL + "/"
	topicID := ""
	if selected != nil {
		topicID = selected.String()
		canonical += "?topic=" + url.QueryEscape(topicID)
	}
	cards := make([]map[string]any, 0, len(posts))
	for _, item := range posts {
		cards = append(cards, map[string]any{"Title": item.Title, "Slug": item.Slug, "Summary": item.Description, "Topic": item.TopicName, "Date": kst(item.PublishedAt, "2006. 1. 2.")})
	}
	options := make([]map[string]string, 0, len(topics))
	for _, item := range topics {
		options = append(options, map[string]string{"ID": item.ID.String(), "Name": item.Name})
	}
	jsonLD := map[string]any{"@context": "https://schema.org", "@type": "Blog", "name": c.config.SiteName, "description": c.config.SiteDescription, "url": canonical}
	return c.render(w, "index.html", map[string]any{"SiteName": c.config.SiteName, "SEO": c.seo(c.config.SiteName, c.config.SiteDescription, canonical, "website", "index,follow", jsonLD), "Posts": cards, "Topics": options, "TopicID": topicID, "TopicName": topicName})
}

func (c *Controller) showPost(w http.ResponseWriter, r *http.Request) error {
	p, err := c.posts.PublicBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		return err
	}
	within := r.URL.Query().Get("topic") == p.TopicID.String() || strings.TrimSpace(r.URL.Query().Get("category")) == p.TopicName
	neighbors, err := c.posts.Adjacent(r.Context(), p, within)
	if err != nil {
		return err
	}
	query := ""
	if within {
		query = "?topic=" + p.TopicID.String()
	}
	canonical := c.config.PublicBaseURL + "/posts/" + p.Slug
	published := p.PublishedAt.UTC().Format(time.RFC3339)
	updated := p.UpdatedAt.UTC().Format(time.RFC3339)
	jsonLD := map[string]any{"@context": "https://schema.org", "@type": "BlogPosting", "headline": p.Title, "description": p.Description, "datePublished": published, "dateModified": updated, "mainEntityOfPage": canonical, "articleSection": p.TopicName, "author": map[string]string{"@type": "Person", "name": c.config.SiteName}}
	data := map[string]any{"SiteName": c.config.SiteName, "SEO": c.seo(p.Title+" — "+c.config.SiteName, p.Description, canonical, "article", "index,follow,max-image-preview:large", jsonLD), "Post": map[string]any{"Title": p.Title, "Topic": p.TopicName, "Date": kst(p.PublishedAt, "2006. 1. 2.")}, "BodyHTML": markdownutil.Render(p.ContentMarkdown), "PublishedISO": published, "UpdatedISO": updated, "UpdatedDisplay": kst(p.UpdatedAt, "2006-01-02 15:04:05"), "Previous": neighbors.Previous, "Next": neighbors.Next, "NavigationQuery": query, "ListURL": "/" + query}
	return c.render(w, "post.html", data)
}

func (c *Controller) aboutPage(w http.ResponseWriter, r *http.Request) error {
	page, err := c.about.Get(r.Context())
	if err != nil {
		return err
	}
	description := markdownutil.Excerpt(page.ContentMarkdown, 80)
	if description == "" {
		description = c.config.SiteDescription
	}
	canonical := c.config.PublicBaseURL + "/about"
	jsonLD := map[string]any{"@context": "https://schema.org", "@type": "ProfilePage", "name": page.Title, "url": canonical, "dateModified": page.UpdatedAt.UTC().Format(time.RFC3339), "mainEntity": map[string]string{"@type": "Person", "name": c.config.SiteName}}
	return c.render(w, "about.html", map[string]any{"SiteName": c.config.SiteName, "SEO": c.seo(page.Title+" — "+c.config.SiteName, description, canonical, "profile", "index,follow,max-image-preview:large", jsonLD), "Title": page.Title, "BodyHTML": markdownutil.Render(page.ContentMarkdown)})
}

func (c *Controller) loginPage(w http.ResponseWriter, r *http.Request) error {
	if _, ok := c.auth.Session(r); ok {
		redirect(w, r, "/admin")
		return nil
	}
	return c.render(w, "login.html", map[string]any{"SiteName": c.config.SiteName})
}
func (c *Controller) login(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if !c.auth.Verify(r.FormValue("username"), r.FormValue("password")) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		return c.render(w, "login.html", map[string]any{"SiteName": c.config.SiteName, "Error": "아이디 또는 비밀번호를 확인해주세요."})
	}
	if err := c.auth.Login(w); err != nil {
		return err
	}
	redirect(w, r, "/admin")
	return nil
}
func (c *Controller) requireAdmin(next app.AppHandler) app.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		session, ok := c.auth.Session(r)
		if !ok {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				redirect(w, r, "/admin/login")
				return nil
			}
			return apperr.New(apperr.Unauthorized, "")
		}
		return next(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, session)))
	}
}
func session(r *http.Request) auth.Session { return r.Context().Value(sessionKey{}).(auth.Session) }
func verifyCSRF(expected, actual string) error {
	if len(expected) != len(actual) || subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) != 1 {
		return apperr.New(apperr.Unauthorized, "")
	}
	return nil
}
func (c *Controller) logout(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if err := verifyCSRF(session(r).CSRF, r.FormValue("csrf_token")); err != nil {
		return err
	}
	c.auth.Logout(w)
	redirect(w, r, "/")
	return nil
}

func (c *Controller) adminIndex(w http.ResponseWriter, r *http.Request) error {
	drafts, err := c.posts.ListUnlinkedTemp(r.Context())
	if err != nil {
		return err
	}
	published, err := c.posts.ListAll(r.Context())
	if err != nil {
		return err
	}
	items := make([]map[string]string, 0, len(drafts)+len(published))
	for _, p := range drafts {
		title := p.Title
		if strings.TrimSpace(title) == "" {
			title = "제목 없는 글"
		}
		items = append(items, map[string]string{"Title": title, "Slug": p.Slug, "Status": "임시저장", "Updated": kst(p.UpdatedAt, "2006. 1. 2."), "EditURL": "/admin/temp-posts/" + p.ID.String() + "/edit", "DeleteURL": "/admin/temp-posts/" + p.ID.String() + "/delete"})
	}
	for _, p := range published {
		items = append(items, map[string]string{"Title": p.Title, "Slug": p.Slug, "Status": "공개", "Updated": kst(p.UpdatedAt, "2006. 1. 2."), "EditURL": "/admin/posts/" + p.ID.String() + "/edit", "DeleteURL": "/admin/posts/" + p.ID.String() + "/delete"})
	}
	return c.render(w, "admin.html", map[string]any{"PageTitle": "글 관리", "SiteName": c.config.SiteName, "CSRF": session(r).CSRF, "Posts": items})
}

func (c *Controller) topicsPage(w http.ResponseWriter, r *http.Request) error {
	items, err := c.topics.ListWithCounts(r.Context())
	if err != nil {
		return err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, t := range items {
		rows = append(rows, map[string]any{"ID": t.ID.String(), "Name": t.Name, "PostCount": t.PostCount, "TempCount": t.TempCount, "InUse": t.PostCount > 0 || t.TempCount > 0})
	}
	return c.render(w, "topics.html", map[string]any{"PageTitle": "주제 관리", "SiteName": c.config.SiteName, "CSRF": session(r).CSRF, "Topics": rows})
}
func (c *Controller) createTopic(w http.ResponseWriter, r *http.Request) error {
	return c.topicMutation(w, r, func(_ uuid.UUID, name string) error { _, err := c.topics.Create(r.Context(), name); return err })
}
func (c *Controller) updateTopic(w http.ResponseWriter, r *http.Request) error {
	return c.topicMutation(w, r, func(id uuid.UUID, name string) error { _, err := c.topics.Update(r.Context(), id, name); return err })
}
func (c *Controller) topicMutation(w http.ResponseWriter, r *http.Request, fn func(uuid.UUID, string) error) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if err := verifyCSRF(session(r).CSRF, r.FormValue("csrf_token")); err != nil {
		return err
	}
	id := uuid.Nil
	if r.PathValue("id") != "" {
		var err error
		id, err = parseUUID(r)
		if err != nil {
			return err
		}
	}
	if err := fn(id, r.FormValue("name")); err != nil {
		return err
	}
	redirect(w, r, "/admin/topics")
	return nil
}
func (c *Controller) deleteTopic(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if err := verifyCSRF(session(r).CSRF, r.FormValue("csrf_token")); err != nil {
		return err
	}
	id, err := parseUUID(r)
	if err != nil {
		return err
	}
	if err := c.topics.Delete(r.Context(), id); err != nil {
		return err
	}
	redirect(w, r, "/admin/topics")
	return nil
}

func (c *Controller) aboutEditor(w http.ResponseWriter, r *http.Request) error {
	page, err := c.about.Get(r.Context())
	if err != nil {
		return err
	}
	return c.render(w, "about_editor.html", map[string]any{"PageTitle": "소개 작성", "SiteName": c.config.SiteName, "BodyClass": "editor-body", "CSRF": session(r).CSRF, "Title": page.Title, "ContentMarkdown": page.ContentMarkdown, "MaxUploadBytes": c.config.MaxUploadBytes})
}
func (c *Controller) saveAbout(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if err := verifyCSRF(session(r).CSRF, r.FormValue("csrf_token")); err != nil {
		return err
	}
	if _, err := c.about.Update(r.Context(), r.FormValue("title"), r.FormValue("content_markdown")); err != nil {
		return err
	}
	redirect(w, r, "/about")
	return nil
}

func (c *Controller) newPost(w http.ResponseWriter, r *http.Request) error {
	p, err := c.posts.NewTemp(r.Context())
	if err != nil {
		return err
	}
	redirect(w, r, "/admin/temp-posts/"+p.ID.String()+"/edit")
	return nil
}
func (c *Controller) editPost(w http.ResponseWriter, r *http.Request) error {
	id, err := parseUUID(r)
	if err != nil {
		return err
	}
	p, err := c.posts.TempForPost(r.Context(), id)
	if err != nil {
		return err
	}
	return c.renderEditor(w, r, p)
}
func (c *Controller) editTemp(w http.ResponseWriter, r *http.Request) error {
	id, err := parseUUID(r)
	if err != nil {
		return err
	}
	p, err := c.posts.TempByID(r.Context(), id)
	if err != nil {
		return err
	}
	return c.renderEditor(w, r, p)
}
func (c *Controller) renderEditor(w http.ResponseWriter, r *http.Request, p post.TempPost) error {
	topics, err := c.topics.List(r.Context())
	if err != nil {
		return err
	}
	options := make([]map[string]any, 0, len(topics))
	for _, t := range topics {
		options = append(options, map[string]any{"ID": t.ID.String(), "Name": t.Name, "Selected": p.TopicID != nil && *p.TopicID == t.ID})
	}
	pageTitle := "새 글"
	if p.PostID != nil {
		pageTitle = "글 수정"
	}
	postData := map[string]any{"ID": p.ID.String(), "Title": p.Title, "Slug": p.Slug, "Description": p.Description, "DescriptionManual": p.DescriptionManual, "ContentMarkdown": p.ContentMarkdown, "HasPublicPost": p.PostID != nil}
	return c.render(w, "editor.html", map[string]any{"PageTitle": pageTitle, "SiteName": c.config.SiteName, "BodyClass": "editor-body", "CSRF": session(r).CSRF, "Action": "/admin/temp-posts/" + p.ID.String() + "/publish", "Post": postData, "Topics": options, "MaxUploadBytes": c.config.MaxUploadBytes})
}

func postForm(r *http.Request) post.Form {
	return post.Form{Title: r.FormValue("title"), Slug: r.FormValue("slug"), Description: r.FormValue("description"), DescriptionManual: r.FormValue("description_manual") == "true" || r.FormValue("description_manual") == "on", TopicID: r.FormValue("topic_id"), ContentMarkdown: r.FormValue("content_markdown"), CSRFToken: r.FormValue("csrf_token")}
}
func (c *Controller) withPostForm(w http.ResponseWriter, r *http.Request, fn func(uuid.UUID, post.Form) error, target string) error {
	if err := parseForm(r); err != nil {
		return err
	}
	form := postForm(r)
	if err := verifyCSRF(session(r).CSRF, form.CSRFToken); err != nil {
		return err
	}
	id, err := parseUUID(r)
	if err != nil {
		return err
	}
	if err := fn(id, form); err != nil {
		return err
	}
	redirect(w, r, target)
	return nil
}
func (c *Controller) saveTemp(w http.ResponseWriter, r *http.Request) error {
	return c.withPostForm(w, r, func(id uuid.UUID, f post.Form) error { _, err := c.posts.SaveTemp(r.Context(), id, f); return err }, "/admin/temp-posts/"+r.PathValue("id")+"/edit")
}
func (c *Controller) publish(w http.ResponseWriter, r *http.Request) error {
	return c.withPostForm(w, r, func(id uuid.UUID, f post.Form) error { _, err := c.posts.PublishTemp(r.Context(), id, f); return err }, "/admin")
}
func (c *Controller) autosave(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	form := postForm(r)
	if err := verifyCSRF(session(r).CSRF, form.CSRFToken); err != nil {
		return err
	}
	id, err := parseUUID(r)
	if err != nil {
		return err
	}
	saved, err := c.posts.SaveTemp(r.Context(), id, form)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"saved_at": saved.UpdatedAt.UTC().Format(time.RFC3339)})
}

func (c *Controller) markdownPreview(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if err := verifyCSRF(session(r).CSRF, r.FormValue("csrf_token")); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	return json.NewEncoder(w).Encode(map[string]string{"html": markdownutil.Render(r.FormValue("markdown"))})
}
func (c *Controller) deletePost(w http.ResponseWriter, r *http.Request) error {
	return c.deleteItem(w, r, c.posts.Delete)
}
func (c *Controller) deleteTemp(w http.ResponseWriter, r *http.Request) error {
	return c.deleteItem(w, r, c.posts.DeleteTemp)
}
func (c *Controller) deleteItem(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID) error) error {
	if err := parseForm(r); err != nil {
		return err
	}
	if err := verifyCSRF(session(r).CSRF, r.FormValue("csrf_token")); err != nil {
		return err
	}
	id, err := parseUUID(r)
	if err != nil {
		return err
	}
	if err := fn(r.Context(), id); err != nil {
		return err
	}
	redirect(w, r, "/admin")
	return nil
}

func (c *Controller) upload(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, c.config.MaxUploadBytes+65536)
	if err := r.ParseMultipartForm(c.config.MaxUploadBytes); err != nil {
		return apperr.New(apperr.Validation, "업로드 요청을 읽지 못했습니다.")
	}
	if err := verifyCSRF(session(r).CSRF, r.FormValue("csrf_token")); err != nil {
		return err
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		return apperr.New(apperr.Validation, "이미지를 선택해주세요.")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, c.config.MaxUploadBytes+1))
	if err != nil {
		return err
	}
	if len(data) == 0 || int64(len(data)) > c.config.MaxUploadBytes {
		return apperr.New(apperr.Validation, "이미지가 업로드 용량 제한을 초과했습니다.")
	}
	normalized, err := c.normalizeUpload(data, header)
	if err != nil {
		return err
	}
	if int64(len(normalized.Bytes)) > c.config.MaxUploadBytes {
		return apperr.New(apperr.Validation, "변환된 이미지가 업로드 용량 제한을 초과했습니다.")
	}
	id := uuid.New()
	filename := id.String() + "." + normalized.Extension
	path := filepath.Join(c.config.UploadDir, filename)
	if err := os.WriteFile(path, normalized.Bytes, 0600); err != nil {
		return err
	}
	if err := c.images.Register(r.Context(), id, filename, header.Filename, normalized.Mime, int64(len(normalized.Bytes))); err != nil {
		_ = os.Remove(path)
		return err
	}
	resultURL := "/uploads/" + filename
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"url": resultURL, "markdown": "![이미지](" + resultURL + ")"})
}
func (c *Controller) normalizeUpload(data []byte, header *multipart.FileHeader) (images.Normalized, error) {
	declared := header.Header.Get("Content-Type")
	isSVG := declared == "image/svg+xml" || strings.EqualFold(filepath.Ext(header.Filename), ".svg")
	if isSVG {
		clean, err := images.SanitizeSVG(data)
		return images.Normalized{Bytes: clean, Mime: "image/svg+xml", Extension: "svg"}, err
	}
	mime := http.DetectContentType(data)
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/gif" && mime != "image/webp" {
		return images.Normalized{}, apperr.New(apperr.Validation, "JPEG, PNG, GIF, WebP, SVG만 업로드할 수 있습니다.")
	}
	return images.NormalizeRaster(data, mime, c.config.ImageMaxDimension, c.config.ImageMaxPixels, c.config.ImageWebPQuality)
}

func (c *Controller) sitemap(w http.ResponseWriter, r *http.Request) error {
	posts, err := c.posts.ListPublic(r.Context(), nil)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>` + template.HTMLEscapeString(c.config.PublicBaseURL) + `/</loc></url><url><loc>` + template.HTMLEscapeString(c.config.PublicBaseURL) + `/about</loc></url>`)
	for _, p := range posts {
		fmt.Fprintf(&b, "<url><loc>%s/posts/%s</loc><lastmod>%s</lastmod></url>", template.HTMLEscapeString(c.config.PublicBaseURL), template.HTMLEscapeString(p.Slug), p.UpdatedAt.Format("2006-01-02"))
	}
	b.WriteString("</urlset>")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, err = io.WriteString(w, b.String())
	return err
}
func (c *Controller) robots(w http.ResponseWriter, _ *http.Request) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, err := fmt.Fprintf(w, "User-agent: *\nAllow: /\nAllow: /posts/\nAllow: /about\nDisallow: /admin\n\nSitemap: %s/sitemap.xml\n", c.config.PublicBaseURL)
	return err
}
func (c *Controller) live(w http.ResponseWriter, _ *http.Request) error {
	_, err := io.WriteString(w, "ok")
	return err
}
func (c *Controller) ready(w http.ResponseWriter, r *http.Request) error {
	if err := c.pool.Ping(r.Context()); err != nil {
		return err
	}
	_, err := io.WriteString(w, "ok")
	return err
}
