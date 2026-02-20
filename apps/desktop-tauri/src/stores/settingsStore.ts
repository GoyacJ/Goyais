import { create } from "zustand";

interface SettingsState {
  runtimeUrl: string;
  setRuntimeUrl: (value: string) => void;
}

export const useSettingsStore = create<SettingsState>((set) => ({
  runtimeUrl: "http://127.0.0.1:8040",
  setRuntimeUrl: (runtimeUrl) => set({ runtimeUrl })
}));
