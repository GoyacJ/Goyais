"""Unit tests for app/agent/loop.py and related components."""
from __future__ import annotations

import json
from unittest.mock import AsyncMock

import pytest

from app.agent.loop import LoopCallbacks, agent_loop
from app.agent.messages import ChatResponse, Message, ToolCall, ToolResult
from app.agent.providers.base import ChatRequest, ProviderAdapter
from app.agent.tool_registry import ToolDef, ToolRegistry


# ─── Helpers ──────────────────────────────────────────────────────────────────

class MockProvider(ProviderAdapter):
    """Provider that returns a preset sequence of ChatResponses."""

    def __init__(self, responses: list[ChatResponse]) -> None:
        self._responses = list(responses)
        self._idx = 0

    async def chat(self, request: ChatRequest) -> ChatResponse:
        resp = self._responses[self._idx]
        self._idx = min(self._idx + 1, len(self._responses) - 1)
        return resp


def _make_tool_use_response(tool_name: str, tool_id: str = "call_abc123", args: dict | None = None) -> ChatResponse:
    return ChatResponse(
        stop_reason="tool_use",
        text="",
        tool_calls=[ToolCall(id=tool_id, name=tool_name, input=args or {})],
    )


def _make_end_response(text: str = "Done.") -> ChatResponse:
    return ChatResponse(stop_reason="end_turn", text=text, tool_calls=[])


# ─── ToolRegistry tests ────────────────────────────────────────────────────────

def test_tool_registry_register_and_get():
    registry = ToolRegistry()
    td = ToolDef(
        name="my_tool",
        description="A test tool",
        input_schema={"type": "object", "properties": {}},
        handler=lambda: "ok",
    )
    registry.register(td)
    assert registry.get("my_tool") is td
    assert registry.get("missing") is None


def test_tool_registry_to_schemas():
    registry = ToolRegistry()
    registry.register(ToolDef(
        name="tool_a",
        description="desc a",
        input_schema={"type": "object"},
        handler=lambda: None,
    ))
    schemas = registry.to_schemas()
    assert len(schemas) == 1
    assert schemas[0].name == "tool_a"
    assert schemas[0].description == "desc a"


@pytest.mark.asyncio
async def test_tool_registry_execute_sync_handler():
    registry = ToolRegistry()
    registry.register(ToolDef(
        name="echo",
        description="echo",
        input_schema={"type": "object", "properties": {"msg": {"type": "string"}}},
        handler=lambda msg: f"echo: {msg}",
    ))
    result = await registry.execute("echo", {"msg": "hello"})
    assert result == "echo: hello"


@pytest.mark.asyncio
async def test_tool_registry_execute_async_handler():
    registry = ToolRegistry()

    async def async_handler(x: int) -> str:
        return f"async: {x}"

    registry.register(ToolDef(
        name="async_tool",
        description="async",
        input_schema={"type": "object"},
        handler=async_handler,
    ))
    result = await registry.execute("async_tool", {"x": 42})
    assert result == "async: 42"


@pytest.mark.asyncio
async def test_tool_registry_execute_unknown_tool():
    registry = ToolRegistry()
    result = await registry.execute("nonexistent", {})
    parsed = json.loads(result)
    assert "error" in parsed


# ─── agent_loop tests ─────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_agent_loop_exits_on_end_turn():
    """Loop exits immediately when LLM returns end_turn with no tools."""
    provider = MockProvider([_make_end_response("Task complete.")])
    registry = ToolRegistry()
    messages = [Message(role="user", content="do something")]

    response = await agent_loop(
        provider=provider,
        model="test-model",
        system_prompt="You are a test agent.",
        messages=messages,
        tools=registry,
    )

    assert response.stop_reason == "end_turn"
    assert response.text == "Task complete."
    # messages: user + assistant
    assert len(messages) == 2
    assert messages[-1].role == "assistant"


@pytest.mark.asyncio
async def test_agent_loop_calls_tool_and_continues():
    """Loop executes one tool call then exits on end_turn."""
    tool_call_response = _make_tool_use_response("read_file", "call_1", {"path": "README.md"})
    end_response = _make_end_response("I read the file.")

    provider = MockProvider([tool_call_response, end_response])

    registry = ToolRegistry()
    registry.register(ToolDef(
        name="read_file",
        description="read",
        input_schema={"type": "object", "properties": {"path": {"type": "string"}}},
        handler=lambda path: f"contents of {path}",
    ))

    messages = [Message(role="user", content="read README.md")]
    tool_calls_seen: list[str] = []
    tool_results_seen: list[str] = []

    cb = LoopCallbacks(
        on_tool_call=lambda tc, conf: tool_calls_seen.append(tc.name),
        on_tool_result=lambda call_id, output, is_err: tool_results_seen.append(output),
    )

    response = await agent_loop(
        provider=provider,
        model="test-model",
        system_prompt="sys",
        messages=messages,
        tools=registry,
        callbacks=cb,
    )

    assert response.stop_reason == "end_turn"
    assert tool_calls_seen == ["read_file"]
    assert len(tool_results_seen) == 1
    assert "README.md" in tool_results_seen[0]
    # messages: user, assistant(tool_use), tool_result, assistant(end)
    assert len(messages) == 4


