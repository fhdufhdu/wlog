package web

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
		`href="/admin/topics"`,
		`class="editor-action-buttons"`,
		`formaction="/admin/temp-posts/draft-id/save"`,
		`action="/admin/temp-posts/draft-id/publish"`,
		`>발행</button>`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("editor missing %q", expected)
		}
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
