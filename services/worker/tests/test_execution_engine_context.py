from __future__ import annotations

import asyncio
from typing import Any

from app import execution_engine
from app.execution_engine import run_execution_loop
from app.model_adapters import ModelInvocation, ModelTurnResult, ToolCall
from app.tool_runtime import ToolExecutionResult


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


def test_resolve_max_turns_prefers_snapshot_and_clamps(monkeypatch) -> None:
    monkeypatch.setenv("WORKER_MAX_MODEL_TURNS", "60")

    assert execution_engine._resolve_max_turns({"agent_config_snapshot": {"max_model_turns": 2}}) == 4
    assert execution_engine._resolve_max_turns({"agent_config_snapshot": {"max_model_turns": 80}}) == 64
    assert execution_engine._resolve_max_turns({"agent_config_snapshot": {"max_model_turns": 18}}) == 18
    assert execution_engine._resolve_max_turns({}) == 60

    monkeypatch.delenv("WORKER_MAX_MODEL_TURNS", raising=False)
    assert execution_engine._resolve_max_turns({}) == 24


def test_run_execution_loop_turn_limit_emits_truncated_done(monkeypatch) -> None:
    events: list[dict[str, Any]] = []
    turn_counter = 0

    async def fake_run_model_turn(invocation, messages, tools):
        del invocation, messages
        nonlocal turn_counter
        turn_counter += 1
        if len(tools) == 0:
            return ModelTurnResult(text="summary after limit", tool_calls=[], raw_response={})
        return ModelTurnResult(
            text=f"turn-{turn_counter}",
            tool_calls=[ToolCall(id=f"tc_{turn_counter}", name="read_file", arguments={"path": "README.md"})],
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

    def fake_execute_tool_call(tool_call, working_directory):
        del tool_call, working_directory
        return ToolExecutionResult(output={"summary": "ok"})

    async def emit_event(execution, event_type, payload):
        del execution
        events.append({"type": event_type, "payload": payload})

    def is_cancelled(execution_id: str) -> bool:
        del execution_id
        return False

    monkeypatch.setattr(execution_engine, "run_model_turn", fake_run_model_turn)
    monkeypatch.setattr(execution_engine, "resolve_model_invocation", fake_resolve_model_invocation)
    monkeypatch.setattr(execution_engine, "execute_tool_call", fake_execute_tool_call)

    execution = {
        "execution_id": "exec_turn_limit",
        "mode_snapshot": "agent",
        "content": "初始化项目并生成大量文件",
        "model_id": "llama3:8b",
        "agent_config_snapshot": {
            "max_model_turns": 4,
        },
    }

    asyncio.run(run_execution_loop(execution, emit_event, is_cancelled))

    done_events = [event for event in events if event["type"] == "execution_done"]
    assert len(done_events) == 1
    payload = done_events[0]["payload"]
    assert payload["truncated"] is True
    assert payload["reason"] == "MAX_TURNS_REACHED"
    assert payload["max_turns"] == 4
    assert "summary after limit" in payload["content"]
    assert all(event["type"] != "execution_error" for event in events)
    assert turn_counter == 5


def test_tool_events_include_call_id_for_tool_and_subagent(monkeypatch) -> None:
    events: list[dict[str, Any]] = []
    turn_counter = 0

    async def fake_run_model_turn(invocation, messages, tools):
        del invocation, messages, tools
        nonlocal turn_counter
        turn_counter += 1
        if turn_counter == 1:
            return ModelTurnResult(
                text="tool call turn",
                tool_calls=[
                    ToolCall(id="tc_read_1", name="read_file", arguments={"path": "README.md"}),
                    ToolCall(id="tc_sub_1", name="run_subagent", arguments={"task": "list files"}),
                ],
                raw_response={},
            )
        return ModelTurnResult(text="done", tool_calls=[], raw_response={})

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

    def fake_execute_tool_call(tool_call, working_directory):
        del working_directory
        return ToolExecutionResult(output={"name": tool_call.name, "ok": True})

    async def fake_run_subagent(arguments, invocation):
        del arguments, invocation
        return {"ok": True, "output": "subagent done"}

    async def emit_event(execution, event_type, payload):
        del execution
        events.append({"type": event_type, "payload": payload})

    def is_cancelled(execution_id: str) -> bool:
        del execution_id
        return False

    monkeypatch.setattr(execution_engine, "run_model_turn", fake_run_model_turn)
    monkeypatch.setattr(execution_engine, "resolve_model_invocation", fake_resolve_model_invocation)
    monkeypatch.setattr(execution_engine, "execute_tool_call", fake_execute_tool_call)
    monkeypatch.setattr(execution_engine, "run_subagent", fake_run_subagent)

    execution = {
        "execution_id": "exec_tool_call_id",
        "mode_snapshot": "agent",
        "content": "读取项目文件并启动子代理",
        "model_id": "llama3:8b",
    }

    asyncio.run(run_execution_loop(execution, emit_event, is_cancelled))

    tool_calls = [event for event in events if event["type"] == "tool_call"]
    tool_results = [event for event in events if event["type"] == "tool_result"]

    assert len(tool_calls) == 2
    assert len(tool_results) == 2
    assert {item["payload"].get("call_id") for item in tool_calls} == {"tc_read_1", "tc_sub_1"}
    assert {item["payload"].get("call_id") for item in tool_results} == {"tc_read_1", "tc_sub_1"}


def test_execution_done_includes_accumulated_usage(monkeypatch) -> None:
    events: list[dict[str, Any]] = []

    async def fake_run_model_turn(invocation, messages, tools):
        del invocation, messages, tools
        return ModelTurnResult(
            text="done with usage",
            tool_calls=[],
            raw_response={},
            usage={"input_tokens": 12, "output_tokens": 8, "total_tokens": 20},
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
        "execution_id": "exec_usage_accumulated",
        "mode_snapshot": "agent",
        "content": "读取当前项目",
        "model_id": "llama3:8b",
    }

    asyncio.run(run_execution_loop(execution, emit_event, is_cancelled))

    done_events = [event for event in events if event["type"] == "execution_done"]
    assert len(done_events) == 1
    assert done_events[0]["payload"]["usage"] == {
        "input_tokens": 12,
        "output_tokens": 8,
        "total_tokens": 20,
    }
