from __future__ import annotations

import importlib
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

import app.api.ops as ops_module
import app.api.runs as runs_module
import app.main as main_module
import app.services.secret_resolver as secret_resolver_module


@pytest.fixture
def hub_auth_client(tmp_path: Path, monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setenv("GOYAIS_DB_PATH", str(tmp_path / "runtime-hub-auth.db"))
    monkeypatch.setenv("GOYAIS_RUNTIME_REQUIRE_HUB_AUTH", "true")
    monkeypatch.setenv("GOYAIS_RUNTIME_SHARED_SECRET", "hub-runtime-secret")
    monkeypatch.setenv("GOYAIS_RUNTIME_WORKSPACE_ID", "ws-test")
    monkeypatch.setenv("GOYAIS_RUNTIME_WORKSPACE_ROOT", str(tmp_path / "workspace"))
    monkeypatch.setenv("GOYAIS_HUB_BASE_URL", "http://127.0.0.1:8787")

    module = importlib.reload(main_module)
    ops_module.settings = module.settings
    runs_module.settings = module.settings
    secret_resolver_module.settings = module.settings
    with TestClient(module.app) as client:
        yield client

    monkeypatch.setenv("GOYAIS_RUNTIME_REQUIRE_HUB_AUTH", "false")
    monkeypatch.delenv("GOYAIS_RUNTIME_SHARED_SECRET", raising=False)
    monkeypatch.delenv("GOYAIS_RUNTIME_WORKSPACE_ID", raising=False)
    monkeypatch.delenv("GOYAIS_RUNTIME_WORKSPACE_ROOT", raising=False)
    monkeypatch.delenv("GOYAIS_HUB_BASE_URL", raising=False)
    module = importlib.reload(main_module)
    ops_module.settings = module.settings
    runs_module.settings = module.settings
    secret_resolver_module.settings = module.settings


def test_rejects_request_without_hub_auth_headers(hub_auth_client: TestClient):
    response = hub_auth_client.get("/v1/health", headers={"X-User-Id": "u1", "X-Trace-Id": "trace-1"})
    assert response.status_code == 401
    body = response.json()
    assert body["error"]["code"] == "E_UNAUTHORIZED"


def test_accepts_request_with_required_hub_headers(hub_auth_client: TestClient):
    response = hub_auth_client.get(
        "/v1/health",
        headers={
            "X-Hub-Auth": "hub-runtime-secret",
            "X-User-Id": "u1",
            "X-Trace-Id": "trace-1",
        },
    )
    assert response.status_code == 200
    body = response.json()
    assert body["workspace_id"] == "ws-test"
