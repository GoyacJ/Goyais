import { computed, reactive } from "vue";

export type ThemePreference = "system" | "dark" | "light";
export type ThemeResolved = "dark" | "light";

type ThemeState = {
  preference: ThemePreference;
  resolved: ThemeResolved;
};

const THEME_STORAGE_KEY = "goyais.theme.preference";
const themeState = reactive<ThemeState>({
  preference: "system",
  resolved: "dark"
});

let mediaQuery: MediaQueryList | null = null;
let mediaListenerBound = false;

export function initializeTheme(): void {
  themeState.preference = readPersistedPreference();
  applyThemeFromPreference();
  bindSystemThemeListener();
}

export function setThemePreference(preference: ThemePreference): void {
  themeState.preference = preference;
  persistThemePreference(preference);
  applyThemeFromPreference();
}

export function useTheme() {
  return {
    preference: computed(() => themeState.preference),
    resolved: computed(() => themeState.resolved),
    setThemePreference
  };
}

function applyThemeFromPreference(): void {
  const resolved = resolveTheme(themeState.preference);
  themeState.resolved = resolved;

  if (typeof document === "undefined") {
    return;
  }
  document.documentElement.setAttribute("data-theme", resolved);
}

function resolveTheme(preference: ThemePreference): ThemeResolved {
  if (preference === "light") {
    return "light";
  }
  if (preference === "dark") {
    return "dark";
  }
  if (typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: light)").matches) {
    return "light";
  }
  return "dark";
}

function bindSystemThemeListener(): void {
  if (typeof window === "undefined") {
    return;
  }

  if (!mediaQuery) {
    mediaQuery = window.matchMedia("(prefers-color-scheme: light)");
  }

  if (mediaListenerBound) {
    return;
  }

  const handleSystemThemeChange = () => {
    if (themeState.preference !== "system") {
      return;
    }
    applyThemeFromPreference();
  };

  mediaQuery.addEventListener("change", handleSystemThemeChange);
  mediaListenerBound = true;
}

function readPersistedPreference(): ThemePreference {
  if (typeof window === "undefined") {
    return "system";
  }

  try {
    const raw = window.localStorage.getItem(THEME_STORAGE_KEY);
    if (raw === "system" || raw === "dark" || raw === "light") {
      return raw;
    }
  } catch {
    return "system";
  }
  return "system";
}

function persistThemePreference(preference: ThemePreference): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, preference);
  } catch {
    // ignore localStorage failures
  }
}
