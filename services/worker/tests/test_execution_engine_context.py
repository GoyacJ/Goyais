from __future__ import annotations

import asyncio
from typing import Any

from app import execution_engine
from app.execution_engine import run_execution_loop
from app.model_adapters import ModelInvocation, ModelTurnResult


def test_run_execution_loop_injects_project_context_into_system_prompt(monkeypatch) -> None:
    captured: dict[str, str] = {}
    events: list[dict[str, Any]] = []

    async def fake_run_model_turn(invocation, messages, tools):
        del invocation, tools
        captured["system_prompt"] = str(messages[0].get("content") or "")
        return ModelTurnResult(text="ok", tool_calls=[], raw_response={})

    def fake_resolve_model_invocation(execution):
        del execution
        return ModelInvocation(
            vendor="local",
            model_id="llama3:8b",
            base_url="http://127.0.0.1:11434/v1",
            api_key="",
            timeout_ms=30_000,
            params={},
        )

    async def emit_event(execution, event_type, payload):
        del execution
        events.append({"type": event_type, "payload": payload})

    async def wait_confirmation(execution_id: str, timeout_seconds: int) -> str:
        del execution_id, timeout_seconds
        return "approve"

    def is_cancelled(execution_id: str) -> bool:
        del execution_id
        return False

    monkeypatch.setattr(execution_engine, "run_model_turn", fake_run_model_turn)
    monkeypatch.setattr(execution_engine, "resolve_model_invocation", fake_resolve_model_invocation)

    execution = {
        "execution_id": "exec_context_1",
        "mode_snapshot": "agent",
        "content": "查看当前项目",
        "model_id": "llama3:8b",
        "project_name": "Test Project",
        "project_path": "/tmp/test-project",
    }

    asyncio.run(run_execution_loop(execution, emit_event, wait_confirmation, is_cancelled))

    system_prompt = captured.get("system_prompt", "")
    assert "Current project name: Test Project." in system_prompt
    assert "Current project path: /tmp/test-project." in system_prompt
    assert any(event["type"] == "execution_done" for event in events)
