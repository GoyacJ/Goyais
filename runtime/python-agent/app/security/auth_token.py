from __future__ import annotations

from fastapi import Header, HTTPException


def require_runtime_token(token: str, x_runtime_token: str = Header(default="")) -> None:
    if x_runtime_token != token:
        raise HTTPException(status_code=401, detail="invalid runtime token")
