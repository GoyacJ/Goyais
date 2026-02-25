from __future__ import annotations

from typing import Any
import urllib.request

from fastapi import FastAPI, Request
from fastapi.testclient import TestClient
import pytest

from app.hub_client import HubClient
from app.internal_api import DEFAULT_INTERNAL_TOKEN, INTERNAL_TOKEN_HEADER, require_internal_token


def _build_auth_client() -> TestClient:
    app = FastAPI()

    @app.get("/check-internal-auth")
    async def check_internal_auth(request: Request):
        auth_error = require_internal_token(request)
        if auth_error is not None:
            return auth_error
        return {"ok": True}

    return TestClient(app)


def test_require_internal_token_returns_503_when_not_configured(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.delenv("WORKER_INTERNAL_TOKEN", raising=False)
    monkeypatch.delenv("GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN", raising=False)
    client = _build_auth_client()

    response = client.get("/check-internal-auth")

    assert response.status_code == 503
    payload = response.json()
    assert payload["code"] == "AUTH_INTERNAL_TOKEN_NOT_CONFIGURED"


def test_require_internal_token_accepts_configured_token(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("WORKER_INTERNAL_TOKEN", "worker-test-token")
    monkeypatch.delenv("GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN", raising=False)
    client = _build_auth_client()

    response = client.get(
        "/check-internal-auth",
        headers={INTERNAL_TOKEN_HEADER: "worker-test-token"},
    )

    assert response.status_code == 200
    assert response.json()["ok"] is True


def test_hub_client_rejects_missing_internal_token(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.delenv("HUB_INTERNAL_TOKEN", raising=False)
    monkeypatch.delenv("GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN", raising=False)
    called = {"urlopen": False}

    def fake_urlopen(*args: Any, **kwargs: Any) -> Any:
        called["urlopen"] = True
        raise AssertionError("urlopen should not be called without HUB_INTERNAL_TOKEN")

    monkeypatch.setattr(urllib.request, "urlopen", fake_urlopen)
    client = HubClient()

    with pytest.raises(RuntimeError, match="HUB_INTERNAL_TOKEN is required"):
        client._request_sync("GET", "/internal/executions/claim", None)

    assert called["urlopen"] is False


def test_hub_client_allows_insecure_default_when_explicitly_enabled(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.delenv("HUB_INTERNAL_TOKEN", raising=False)
    monkeypatch.setenv("GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN", "1")

    client = HubClient()

    assert client.internal_token == DEFAULT_INTERNAL_TOKEN
