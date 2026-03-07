import { defineStore } from "pinia";

import { pinia } from "@/shared/stores/pinia";

export type ToastTone = "error" | "warning" | "info" | "403" | "disconnected" | "retrying";

export type GlobalToastOptions = {
  tone: ToastTone;
  message: string;
  key?: string;
  durationMs?: number;
  persistent?: boolean;
};

export type GlobalToastItem = {
  id: string;
  key: string;
  tone: ToastTone;
  message: string;
  persistent: boolean;
  durationMs: number;
  createdAt: number;
};

type ToastState = {
  items: GlobalToastItem[];
};

const DEFAULT_DURATION_MS = 3200;
const MAX_TOAST_ITEMS = 3;

const toastTimers = new Map<string, ReturnType<typeof setTimeout>>();

const useToastStoreDefinition = defineStore("toast", {
  state: (): ToastState => ({
    items: []
  })
});

export const useToastStore = useToastStoreDefinition;
export const toastStore = useToastStoreDefinition(pinia);

export function showToast(options: GlobalToastOptions): string {
  const message = options.message.trim();
  if (message === "") {
    return "";
  }

  const key = (options.key ?? "").trim();
  const persistent = options.persistent === true;
  const durationMs = resolveDuration(options.durationMs);

  if (key !== "") {
    const existing = toastStore.items.find((item) => item.key === key);
    if (existing) {
      clearToastTimer(existing.id);
      existing.tone = options.tone;
      existing.message = message;
      existing.persistent = persistent;
      existing.durationMs = durationMs;
      existing.createdAt = Date.now();
      scheduleDismiss(existing);
      return existing.id;
    }
  }

  const nextItem: GlobalToastItem = {
    id: createToastId(),
    key,
    tone: options.tone,
    message,
    persistent,
    durationMs,
    createdAt: Date.now()
  };

  toastStore.items.push(nextItem);
  scheduleDismiss(nextItem);

  while (toastStore.items.length > MAX_TOAST_ITEMS) {
    const oldest = toastStore.items[0];
    if (!oldest) {
      break;
    }
    dismissToast(oldest.id);
  }

  return nextItem.id;
}

export function dismissToast(id: string): void {
  const normalizedId = id.trim();
  if (normalizedId === "") {
    return;
  }

  const index = toastStore.items.findIndex((item) => item.id === normalizedId);
  if (index === -1) {
    clearToastTimer(normalizedId);
    return;
  }

  clearToastTimer(normalizedId);
  toastStore.items.splice(index, 1);
}

export function dismissToastByKey(key: string): void {
  const normalizedKey = key.trim();
  if (normalizedKey === "") {
    return;
  }

  const targetIds = toastStore.items.filter((item) => item.key === normalizedKey).map((item) => item.id);
  for (const id of targetIds) {
    dismissToast(id);
  }
}

export function clearToasts(): void {
  for (const item of toastStore.items) {
    clearToastTimer(item.id);
  }
  toastStore.items = [];
}

export function resetToastStore(): void {
  clearToasts();
}

function resolveDuration(durationMs: number | undefined): number {
  if (typeof durationMs !== "number" || !Number.isFinite(durationMs)) {
    return DEFAULT_DURATION_MS;
  }
  if (durationMs <= 0) {
    return DEFAULT_DURATION_MS;
  }
  return Math.floor(durationMs);
}

function scheduleDismiss(item: GlobalToastItem): void {
  clearToastTimer(item.id);
  if (item.persistent) {
    return;
  }

  const timer = setTimeout(() => {
    dismissToast(item.id);
  }, item.durationMs);
  toastTimers.set(item.id, timer);
}

function clearToastTimer(id: string): void {
  const timer = toastTimers.get(id);
  if (timer) {
    clearTimeout(timer);
    toastTimers.delete(id);
  }
}

function createToastId(): string {
  return `toast_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
}
