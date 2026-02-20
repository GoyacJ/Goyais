from __future__ import annotations

import httpx

from app.config import load_settings

settings = load_settings()


async def resolve_secret_via_hub(secret_ref: str, trace_id: str) -> str:
    if not settings.runtime_require_hub_auth:
        raise RuntimeError("secret:* refs require GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true")

    if not settings.runtime_shared_secret:
        raise RuntimeError("GOYAIS_RUNTIME_SHARED_SECRET is required for hub secret resolution")

    base_url = settings.hub_base_url.rstrip("/")
    if not base_url:
        raise RuntimeError("GOYAIS_HUB_BASE_URL is required for hub secret resolution")

    async with httpx.AsyncClient(timeout=10.0) as client:
        response = await client.post(
            f"{base_url}/internal/secrets/resolve",
            headers={
                "X-Hub-Auth": settings.runtime_shared_secret,
                "X-Trace-Id": trace_id,
            },
            json={
                "workspace_id": settings.runtime_workspace_id,
                "secret_ref": secret_ref,
            },
        )

    if response.status_code != 200:
        raise RuntimeError("failed to resolve secret_ref via hub")

    payload = response.json()
    value = payload.get("value")
    if not isinstance(value, str) or not value:
        raise RuntimeError("invalid secret resolve response")
    return value
