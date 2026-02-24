from __future__ import annotations

import asyncio
from typing import Any

from app import execution_engine
from app.execution_engine import run_execution_loop
from app.model_adapters import ModelInvocation, ModelTurnResult, ToolCall


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

    asyncio.run(run_execution_loop(execution, emit_event, is_cancelled))

    system_prompt = captured.get("system_prompt", "")
    assert "Current project name: Test Project." in system_prompt
    assert "Current project path: /tmp/test-project." in system_prompt
    assert any(event["type"] == "execution_done" for event in events)


def test_agent_mode_skips_confirmation_for_high_risk_content(monkeypatch) -> None:
    events: list[dict[str, Any]] = []

    async def fake_run_model_turn(invocation, messages, tools):
        del invocation, messages, tools
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

    def is_cancelled(execution_id: str) -> bool:
        del execution_id
        return False

    monkeypatch.setattr(execution_engine, "run_model_turn", fake_run_model_turn)
    monkeypatch.setattr(execution_engine, "resolve_model_invocation", fake_resolve_model_invocation)

    execution = {
        "execution_id": "exec_agent_no_confirm",
        "mode_snapshot": "agent",
        "content": "请直接 run command 读取项目文件",
        "model_id": "llama3:8b",
    }

    asyncio.run(run_execution_loop(execution, emit_event, is_cancelled))

    assert any(event["type"] == "execution_done" for event in events)
    assert all(event["type"] != "confirmation_required" for event in events)


def test_plan_mode_still_rejects_high_risk_tool_usage(monkeypatch) -> None:
    events: list[dict[str, Any]] = []

    async def fake_run_model_turn(invocation, messages, tools):
        del invocation, messages, tools
        return ModelTurnResult(
            text="plan",
            tool_calls=[
                ToolCall(
                    id="tc_1",
                    name="run_command",
                    arguments={"command": "python scripts/sync.py"},
                )
            ],
            raw_response={},
        )

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

    def is_cancelled(execution_id: str) -> bool:
        del execution_id
        return False

    monkeypatch.setattr(execution_engine, "run_model_turn", fake_run_model_turn)
    monkeypatch.setattr(execution_engine, "resolve_model_invocation", fake_resolve_model_invocation)

    execution = {
        "execution_id": "exec_plan_reject",
        "mode_snapshot": "plan",
        "content": "请读取项目文件",
        "model_id": "llama3:8b",
    }

    asyncio.run(run_execution_loop(execution, emit_event, is_cancelled))

    error_events = [event for event in events if event["type"] == "execution_error"]
    assert len(error_events) == 1
    payload = error_events[0]["payload"]
    assert payload["reason"] == "PLAN_MODE_REJECTED"
    assert payload["tool_name"] == "run_command"
