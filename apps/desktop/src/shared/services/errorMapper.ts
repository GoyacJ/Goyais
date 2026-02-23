import { ApiError } from "@/shared/services/http";

export type UiError = {
  code: string;
  message: string;
  traceId: string;
  isForbidden: boolean;
};

export function mapToUiError(error: unknown): UiError {
  if (error instanceof ApiError) {
    return {
      code: error.code,
      message: error.message,
      traceId: error.traceId,
      isForbidden: error.status === 403
    };
  }

  if (error instanceof Error) {
    return {
      code: "UNKNOWN_ERROR",
      message: error.message,
      traceId: "n/a",
      isForbidden: false
    };
  }

  return {
    code: "UNKNOWN_ERROR",
    message: "Unknown error",
    traceId: "n/a",
    isForbidden: false
  };
}

export function toDisplayError(error: unknown): string {
  const mapped = mapToUiError(error);
  return `${mapped.code}: ${mapped.message} (trace_id: ${mapped.traceId})`;
}
