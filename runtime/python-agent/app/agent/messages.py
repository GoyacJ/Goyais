"""Unified message types shared by both vanilla and LangGraph backends."""
from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(slots=True)
class ToolCall:
    id: str
    name: str
    input: dict


@dataclass(slots=True)
class ToolResult:
    tool_call_id: str
    content: str
    is_error: bool = False


@dataclass(slots=True)
class Message:
    role: str  # "user" | "assistant" | "tool_result"
    content: str = ""
    tool_calls: list[ToolCall] = field(default_factory=list)
    tool_results: list[ToolResult] = field(default_factory=list)


@dataclass(slots=True)
class ChatResponse:
    stop_reason: str  # "end_turn" | "tool_use" | "max_tokens"
    text: str
    tool_calls: list[ToolCall]
    usage: dict[str, int] = field(default_factory=dict)
