from __future__ import annotations

import asyncio
import importlib
import time
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

import app.api.ops as ops_module
import app.api.runs as runs_module
import app.main as main_module
from app.deps import get_repo
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


def _auth_headers(user_id: str, trace_id: str) -> dict[str, str]:
    return {
        "X-Hub-Auth": "hub-runtime-secret",
        "X-User-Id": user_id,
        "X-Trace-Id": trace_id,
    }


def test_events_trace_and_audit_user_are_injected(remote_client: TestClient):
    create_response = remote_client.post(
        "/v1/runs",
        headers=_auth_headers("runner-1", "trace-run-1"),
        json={
            "project_id": "project-1",
            "session_id": "session-1",
            "input": "update readme",
            "model_config_id": "",
            "workspace_path": "/tmp/ignored-by-remote",
            "options": {"use_worktree": False},
        },
    )
    assert create_response.status_code == 200
    run_id = create_response.json()["run_id"]

    call_id = None
    replay_events = []
    for _ in range(80):
        replay_response = remote_client.get(
            f"/v1/runs/{run_id}/events/replay",
            headers=_auth_headers("runner-1", "trace-run-1"),
        )
        assert replay_response.status_code == 200
        replay_events = replay_response.json()["events"]
        for event in replay_events:
            assert event["trace_id"] == "trace-run-1"
            assert event["payload"]["trace_id"] == "trace-run-1"
            if event["type"] == "tool_call" and event["payload"].get("requires_confirmation") is True:
                call_id = event["payload"]["call_id"]
        if call_id:
            break
        time.sleep(0.05)

    assert call_id is not None

    confirm_response = remote_client.post(
        "/v1/tool-confirmations",
        headers=_auth_headers("reviewer-1", "trace-confirm-1"),
        json={
            "run_id": run_id,
            "call_id": call_id,
            "approved": True,
        },
    )
    assert confirm_response.status_code == 200

    repo = get_repo()

    async def read_audit_and_confirmation():
        confirmation_cursor = await repo.conn.execute(
            "SELECT decided_by FROM tool_confirmations WHERE run_id=? AND call_id=?",
            (run_id, call_id),
        )
        confirmation = await confirmation_cursor.fetchone()

        audit_cursor = await repo.conn.execute(
            """
            SELECT user_id, trace_id
            FROM audit_logs
            WHERE run_id=? AND action='tool_confirmation'
            ORDER BY created_at DESC
            LIMIT 1
            """,
            (run_id,),
        )
        audit_row = await audit_cursor.fetchone()
        return confirmation, audit_row

    confirmation, audit_row = asyncio.run(read_audit_and_confirmation())
    assert confirmation["decided_by"] == "reviewer-1"
    assert audit_row["user_id"] == "reviewer-1"
    assert isinstance(audit_row["trace_id"], str) and len(audit_row["trace_id"]) > 0
