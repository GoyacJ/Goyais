from __future__ import annotations

import asyncio
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any

import httpx
from fastapi import HTTPException
from fastapi.exceptions import RequestValidationError

from app.security.command_guard import CommandGuardError
from app.security.path_guard import PathGuardError
from app.tools.patch_tools import PatchApplyError

DEFAULT_INTERNAL_MESSAGE = "Internal server error"


@dataclass(slots=True)
class GoyaisApiError(Exception):
    code: str
    message: str
    retryable: bool
    status_code: int
    details: dict[str, Any] | None = None
    cause: str | None = None


def build_goyais_error(
    *,
    code: str,
    message: str,
    trace_id: str,
    retryable: bool,
    details: dict[str, Any] | None = None,
    cause: str | None = None,
) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "code": code,
        "message": message,
        "trace_id": trace_id,
        "retryable": retryable,
        "ts": datetime.now(tz=timezone.utc).isoformat(),
    }
    if details:
        payload["details"] = details
    if cause:
        payload["cause"] = cause
    return payload


def error_response(
    *,
    code: str,
    message: str,
    trace_id: str,
    retryable: bool,
    details: dict[str, Any] | None = None,
    cause: str | None = None,
) -> dict[str, Any]:
    return {
        "error": build_goyais_error(
            code=code,
            message=message,
            trace_id=trace_id,
            retryable=retryable,
            details=details,
            cause=cause,
        )
    }


def error_from_exception(exc: Exception, trace_id: str) -> tuple[int, dict[str, Any]]:
    if isinstance(exc, GoyaisApiError):
        return (
            exc.status_code,
            error_response(
                code=exc.code,
                message=exc.message,
                trace_id=trace_id,
                retryable=exc.retryable,
                details=exc.details,
                cause=exc.cause,
            ),
        )

    if isinstance(exc, RequestValidationError):
        return (
            422,
            error_response(
                code="E_SCHEMA_INVALID",
                message="Request validation failed.",
                trace_id=trace_id,
                retryable=False,
                details={"errors": exc.errors()},
                cause="request_validation_error",
            ),
        )

    if isinstance(exc, HTTPException):
        retryable = exc.status_code >= 500
        code = "E_INTERNAL" if retryable else "E_SCHEMA_INVALID"
        if exc.status_code in {401, 403}:
            code = "E_SYNC_AUTH"
        return (
            exc.status_code,
            error_response(
                code=code,
                message=str(exc.detail),
                trace_id=trace_id,
                retryable=retryable,
                cause="http_exception",
            ),
        )

    if isinstance(exc, PathGuardError):
        return (
            400,
            error_response(
                code="E_PATH_ESCAPE",
                message="Path escapes workspace boundary.",
                trace_id=trace_id,
                retryable=False,
                cause="path_guard",
            ),
        )

    if isinstance(exc, CommandGuardError):
        return (
            403,
            error_response(
                code="E_TOOL_DENIED",
                message="Tool execution denied by policy.",
                trace_id=trace_id,
                retryable=False,
                cause="command_guard",
            ),
        )

    if isinstance(exc, PatchApplyError):
        return (
            400,
            error_response(
                code="E_SCHEMA_INVALID",
                message="Patch could not be applied.",
                trace_id=trace_id,
                retryable=False,
                cause="patch_apply",
            ),
        )

    if isinstance(exc, asyncio.TimeoutError):
        return (
            504,
            error_response(
                code="E_TOOL_TIMEOUT",
                message="Operation timed out.",
                trace_id=trace_id,
                retryable=True,
                cause="timeout",
            ),
        )

    if isinstance(exc, httpx.TimeoutException):
        return (
            503,
            error_response(
                code="E_NETWORK_TIMEOUT",
                message="Network timeout.",
                trace_id=trace_id,
                retryable=True,
                cause="network_timeout",
            ),
        )

    if isinstance(exc, httpx.HTTPStatusError):
        status = int(exc.response.status_code)
        if status in {401, 403}:
            return (
                status,
                error_response(
                    code="E_SYNC_AUTH",
                    message="Sync authentication failed.",
                    trace_id=trace_id,
                    retryable=False,
                    cause="sync_auth",
                ),
            )
        if status == 429:
            return (
                status,
                error_response(
                    code="E_PROVIDER_RATE_LIMIT",
                    message="Provider rate limited request.",
                    trace_id=trace_id,
                    retryable=True,
                    cause="provider_rate_limit",
                ),
            )
        return (
            status,
            error_response(
                code="E_NETWORK",
                message="Network request failed.",
                trace_id=trace_id,
                retryable=status >= 500,
                cause="http_status_error",
            ),
        )

    message = str(exc)
    if "model_config_id is required" in message:
        return (
            400,
            error_response(
                code="E_SCHEMA_INVALID",
                message="model_config_id is required for graph/deepagents mode.",
                trace_id=trace_id,
                retryable=False,
                cause="missing_model_config",
            ),
        )
    if "API key not found" in message or "missing secret_ref" in message:
        return (
            401,
            error_response(
                code="E_PROVIDER_AUTH",
                message="Provider credentials are not configured.",
                trace_id=trace_id,
                retryable=False,
                cause="provider_auth",
            ),
        )

    if isinstance(exc, (KeyError, ValueError, TypeError)):
        return (
            400,
            error_response(
                code="E_SCHEMA_INVALID",
                message="Invalid request or payload shape.",
                trace_id=trace_id,
                retryable=False,
                cause=exc.__class__.__name__,
            ),
        )

    return (
        500,
        error_response(
            code="E_INTERNAL",
            message=DEFAULT_INTERNAL_MESSAGE,
            trace_id=trace_id,
            retryable=False,
            cause=exc.__class__.__name__,
        ),
    )
