import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import type { SupportedLocale } from "@/i18n/types";

import enUSCommon from "./locales/en-US/common.json";
import zhCNCommon from "./locales/zh-CN/common.json";

export const SUPPORTED_LOCALES: SupportedLocale[] = ["zh-CN", "en-US"];
export const DEFAULT_LOCALE: SupportedLocale = "zh-CN";
export const LOCALE_STORAGE_KEY = "goyais.locale";

function normalizeLocale(input?: string | null): SupportedLocale | undefined {
  if (!input) return undefined;
  if (input.startsWith("zh")) return "zh-CN";
  if (input.startsWith("en")) return "en-US";
  return undefined;
}

interface DetectOptions {
  storedLocale?: string | null;
  preferredLanguages?: string[];
}

export function detectInitialLocale(options?: DetectOptions): SupportedLocale {
  const storedLocale = normalizeLocale(options?.storedLocale ?? localStorage.getItem(LOCALE_STORAGE_KEY));
  if (storedLocale) return storedLocale;

  const preferredLanguages = options?.preferredLanguages ?? navigator.languages;
  for (const locale of preferredLanguages) {
    const normalized = normalizeLocale(locale);
    if (normalized) {
      return normalized;
    }
  }

  return DEFAULT_LOCALE;
}

export async function initializeI18n(locale: SupportedLocale) {
  if (i18n.isInitialized) {
    if (i18n.language !== locale) {
      await i18n.changeLanguage(locale);
    }
    return i18n;
  }

  await i18n.use(initReactI18next).init({
    lng: locale,
    fallbackLng: DEFAULT_LOCALE,
    defaultNS: "common",
    ns: ["common"],
    resources: {
      "zh-CN": { common: zhCNCommon },
      "en-US": { common: enUSCommon }
    },
    interpolation: {
      escapeValue: false
    }
  });

  return i18n;
}

export async function applyLocale(locale: SupportedLocale) {
  localStorage.setItem(LOCALE_STORAGE_KEY, locale);
  if (i18n.isInitialized) {
    await i18n.changeLanguage(locale);
  }
}

export default i18n;
