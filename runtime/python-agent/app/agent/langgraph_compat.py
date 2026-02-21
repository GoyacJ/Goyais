"""
langgraph_compat.py — bidirectional conversion between internal Message types
and langchain_core.messages (HumanMessage, AIMessage, ToolMessage).
"""
from __future__ import annotations

import json

from langchain_core.messages import AIMessage, BaseMessage, HumanMessage, ToolMessage

from app.agent.messages import ChatResponse, Message, ToolCall, ToolResult


def internal_to_langgraph(msg: Message) -> BaseMessage:
    if msg.role == "user":
        return HumanMessage(content=msg.content)

    if msg.role == "assistant":
        if msg.tool_calls:
            tool_calls_lc = [
                {
                    "id": tc.id,
                    "name": tc.name,
                    "args": tc.input,
                    "type": "tool_call",
                }
                for tc in msg.tool_calls
            ]
            return AIMessage(content=msg.content or "", tool_calls=tool_calls_lc)
        return AIMessage(content=msg.content)

    if msg.role == "tool_result":
        # Convert first tool_result; multiple results need separate messages
        if msg.tool_results:
            tr = msg.tool_results[0]
            return ToolMessage(
                content=tr.content,
                tool_call_id=tr.tool_call_id,
                status="error" if tr.is_error else "success",
            )

    return HumanMessage(content=msg.content or "")


def langgraph_to_internal(msg: BaseMessage) -> Message:
    if isinstance(msg, HumanMessage):
        content = msg.content if isinstance(msg.content, str) else json.dumps(msg.content)
        return Message(role="user", content=content)

    if isinstance(msg, AIMessage):
        text = msg.content if isinstance(msg.content, str) else ""
        tool_calls: list[ToolCall] = []
        for tc in getattr(msg, "tool_calls", []):
            tool_calls.append(ToolCall(
                id=tc.get("id", ""),
                name=tc.get("name", ""),
                input=tc.get("args", {}),
            ))
        return Message(role="assistant", content=text, tool_calls=tool_calls)

    if isinstance(msg, ToolMessage):
        content = msg.content if isinstance(msg.content, str) else json.dumps(msg.content)
        is_error = getattr(msg, "status", "success") == "error"
        return Message(
            role="tool_result",
            tool_results=[ToolResult(
                tool_call_id=msg.tool_call_id,
                content=content,
                is_error=is_error,
            )],
        )

    content = msg.content if isinstance(msg.content, str) else json.dumps(msg.content)
    return Message(role="user", content=content)


def chat_response_to_ai_message(resp: ChatResponse) -> AIMessage:
    tool_calls_lc = [
        {
            "id": tc.id,
            "name": tc.name,
            "args": tc.input,
            "type": "tool_call",
        }
        for tc in resp.tool_calls
    ]
    return AIMessage(content=resp.text, tool_calls=tool_calls_lc)
