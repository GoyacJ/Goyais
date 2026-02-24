import { computed, reactive } from "vue";

export type ThemeMode = "system" | "dark" | "light";
export type ThemePreference = ThemeMode;
export type ThemeResolved = "dark" | "light";
export type FontStyle = "neutral" | "reading" | "coding";
export type FontScale = "sm" | "md" | "lg";
export type ThemePreset = "aurora_forge" | "obsidian_pulse" | "paper_focus";

export type ThemeSettings = {
  mode: ThemeMode;
  fontStyle: FontStyle;
  fontScale: FontScale;
  preset: ThemePreset;
  resolved: ThemeResolved;
};

type ThemePresetProfile = {
  mode: ThemeMode;
  fontStyle: FontStyle;
  fontScale: FontScale;
};

type PersistedThemeSettings = {
  mode?: ThemeMode;
  fontStyle?: FontStyle;
  fontScale?: FontScale;
  preset?: ThemePreset;
};

const THEME_SETTINGS_STORAGE_KEY = "goyais.theme.settings.v1";

const THEME_PRESET_PROFILES: Record<ThemePreset, ThemePresetProfile> = {
  aurora_forge: {
    mode: "system",
    fontStyle: "neutral",
    fontScale: "md"
  },
  obsidian_pulse: {
    mode: "dark",
    fontStyle: "coding",
    fontScale: "sm"
  },
  paper_focus: {
    mode: "light",
    fontStyle: "reading",
    fontScale: "lg"
  }
};

const DEFAULT_THEME_SETTINGS: Omit<ThemeSettings, "resolved"> = {
  mode: "system",
  fontStyle: "neutral",
  fontScale: "md",
  preset: "aurora_forge"
};

const themeState = reactive<ThemeSettings>({
  ...DEFAULT_THEME_SETTINGS,
  resolved: "dark"
});

let mediaQuery: MediaQueryList | null = null;
let mediaListenerBound = false;

export function initializeTheme(): void {
  const persisted = readPersistedThemeSettings();
  themeState.mode = persisted.mode ?? DEFAULT_THEME_SETTINGS.mode;
  themeState.fontStyle = persisted.fontStyle ?? DEFAULT_THEME_SETTINGS.fontStyle;
  themeState.fontScale = persisted.fontScale ?? DEFAULT_THEME_SETTINGS.fontScale;
  themeState.preset = persisted.preset ?? DEFAULT_THEME_SETTINGS.preset;
  applyThemeFromSettings();
  bindSystemThemeListener();
}

export function setThemeMode(mode: ThemeMode): void {
  themeState.mode = mode;
  syncPresetFromValues();
  persistThemeSettings();
  applyThemeFromSettings();
}

export function setThemePreference(preference: ThemePreference): void {
  setThemeMode(preference);
}

export function setFontStyle(fontStyle: FontStyle): void {
  themeState.fontStyle = fontStyle;
  syncPresetFromValues();
  persistThemeSettings();
  applyThemeFromSettings();
}

export function setFontScale(fontScale: FontScale): void {
  themeState.fontScale = fontScale;
  syncPresetFromValues();
  persistThemeSettings();
  applyThemeFromSettings();
}

export function setThemePreset(preset: ThemePreset): void {
  const profile = THEME_PRESET_PROFILES[preset];
  themeState.preset = preset;
  themeState.mode = profile.mode;
  themeState.fontStyle = profile.fontStyle;
  themeState.fontScale = profile.fontScale;
  persistThemeSettings();
  applyThemeFromSettings();
}

export function resetThemeSettings(): void {
  themeState.mode = DEFAULT_THEME_SETTINGS.mode;
  themeState.fontStyle = DEFAULT_THEME_SETTINGS.fontStyle;
  themeState.fontScale = DEFAULT_THEME_SETTINGS.fontScale;
  themeState.preset = DEFAULT_THEME_SETTINGS.preset;
  persistThemeSettings();
  applyThemeFromSettings();
}

