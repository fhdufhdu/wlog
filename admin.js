import markdownit from "https://cdn.jsdelivr.net/npm/markdown-it@15.0.0/+esm";
import taskLists from "https://cdn.jsdelivr.net/npm/markdown-it-task-lists@2.1.1/+esm";
import footnote from "https://cdn.jsdelivr.net/npm/markdown-it-footnote@4.0.0/+esm";
import DOMPurify from "https://cdn.jsdelivr.net/npm/dompurify@3.4.14/+esm";
import hljs from "https://cdn.jsdelivr.net/npm/highlight.js@11.12.0/lib/common/+esm";

const form = document.querySelector("#editor-form");
const upload = document.querySelector("#image-upload");
const editor = document.querySelector("#content_markdown");
const uploadStatus = document.querySelector("#upload-status");
const saveStatus = document.querySelector("#save-status");
const title = document.querySelector("#title");
const slug = document.querySelector("#slug");
const description = document.querySelector("#description");
const descriptionCount = document.querySelector("#description-count");
const descriptionManual = document.querySelector("#description-manual");
const writeTab = document.querySelector("#write-tab");
const previewTab = document.querySelector("#preview-tab");
const previewBody = document.querySelector("#preview-body");
const previewTitle = document.querySelector("#preview-title");
const csrfToken = document.querySelector("#csrf-token");
const contentHtml = document.querySelector("#content-html");
const writePane = document.querySelector("#write-panel");
const operationDialog = document.querySelector("#operation-dialog");
const operationTitle = document.querySelector("#operation-title");
const operationDetail = document.querySelector("#operation-detail");

let dirty = false;
let changeVersion = 0;
let descriptionEdited = descriptionManual?.value === "true";
let autosaveController = null;
let previewTimer = null;
let isComposing = false;
let lastPreviewMarkdown = null;
let editorResizeFrame = null;
let submitting = false;
const autosaveUrl = form?.dataset.autosaveUrl;
const previewDelay = 120;
const editorHeightBuffer = 48;

function showOperation(titleText, detailText) {
  if (!operationDialog) return;
  if (operationTitle) operationTitle.textContent = titleText;
  if (operationDetail) operationDetail.textContent = detailText;
  document.body.setAttribute("aria-busy", "true");
  if (!operationDialog.open) operationDialog.showModal();
}

function hideOperation() {
  document.body.removeAttribute("aria-busy");
  if (operationDialog?.open) operationDialog.close();
}

operationDialog?.addEventListener("cancel", (event) => {
  event.preventDefault();
});

function mathPlugin(md) {
  md.inline.ruler.after("escape", "math_inline", (state, silent) => {
    const start = state.pos;
    if (state.src[start] !== "$" || state.src[start + 1] === "$") return false;
    let end = start + 1;
    while ((end = state.src.indexOf("$", end)) !== -1) {
      let escapes = 0;
      for (let index = end - 1; index > start && state.src[index] === "\\"; index -= 1) escapes += 1;
      if (escapes % 2 === 0) break;
      end += 1;
    }
    if (end === -1 || end === start + 1 || state.src.slice(start + 1, end).includes("\n")) return false;
    if (!silent) {
      const token = state.push("math_inline", "span", 0);
      token.content = state.src.slice(start + 1, end);
    }
    state.pos = end + 1;
    return true;
  });

  md.block.ruler.after("blockquote", "math_block", (state, startLine, endLine, silent) => {
    const start = state.bMarks[startLine] + state.tShift[startLine];
    const firstLine = state.src.slice(start, state.eMarks[startLine]);
    if (!firstLine.startsWith("$$")) return false;
    if (silent) return true;

    const lines = [];
    const opening = firstLine.slice(2);
    if (opening.trimEnd().endsWith("$$")) {
      lines.push(opening.trimEnd().slice(0, -2));
      state.line = startLine + 1;
    } else {
      if (opening) lines.push(opening);
      let line = startLine + 1;
      for (; line < endLine; line += 1) {
        const value = state.src.slice(state.bMarks[line] + state.tShift[line], state.eMarks[line]);
        if (value.trimEnd().endsWith("$$")) {
          lines.push(value.trimEnd().slice(0, -2));
          line += 1;
          break;
        }
        lines.push(value);
      }
      state.line = line;
    }
    const token = state.push("math_block", "span", 0);
    token.block = true;
    token.content = lines.join("\n").trim();
    return true;
  });

  md.renderer.rules.math_inline = (tokens, index) =>
    `<span data-math-style="inline">${md.utils.escapeHtml(tokens[index].content)}</span>`;
  md.renderer.rules.math_block = (tokens, index) =>
    `<span data-math-style="display">${md.utils.escapeHtml(tokens[index].content)}</span>\n`;
}

