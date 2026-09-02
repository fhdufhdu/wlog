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
const previewDescription = document.querySelector("#preview-description");
const csrfToken = document.querySelector("#csrf-token");
const writePane = document.querySelector("#write-panel");

let dirty = false;
let changeVersion = 0;
let descriptionEdited = descriptionManual?.value === "true";
let autosaveController = null;
let previewController = null;
let previewTimer = null;
let previewVersion = 0;
let isComposing = false;
const autosaveUrl = form?.dataset.autosaveUrl;

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
  if (previewDescription) {
    previewDescription.textContent = description?.value.trim() || "";
    previewDescription.hidden = !description?.value.trim();
  }
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

async function requestPreview() {
  if (!previewBody || !editor || !csrfToken) return;
  window.clearTimeout(previewTimer);
  previewController?.abort();
  const requestVersion = ++previewVersion;

  if (!editor.value.trim()) {
    previewBody.removeAttribute("aria-busy");
    showPreviewMessage("본문을 입력하면 여기에 미리보기가 표시됩니다.");
    return;
  }

  previewController = new AbortController();
  previewBody.setAttribute("aria-busy", "true");
  const data = new URLSearchParams();
  data.set("csrf_token", csrfToken.value);
  data.set("content_markdown", editor.value);

  try {
    const response = await fetch("/admin/preview", {
      method: "POST",
      body: data,
      credentials: "same-origin",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      signal: previewController.signal,
    });
    if (!response.ok) throw new Error(await response.text());
    const html = await response.text();
    if (requestVersion === previewVersion) {
      previewBody.innerHTML = html;
      window.dispatchEvent(new CustomEvent("wlog:markdown-rendered", {
        detail: { root: previewBody },
      }));
    }
  } catch (error) {
    if (error.name !== "AbortError" && requestVersion === previewVersion) {
      showPreviewMessage(error.message || "미리보기를 불러오지 못했습니다.", "form-alert");
    }
  } finally {
    if (requestVersion === previewVersion) previewBody.removeAttribute("aria-busy");
  }
}

function schedulePreview(delay = 320) {
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
  descriptionEdited = Boolean(description.value.trim());
  descriptionManual.value = descriptionEdited ? "true" : "false";
  if (descriptionCount) descriptionCount.textContent = Array.from(description.value).length;
  if (!descriptionEdited) updateDescription();
});

editor?.addEventListener("compositionstart", () => { isComposing = true; });
editor?.addEventListener("compositionend", () => {
  isComposing = false;
  updateDescription();
  updatePreviewMeta();
  schedulePreview(0);
});
editor?.addEventListener("input", updateDescription);

form?.addEventListener("input", () => {
  markDirty();
  updatePreviewMeta();
  schedulePreview();
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

form?.addEventListener("submit", () => {
  autosaveController?.abort();
  previewController?.abort();
  window.clearTimeout(previewTimer);
  form.querySelectorAll("button[type='submit']").forEach((button) => { button.disabled = true; });
  setSaveStatus("저장 중…", "saving");
  dirty = false;
});

window.addEventListener("beforeunload", (event) => {
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

async function handleImageFiles(fileList) {
  if (!editor || !csrfToken) return;
  const files = [...fileList].filter((file) => file.type.startsWith("image/"));
  if (!files.length) {
    if (uploadStatus) uploadStatus.textContent = "JPEG, PNG, GIF, WebP 사진 파일만 추가할 수 있습니다.";
    return;
  }

  if (uploadStatus) uploadStatus.textContent = `사진 ${files.length}장을 올리는 중입니다…`;
  let uploaded = 0;
  try {
    for (const file of files) {
      const result = await uploadImage(file);
      replaceEditorSelection(`\n${result.markdown}\n`);
      uploaded += 1;
    }
    if (uploadStatus) {
      uploadStatus.textContent = `사진 ${uploaded}장을 본문에 추가했습니다. 대체 텍스트를 수정해주세요.`;
    }
  } catch (error) {
    if (uploadStatus) {
      const progress = uploaded ? `${uploaded}장은 추가했습니다. ` : "";
      uploadStatus.textContent = `${progress}${error.message || "사진을 올리지 못했습니다."}`;
    }
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
updatePreviewMeta();
requestPreview();