export function useTheme() {
  return {
    preference: computed(() => themeState.mode),
    mode: computed(() => themeState.mode),
    fontStyle: computed(() => themeState.fontStyle),
    fontScale: computed(() => themeState.fontScale),
    preset: computed(() => themeState.preset),
    resolved: computed(() => themeState.resolved),
    setThemePreference,
    setThemeMode,
    setFontStyle,
    setFontScale,
    setThemePreset,
    resetThemeSettings
  };
}

function applyThemeFromSettings(): void {
  const resolved = resolveTheme(themeState.mode);
  themeState.resolved = resolved;
  applyThemeAttributes();
}

function applyThemeAttributes(): void {
  if (typeof document === "undefined") {
    return;
  }

  const root = document.documentElement;
  root.setAttribute("data-theme", themeState.resolved);
  root.setAttribute("data-theme-mode", themeState.mode);
  root.setAttribute("data-font-style", themeState.fontStyle);
  root.setAttribute("data-font-scale", themeState.fontScale);
  root.setAttribute("data-theme-preset", themeState.preset);
}

function resolveTheme(mode: ThemeMode): ThemeResolved {
  if (mode === "light") {
    return "light";
  }
  if (mode === "dark") {
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
    if (themeState.mode !== "system") {
      return;
    }
    applyThemeFromSettings();
  };

  mediaQuery.addEventListener("change", handleSystemThemeChange);
  mediaListenerBound = true;
}

function readPersistedThemeSettings(): PersistedThemeSettings {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const raw = window.localStorage.getItem(THEME_SETTINGS_STORAGE_KEY);
    if (!raw) {
      return {};
    }

    const parsed = JSON.parse(raw) as PersistedThemeSettings;
    return {
      mode: asThemeMode(parsed.mode),
      fontStyle: asFontStyle(parsed.fontStyle),
      fontScale: asFontScale(parsed.fontScale),
      preset: asThemePreset(parsed.preset)
    };
  } catch {
    return {};
  }
}

function persistThemeSettings(): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    const payload: PersistedThemeSettings = {
      mode: themeState.mode,
      fontStyle: themeState.fontStyle,
      fontScale: themeState.fontScale,
      preset: themeState.preset
    };
    window.localStorage.setItem(THEME_SETTINGS_STORAGE_KEY, JSON.stringify(payload));
  } catch {
    // ignore localStorage failures
  }
}

function syncPresetFromValues(): void {
  if (matchesPreset(themeState.preset, themeState.mode, themeState.fontStyle, themeState.fontScale)) {
    return;
  }

  const matchedPreset = (Object.keys(THEME_PRESET_PROFILES) as ThemePreset[]).find((preset) =>
    matchesPreset(preset, themeState.mode, themeState.fontStyle, themeState.fontScale)
  );

  if (matchedPreset) {
    themeState.preset = matchedPreset;
  }
}

function matchesPreset(preset: ThemePreset, mode: ThemeMode, fontStyle: FontStyle, fontScale: FontScale): boolean {
  const profile = THEME_PRESET_PROFILES[preset];
  return profile.mode === mode && profile.fontStyle === fontStyle && profile.fontScale === fontScale;
}

function asThemeMode(value: unknown): ThemeMode | undefined {
  return isOneOf(value, ["system", "dark", "light"]);
}

function asFontStyle(value: unknown): FontStyle | undefined {
  return isOneOf(value, ["neutral", "reading", "coding"]);
}

function asFontScale(value: unknown): FontScale | undefined {
  return isOneOf(value, ["sm", "md", "lg"]);
}

function asThemePreset(value: unknown): ThemePreset | undefined {
  return isOneOf(value, ["aurora_forge", "obsidian_pulse", "paper_focus"]);
}

function isOneOf<T extends string>(value: unknown, options: readonly T[]): T | undefined {
  if (typeof value !== "string") {
    return undefined;
  }
  return options.includes(value as T) ? (value as T) : undefined;
}