@pytest.mark.asyncio
async def test_agent_loop_confirmation_denied():
    """When confirmation denied, tool_result is error and loop continues."""
    tool_call_response = _make_tool_use_response("write_file", "call_w", {"path": "x.txt", "content": "hi"})
    end_response = _make_end_response("Okay, skipped.")

    provider = MockProvider([tool_call_response, end_response])

    registry = ToolRegistry()
    executed: list[str] = []

    def write_handler(path: str, content: str) -> str:
        executed.append(path)
        return f"wrote {path}"

    registry.register(ToolDef(
        name="write_file",
        description="write",
        input_schema={"type": "object"},
        handler=write_handler,
        requires_confirmation=True,
    ))

    messages = [Message(role="user", content="write a file")]

    cb = LoopCallbacks(
        on_confirmation_needed=AsyncMock(return_value=False),  # user denies
    )

    response = await agent_loop(
        provider=provider,
        model="test-model",
        system_prompt="sys",
        messages=messages,
        tools=registry,
        callbacks=cb,
    )

    assert response.stop_reason == "end_turn"
    # Tool should NOT have been executed
    assert executed == []
    # tool_result message should contain denied error
    tool_result_msg = next(m for m in messages if m.role == "tool_result")
    assert tool_result_msg.tool_results[0].is_error is True


@pytest.mark.asyncio
async def test_agent_loop_max_iterations():
    """Loop terminates after max_iterations."""
    # Provider always returns tool_use (infinite loop guard)
    always_tool = _make_tool_use_response("echo", "call_x")
    provider = MockProvider([always_tool] * 200)

    registry = ToolRegistry()
    registry.register(ToolDef(
        name="echo",
        description="echo",
        input_schema={"type": "object"},
        handler=lambda: "pong",
    ))

    messages = [Message(role="user", content="loop forever")]

    response = await agent_loop(
        provider=provider,
        model="test-model",
        system_prompt="sys",
        messages=messages,
        tools=registry,
        max_iterations=3,
    )

    assert response.stop_reason == "max_tokens"


# ─── Provider message building tests ──────────────────────────────────────────

def test_anthropic_build_messages_user():
    from app.agent.providers.anthropic_provider import _build_messages
    msgs = [Message(role="user", content="hello")]
    result = _build_messages(msgs)
    assert result == [{"role": "user", "content": "hello"}]


def test_anthropic_build_messages_assistant_with_tool_use():
    from app.agent.providers.anthropic_provider import _build_messages
    msgs = [Message(
        role="assistant",
        content="thinking",
        tool_calls=[ToolCall(id="c1", name="read_file", input={"path": "x.py"})],
    )]
    result = _build_messages(msgs)
    assert len(result) == 1
    assert result[0]["role"] == "assistant"
    content = result[0]["content"]
    assert {"type": "text", "text": "thinking"} in content
    tool_block = next(b for b in content if b.get("type") == "tool_use")
    assert tool_block["name"] == "read_file"


def test_anthropic_build_messages_tool_result():
    from app.agent.providers.anthropic_provider import _build_messages
    msgs = [Message(
        role="tool_result",
        tool_results=[ToolResult(tool_call_id="c1", content="file content", is_error=False)],
    )]
    result = _build_messages(msgs)
    assert result[0]["role"] == "user"
    content = result[0]["content"]
    assert content[0]["type"] == "tool_result"
    assert content[0]["tool_use_id"] == "c1"


def test_openai_build_messages_system_and_user():
    from app.agent.providers.openai_provider import _build_messages
    msgs = [Message(role="user", content="hello")]
    result = _build_messages("You are a bot.", msgs)
    assert result[0] == {"role": "system", "content": "You are a bot."}
    assert result[1] == {"role": "user", "content": "hello"}


def test_openai_build_messages_tool_result():
    from app.agent.providers.openai_provider import _build_messages
    msgs = [Message(
        role="tool_result",
        tool_results=[ToolResult(tool_call_id="c1", content="result text", is_error=False)],
    )]
    result = _build_messages("", msgs)
    # system (empty string skipped) + tool message
    tool_msg = next(m for m in result if m["role"] == "tool")
    assert tool_msg["tool_call_id"] == "c1"
    assert tool_msg["content"] == "result text"
