import { reactive } from "vue";

type FallbackHit = {
  scope: string;
  reason: string;
  status: number;
  at: string;
};

type RuntimeState = {
  fallbackHits: Record<string, FallbackHit>;
};

const initialState: RuntimeState = {
  fallbackHits: {}
};

export const runtimeStore = reactive<RuntimeState>({ ...initialState });

export function resetRuntimeStore(): void {
  runtimeStore.fallbackHits = {};
}

export function markFallback(scope: string, status: number, reason: string): void {
  runtimeStore.fallbackHits[scope] = {
    scope,
    status,
    reason,
    at: new Date().toISOString()
  };
}

export function clearFallback(scope: string): void {
  delete runtimeStore.fallbackHits[scope];
}

export function isFallbackActive(scope: string): boolean {
  return runtimeStore.fallbackHits[scope] !== undefined;
}
