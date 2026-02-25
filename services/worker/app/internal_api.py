from __future__ import annotations

from datetime import UTC, datetime
import os
from typing import Any

from fastapi import Request
from fastapi.responses import JSONResponse

from app.errors import standard_error_response

INTERNAL_TOKEN_HEADER = "X-Internal-Token"
AUTHORIZATION_HEADER = "Authorization"
BEARER_PREFIX = "Bearer "
DEFAULT_INTERNAL_TOKEN = "goyais-internal-token"


async def decode_json(request: Request) -> tuple[dict[str, Any], JSONResponse | None]:
    try:
        payload = await request.json()
    except Exception:
        return {}, standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="Invalid JSON request body",
            details={},
        )

    if not isinstance(payload, dict):
        return {}, standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="Request body must be a JSON object",
            details={},
        )

    return payload, None


def parse_queue_index(raw_value: Any, request: Request) -> int | JSONResponse:
    if raw_value is None:
        return 0
    return parse_int(raw_value, "queue_index", request)


def parse_int(raw_value: Any, field: str, request: Request) -> int | JSONResponse:
    try:
        value = int(raw_value)
    except (TypeError, ValueError):
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message=f"{field} must be an integer",
            details={"field": field},
        )

    if value < 0:
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message=f"{field} must be >= 0",
            details={"field": field},
        )
    return value


def safe_int(raw_value: Any, default: int) -> int:
    try:
        value = int(raw_value)
    except (TypeError, ValueError):
        return default
    if value < 0:
        return default
    return value


def is_blank(value: Any) -> bool:
    if value is None:
        return True
    if isinstance(value, str):
        return value.strip() == ""
    return False


def now_iso() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def require_internal_token(request: Request) -> JSONResponse | None:
    expected_token = _resolve_worker_internal_token()
    if expected_token == "":
        return standard_error_response(
            request=request,
            status_code=503,
            code="AUTH_INTERNAL_TOKEN_NOT_CONFIGURED",
            message="Internal token is not configured",
            details={"env": "WORKER_INTERNAL_TOKEN"},
        )

    provided_token = extract_internal_token(request)
    if provided_token == "":
        return standard_error_response(
            request=request,
            status_code=401,
            code="AUTH_INTERNAL_TOKEN_REQUIRED",
            message="Internal token is required",
            details={"header": INTERNAL_TOKEN_HEADER},
        )
    if provided_token != expected_token:
        return standard_error_response(
            request=request,
            status_code=401,
            code="AUTH_INVALID_INTERNAL_TOKEN",
            message="Internal token is invalid",
            details={},
        )
    return None


def extract_internal_token(request: Request) -> str:
    token = request.headers.get(INTERNAL_TOKEN_HEADER, "").strip()
    if token != "":
        return token

    authorization = request.headers.get(AUTHORIZATION_HEADER, "").strip()
    if not authorization.startswith(BEARER_PREFIX):
        return ""
    return authorization[len(BEARER_PREFIX) :].strip()


def _resolve_worker_internal_token() -> str:
    token = os.getenv("WORKER_INTERNAL_TOKEN", "").strip()
    if token != "":
        return token
    if _allow_insecure_internal_token_default():
        return DEFAULT_INTERNAL_TOKEN
    return ""


def _allow_insecure_internal_token_default() -> bool:
    flag = os.getenv("GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN", "").strip().lower()
    return flag in {"1", "true", "yes"}
