import asyncio

import pytest

from app.model_adapters import ModelInvocation, ModelTurnResult
from app.safety.risk_gate import classify_tool_risk
from app.tool_runtime import default_tools
from app.tools import subagent_tools


def test_default_tools_include_run_subagent() -> None:
    names = [item["name"] for item in default_tools()]
    assert "run_subagent" in names


def test_subagent_tool_risk_is_low() -> None:
    assert classify_tool_risk("run_subagent", {"task": "analyze files"}) == "low"


def test_run_subagent_requires_task() -> None:
    invocation = ModelInvocation(
        vendor="openai",
        model_id="gpt-4.1",
        base_url="https://api.openai.com/v1",
        api_key="sk-test",
        timeout_ms=10_000,
        params={},
    )
    result = asyncio.run(subagent_tools.run_subagent({}, invocation))
    assert result["ok"] is False
    assert "task is required" in str(result["error"])


def test_run_subagent_success(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_run_model_turn(invocation, messages, tools):
        return ModelTurnResult(text="subagent summary", tool_calls=[], raw_response={})

    monkeypatch.setattr(subagent_tools, "run_model_turn", fake_run_model_turn)
    invocation = ModelInvocation(
        vendor="openai",
        model_id="gpt-4.1",
        base_url="https://api.openai.com/v1",
        api_key="sk-test",
        timeout_ms=10_000,
        params={},
    )
    result = asyncio.run(subagent_tools.run_subagent({"task": "analyze README"}, invocation))
    assert result["ok"] is True
    assert result["summary"] == "subagent summary"
    assert result["model_id"] == "gpt-4.1"