function headingIdPlugin(md) {
  md.renderer.rules.heading_open = (tokens, index, options, environment, renderer) => {
    const text = tokens[index + 1]?.content || "heading";
    const base = text
      .normalize("NFKC")
      .toLowerCase()
      .replace(/[^\p{Letter}\p{Number}]+/gu, "-")
      .replace(/^-|-$/g, "") || "heading";
    const headingIds = environment.headingIds || (environment.headingIds = new Map());
    const sequence = (headingIds.get(base) || 0) + 1;
    headingIds.set(base, sequence);
    tokens[index].attrSet("id", `section-${base}${sequence > 1 ? `-${sequence}` : ""}`);
    return renderer.renderToken(tokens, index, options);
  };
}

const markdown = markdownit({
  html: true,
  breaks: true,
  linkify: true,
  highlight(source, language) {
    if (language === "mermaid") return markdown.utils.escapeHtml(source);
    if (language && hljs.getLanguage(language)) {
      return hljs.highlight(source, { language, ignoreIllegals: true }).value;
    }
    return markdown.utils.escapeHtml(source);
  },
}).use(taskLists, { enabled: true }).use(footnote).use(mathPlugin).use(headingIdPlugin);

const renderImage = markdown.renderer.rules.image;
markdown.renderer.rules.image = (tokens, index, options, environment, renderer) => {
  tokens[index].attrSet("loading", "lazy");
  tokens[index].attrSet("decoding", "async");
  return renderImage(tokens, index, options, environment, renderer);
};

function renderMarkdown(source) {
  return DOMPurify.sanitize(markdown.render(source, {}), {
    USE_PROFILES: { html: true },
    ADD_TAGS: ["details", "summary", "figure", "figcaption", "mark", "kbd", "samp"],
    ADD_ATTR: ["data-math-style", "loading", "decoding", "width", "height"],
  });
}

function resizeEditor() {
  if (!editor) return;
  window.cancelAnimationFrame(editorResizeFrame);
  editorResizeFrame = window.requestAnimationFrame(() => {
    editor.style.height = "0px";
    editor.style.height = `${editor.scrollHeight + editorHeightBuffer}px`;
  });
}

function setSaveStatus(message, state = "") {
  if (!saveStatus) return;
  saveStatus.textContent = message;
  saveStatus.dataset.state = state;
}

