from __future__ import annotations

import pytest

import app.services.execution_service as execution_service_module
from app.services.execution_service import ExecutionService


class _DummyRepo:
    pass


class _DummyReporter:
    def __init__(self, *args, **kwargs) -> None:  # noqa: D401, ANN002, ANN003
        del args, kwargs

    def start(self) -> None:
        return None

    async def stop(self) -> None:
        return None

    async def report(self, event_type: str, payload: dict) -> dict:  # noqa: ANN001
        return {"type": event_type, "payload": payload}


@pytest.mark.asyncio
async def test_execute_uses_worktree_root_when_enabled(monkeypatch: pytest.MonkeyPatch):
    captured: dict[str, str] = {}

    monkeypatch.setattr(execution_service_module, "HubReporter", _DummyReporter)

    async def fake_create(cls, repo_root: str, execution_id: str) -> str:
        captured["repo_root"] = repo_root
        captured["execution_id"] = execution_id
        return "/tmp/worktree-e1"

    monkeypatch.setattr(execution_service_module.WorktreeManager, "create", classmethod(fake_create))

    async def fake_execute_mock(self, execution_id: str, context: dict, workspace_path: str, reporter) -> None:  # noqa: ANN001
        del self, execution_id, context, reporter
        captured["workspace_path"] = workspace_path

    monkeypatch.setattr(ExecutionService, "_execute_mock", fake_execute_mock)

    service = ExecutionService(repo=_DummyRepo(), agent_mode="mock")
    await service.execute(
        {
            "execution_id": "e1",
            "session_id": "s1",
            "trace_id": "t1",
            "user_message": "update readme",
            "repo_root": "/repo/main",
            "use_worktree": True,
        }
    )

    assert captured["repo_root"] == "/repo/main"
    assert captured["execution_id"] == "e1"
    assert captured["workspace_path"] == "/tmp/worktree-e1"


@pytest.mark.asyncio
async def test_execute_uses_repo_root_when_worktree_disabled(monkeypatch: pytest.MonkeyPatch):
    captured: dict[str, str] = {}

    monkeypatch.setattr(execution_service_module, "HubReporter", _DummyReporter)

    async def fake_create(cls, repo_root: str, execution_id: str) -> str:  # pragma: no cover - should not run
        del cls, repo_root, execution_id
        raise AssertionError("WorktreeManager.create should not be called when use_worktree is false")

    monkeypatch.setattr(execution_service_module.WorktreeManager, "create", classmethod(fake_create))

    async def fake_execute_mock(self, execution_id: str, context: dict, workspace_path: str, reporter) -> None:  # noqa: ANN001
        del self, execution_id, context, reporter
        captured["workspace_path"] = workspace_path

    monkeypatch.setattr(ExecutionService, "_execute_mock", fake_execute_mock)

    service = ExecutionService(repo=_DummyRepo(), agent_mode="mock")
    await service.execute(
        {
            "execution_id": "e2",
            "session_id": "s2",
            "trace_id": "t2",
            "user_message": "update readme",
            "repo_root": "/repo/main",
            "use_worktree": False,
        }
    )

    assert captured["workspace_path"] == "/repo/main"
