from __future__ import annotations

import pytest

import app.services.execution_service as execution_service_module
from app.services.execution_service import ExecutionService


class _DummyRepo:
    pass


class _DummyReporter:
    def __init__(self, *args, **kwargs) -> None:
        del args, kwargs

    def start(self) -> None:
        return None

    async def stop(self) -> None:
        return None

    async def report(self, event_type: str, payload: dict) -> dict:
        return {"type": event_type, "payload": payload}


class _CaptureInitReporter:
    last_init: dict[str, dict] = {}

    def __init__(self, *args, **kwargs) -> None:
        self.args = args
        self.kwargs = kwargs
        _CaptureInitReporter.last_init = {"args": args, "kwargs": kwargs}

    def start(self) -> None:
        return None

    async def stop(self) -> None:
        return None

    async def report(self, event_type: str, payload: dict) -> dict:
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

    async def fake_execute_agent(self, execution_id: str, context: dict, workspace_path: str, reporter) -> None:
        del self, execution_id, context, reporter
        captured["workspace_path"] = workspace_path

    monkeypatch.setattr(ExecutionService, "_execute_agent", fake_execute_agent)

    service = ExecutionService(repo=_DummyRepo(), agent_mode="vanilla")
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

    async def fake_create(cls, repo_root: str, execution_id: str) -> str:  # pragma: no cover
        del cls, repo_root, execution_id
        raise AssertionError("WorktreeManager.create should not be called when use_worktree is false")

    monkeypatch.setattr(execution_service_module.WorktreeManager, "create", classmethod(fake_create))

    async def fake_execute_agent(self, execution_id: str, context: dict, workspace_path: str, reporter) -> None:
        del self, execution_id, context, reporter
        captured["workspace_path"] = workspace_path

    monkeypatch.setattr(ExecutionService, "_execute_agent", fake_execute_agent)

    service = ExecutionService(repo=_DummyRepo(), agent_mode="vanilla")
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


@pytest.mark.asyncio
async def test_execute_uses_runtime_shared_secret_for_hub_reporter(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(execution_service_module, "HubReporter", _CaptureInitReporter)
    monkeypatch.setenv("GOYAIS_RUNTIME_SHARED_SECRET", "dev-shared")
    monkeypatch.delenv("GOYAIS_HUB_INTERNAL_SECRET", raising=False)
    monkeypatch.setenv("GOYAIS_HUB_BASE_URL", "http://127.0.0.1:8787")

    async def fake_execute_agent(self, execution_id: str, context: dict, workspace_path: str, reporter) -> None:
        del self, execution_id, context, workspace_path, reporter

    monkeypatch.setattr(ExecutionService, "_execute_agent", fake_execute_agent)

    service = ExecutionService(repo=_DummyRepo(), agent_mode="vanilla")
    await service.execute(
        {
            "execution_id": "e3",
            "session_id": "s3",
            "trace_id": "t3",
            "user_message": "hello",
            "repo_root": "/repo/main",
            "use_worktree": False,
        }
    )

    init = _CaptureInitReporter.last_init["kwargs"]
    assert init["hub_internal_secret"] == "dev-shared"


@pytest.mark.asyncio
async def test_execute_defaults_hub_base_url_to_8787(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(execution_service_module, "HubReporter", _CaptureInitReporter)
    monkeypatch.delenv("GOYAIS_HUB_BASE_URL", raising=False)

    async def fake_execute_agent(self, execution_id: str, context: dict, workspace_path: str, reporter) -> None:
        del self, execution_id, context, workspace_path, reporter

    monkeypatch.setattr(ExecutionService, "_execute_agent", fake_execute_agent)

    service = ExecutionService(repo=_DummyRepo(), agent_mode="vanilla")
    await service.execute(
        {
            "execution_id": "e4",
            "session_id": "s4",
            "trace_id": "t4",
            "user_message": "hello",
            "repo_root": "/repo/main",
            "use_worktree": False,
        }
    )

    init = _CaptureInitReporter.last_init["kwargs"]
    assert init["hub_base_url"] == "http://127.0.0.1:8787"
