import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { applyLocale, detectInitialLocale } from "@/i18n";
import type { SupportedLocale } from "@/i18n/types";

export type ThemeMode = "dark" | "light";

const LEGACY_RUNTIME_URL_STORAGE_KEY = "goyais.runtimeUrl";
const LEGACY_LOCAL_AUTH_PASSWORD_KEY = "goyais.localAutoPassword";

interface SettingsState {
  locale: SupportedLocale;
  theme: ThemeMode;
  defaultModelConfigId?: string;
  setLocale: (locale: SupportedLocale) => Promise<void>;
  setTheme: (theme: ThemeMode) => void;
  setDefaultModelConfigId: (modelConfigId?: string) => void;
  hydrate: () => Promise<void>;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      locale: "zh-CN",
      theme: "dark",
      defaultModelConfigId: undefined,
      setLocale: async (locale) => {
        await applyLocale(locale);
        localStorage.setItem("goyais.locale", locale);
        set({ locale });
      },
      setTheme: (theme) => set({ theme }),
      setDefaultModelConfigId: (defaultModelConfigId) => set({ defaultModelConfigId }),
      hydrate: async () => {
        const locale = detectInitialLocale();
        await applyLocale(locale);

        // One-time v0.2.0 cleanup for removed runtime-direct/local-auth settings.
        localStorage.removeItem(LEGACY_RUNTIME_URL_STORAGE_KEY);
        localStorage.removeItem(LEGACY_LOCAL_AUTH_PASSWORD_KEY);

        set({
          locale
        });
      }
    }),
    {
      name: "goyais.settings.v3",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        locale: state.locale,
        theme: state.theme,
        defaultModelConfigId: state.defaultModelConfigId
      })
    }
  )
);
