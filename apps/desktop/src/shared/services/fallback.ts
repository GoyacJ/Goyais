import { ApiError } from "@/shared/services/http";
import { clearFallback, markFallback } from "@/shared/stores/runtimeStore";

const apiMode = (import.meta.env.VITE_API_MODE ?? "hybrid").toLowerCase();
const fallbackEnabled = (import.meta.env.VITE_ENABLE_MOCK_FALLBACK ?? "true").toLowerCase() !== "false";

type AsyncOrSync<T> = Promise<T> | T;

export async function withApiFallback<T>(scope: string, realCall: () => Promise<T>, fallbackCall: () => AsyncOrSync<T>): Promise<T> {
  if (apiMode === "mock") {
    markFallback(scope, 0, "mock_mode");
    return await fallbackCall();
  }

  try {
    const result = await realCall();
    clearFallback(scope);
    return result;
  } catch (error) {
    if (!shouldFallback(error)) {
      throw error;
    }

    const { status, reason } = normalizeError(error);
    markFallback(scope, status, reason);
    return await fallbackCall();
  }
}

function shouldFallback(error: unknown): boolean {
  if (fallbackEnabled === false) {
    return false;
  }

  if (apiMode === "hybrid") {
    if (error instanceof ApiError) {
      return [0, 404, 405, 501, 502, 503].includes(error.status);
    }
    return error instanceof Error;
  }

  return false;
}

function normalizeError(error: unknown): { status: number; reason: string } {
  if (error instanceof ApiError) {
    return { status: error.status, reason: error.code };
  }

  if (error instanceof Error) {
    return { status: 0, reason: error.message };
  }

  return { status: 0, reason: "unknown" };
}
