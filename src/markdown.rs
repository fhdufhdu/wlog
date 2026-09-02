use ammonia::Builder;
use comrak::plugins::syntect::SyntectAdapter;
use comrak::{
    Arena, Options, markdown_to_html_with_plugins,
    nodes::NodeValue,
    options::{Plugins, RenderPlugins},
    parse_document,
};

pub fn render(markdown: &str) -> String {
    let mut options = Options::default();
    options.extension.strikethrough = true;
    options.extension.table = true;
    options.extension.tasklist = true;
    options.extension.footnotes = true;
    options.extension.header_id_prefix = Some("section-".into());
    options.extension.header_id_prefix_in_href = true;
    options.render.hardbreaks = true;
    options.render.r#unsafe = false;
    let highlighter = SyntectAdapter::new(Some("base16-ocean.dark"));
    let plugins = Plugins {
        render: RenderPlugins {
            codefence_syntax_highlighter: Some(&highlighter),
            ..Default::default()
        },
    };
    let html = markdown_to_html_with_plugins(markdown, &options, &plugins);
    let mut cleaner = Builder::default();
    cleaner
        .add_tag_attributes("span", &["class", "style"])
        .add_tag_attributes("pre", &["class", "style"])
        .add_tag_attributes("code", &["class"])
        .add_tag_attributes("img", &["loading"])
        .add_tag_attributes("input", &["type", "checked", "disabled"]);
    for heading in ["h1", "h2", "h3", "h4", "h5", "h6"] {
        cleaner.add_tag_attributes(heading, &["id"]);
    }
    cleaner.clean(&html).to_string()
}

pub fn excerpt(markdown: &str, limit: usize) -> String {
    let arena = Arena::new();
    let root = parse_document(&arena, markdown, &Options::default());
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
    fn creates_plain_text_excerpt_from_markdown() {
        let excerpt = super::excerpt("# 제목\n\n**강조한 본문**과 `코드`입니다.", 80);
        assert_eq!(excerpt, "제목 강조한 본문 과 코드 입니다.");
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
}
