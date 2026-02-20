from __future__ import annotations

from fastapi import Header

from app.errors import GoyaisApiError


def require_runtime_token(token: str, x_runtime_token: str = Header(default="")) -> None:
    if x_runtime_token != token:
        raise GoyaisApiError(
            code="E_SYNC_AUTH",
            message="Invalid runtime token.",
            retryable=False,
            status_code=401,
            cause="runtime_token",
        )
