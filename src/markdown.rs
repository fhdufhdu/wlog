use ammonia::Builder;
use comrak::plugins::syntect::SyntectAdapter;
use comrak::{
    Arena, Options, markdown_to_html_with_plugins,
    nodes::NodeValue,
    options::{Plugins, RenderPlugins},
    parse_document,
};
use std::{collections::HashSet, sync::LazyLock};

static HIGHLIGHTER: LazyLock<SyntectAdapter> =
    LazyLock::new(|| SyntectAdapter::new(Some("base16-ocean.dark")));

pub fn render(markdown: &str) -> String {
    let mut options = Options::default();
    options.extension.strikethrough = true;
    options.extension.table = true;
    options.extension.tasklist = true;
    options.extension.footnotes = true;
    options.extension.math_dollars = true;
    options.extension.math_latex = true;
    options.extension.header_id_prefix = Some("section-".into());
    options.extension.header_id_prefix_in_href = true;
    options.render.hardbreaks = true;
    options.render.r#unsafe = true;
    let plugins = Plugins {
        render: RenderPlugins {
            codefence_syntax_highlighter: Some(&*HIGHLIGHTER),
            ..Default::default()
        },
    };
    let html = markdown_to_html_with_plugins(markdown, &options, &plugins);
    sanitize_html(&html)
}

pub fn sanitize_html(html: &str) -> String {
    let mut cleaner = Builder::default();
    cleaner
        .add_tags(&[
            "details",
            "summary",
            "figure",
            "figcaption",
            "mark",
            "kbd",
            "samp",
            "sub",
            "sup",
        ])
        .add_generic_attributes(&["class", "id", "title"])
        .add_tag_attributes("span", &["class", "style"])
        .add_tag_attributes("pre", &["class", "style"])
        .add_tag_attributes("span", &["data-math-style"])
        .add_tag_attributes("pre", &["data-math-style"])
        .add_tag_attributes("code", &["class", "data-math-style"])
        .add_tag_attributes("details", &["open"])
        .add_tag_attributes("img", &["loading", "decoding", "width", "height"])
        .add_tag_attributes("input", &["type", "checked", "disabled"]);
    for heading in ["h1", "h2", "h3", "h4", "h5", "h6"] {
        cleaner.add_tag_attributes(heading, &["id"]);
    }
    cleaner.clean(html).to_string()
}

pub fn excerpt(markdown: &str, limit: usize) -> String {
    let arena = Arena::new();
    let mut options = Options::default();
    options.extension.math_dollars = true;
    options.extension.math_latex = true;
    let root = parse_document(&arena, markdown, &options);
    let mut text = String::new();
    for node in root.descendants() {
        match &node.data().value {
            NodeValue::Text(value) => {
                text.push_str(value);
                text.push(' ');
            }
            NodeValue::Code(value) => {
                text.push_str(&value.literal);
                text.push(' ');
            }
            NodeValue::Math(value) => {
                text.push_str(&value.literal);
                text.push(' ');
            }
            NodeValue::SoftBreak | NodeValue::LineBreak => text.push(' '),
            _ => {}
        }
    }
    text.split_whitespace()
        .collect::<Vec<_>>()
        .join(" ")
        .chars()
        .take(limit)
        .collect()
}

pub fn upload_names(markdown: &str) -> HashSet<String> {
    let arena = Arena::new();
    let root = parse_document(&arena, markdown, &Options::default());
    root.descendants()
        .filter_map(|node| match &node.data().value {
            NodeValue::Image(link) => upload_name(&link.url),
            _ => None,
        })
        .collect()
}

fn upload_name(url: &str) -> Option<String> {
    let path = url.split(['?', '#']).next()?;
    let name = path.strip_prefix("/uploads/")?;
    if name.is_empty() || name.contains('/') || name.contains('\\') {
        return None;
    }
    Some(name.to_owned())
}

#[cfg(test)]
mod tests {
    #[test]
    fn blocks_raw_script_and_highlights_code() {
        let html = super::render("<script>alert(1)</script>\n```rust\nfn main() {}\n```");
        assert!(!html.contains("<script"));
        assert!(html.contains("<pre"));
        assert!(html.contains("class="));
    }

    #[test]
    fn sanitizes_client_rendered_html() {
        let html = super::sanitize_html(
            r#"<p><span class="hljs-keyword">fn</span></p><span data-math-style="inline">x</span><script>alert(1)</script>"#,
        );
        assert!(html.contains("hljs-keyword"));
        assert!(html.contains("data-math-style="));
        assert!(!html.contains("<script"));
    }

    #[test]
    fn creates_plain_text_excerpt_from_markdown() {
        let excerpt = super::excerpt("# 제목\n\n**강조한 본문**과 `코드`, $E=mc^2$입니다.", 80);
        assert_eq!(excerpt, "제목 강조한 본문 과 코드 , E=mc^2 입니다.");
    }

    #[test]
    fn preserves_mermaid_code_fence_for_client_rendering() {
        let html = super::render("```mermaid\ngraph LR\nA --> B\n```");
        assert!(html.contains("language-mermaid"));
        assert!(html.contains("graph LR"));
    }

    #[test]
    fn renders_a_single_newline_as_a_line_break() {
        let html = super::render("첫째 줄\n둘째 줄");
        assert!(html.contains("첫째 줄<br"));
        assert!(html.contains("둘째 줄"));
    }

    #[test]
    fn preserves_math_nodes_for_katex() {
        let html = super::render("인라인 $E = mc^2$\n\n$$\\int_0^1 x^2 dx$$");
        assert!(html.contains("data-math-style=\"inline\""));
        assert!(html.contains("data-math-style=\"display\""));
        assert!(html.contains("E = mc^2"));
    }

    #[test]
    fn allows_safe_html_and_removes_executable_html() {
        let html = super::render(
            "<details open onclick=\"alert(1)\"><summary>설명</summary><mark>본문</mark></details><a href=\"javascript:alert(2)\">위험한 링크</a><img src=\"/uploads/safe.png\" onerror=\"alert(3)\"><script>alert(4)</script>",
        );
        assert!(html.contains("<details open=\"\">"));
        assert!(html.contains("<summary>설명</summary>"));
        assert!(html.contains("<mark>본문</mark>"));
        assert!(!html.contains("onclick"));
        assert!(!html.contains("javascript:"));
        assert!(!html.contains("onerror"));
        assert!(!html.contains("<script"));
    }

    #[test]
    fn extracts_only_managed_markdown_images() {
        let names = super::upload_names(
            "![첫째](/uploads/one.webp)\n![외부](https://example.com/two.png)\n![잘못됨](/uploads/sub/three.png)\n![넷째](/uploads/four.jpg?size=2)",
        );
        assert_eq!(names.len(), 2);
        assert!(names.contains("one.webp"));
        assert!(names.contains("four.jpg"));
    }
}
