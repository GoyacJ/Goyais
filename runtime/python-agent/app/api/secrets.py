from __future__ import annotations

import os

from fastapi import APIRouter, Header, HTTPException

from app.config import load_settings

router = APIRouter(prefix="/v1", tags=["secrets"])
settings = load_settings()


@router.get("/secrets/{provider}/{profile}")
async def get_secret(provider: str, profile: str, x_runtime_token: str = Header(default="")):
    if x_runtime_token != settings.runtime_secret_token:
        raise HTTPException(status_code=401, detail="invalid runtime token")

    env_key = f"GOYAIS_SECRET_{provider.upper()}_{profile.upper()}"
    secret = os.getenv(env_key)
    if not secret:
        raise HTTPException(status_code=404, detail="secret not found")

    return {"secret_ref": f"keychain:{provider}:{profile}", "secret": secret}
