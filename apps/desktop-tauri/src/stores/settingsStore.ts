import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { applyLocale, detectInitialLocale } from "@/i18n";
import type { SupportedLocale } from "@/i18n/types";

export type ThemeMode = "dark" | "light";

export const RUNTIME_URL_STORAGE_KEY = "goyais.runtimeUrl";
const DEFAULT_RUNTIME_URL = "http://127.0.0.1:8040";

export function normalizeRuntimeUrl(input: string): string {
  const trimmed = input.trim();
  if (!trimmed) {
    return DEFAULT_RUNTIME_URL;
  }

  let parsed: URL;
  try {
    parsed = new URL(trimmed);
  } catch {
    throw new Error("Runtime URL must be a valid http/https URL");
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("Runtime URL must use http or https");
  }

  return parsed.toString().replace(/\/+$/, "");
}

interface SettingsState {
  runtimeUrl: string;
  locale: SupportedLocale;
  theme: ThemeMode;
  defaultModelConfigId?: string;
  setRuntimeUrl: (value: string) => void;
  setLocale: (locale: SupportedLocale) => Promise<void>;
  setTheme: (theme: ThemeMode) => void;
  setDefaultModelConfigId: (modelConfigId?: string) => void;
  hydrate: () => Promise<void>;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set, get) => ({
      runtimeUrl: localStorage.getItem(RUNTIME_URL_STORAGE_KEY) ?? DEFAULT_RUNTIME_URL,
      locale: "zh-CN",
      theme: "dark",
      defaultModelConfigId: undefined,
      setRuntimeUrl: (runtimeUrl) => {
        const normalized = normalizeRuntimeUrl(runtimeUrl);
        localStorage.setItem(RUNTIME_URL_STORAGE_KEY, normalized);
        set({ runtimeUrl: normalized });
      },
      setLocale: async (locale) => {
        await applyLocale(locale);
        set({ locale });
      },
      setTheme: (theme) => set({ theme }),
      setDefaultModelConfigId: (defaultModelConfigId) => set({ defaultModelConfigId }),
      hydrate: async () => {
        const state = get();
        const locale = detectInitialLocale();
        await applyLocale(locale);
        const runtimeValue = localStorage.getItem(RUNTIME_URL_STORAGE_KEY) ?? state.runtimeUrl ?? DEFAULT_RUNTIME_URL;
        let runtimeUrl = DEFAULT_RUNTIME_URL;
        try {
          runtimeUrl = normalizeRuntimeUrl(runtimeValue);
        } catch {
          runtimeUrl = DEFAULT_RUNTIME_URL;
          localStorage.setItem(RUNTIME_URL_STORAGE_KEY, runtimeUrl);
        }
        set({
          locale,
          runtimeUrl
        });
      }
    }),
    {
      name: "goyais.settings.v2",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        runtimeUrl: state.runtimeUrl,
        locale: state.locale,
        theme: state.theme,
        defaultModelConfigId: state.defaultModelConfigId
      })
    }
  )
);
