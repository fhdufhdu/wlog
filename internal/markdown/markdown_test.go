package markdown

import (
	"html"
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

func TestMathLatexCommandsAreNotConsumedByMarkdown(t *testing.T) {
	html := Render("- 암호화: $E = N^A\\;(mod\\;C)$\n\n$$x\\,y\\!z$$\n\n`WLOGMATHTOKEN0X`")
	for _, want := range []string{`E = N^A\;(mod\;C)`, `x\,y\!z`, `WLOGMATHTOKEN0X`} {
		if !strings.Contains(html, want) {
			t.Fatalf("LaTeX command %q was not preserved: %s", want, html)
		}
	}
}

func TestRenderPreservesSpacingEntities(t *testing.T) {
	rendered := html.UnescapeString(Render("&emsp;들여쓰기\n\n&nbsp;줄바꿈 방지\n\n&#12288;한글 한 칸"))
	for _, want := range []string{"\u2003들여쓰기", "\u00a0줄바꿈 방지", "\u3000한글 한 칸"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("spacing entity %q was not preserved: %s", want, rendered)
		}
	}
}
