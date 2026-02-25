import { computed, watch } from "vue";
import { defineStore } from "pinia";
import { useMediaQuery, useStorage } from "@vueuse/core";

import { pinia } from "@/shared/stores/pinia";

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
const persistedThemeSettings = useStorage<PersistedThemeSettings>(THEME_SETTINGS_STORAGE_KEY, {}, undefined, {
  writeDefaults: false,
  flush: "sync"
});
const prefersLightColorScheme = useMediaQuery("(prefers-color-scheme: light)");

const THEME_PRESET_PROFILES: Record<ThemePreset, ThemePresetProfile> = {
  aurora_forge: {
    mode: "system",
    fontStyle: "neutral",
    fontScale: "md"
  },
  obsidian_pulse: {
    mode: "dark",
    fontStyle: "neutral",
    fontScale: "md"
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

const useThemeStoreDefinition = defineStore("theme", {
  state: (): ThemeSettings => ({
    ...DEFAULT_THEME_SETTINGS,
    resolved: "dark"
  })
});

export const useThemeStore = useThemeStoreDefinition;
export const themeStore = useThemeStoreDefinition(pinia);

let stopColorSchemeWatch: (() => void) | null = null;

export function initializeTheme(): void {
  const persisted = readPersistedThemeSettings();
  themeStore.mode = persisted.mode ?? DEFAULT_THEME_SETTINGS.mode;
  themeStore.fontStyle = persisted.fontStyle ?? DEFAULT_THEME_SETTINGS.fontStyle;
  themeStore.fontScale = persisted.fontScale ?? DEFAULT_THEME_SETTINGS.fontScale;
  themeStore.preset = persisted.preset ?? DEFAULT_THEME_SETTINGS.preset;
  applyThemeFromSettings();
  ensureColorSchemeWatcher();
}

export function setThemeMode(mode: ThemeMode): void {
  themeStore.mode = mode;
  syncPresetFromValues();
  persistThemeSettings();
  applyThemeFromSettings();
}

export function setThemePreference(preference: ThemePreference): void {
  setThemeMode(preference);
}

export function setFontStyle(fontStyle: FontStyle): void {
  themeStore.fontStyle = fontStyle;
  syncPresetFromValues();
  persistThemeSettings();
  applyThemeFromSettings();
}

export function setFontScale(fontScale: FontScale): void {
  themeStore.fontScale = fontScale;
  syncPresetFromValues();
  persistThemeSettings();
  applyThemeFromSettings();
}

export function setThemePreset(preset: ThemePreset): void {
  const profile = THEME_PRESET_PROFILES[preset];
  themeStore.preset = preset;
  themeStore.mode = profile.mode;
  themeStore.fontStyle = profile.fontStyle;
  themeStore.fontScale = profile.fontScale;
  persistThemeSettings();
  applyThemeFromSettings();
}

export function resetThemeSettings(): void {
  themeStore.mode = DEFAULT_THEME_SETTINGS.mode;
  themeStore.fontStyle = DEFAULT_THEME_SETTINGS.fontStyle;
  themeStore.fontScale = DEFAULT_THEME_SETTINGS.fontScale;
  themeStore.preset = DEFAULT_THEME_SETTINGS.preset;
  persistThemeSettings();
  applyThemeFromSettings();
}

export function useTheme() {
  return {
    preference: computed(() => themeStore.mode),
    mode: computed(() => themeStore.mode),
    fontStyle: computed(() => themeStore.fontStyle),
    fontScale: computed(() => themeStore.fontScale),
    preset: computed(() => themeStore.preset),
    resolved: computed(() => themeStore.resolved),
    setThemePreference,
    setThemeMode,
    setFontStyle,
    setFontScale,
    setThemePreset,
    resetThemeSettings
  };
}

function applyThemeFromSettings(): void {
  const resolved = resolveTheme(themeStore.mode);
  themeStore.resolved = resolved;
  applyThemeAttributes();
}

function applyThemeAttributes(): void {
  if (typeof document === "undefined") {
    return;
  }

  const root = document.documentElement;
  root.setAttribute("data-theme", themeStore.resolved);
  root.setAttribute("data-theme-mode", themeStore.mode);
  root.setAttribute("data-font-style", themeStore.fontStyle);
  root.setAttribute("data-font-scale", themeStore.fontScale);
  root.setAttribute("data-theme-preset", themeStore.preset);
}

function resolveTheme(mode: ThemeMode): ThemeResolved {
  if (mode === "light") {
    return "light";
  }
  if (mode === "dark") {
    return "dark";
  }
  if (prefersLightColorScheme.value) {
    return "light";
  }
  return "dark";
}

function ensureColorSchemeWatcher(): void {
  if (stopColorSchemeWatch) {
    return;
  }

  stopColorSchemeWatch = watch(
    prefersLightColorScheme,
    () => {
      if (themeStore.mode !== "system") {
        return;
      }
      applyThemeFromSettings();
    },
    { flush: "sync" }
  );
}

function readPersistedThemeSettings(): PersistedThemeSettings {
  const persisted = migrateLegacyPresetDefaults(persistedThemeSettings.value);
  return {
    mode: asThemeMode(persisted?.mode),
    fontStyle: asFontStyle(persisted?.fontStyle),
    fontScale: asFontScale(persisted?.fontScale),
    preset: asThemePreset(persisted?.preset)
  };
}

function migrateLegacyPresetDefaults(persisted: PersistedThemeSettings | undefined): PersistedThemeSettings {
  const snapshot: PersistedThemeSettings = { ...(persisted ?? {}) };
  if (
    snapshot.preset === "obsidian_pulse" &&
    (snapshot.mode === undefined || snapshot.mode === "dark") &&
    (snapshot.fontStyle === undefined || snapshot.fontStyle === "coding") &&
    (snapshot.fontScale === undefined || snapshot.fontScale === "sm")
  ) {
    snapshot.mode = "dark";
    snapshot.fontStyle = "neutral";
    snapshot.fontScale = "md";
  }
  return snapshot;
}

function persistThemeSettings(): void {
  const payload: PersistedThemeSettings = {
    mode: themeStore.mode,
    fontStyle: themeStore.fontStyle,
    fontScale: themeStore.fontScale,
    preset: themeStore.preset
  };
  persistedThemeSettings.value = payload;
}

function syncPresetFromValues(): void {
  if (matchesPreset(themeStore.preset, themeStore.mode, themeStore.fontStyle, themeStore.fontScale)) {
    return;
  }

  const matchedPreset = (Object.keys(THEME_PRESET_PROFILES) as ThemePreset[]).find((preset) =>
    matchesPreset(preset, themeStore.mode, themeStore.fontStyle, themeStore.fontScale)
  );

  if (matchedPreset) {
    themeStore.preset = matchedPreset;
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
