package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/fhdufhdu/wlog/internal/auth"
	"github.com/fhdufhdu/wlog/internal/config"
)

func TestTemplatesParse(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })
	if _, err := NewController(config.Config{}, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestEditorTemplateIncludesPublishingControls(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })

	controller, err := NewController(config.Config{}, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	err = controller.render(response, "editor.html", map[string]any{
		"PageTitle":      "새 글",
		"SiteName":       "Wlog",
		"BodyClass":      "editor-body",
		"CSRF":           "csrf-token",
		"Action":         "/admin/temp-posts/draft-id/publish",
		"Post":           map[string]any{"ID": "draft-id", "Title": "", "Slug": "", "Description": "", "DescriptionManual": false, "ContentMarkdown": "", "HasPublicPost": false},
		"Topics":         []map[string]any{},
		"MaxUploadBytes": int64(10 << 20),
	})
	if err != nil {
		t.Fatal(err)
	}
	body := response.Body.String()
	for _, expected := range []string{
		`class="admin-body editor-body"`,
		`href="/">블로그 보기</a>`,
		`href="/admin/topics"`,
		`class="editor-action-buttons"`,
		`data-preview-url="/admin/markdown-preview"`,
		`data-insert="&amp;emsp;"`,
		`>한 칸 들여쓰기</button>`,
		`>줄바꿈 방지</button>`,
		`>한글 한 칸</button>`,
		`href="/styles.css?v=20260911-5"`,
		`src="/admin.js?v=20260911-6"`,
		`formaction="/admin/temp-posts/draft-id/save"`,
		`action="/admin/temp-posts/draft-id/publish"`,
		`>발행</button>`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("editor missing %q", expected)
		}
	}
}

func TestMutableStaticAssetsAlwaysRevalidate(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })

	handler := (&Controller{}).StaticRoutes().Apply()
	for _, path := range []string{"/admin.js?v=20260911-6", "/styles.css?v=20260911-5"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d", response.Code)
			}
			if got := response.Header().Get("Cache-Control"); got != "no-cache" {
				t.Errorf("Cache-Control = %q", got)
			}
		})
	}
}

func TestMarkdownPreviewUsesServerRenderer(t *testing.T) {
	form := url.Values{
		"csrf_token": {"csrf-token"},
		"markdown":   {"$E = N^A\\;(mod\\;C)$\n\n<script>alert(1)</script>"},
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/markdown-preview", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(context.WithValue(request.Context(), sessionKey{}, auth.Session{CSRF: "csrf-token"}))
	response := httptest.NewRecorder()

	if err := (&Controller{}).markdownPreview(response, request); err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload["html"], `E = N^A\;(mod\;C)`) {
		t.Fatalf("LaTeX command was not preserved: %s", payload["html"])
	}
	if strings.Contains(payload["html"], "<script") {
		t.Fatalf("preview HTML was not sanitized: %s", payload["html"])
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q", got)
	}
}

func TestRobotsAllowsPublicPagesAndAdvertisesSitemap(t *testing.T) {
	c := &Controller{config: config.Config{PublicBaseURL: "https://example.com"}}
	response := httptest.NewRecorder()
	if err := c.robots(response, httptest.NewRequest("GET", "/robots.txt", nil)); err != nil {
		t.Fatal(err)
	}
	body := response.Body.String()
	for _, expected := range []string{"User-agent: *", "Allow: /posts/", "Allow: /about", "Disallow: /admin", "Sitemap: https://example.com/sitemap.xml"} {
		if !strings.Contains(body, expected) {
			t.Errorf("robots.txt missing %q: %s", expected, body)
		}
	}
	if got := response.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
}
