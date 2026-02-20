import { create } from "zustand";

import { applyLocale, detectInitialLocale } from "@/i18n";
import type { SupportedLocale } from "@/i18n/types";

interface SettingsState {
  runtimeUrl: string;
  locale: SupportedLocale;
  setRuntimeUrl: (value: string) => void;
  setLocale: (locale: SupportedLocale) => Promise<void>;
  hydrateLocale: () => Promise<void>;
}

export const useSettingsStore = create<SettingsState>((set) => ({
  runtimeUrl: "http://127.0.0.1:8040",
  locale: "zh-CN",
  setRuntimeUrl: (runtimeUrl) => set({ runtimeUrl }),
  setLocale: async (locale) => {
    await applyLocale(locale);
    set({ locale });
  },
  hydrateLocale: async () => {
    const locale = detectInitialLocale();
    await applyLocale(locale);
    set({ locale });
  }
}));
