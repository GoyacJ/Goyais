from __future__ import annotations

import asyncio
from types import SimpleNamespace

from app.services import secret_resolver


class _MockResponse:
    def __init__(self, status_code: int, payload: dict):
        self.status_code = status_code
        self._payload = payload

    def json(self):
        return self._payload


def test_resolve_secret_via_hub_uses_internal_endpoint(monkeypatch):
    observed = {}

    class _MockAsyncClient:
        def __init__(self, timeout: float):
            observed["timeout"] = timeout

        async def __aenter__(self):
            return self

        async def __aexit__(self, exc_type, exc, tb):
            return False

        async def post(self, url: str, headers: dict, json: dict):
            observed["url"] = url
            observed["headers"] = headers
            observed["json"] = json
            return _MockResponse(200, {"value": "sk-resolved"})

    monkeypatch.setattr(
        secret_resolver,
        "settings",
        SimpleNamespace(
            runtime_require_hub_auth=True,
            runtime_shared_secret="hub-runtime-secret",
            hub_base_url="http://127.0.0.1:8787",
            runtime_workspace_id="ws-remote",
        ),
    )
    monkeypatch.setattr(secret_resolver.httpx, "AsyncClient", _MockAsyncClient)

    resolved = asyncio.run(secret_resolver.resolve_secret_via_hub("secret:abc", "trace-xyz"))
    assert resolved == "sk-resolved"
    assert observed["url"] == "http://127.0.0.1:8787/internal/secrets/resolve"
    assert observed["headers"]["X-Hub-Auth"] == "hub-runtime-secret"
    assert observed["headers"]["X-Trace-Id"] == "trace-xyz"
    assert observed["json"] == {
        "workspace_id": "ws-remote",
        "secret_ref": "secret:abc",
    }
