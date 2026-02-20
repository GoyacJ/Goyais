export interface ApiErrorShape {
  code: string;
  message: string;
  retryable: boolean;
  status?: number;
  detail?: string;
}

export class ApiError extends Error {
  code: string;
  retryable: boolean;
  status?: number;
  detail?: string;

  constructor(shape: ApiErrorShape) {
    super(shape.message);
    this.name = "ApiError";
    this.code = shape.code;
    this.retryable = shape.retryable;
    this.status = shape.status;
    this.detail = shape.detail;
  }
}

async function parseResponseDetail(response: Response): Promise<string> {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      const payload = (await response.json()) as Record<string, unknown>;
      if (typeof payload.detail === "string") return payload.detail;
      return JSON.stringify(payload);
    } catch {
      return response.statusText;
    }
  }

  try {
    return await response.text();
  } catch {
    return response.statusText;
  }
}

export async function normalizeHttpError(response: Response): Promise<ApiError> {
  const detail = await parseResponseDetail(response);
  return new ApiError({
    code: `HTTP_${response.status}`,
    message: detail || `HTTP ${response.status}`,
    status: response.status,
    retryable: response.status >= 500,
    detail
  });
}

export function normalizeUnknownError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }

  if (error instanceof Error) {
    return new ApiError({
      code: "NETWORK_OR_RUNTIME_ERROR",
      message: error.message,
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
