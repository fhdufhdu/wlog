const THEME_KEY = "wlog-theme";
const root = document.documentElement;
const systemTheme = window.matchMedia("(prefers-color-scheme: dark)");

function storedTheme() {
  try {
    const value = localStorage.getItem(THEME_KEY);
    return value === "light" || value === "dark" ? value : null;
  } catch (_) {
    return null;
  }
}

const savedTheme = storedTheme();
if (savedTheme) root.dataset.theme = savedTheme;

function currentTheme() {
  return root.dataset.theme || (systemTheme.matches ? "dark" : "light");
}

function updateThemeUi(notify = false) {
  const theme = currentTheme();
  const dark = theme === "dark";
  document.querySelectorAll("[data-theme-toggle]").forEach((button) => {
    const label = dark ? "라이트 모드로 전환" : "다크 모드로 전환";
    button.dataset.theme = theme;
    button.setAttribute("aria-label", label);
    button.setAttribute("aria-pressed", String(dark));
    const hiddenLabel = button.querySelector("[data-theme-label]");
    if (hiddenLabel) hiddenLabel.textContent = label;
  });
  document.querySelectorAll('meta[name="theme-color"]').forEach((meta) => {
    meta.content = dark ? "#171a1f" : "#f2f4f7";
  });
  if (notify) {
    window.dispatchEvent(new CustomEvent("wlog:theme-changed", { detail: { theme } }));
  }
}

function toggleTheme() {
  const theme = currentTheme() === "dark" ? "light" : "dark";
  root.dataset.theme = theme;
  try {
    localStorage.setItem(THEME_KEY, theme);
  } catch (_) {
    // Storage may be unavailable in private browsing; the current page still updates.
  }
  updateThemeUi(true);
}

function initializeThemeControls() {
  document.querySelectorAll("[data-theme-toggle]").forEach((button) => {
    button.addEventListener("click", toggleTheme);
  });
  updateThemeUi();
}

systemTheme.addEventListener("change", () => {
  if (!root.dataset.theme) updateThemeUi(true);
});

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initializeThemeControls);
} else {
  initializeThemeControls();
}
