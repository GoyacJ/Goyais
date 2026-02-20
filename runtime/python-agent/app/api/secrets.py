from __future__ import annotations

import os

from fastapi import APIRouter, Header

from app.config import load_settings
from app.errors import GoyaisApiError

router = APIRouter(prefix="/v1", tags=["secrets"])
settings = load_settings()


@router.get("/secrets/{provider}/{profile}")
async def get_secret(provider: str, profile: str, x_runtime_token: str = Header(default="")):
    if x_runtime_token != settings.runtime_secret_token:
        raise GoyaisApiError(
            code="E_SYNC_AUTH",
            message="Invalid runtime token.",
            retryable=False,
            status_code=401,
            cause="runtime_token",
        )

    env_key = f"GOYAIS_SECRET_{provider.upper()}_{profile.upper()}"
    secret = os.getenv(env_key)
    if not secret:
        raise GoyaisApiError(
            code="E_PROVIDER_AUTH",
            message="Secret not found.",
            retryable=False,
            status_code=404,
            cause="secret_lookup",
        )

    return {"secret_ref": f"keychain:{provider}:{profile}", "secret": secret}
