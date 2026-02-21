"""OpenAI Chat Completions API provider with function-calling support."""
from __future__ import annotations

import json
import uuid

from openai import AsyncOpenAI

from app.agent.messages import ChatResponse, Message, ToolCall
from app.agent.providers.base import ChatRequest, ProviderAdapter


class OpenAIProvider(ProviderAdapter):
    def __init__(self, api_key: str, base_url: str | None = None) -> None:
        self.client = AsyncOpenAI(api_key=api_key, base_url=base_url)

    async def chat(self, request: ChatRequest) -> ChatResponse:
        messages = _build_messages(request.system_prompt, request.messages)
        tools = _build_tools(request.tools) if request.tools else None

        payload: dict = {
            "model": request.model,
            "messages": messages,
            "max_tokens": request.max_tokens,
            "temperature": request.temperature,
        }
        if tools:
            payload["tools"] = tools

        response = await self.client.chat.completions.create(**payload)
        choice = response.choices[0]
        message = choice.message

        text = message.content or ""
        tool_calls: list[ToolCall] = []
        if message.tool_calls:
            for tc in message.tool_calls:
                try:
                    args = json.loads(tc.function.arguments)
                except (json.JSONDecodeError, TypeError):
                    args = {}
                tool_calls.append(ToolCall(
                    id=tc.id or f"call_{uuid.uuid4().hex[:8]}",
                    name=tc.function.name,
                    input=args,
                ))

        finish = choice.finish_reason
        if finish == "tool_calls":
            stop_reason = "tool_use"
        elif finish == "length":
            stop_reason = "max_tokens"
        else:
            stop_reason = "end_turn"

        usage_data: dict[str, int] = {}
        if response.usage:
            usage_data = {
                "input_tokens": response.usage.prompt_tokens,
                "output_tokens": response.usage.completion_tokens or 0,
            }

        return ChatResponse(
            stop_reason=stop_reason,
            text=text,
            tool_calls=tool_calls,
            usage=usage_data,
        )


def _build_messages(system_prompt: str, messages: list[Message]) -> list[dict]:
    """Convert internal Messages to OpenAI chat format."""
    result: list[dict] = []
    if system_prompt:
        result.append({"role": "system", "content": system_prompt})

    for msg in messages:
        if msg.role == "user":
            result.append({"role": "user", "content": msg.content})
        elif msg.role == "assistant":
            entry: dict = {"role": "assistant"}
            if msg.content:
                entry["content"] = msg.content
            if msg.tool_calls:
                entry["tool_calls"] = [
                    {
                        "id": tc.id,
                        "type": "function",
                        "function": {
                            "name": tc.name,
                            "arguments": json.dumps(tc.input),
                        },
                    }
                    for tc in msg.tool_calls
                ]
            result.append(entry)
        elif msg.role == "tool_result":
            for tr in msg.tool_results:
                result.append({
                    "role": "tool",
                    "tool_call_id": tr.tool_call_id,
                    "content": tr.content,
                })
    return result


def _build_tools(tools: list) -> list[dict]:
    """Convert ToolSchema list to OpenAI function-calling format."""
    return [
        {
            "type": "function",
            "function": {
                "name": t.name,
                "description": t.description,
                "parameters": t.input_schema,
            },
        }
        for t in tools
    ]
