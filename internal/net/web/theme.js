const CSS_VAR_DEFAULTS = {
  "--mud-bg": "#0f0f10",
  "--mud-surface": "#151517",
  "--mud-border": "#1f2024",
  "--mud-text": "#f5f6f7",
  "--mud-subtle-text": "#a1a7b3",
  "--mud-accent": "#f97316",
  "--mud-accent-text": "#0b0c0f",
  "--mud-success": "#16a34a",
  "--mud-error": "#ef4444",
};

const CSS_VAR_CANDIDATES = {
  "--mud-bg": ["--mud-bg", "--bg", "--background", "--color-bg", "--surface-base"],
  "--mud-surface": ["--mud-surface", "--surface", "--panel", "--surface-100", "--muted"],
  "--mud-border": ["--mud-border", "--border", "--border-color", "--surface-200"],
  "--mud-text": ["--mud-text", "--text", "--foreground", "--color-text"],
  "--mud-subtle-text": ["--mud-subtle-text", "--muted-foreground", "--muted-text", "--text-muted", "--color-muted"],
  "--mud-accent": ["--mud-accent", "--accent", "--primary", "--color-accent", "--brand"],
  "--mud-accent-text": ["--mud-accent-text", "--accent-foreground", "--on-accent", "--color-on-accent"],
  "--mud-success": ["--mud-success", "--success", "--color-success", "--green"],
  "--mud-error": ["--mud-error", "--danger", "--color-danger", "--red"],
};

const PARAM_ALIASES = {
  "--mud-bg": ["bg", "background"],
  "--mud-surface": ["surface", "panel", "card"],
  "--mud-border": ["border", "outline"],
  "--mud-text": ["text", "fg", "foreground"],
  "--mud-subtle-text": ["muted", "subtle", "fgMuted"],
  "--mud-accent": ["accent", "primary", "highlight"],
  "--mud-accent-text": ["accentText", "onAccent", "accent-foreground"],
  "--mud-success": ["success"],
  "--mud-error": ["error", "danger"],
};

const THEME_MESSAGE_TYPES = new Set([
  "dmud:theme",
  "dmud-theme",
  "dmud:theme:update",
]);

function normaliseColor(value) {
  if (!value) {
    return "";
  }
  const trimmed = String(value).trim();
  if (!trimmed) {
    return "";
  }
  if (/^[0-9a-fA-F]{6}$/.test(trimmed)) {
    return `#${trimmed}`;
  }
  if (/^[0-9a-fA-F]{3}$/.test(trimmed)) {
    return `#${trimmed}`;
  }
  return trimmed;
}

function resolveCandidate(computed, candidates) {
  for (const name of candidates) {
    const value = computed.getPropertyValue(name).trim();
    if (value) {
      return value;
    }
  }
  return "";
}

function applyCssVariables(root, values) {
  if (!values) {
    return;
  }
  for (const [key, value] of Object.entries(values)) {
    if (typeof value === "string" && value.trim() !== "") {
      root.style.setProperty(key, value.trim());
    }
  }
}

function parseMessageTheme(data) {
  if (!data || typeof data !== "object") {
    return { vars: {}, scheme: "" };
  }

  const theme = { vars: {}, scheme: "" };
  const cssVars = data.cssVariables || data.css || data.vars;
  if (cssVars && typeof cssVars === "object") {
    for (const [key, value] of Object.entries(cssVars)) {
      theme.vars[key] = normaliseColor(value);
    }
  }

  const aliasToVar = {
    bg: "--mud-bg",
    background: "--mud-bg",
    surface: "--mud-surface",
    panel: "--mud-surface",
    card: "--mud-surface",
    border: "--mud-border",
    outline: "--mud-border",
    text: "--mud-text",
    foreground: "--mud-text",
    fg: "--mud-text",
    muted: "--mud-subtle-text",
    subtle: "--mud-subtle-text",
    accent: "--mud-accent",
    primary: "--mud-accent",
    highlight: "--mud-accent",
    accentText: "--mud-accent-text",
    onAccent: "--mud-accent-text",
    success: "--mud-success",
    error: "--mud-error",
    danger: "--mud-error",
  };

  for (const [key, cssVar] of Object.entries(aliasToVar)) {
    if (key in data) {
      theme.vars[cssVar] = normaliseColor(data[key]);
    }
  }

  if (typeof data.colorScheme === "string") {
    theme.scheme = data.colorScheme;
  } else if (typeof data.scheme === "string") {
    theme.scheme = data.scheme;
  } else if (typeof data.mode === "string") {
    theme.scheme = data.mode;
  }

  return theme;
}

