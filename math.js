const KATEX_MODULE = "https://cdn.jsdelivr.net/npm/katex@0.18.5/dist/katex.mjs";
const KATEX_STYLES = "https://cdn.jsdelivr.net/npm/katex@0.18.5/dist/katex.css";
const KATEX_STYLES_INTEGRITY = "sha384-zfuvpaZ3OjmRVaLPkCCrb0YbSTd5NgPmLIum+Cmsr06+WhmECuyum4BEipwHyWW4";

let katexPromise;
let katexStylesPromise;

function loadKatex() {
  if (!katexPromise) {
    katexPromise = import(KATEX_MODULE).then(({ default: katex }) => katex);
  }
  return katexPromise;
}

function loadKatexStyles() {
  if (!katexStylesPromise) {
    katexStylesPromise = new Promise((resolve, reject) => {
      const existing = document.querySelector(`link[href="${KATEX_STYLES}"]`);
      if (existing) {
        resolve();
        return;
      }
      const link = document.createElement("link");
      link.rel = "stylesheet";
      link.href = KATEX_STYLES;
      link.integrity = KATEX_STYLES_INTEGRITY;
      link.crossOrigin = "anonymous";
      link.addEventListener("load", resolve, { once: true });
      link.addEventListener("error", () => reject(new Error("KaTeX 스타일을 불러오지 못했습니다.")), { once: true });
      document.head.append(link);
    });
  }
  return katexStylesPromise;
}

async function renderMath(root = document) {
  if (!root?.querySelectorAll) return;
  const nodes = [...root.querySelectorAll("[data-math-style]:not([data-math-rendered])")];
  if (!nodes.length) return;

  let katex;
  try {
    [katex] = await Promise.all([loadKatex(), loadKatexStyles()]);
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
