export const TRACE_HEADER = "X-Trace-Id";

export const DOMAIN_ERROR_CODES = {
  VALIDATION: "E_VALIDATION",
  UNAUTHORIZED: "E_UNAUTHORIZED",
  FORBIDDEN: "E_FORBIDDEN",
  NOT_FOUND: "E_NOT_FOUND",
  INTERNAL: "E_INTERNAL"
} as const;

export interface GoyaisError {
  code: string;
  message: string;
  trace_id: string;
  retryable: boolean;
  details?: Record<string, unknown>;
  cause?: string;
  ts?: string;
}

export class HubServerError extends Error {
  readonly code: string;
  readonly retryable: boolean;
  readonly statusCode: number;
  readonly details?: Record<string, unknown>;
  readonly causeType?: string;

  constructor(params: {
    code: string;
    message: string;
    retryable: boolean;
    statusCode: number;
    details?: Record<string, unknown>;
    causeType?: string;
  }) {
    super(params.message);
    this.name = "HubServerError";
    this.code = params.code;
    this.retryable = params.retryable;
    this.statusCode = params.statusCode;
    this.details = params.details;
    this.causeType = params.causeType;
  }
}

export function buildError(
  traceId: string,
  params: {
    code: string;
    message: string;
    retryable: boolean;
    details?: Record<string, unknown>;
    cause?: string;
  }
): GoyaisError {
  const payload: GoyaisError = {
    code: params.code,
    message: params.message,
    trace_id: traceId,
    retryable: params.retryable,
    ts: new Date().toISOString()
  };

  if (params.details) {
    payload.details = params.details;
  }
  if (params.cause) {
    payload.cause = params.cause;
  }

  return payload;
}

export function errorFromUnknown(error: unknown, traceId: string): { statusCode: number; error: GoyaisError } {
  if (error instanceof HubServerError) {
    return {
      statusCode: error.statusCode,
      error: buildError(traceId, {
        code: error.code,
        message: error.message,
        retryable: error.retryable,
        details: error.details,
        cause: error.causeType
      })
    };
  }

  if (error instanceof Error) {
    return {
      statusCode: 500,
      error: buildError(traceId, {
        code: "E_INTERNAL",
        message: "Internal server error.",
        retryable: false,
        cause: error.name
      })
    };
  }

  return {
    statusCode: 500,
    error: buildError(traceId, {
      code: "E_INTERNAL",
      message: "Internal server error.",
      retryable: false,
      cause: "unknown"
    })
  };
}