function applyColorScheme(root, scheme) {
  const value = typeof scheme === "string" ? scheme.trim().toLowerCase() : "";
  if (!value) {
    return;
  }
  if (value === "light" || value === "dark") {
    root.style.colorScheme = value;
    root.dataset.colorScheme = value;
  }
}

export function setupThemeSync(options = {}) {
  const root = document.documentElement;
  const params = new URLSearchParams(window.location.search);
  const requestParent = options.requestParent !== false;

  function refreshFromComputed() {
    const computed = getComputedStyle(root);
    const updates = {};
    for (const [cssVar, fallback] of Object.entries(CSS_VAR_DEFAULTS)) {
      const candidates = CSS_VAR_CANDIDATES[cssVar] || [cssVar];
      const value = resolveCandidate(computed, candidates) || fallback;
      updates[cssVar] = value;
    }
    applyCssVariables(root, updates);
  }

  function applyFromParams() {
    const updates = {};
    for (const [cssVar, aliases] of Object.entries(PARAM_ALIASES)) {
      for (const alias of aliases) {
        if (params.has(alias)) {
          updates[cssVar] = normaliseColor(params.get(alias));
          break;
        }
      }
    }
    if (params.has("colorScheme")) {
      applyColorScheme(root, params.get("colorScheme"));
    } else if (params.has("scheme")) {
      applyColorScheme(root, params.get("scheme"));
    } else if (params.has("theme")) {
      applyColorScheme(root, params.get("theme"));
    }
    applyCssVariables(root, updates);
  }

  function handleMessage(event) {
    const data = event.data;
    if (!data) {
      return;
    }
    if (typeof data === "string") {
      try {
        const parsed = JSON.parse(data);
        handleParsedMessage(parsed);
        return;
      } catch (_) {
        return;
      }
    }
    handleParsedMessage(data);
  }

  function handleParsedMessage(data) {
    if (!data || typeof data !== "object") {
      return;
    }
    const type = typeof data.type === "string" ? data.type : "";
    if (THEME_MESSAGE_TYPES.has(type)) {
      const payload = "theme" in data ? data.theme : data.payload || data;
      const { vars, scheme } = parseMessageTheme(payload);
      applyCssVariables(root, vars);
      applyColorScheme(root, scheme);
      return;
    }
    if (type === "dmud:theme:request") {
      if (typeof options.onThemeRequest === "function") {
        options.onThemeRequest();
      }
    }
  }

  function requestTheme() {
    if (!requestParent) {
      return;
    }
    if (window.parent && window.parent !== window) {
      try {
        window.parent.postMessage({ type: "dmud:request-theme" }, "*");
      } catch (_) {
        /* no-op */
      }
    }
  }

  refreshFromComputed();
  applyFromParams();
  if (requestParent) {
    requestTheme();
    window.setTimeout(requestTheme, 250);
  }

  window.addEventListener("message", handleMessage);
  const mq = window.matchMedia("(prefers-color-scheme: dark)");
  if (typeof mq.addEventListener === "function") {
    mq.addEventListener("change", () => {
      refreshFromComputed();
    });
  }

  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      if (mutation.type === "attributes") {
        refreshFromComputed();
        break;
      }
    }
  });
  observer.observe(root, { attributes: true, attributeFilter: ["class", "data-theme", "data-color-mode"] });

  return {
    applyUpdate: (update) => {
      const { vars, scheme } = parseMessageTheme(update);
      applyCssVariables(root, vars);
      applyColorScheme(root, scheme);
    },
    refreshFromComputed,
    requestTheme,
  };
}
