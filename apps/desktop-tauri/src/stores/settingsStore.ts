import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { localConfigRead } from "@/api/localConfigClient";
import { applyLocale, detectInitialLocale } from "@/i18n";
import type { SupportedLocale } from "@/i18n/types";
import {
  createDefaultLocalProcessConfig,
  type LocalProcessConfigV1,
  normalizeHubUrl} from "@/types/localProcessConfig";

export type ThemeMode = "dark" | "light";

const LEGACY_RUNTIME_URL_STORAGE_KEY = "goyais.runtimeUrl";
const LEGACY_LOCAL_AUTH_PASSWORD_KEY = "goyais.localAutoPassword";
const LEGACY_LOCAL_HUB_URL_STORAGE_KEY = "goyais.localHubUrl";

interface SettingsState {
  locale: SupportedLocale;
  theme: ThemeMode;
  defaultModelConfigId?: string;
  localProcessConfig: LocalProcessConfigV1;
  setLocale: (locale: SupportedLocale) => Promise<void>;
  setTheme: (theme: ThemeMode) => void;
  setDefaultModelConfigId: (modelConfigId?: string) => void;
  setLocalProcessConfig: (updater: (current: LocalProcessConfigV1) => LocalProcessConfigV1) => void;
  hydrate: () => Promise<void>;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      locale: "zh-CN",
      theme: "dark",
      defaultModelConfigId: undefined,
      localProcessConfig: createDefaultLocalProcessConfig(),
      setLocale: async (locale) => {
        await applyLocale(locale);
        localStorage.setItem("goyais.locale", locale);
        set({ locale });
      },
      setTheme: (theme) => set({ theme }),
      setDefaultModelConfigId: (defaultModelConfigId) => set({ defaultModelConfigId }),
      setLocalProcessConfig: (updater) =>
        set((state) => ({
          localProcessConfig: updater(state.localProcessConfig)
        })),
      hydrate: async () => {
        const locale = detectInitialLocale();
        await applyLocale(locale);
        const localProcessConfig = await localConfigRead();

        // One-time v0.2.0 cleanup for removed runtime-direct/local-auth settings.
        localStorage.removeItem(LEGACY_RUNTIME_URL_STORAGE_KEY);
        localStorage.removeItem(LEGACY_LOCAL_AUTH_PASSWORD_KEY);
        const legacyLocalHubUrl = localStorage.getItem(LEGACY_LOCAL_HUB_URL_STORAGE_KEY);

        set(() => {
          const nextConfig = localProcessConfig;
          if (!legacyLocalHubUrl?.trim()) {
            return { locale, localProcessConfig: nextConfig };
          }

          const normalized = normalizeHubUrl(legacyLocalHubUrl);
          return {
            locale,
            localProcessConfig: {
              ...nextConfig,
              connections: {
                ...nextConfig.connections,
                localHubUrl: normalized
              }
            }
          };
        });
      }
    }),
    {
      name: "goyais.settings.v4",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        locale: state.locale,
        theme: state.theme,
        defaultModelConfigId: state.defaultModelConfigId,
        localProcessConfig: state.localProcessConfig
      })
    }
  )
);
