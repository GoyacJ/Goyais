export interface ApiErrorShape {
  code: string;
  message: string;
  retryable: boolean;
  traceId?: string;
  status?: number;
  detail?: string;
}

export class ApiError extends Error {
  code: string;
  retryable: boolean;
  traceId?: string;
  status?: number;
  detail?: string;

  constructor(shape: ApiErrorShape) {
    super(shape.message);
    this.name = "ApiError";
    this.code = shape.code;
    this.retryable = shape.retryable;
    this.traceId = shape.traceId;
    this.status = shape.status;
    this.detail = shape.detail;
  }
}

async function parseResponseError(response: Response): Promise<ApiErrorShape> {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      const payload = (await response.json()) as Record<string, unknown>;
      const goyaisError = payload.error as Record<string, unknown> | undefined;
      if (goyaisError && typeof goyaisError === "object") {
        return {
          code: String(goyaisError.code ?? `HTTP_${response.status}`),
          message: String(goyaisError.message ?? `HTTP ${response.status}`),
          retryable: Boolean(goyaisError.retryable),
          traceId: typeof goyaisError.trace_id === "string" ? goyaisError.trace_id : undefined,
          status: response.status,
          detail: JSON.stringify(payload)
        };
      }
      return {
        code: `HTTP_${response.status}`,
        message: JSON.stringify(payload),
        retryable: response.status >= 500,
        status: response.status,
        detail: JSON.stringify(payload)
      };
    } catch {
      return {
        code: `HTTP_${response.status}`,
        message: response.statusText || `HTTP ${response.status}`,
        retryable: response.status >= 500,
        status: response.status,
        detail: response.statusText
      };
    }
  }

  try {
    const detail = await response.text();
    return {
      code: `HTTP_${response.status}`,
      message: detail || `HTTP ${response.status}`,
      retryable: response.status >= 500,
      status: response.status,
      detail
    };
  } catch {
    return {
      code: `HTTP_${response.status}`,
      message: response.statusText || `HTTP ${response.status}`,
      retryable: response.status >= 500,
      status: response.status,
      detail: response.statusText
    };
  }
}

export async function normalizeHttpError(response: Response): Promise<ApiError> {
  return new ApiError(await parseResponseError(response));
}

function normalizeRuntimeMessage(message: string): string {
  const normalized = message.trim().toLowerCase();
  if (
    normalized === "load failed"
    || normalized === "failed to fetch"
    || normalized.includes("networkerror")
  ) {
    return "Network request failed. Please verify the Hub server URL and make sure the service is running.";
  }
  return message;
}

export function normalizeUnknownError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }

  if (error instanceof Error) {
    return new ApiError({
      code: "NETWORK_OR_RUNTIME_ERROR",
      message: normalizeRuntimeMessage(error.message),
      retryable: true,
      detail: error.stack
    });
  }

  return new ApiError({
    code: "UNKNOWN_ERROR",
    message: "Unknown error",
    retryable: false
  });
}
