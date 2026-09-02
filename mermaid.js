const MERMAID_MODULE = "https://cdn.jsdelivr.net/npm/mermaid@11.17.2/dist/mermaid.esm.min.mjs";

let mermaidPromise;
let renderQueue = Promise.resolve();

function loadMermaid() {
  if (!mermaidPromise) {
    mermaidPromise = import(MERMAID_MODULE).then(({ default: mermaid }) => {
      const dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: dark ? "dark" : "base",
        fontFamily: "SUIT, system-ui, sans-serif",
        suppressErrorRendering: true,
        themeVariables: dark ? undefined : {
          primaryColor: "#dff3f8",
          primaryTextColor: "#1b1d20",
          primaryBorderColor: "#117696",
          lineColor: "#626974",
          secondaryColor: "#f2f4f7",
          tertiaryColor: "#fafbfc",
          fontSize: "15px",
        },
      });
      return mermaid;
    });
  }
  return mermaidPromise;
}

function collectDiagrams(root) {
  return [...root.querySelectorAll("pre > code.language-mermaid")].map((code) => {
    const source = code.textContent;
    const diagram = document.createElement("div");
    diagram.className = "mermaid mermaid-loading";
    diagram.setAttribute("aria-busy", "true");
    diagram.textContent = source;
    code.parentElement.replaceWith(diagram);
    return { diagram, source };
  });
}

function showDiagramError(diagram, source) {
  const error = document.createElement("div");
  error.className = "mermaid-error";
  const message = document.createElement("p");
  message.textContent = "Mermaid 문법을 확인해주세요.";
  const pre = document.createElement("pre");
  const code = document.createElement("code");
  code.className = "language-mermaid";
  code.textContent = source;
  pre.append(code);
  error.append(message, pre);
  diagram.replaceWith(error);
}

async function renderRoot(root) {
  if (!root?.querySelectorAll) return;
  const diagrams = collectDiagrams(root);
  if (!diagrams.length) return;

  let mermaid;
  try {
    mermaid = await loadMermaid();
  } catch (error) {
    console.error("Mermaid 모듈을 불러오지 못했습니다.", error);
    diagrams.forEach(({ diagram, source }) => showDiagramError(diagram, source));
    return;
  }

  for (const { diagram, source } of diagrams) {
    if (!diagram.isConnected) continue;
    try {
      await mermaid.parse(source, { suppressErrors: true });
      await mermaid.run({ nodes: [diagram], suppressErrors: true });
      diagram.classList.remove("mermaid-loading");
      diagram.removeAttribute("aria-busy");
    } catch (error) {
      console.error("Mermaid 다이어그램을 렌더링하지 못했습니다.", error);
      showDiagramError(diagram, source);
    }
  }
}

function renderMermaid(root = document) {
  renderQueue = renderQueue.then(() => renderRoot(root));
  return renderQueue;
}

window.addEventListener("wlog:markdown-rendered", (event) => {
  renderMermaid(event.detail?.root || document);
});

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => renderMermaid());
} else {
  renderMermaid();
}