function markdownExcerpt(value, limit = 80) {
  const plainText = value
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/[`*_>#~-]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
  return Array.from(plainText).slice(0, limit).join("");
}

function updateDescription() {
  if (!description || descriptionEdited) return;
  description.value = markdownExcerpt(editor?.value || "");
  if (descriptionCount) descriptionCount.textContent = Array.from(description.value).length;
}

function updatePreviewMeta() {
  if (previewTitle) previewTitle.textContent = title?.value.trim() || "제목을 입력하세요";
}

function markDirty() {
  dirty = true;
  changeVersion += 1;
  setSaveStatus("저장되지 않은 변경사항", "dirty");
}

function showPreviewMessage(message, className = "preview-empty") {
  if (!previewBody) return;
  const element = document.createElement("p");
  element.className = className;
  element.textContent = message;
  previewBody.replaceChildren(element);
}

function requestPreview() {
  if (!previewBody || !editor || !contentHtml) return;
  window.clearTimeout(previewTimer);
  const markdown = editor.value;
  if (markdown === lastPreviewMarkdown) return;

  if (!markdown.trim()) {
    lastPreviewMarkdown = markdown;
    contentHtml.value = "";
    showPreviewMessage("본문을 입력하면 여기에 미리보기가 표시됩니다.");
    return;
  }

  const html = renderMarkdown(markdown);
  lastPreviewMarkdown = markdown;
  contentHtml.value = html;
  previewBody.innerHTML = html;
  window.requestAnimationFrame(() => {
    window.dispatchEvent(new CustomEvent("wlog:markdown-rendered", {
      detail: { root: previewBody },
    }));
  });
}

function schedulePreview(delay = previewDelay) {
  if (isComposing) return;
  window.clearTimeout(previewTimer);
  previewTimer = window.setTimeout(requestPreview, delay);
}

function selectEditorTab(tab, focusTab = false) {
  const preview = tab === "preview";
  if (form) form.dataset.mobileView = preview ? "preview" : "write";
  writeTab?.classList.toggle("is-current", !preview);
  previewTab?.classList.toggle("is-current", preview);
  writeTab?.setAttribute("aria-selected", String(!preview));
  previewTab?.setAttribute("aria-selected", String(preview));
  writeTab?.setAttribute("tabindex", preview ? "-1" : "0");
  previewTab?.setAttribute("tabindex", preview ? "0" : "-1");
  if (focusTab) (preview ? previewTab : writeTab)?.focus();
  if (preview) requestPreview();
}

if (title && slug) {
  title.addEventListener("input", () => {
    if (slug.dataset.edited === "true") return;
    slug.value = title.value.trim().toLowerCase()
      .replace(/[^a-z0-9가-힣]+/g, "-").replace(/^-|-$/g, "");
  });
  slug.addEventListener("input", () => { slug.dataset.edited = "true"; });
}

description?.addEventListener("input", () => {
  descriptionEdited = true;
  descriptionManual.value = "true";
  if (descriptionCount) descriptionCount.textContent = Array.from(description.value).length;
});

editor?.addEventListener("compositionstart", () => { isComposing = true; });
editor?.addEventListener("compositionend", () => {
  isComposing = false;
  updateDescription();
  updatePreviewMeta();
  schedulePreview(0);
});
editor?.addEventListener("input", () => {
  resizeEditor();
  updateDescription();
  schedulePreview();
});
window.addEventListener("resize", resizeEditor, { passive: true });

form?.addEventListener("input", () => {
  markDirty();
  updatePreviewMeta();
});

writeTab?.addEventListener("click", () => selectEditorTab("write"));
previewTab?.addEventListener("click", () => selectEditorTab("preview"));
[writeTab, previewTab].forEach((tab) => {
  tab?.addEventListener("keydown", (event) => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    selectEditorTab(tab === writeTab ? "preview" : "write", true);
  });
});

function replaceEditorSelection(replacement, selectionStart, selectionEnd) {
  if (!editor) return;
  const start = editor.selectionStart;
  const end = editor.selectionEnd;
  let inputObserved = false;
  editor.addEventListener("input", () => { inputObserved = true; }, { once: true });
  editor.focus();
  editor.setSelectionRange(start, end);

  let insertedWithHistory = false;
  try {
    insertedWithHistory = document.execCommand("insertText", false, replacement);
  } catch (_) {
    insertedWithHistory = false;
  }
  if (!insertedWithHistory) editor.setRangeText(replacement, start, end, "end");

  if (selectionStart !== undefined && selectionEnd !== undefined) {
    editor.setSelectionRange(start + selectionStart, start + selectionEnd);
  }
  editor.focus();
  if (!inputObserved) editor.dispatchEvent(new Event("input", { bubbles: true }));
}

function wrapSelection(prefix, suffix, placeholder) {
  const selected = editor.value.slice(editor.selectionStart, editor.selectionEnd) || placeholder;
  replaceEditorSelection(`${prefix}${selected}${suffix}`, prefix.length, prefix.length + selected.length);
}

function prefixLines(prefix, placeholder) {
  const selected = editor.value.slice(editor.selectionStart, editor.selectionEnd) || placeholder;
  const replacement = selected.split("\n").map((line) => `${prefix}${line}`).join("\n");
  replaceEditorSelection(replacement, prefix.length, replacement.length);
}

document.querySelectorAll("[data-markdown]").forEach((button) => {
  button.addEventListener("click", () => {
    const action = button.dataset.markdown;
    if (action === "h1") prefixLines("# ", "제목");
    if (action === "h2") prefixLines("## ", "제목");
    if (action === "h3") prefixLines("### ", "제목");
    if (action === "bold") wrapSelection("**", "**", "굵은 글씨");
    if (action === "italic") wrapSelection("_", "_", "기울임 글씨");
    if (action === "quote") prefixLines("> ", "인용문");
    if (action === "link") wrapSelection("[", "](https://)", "링크 이름");
    if (action === "code") {
      const selected = editor.value.slice(editor.selectionStart, editor.selectionEnd) || "코드";
      if (selected.includes("\n")) wrapSelection("```\n", "\n```", "코드");
      else wrapSelection("`", "`", "코드");
    }
    if (action === "mermaid") {
      wrapSelection("```mermaid\n", "\n```", "flowchart LR\n  A[시작] --> B[끝]");
    }
    if (action === "math") {
      const selected = editor.value.slice(editor.selectionStart, editor.selectionEnd) || "E = mc^2";
      if (selected.includes("\n")) wrapSelection("$$\n", "\n$$", "E = mc^2");
      else wrapSelection("$", "$", "E = mc^2");
    }
  });
});

async function autosave() {
  if (!dirty || !form || !autosaveUrl || autosaveController) return;
  requestPreview();
  const savingVersion = changeVersion;
  autosaveController = new AbortController();
  setSaveStatus("저장 중…", "saving");
  const body = new URLSearchParams();
  for (const [key, value] of new FormData(form)) {
    if (typeof value === "string") body.append(key, value);
  }
  try {
    const response = await fetch(autosaveUrl, {
      method: "POST",
      body,
      credentials: "same-origin",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      signal: autosaveController.signal,
    });
    if (!response.ok) throw new Error(await response.text());
    const result = await response.json();
    if (contentHtml && contentHtml.value !== result.content_html) {
      contentHtml.value = result.content_html;
      if (previewBody) {
        previewBody.innerHTML = result.content_html;
        window.dispatchEvent(new CustomEvent("wlog:markdown-rendered", {
          detail: { root: previewBody },
        }));
      }
    }
    dirty = changeVersion !== savingVersion;
    if (dirty) {
      setSaveStatus("새 변경사항 저장 대기", "dirty");
    } else {
      const savedAt = new Date(result.saved_at).toLocaleTimeString("ko-KR", {
        hour: "numeric",
        minute: "2-digit",
      });
      setSaveStatus(`${savedAt} 임시저장됨`, "saved");
    }
  } catch (error) {
    if (error.name !== "AbortError") {
      setSaveStatus(`자동저장 실패 · ${error.message || "다시 시도해주세요."}`, "error");
    }
  } finally {
    autosaveController = null;
  }
}

if (autosaveUrl) window.setInterval(autosave, 10_000);

form?.addEventListener("invalid", (event) => {
  const control = event.target;
  if (!(control instanceof HTMLElement)) return;
  if (writePane?.contains(control)) selectEditorTab("write");
  const settings = control.closest("details");
  if (settings) settings.open = true;
  setSaveStatus("필수 항목을 확인해주세요.", "error");
}, true);

form?.addEventListener("submit", (event) => {
  submitting = true;
  requestPreview();
  autosaveController?.abort();
  window.clearTimeout(previewTimer);
  form.querySelectorAll("button[type='submit']").forEach((button) => { button.disabled = true; });
  setSaveStatus("저장 중…", "saving");
  dirty = false;
  const submitter = event.submitter;
  showOperation(
    submitter?.dataset.loadingTitle || "내용을 저장하고 있습니다",
    submitter?.dataset.loadingDetail || "처리가 끝날 때까지 잠시만 기다려주세요.",
  );
});

window.addEventListener("beforeunload", (event) => {
  if (submitting) return;
  if (!dirty && !autosaveController) return;
  event.preventDefault();
  event.returnValue = "";
});

async function uploadImage(file) {
  const data = new FormData();
  data.append("csrf_token", csrfToken.value);
  data.append("image", file);
  const response = await fetch("/admin/uploads", {
    method: "POST",
    body: data,
    credentials: "same-origin",
  });
  if (!response.ok) throw new Error(await response.text());
  return response.json();
}

async function uploadImageFiles(files, concurrency = 3) {
  const results = new Array(files.length);
  let cursor = 0;
  let completed = 0;

  async function worker() {
    while (cursor < files.length) {
      const index = cursor;
      cursor += 1;
      try {
        results[index] = { result: await uploadImage(files[index]) };
      } catch (error) {
        results[index] = { error };
      }
      completed += 1;
      if (uploadStatus) uploadStatus.textContent = `사진 ${completed}/${files.length}장을 처리했습니다…`;
      showOperation(
        "사진을 올리고 있습니다",
        `${completed}/${files.length}장 처리됨 · 이미지 크기와 형식을 최적화하고 있습니다.`,
      );
    }
  }

  const workerCount = Math.min(concurrency, files.length);
  await Promise.all(Array.from({ length: workerCount }, () => worker()));
  return results;
}

async function handleImageFiles(fileList) {
  if (!editor || !csrfToken) return;
  const files = [...fileList].filter((file) => file.type.startsWith("image/"));
  if (!files.length) {
    if (uploadStatus) uploadStatus.textContent = "JPEG, PNG, GIF, WebP, SVG 파일만 추가할 수 있습니다.";
    return;
  }

  if (uploadStatus) uploadStatus.textContent = `사진 ${files.length}장을 올리는 중입니다…`;
  showOperation(
    "사진을 올리고 있습니다",
    `0/${files.length}장 처리됨 · 이미지 크기와 형식을 최적화하고 있습니다.`,
  );
  try {
    const results = await uploadImageFiles(files);
    const uploaded = results.flatMap((entry) => entry.result ? [entry.result] : []);
    const failed = results.length - uploaded.length;
    if (uploaded.length) {
      replaceEditorSelection(`\n${uploaded.map((result) => result.markdown).join("\n")}\n`);
    }
    if (uploadStatus) {
      uploadStatus.textContent = failed
        ? `사진 ${uploaded.length}장은 추가했고 ${failed}장은 올리지 못했습니다.`
        : `사진 ${uploaded.length}장을 본문에 추가했습니다. 대체 텍스트를 수정해주세요.`;
    }
  } finally {
    hideOperation();
  }
}

upload?.addEventListener("change", async () => {
  await handleImageFiles(upload.files || []);
  upload.value = "";
});

let dragDepth = 0;

function hasDraggedFiles(event) {
  return [...(event.dataTransfer?.types || [])].includes("Files");
}

writePane?.addEventListener("dragenter", (event) => {
  if (!hasDraggedFiles(event)) return;
  event.preventDefault();
  dragDepth += 1;
  writePane.classList.add("is-dragging-files");
});

writePane?.addEventListener("dragover", (event) => {
  if (!hasDraggedFiles(event)) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = "copy";
});

writePane?.addEventListener("dragleave", () => {
  dragDepth = Math.max(0, dragDepth - 1);
  if (!dragDepth) writePane.classList.remove("is-dragging-files");
});

writePane?.addEventListener("drop", async (event) => {
  if (!hasDraggedFiles(event)) return;
  event.preventDefault();
  dragDepth = 0;
  writePane.classList.remove("is-dragging-files");
  await handleImageFiles(event.dataTransfer.files);
});

selectEditorTab("write");
resizeEditor();
updatePreviewMeta();
requestPreview();
