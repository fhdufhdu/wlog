package markdown

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
)

var (
	imagePattern         = regexp.MustCompile(`!\[[^\]]*\]\(([^\s)]+)(?:\s+"[^"]*")?\)`)
	markupPattern        = regexp.MustCompile("!\\[[^\\]]*\\]\\([^)]*\\)|\\[([^\\]]+)\\]\\([^)]*\\)|<[^>]+>|[#>*_~|$]")
	spacePattern         = regexp.MustCompile(`\s+`)
	fencedCodePattern    = regexp.MustCompile("(?s)```[^\\n]*\\n(.*?)```")
	inlineCodePattern    = regexp.MustCompile("`([^`\\n]*)`")
	protectedCodePattern = regexp.MustCompile("(?s)```.*?```|`[^`\\n]*`")
	mermaidPattern       = regexp.MustCompile("(?ms)^```mermaid[ \\t]*\\n(.*?)^```[ \\t]*$")
	displayMathPattern   = regexp.MustCompile(`(?s)\$\$(.+?)\$\$`)
	inlineMathPattern    = regexp.MustCompile(`\$([^$\n]+)\$`)
)

func Render(source string) string {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, highlighting.NewHighlighting(highlighting.WithStyle("base16-snazzy"))),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(rendererhtml.WithUnsafe(), rendererhtml.WithHardWraps()),
	)
	var out bytes.Buffer
	mathTokenPrefix := "WLOGMATHTOKEN"
	for strings.Contains(source, mathTokenPrefix) {
		mathTokenPrefix += "X"
	}
	protected := []string{}
	source = protectedCodePattern.ReplaceAllStringFunc(source, func(value string) string {
		token := fmt.Sprintf("WLOGCODETOKEN%dX", len(protected))
		protected = append(protected, value)
		return token
	})
	protectedMath := []string{}
	source = displayMathPattern.ReplaceAllStringFunc(source, func(value string) string {
		match := displayMathPattern.FindStringSubmatch(value)
		token := fmt.Sprintf("%s%dX", mathTokenPrefix, len(protectedMath))
		protectedMath = append(protectedMath, `<span data-math-style="display">`+html.EscapeString(match[1])+`</span>`)
		return token
	})
	source = inlineMathPattern.ReplaceAllStringFunc(source, func(value string) string {
		match := inlineMathPattern.FindStringSubmatch(value)
		token := fmt.Sprintf("%s%dX", mathTokenPrefix, len(protectedMath))
		protectedMath = append(protectedMath, `<span data-math-style="inline">`+html.EscapeString(match[1])+`</span>`)
		return token
	})
	for index, value := range protected {
		source = strings.ReplaceAll(source, fmt.Sprintf("WLOGCODETOKEN%dX", index), value)
	}
	source = mermaidPattern.ReplaceAllStringFunc(source, func(block string) string {
		match := mermaidPattern.FindStringSubmatch(block)
		return `<pre><code class="language-mermaid">` + html.EscapeString(match[1]) + `</code></pre>`
	})
	if err := md.Convert([]byte(source), &out); err != nil {
		return ""
	}
	rendered := out.String()
	for index, value := range protectedMath {
		rendered = strings.ReplaceAll(rendered, fmt.Sprintf("%s%dX", mathTokenPrefix, index), value)
	}
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("details", "summary", "figure", "figcaption", "mark", "kbd", "samp", "sub", "sup", "input")
	policy.AllowAttrs("class", "id", "title").Globally()
	policy.AllowAttrs("style").OnElements("span", "pre")
	policy.AllowAttrs("data-math-style").OnElements("span", "pre", "code")
	policy.AllowAttrs("open").OnElements("details")
	policy.AllowAttrs("loading", "decoding", "width", "height").OnElements("img")
	policy.AllowAttrs("type", "checked", "disabled").OnElements("input")
	return policy.Sanitize(rendered)
}

func Excerpt(source string, limit int) string {
	source = fencedCodePattern.ReplaceAllString(source, "$1")
	source = inlineCodePattern.ReplaceAllString(source, "$1")
	text := markupPattern.ReplaceAllString(source, "$1")
	text = html.UnescapeString(spacePattern.ReplaceAllString(strings.TrimSpace(text), " "))
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit])
}

func UploadNames(source string) []string {
	unique := map[string]struct{}{}
	for _, match := range imagePattern.FindAllStringSubmatch(source, -1) {
		path := strings.SplitN(strings.SplitN(match[1], "?", 2)[0], "#", 2)[0]
		name, ok := strings.CutPrefix(path, "/uploads/")
		if ok && name != "" && !strings.ContainsAny(name, "/\\") {
			unique[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
