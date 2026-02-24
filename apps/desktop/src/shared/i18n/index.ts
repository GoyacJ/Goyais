import { computed, reactive } from "vue";

import type { Locale } from "@/shared/i18n/messages";
import { messages } from "@/shared/i18n/messages";

type I18nState = {
  locale: Locale;
};

const LOCALE_STORAGE_KEY = "goyais.locale";
const initialLocale = detectInitialLocale();

export const i18nState = reactive<I18nState>({
  locale: initialLocale
});

export const availableLocales: Locale[] = ["zh-CN", "en-US"];

export function setLocale(locale: Locale): void {
  i18nState.locale = locale;
  persistLocale(locale);
}

export function useI18n() {
  const locale = computed(() => i18nState.locale);
  return {
    locale,
    setLocale,
    t
  };
}

export function t(key: string): string {
  const current = messages[i18nState.locale][key];
  if (current !== undefined) {
    return current;
  }

  const fallback = messages["zh-CN"][key];
  return fallback ?? key;
}

function detectInitialLocale(): Locale {
  const persisted = readPersistedLocale();
  if (persisted) {
    return persisted;
  }

  const language = typeof navigator !== "undefined" ? navigator.language : "zh-CN";
  return language.toLowerCase().startsWith("en") ? "en-US" : "zh-CN";
}

function readPersistedLocale(): Locale | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const value = window.localStorage.getItem(LOCALE_STORAGE_KEY);
    if (value === "zh-CN" || value === "en-US") {
      return value;
    }
  } catch {
    return null;
  }
  return null;
}

function persistLocale(locale: Locale): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, locale);
  } catch {
    // ignore localStorage failures
  }
}
