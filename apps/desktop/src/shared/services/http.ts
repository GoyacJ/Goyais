import type { StandardError } from "@/shared/types/api";

const DEFAULT_TIMEOUT_MS = 10_000;

export class ApiError extends Error {
  status: number;
  code: string;
  traceId: string;
  details: Record<string, unknown>;

  constructor(input: {
    status: number;
    code: string;
    message: string;
    traceId: string;
    details?: Record<string, unknown>;
  }) {
    super(input.message);
    this.name = "ApiError";
    this.status = input.status;
    this.code = input.code;
    this.traceId = input.traceId;
    this.details = input.details ?? {};
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  headers?: Record<string, string>;
  token?: string;
};

export type ApiClient = {
  request<T>(path: string, options?: RequestOptions): Promise<T>;
  get<T>(path: string, options?: Omit<RequestOptions, "method" | "body">): Promise<T>;
  post<T>(path: string, body?: unknown, options?: Omit<RequestOptions, "method" | "body">): Promise<T>;
};

export function createApiClient(baseURL: string): ApiClient {
  const normalizedBaseURL = baseURL.replace(/\/$/, "");

  async function request<T>(path: string, options?: RequestOptions): Promise<T> {
    const traceId = createTraceId();
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS);

    try {
      const headers: Record<string, string> = {
        "X-Trace-Id": traceId,
        ...(options?.headers ?? {})
      };

      if (options?.token) {
        headers.Authorization = `Bearer ${options.token}`;
      }

      const hasBody = options?.body !== undefined;
      if (hasBody && headers["Content-Type"] === undefined) {
        headers["Content-Type"] = "application/json";
      }

      const response = await fetch(`${normalizedBaseURL}${path}`, {
        method: options?.method ?? "GET",
        headers,
        body: hasBody ? JSON.stringify(options?.body) : undefined,
        signal: controller.signal
      });

      if (!response.ok) {
        throw await buildApiError(response, traceId);
      }

      if (response.status === 204) {
        return undefined as T;
      }

      return (await response.json()) as T;
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }

      const message = error instanceof Error ? error.message : "Unknown request error";
      throw new ApiError({
        status: 0,
        code: "NETWORK_ERROR",
        message,
        traceId,
        details: {}
      });
    } finally {
      clearTimeout(timeout);
    }
  }

  return {
    request,
    get: <T>(path: string, options?: Omit<RequestOptions, "method" | "body">) => request<T>(path, options),
    post: <T>(path: string, body?: unknown, options?: Omit<RequestOptions, "method" | "body">) =>
      request<T>(path, { ...(options ?? {}), method: "POST", body })
  };
}

function createTraceId(): string {
  const randomPart = Math.random().toString(36).slice(2, 10);
  return `tr_web_${Date.now().toString(36)}_${randomPart}`;
}

async function buildApiError(response: Response, fallbackTraceId: string): Promise<ApiError> {
  const responseTraceId = response.headers.get("X-Trace-Id") ?? fallbackTraceId;

  try {
    const payload = (await response.json()) as Partial<StandardError>;
    const code = typeof payload.code === "string" ? payload.code : "API_ERROR";
    const message = typeof payload.message === "string" ? payload.message : `HTTP ${response.status}`;
    const traceId = typeof payload.trace_id === "string" ? payload.trace_id : responseTraceId;
    const details = isRecord(payload.details) ? payload.details : {};

    return new ApiError({
      status: response.status,
      code,
      message,
      traceId,
      details
    });
  } catch {
    return new ApiError({
      status: response.status,
      code: "API_ERROR",
      message: `HTTP ${response.status}`,
      traceId: responseTraceId,
      details: {}
    });
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
