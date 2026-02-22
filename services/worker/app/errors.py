from typing import Any

from fastapi import Request
from fastapi.responses import JSONResponse

from app.trace import TRACE_HEADER, generate_trace_id


def _resolve_trace_id(request: Request) -> str:
    trace_id = getattr(request.state, "trace_id", "")
    if trace_id:
        return trace_id

    trace_from_header = request.headers.get(TRACE_HEADER, "").strip()
    if trace_from_header:
        return trace_from_header

    return generate_trace_id()


def standard_error_response(
    request: Request,
    status_code: int,
    code: str,
    message: str,
    details: dict[str, Any] | None = None,
) -> JSONResponse:
    trace_id = _resolve_trace_id(request)
    payload = {
        "code": code,
        "message": message,
        "details": details or {},
        "trace_id": trace_id,
    }
    return JSONResponse(
        status_code=status_code,
        content=payload,
        headers={TRACE_HEADER: trace_id},
    )
