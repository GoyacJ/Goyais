from __future__ import annotations

import importlib
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

import app.api.ops as ops_module
import app.main as main_module
import app.services.secret_resolver as secret_resolver_module


@pytest.fixture
def remote_client(tmp_path: Path, monkeypatch: pytest.MonkeyPatch):
    workspace = tmp_path / "workspace"
    workspace.mkdir(parents=True, exist_ok=True)
    (workspace / "README.md").write_text("# Remote Workspace\n", encoding="utf-8")

    monkeypatch.setenv("GOYAIS_DB_PATH", str(tmp_path / "runtime-remote.db"))
    monkeypatch.setenv("GOYAIS_RUNTIME_REQUIRE_HUB_AUTH", "true")
    monkeypatch.setenv("GOYAIS_RUNTIME_SHARED_SECRET", "hub-runtime-secret")
    monkeypatch.setenv("GOYAIS_RUNTIME_WORKSPACE_ID", "ws-remote")
    monkeypatch.setenv("GOYAIS_RUNTIME_WORKSPACE_ROOT", str(workspace))
    monkeypatch.setenv("GOYAIS_HUB_BASE_URL", "http://127.0.0.1:8787")

    module = importlib.reload(main_module)
    ops_module.settings = module.settings
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
    secret_resolver_module.settings = module.settings


def _auth_headers(user_id: str, trace_id: str) -> dict[str, str]:
    return {
        "X-Hub-Auth": "hub-runtime-secret",
        "X-User-Id": user_id,
        "X-Trace-Id": trace_id,
    }


def test_internal_execution_request_preserves_trace_header(remote_client: TestClient):
    response = remote_client.post(
        "/internal/executions",
        headers=_auth_headers("runner-1", "trace-exec-1"),
        json={
            "execution_id": "exec-1",
            "trace_id": "trace-exec-1",
            "workspace_id": "ws-remote",
            "project_id": "project-1",
            "session_id": "session-1",
            "user_message": "update readme",
            "repo_root": "/tmp/ignored-by-remote",
            "use_worktree": False,
        },
    )

    assert response.status_code == 202
    assert response.headers.get("X-Trace-Id") == "trace-exec-1"


def test_internal_confirmation_request_preserves_trace_header(remote_client: TestClient):
    response = remote_client.post(
        "/internal/confirmations",
        headers=_auth_headers("reviewer-1", "trace-confirm-1"),
        json={
            "execution_id": "exec-1",
            "call_id": "call-1",
            "decision": "approved",
        },
    )

    assert response.status_code == 200
    assert response.headers.get("X-Trace-Id") == "trace-confirm-1"
