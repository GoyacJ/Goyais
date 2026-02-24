import {
  GENERAL_SETTINGS_STORAGE_KEY,
  type GeneralSettings,
  normalizeGeneralSettings
} from "@/modules/workspace/schemas/generalSettings";

type TauriStore = {
  get: (key: string) => Promise<unknown>;
  set: (key: string, value: unknown) => Promise<void>;
  save: () => Promise<void>;
};

type TauriStoreModule = {
  load: (path: string) => Promise<TauriStore>;
};

export type GeneralSettingsPersistenceDriver = {
  load: () => Promise<GeneralSettings | null>;
  save: (value: GeneralSettings) => Promise<void>;
};

const STORE_FILE = "goyais.settings.dat";

let cachedStorePromise: Promise<TauriStore | null> | null = null;

export const generalSettingsPersistenceDriver: GeneralSettingsPersistenceDriver = {
  load: loadGeneralSettings,
  save: saveGeneralSettings
};

export async function loadGeneralSettings(): Promise<GeneralSettings | null> {
  const store = await getTauriStore();
  if (store) {
    const value = await store.get(GENERAL_SETTINGS_STORAGE_KEY);
    return value == null ? null : normalizeGeneralSettings(value);
  }

  if (!canUseLocalStorage()) {
    return null;
  }

  const raw = window.localStorage.getItem(GENERAL_SETTINGS_STORAGE_KEY);
  if (!raw) {
    return null;
  }

  try {
    return normalizeGeneralSettings(JSON.parse(raw));
  } catch {
    return null;
  }
}

export async function saveGeneralSettings(value: GeneralSettings): Promise<void> {
  const store = await getTauriStore();
  if (store) {
    await store.set(GENERAL_SETTINGS_STORAGE_KEY, value);
    await store.save();
    return;
  }

  if (!canUseLocalStorage()) {
    return;
  }

  window.localStorage.setItem(GENERAL_SETTINGS_STORAGE_KEY, JSON.stringify(value));
}

async function getTauriStore(): Promise<TauriStore | null> {
  if (cachedStorePromise) {
    return cachedStorePromise;
  }

  cachedStorePromise = loadTauriStore();
  return cachedStorePromise;
}

async function loadTauriStore(): Promise<TauriStore | null> {
  const module = await importStoreModule();
  if (!module || typeof module.load !== "function") {
    return null;
  }

  try {
    return await module.load(STORE_FILE);
  } catch {
    return null;
  }
}

async function importStoreModule(): Promise<TauriStoreModule | null> {
  try {
    const dynamicImport = new Function("specifier", "return import(specifier);") as (specifier: string) => Promise<unknown>;
    const module = await dynamicImport("@tauri-apps/plugin-store");
    return module as TauriStoreModule;
  } catch {
    return null;
  }
}

function canUseLocalStorage(): boolean {
  return typeof window !== "undefined" && typeof window.localStorage !== "undefined";
}
