const KATEX_MODULE = "https://cdn.jsdelivr.net/npm/katex@0.18.5/dist/katex.mjs";

let katexPromise;

function loadKatex() {
  if (!katexPromise) {
    katexPromise = import(KATEX_MODULE).then(({ default: katex }) => katex);
  }
  return katexPromise;
}

async function renderMath(root = document) {
  if (!root?.querySelectorAll) return;
  const nodes = [...root.querySelectorAll("[data-math-style]:not([data-math-rendered])")];
  if (!nodes.length) return;

  let katex;
  try {
    katex = await loadKatex();
  } catch (error) {
    console.error("KaTeX 모듈을 불러오지 못했습니다.", error);
    nodes.forEach((node) => node.classList.add("math-error"));
    return;
  }

  nodes.forEach((node) => {
    const source = node.textContent;
    const displayMode = node.dataset.mathStyle === "display";
    node.classList.toggle("math-display", displayMode);
    try {
      katex.render(source, node, {
        displayMode,
        output: "htmlAndMathml",
        throwOnError: true,
        trust: false,
        strict: "warn",
      });
      node.dataset.mathRendered = "true";
    } catch (error) {
      node.classList.add("math-error");
      node.title = error.message || "LaTeX 문법을 확인해주세요.";
    }
  });
}

window.addEventListener("wlog:markdown-rendered", (event) => {
  renderMath(event.detail?.root || document);
});

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => renderMath());
} else {
  renderMath();
}
