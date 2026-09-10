package markdown

import (
	"slices"
	"strings"
	"testing"
)

func TestRenderSanitizesAndHighlights(t *testing.T) {
	html := Render("<script>alert(1)</script>\n```go\nfunc main() {}\n```")
	if strings.Contains(html, "<script") {
		t.Fatal("script element survived sanitization")
	}
	if !strings.Contains(html, "<pre") || !strings.Contains(html, "<code") {
		t.Fatalf("code block missing: %s", html)
	}
}
func TestExcerptAndUploadNames(t *testing.T) {
	if got := Excerpt("# 제목\n\n**강조한 본문**과 `코드`, $E=mc^2$", 80); got != "제목 강조한 본문과 코드, E=mc^2" {
		t.Fatalf("excerpt = %q", got)
	}
	got := UploadNames("![a](/uploads/one.webp) ![x](https://example.com/x.png) ![b](/uploads/two.png?q=1)")
	if !slices.Equal(got, []string{"one.webp", "two.png"}) {
		t.Fatalf("names = %#v", got)
	}
}

func TestMathAndMermaidHooksArePreserved(t *testing.T) {
	html := Render("인라인 $E=mc^2$\n\n```mermaid\ngraph LR\nA --> B\n```")
	if !strings.Contains(html, `data-math-style="inline"`) || !strings.Contains(html, `class="language-mermaid"`) {
		t.Fatalf("client hooks missing: %s", html)
	}
}
