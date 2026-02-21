"""Anthropic Messages API provider with native tool_use support."""
from __future__ import annotations

from anthropic import AsyncAnthropic

from app.agent.messages import ChatResponse, Message, ToolCall
from app.agent.providers.base import ChatRequest, ProviderAdapter


class AnthropicProvider(ProviderAdapter):
    def __init__(self, api_key: str) -> None:
        self.client = AsyncAnthropic(api_key=api_key)

    async def chat(self, request: ChatRequest) -> ChatResponse:
        messages = _build_messages(request.messages)
        tools = _build_tools(request.tools) if request.tools else []

        payload: dict = {
            "model": request.model,
            "max_tokens": request.max_tokens,
            "messages": messages,
        }
        if request.system_prompt:
            payload["system"] = request.system_prompt
        if tools:
            payload["tools"] = tools
        if request.temperature is not None:
            payload["temperature"] = request.temperature

        response = await self.client.messages.create(**payload)

        text_parts: list[str] = []
        tool_calls: list[ToolCall] = []
        for block in response.content:
            if block.type == "text":
                text_parts.append(block.text)
            elif block.type == "tool_use":
                tool_calls.append(ToolCall(
                    id=block.id,
                    name=block.name,
                    input=block.input if isinstance(block.input, dict) else {},
                ))

        stop_reason = "tool_use" if response.stop_reason == "tool_use" else "end_turn"

        return ChatResponse(
            stop_reason=stop_reason,
            text="\n".join(text_parts),
            tool_calls=tool_calls,
            usage={
                "input_tokens": response.usage.input_tokens,
                "output_tokens": response.usage.output_tokens,
            },
        )


def _build_messages(messages: list[Message]) -> list[dict]:
    """Convert internal Messages to Anthropic API message format."""
    result: list[dict] = []
    for msg in messages:
        if msg.role == "user":
            result.append({"role": "user", "content": msg.content})
        elif msg.role == "assistant":
            content: list[dict] = []
            if msg.content:
                content.append({"type": "text", "text": msg.content})
            for tc in msg.tool_calls:
                content.append({
                    "type": "tool_use",
                    "id": tc.id,
                    "name": tc.name,
                    "input": tc.input,
                })
            result.append({"role": "assistant", "content": content})
        elif msg.role == "tool_result":
            content_blocks: list[dict] = []
            for tr in msg.tool_results:
                content_blocks.append({
                    "type": "tool_result",
                    "tool_use_id": tr.tool_call_id,
                    "content": tr.content,
                    "is_error": tr.is_error,
                })
            result.append({"role": "user", "content": content_blocks})
    return result


def _build_tools(tools: list) -> list[dict]:
    """Convert ToolSchema list to Anthropic tools format."""
    return [
        {
            "name": t.name,
            "description": t.description,
            "input_schema": t.input_schema,
        }
        for t in tools
    ]
