"""
loop.py — Vanilla agentic loop

Immutable while-loop pattern: all functionality injected via tools and callbacks.
LLM calls tools until stop_reason != "tool_use", then returns.
"""
from __future__ import annotations

import json
import logging
import inspect
from dataclasses import dataclass, field
from typing import Awaitable, Callable

from app.agent.context_manager import truncate_old_tool_results
from app.agent.messages import ChatResponse, Message, ToolCall, ToolResult
from app.agent.providers.base import ChatRequest, ProviderAdapter
from app.agent.tool_registry import ToolRegistry

logger = logging.getLogger(__name__)

DENIED_CONTENT = json.dumps({"error": "Tool call denied by user."})


@dataclass(slots=True)
class LoopCallbacks:
    on_tool_call: Callable[[ToolCall, bool], Awaitable[None] | None] | None = None
    on_tool_result: Callable[[str, str, bool], Awaitable[None] | None] | None = None
    on_text_delta: Callable[[str], Awaitable[None] | None] | None = None
    on_confirmation_needed: Callable[[ToolCall], Awaitable[bool] | bool] | None = None


async def agent_loop(
    *,
    provider: ProviderAdapter,
    model: str,
    system_prompt: str,
    messages: list[Message],
    tools: ToolRegistry,
    callbacks: LoopCallbacks | None = None,
    max_iterations: int = 100,
) -> ChatResponse:
    """
    Run the agentic loop until the LLM stops requesting tool calls.

    Args:
        provider: LLM provider adapter.
        model: Model identifier string.
        system_prompt: System prompt for the LLM.
        messages: Mutable message list (modified in place).
        tools: Tool registry for dispatch.
        callbacks: Optional event hooks for observability.
        max_iterations: Safety cap on loop iterations.

    Returns:
        The final ChatResponse (stop_reason != "tool_use").
    """
    cb = callbacks or LoopCallbacks()
    tool_schemas = tools.to_schemas()

    for iteration in range(max_iterations):
        request = ChatRequest(
            model=model,
            system_prompt=system_prompt,
            messages=list(messages),
            tools=tool_schemas,
            max_tokens=4096,
            temperature=0.0,
        )

        logger.debug("agent_loop iteration=%d messages=%d", iteration, len(messages))
        response = await provider.chat(request)

        # Append the assistant turn to history
        messages.append(Message(
            role="assistant",
            content=response.text,
            tool_calls=response.tool_calls,
        ))

        # Fire text callback
        if response.text and cb.on_text_delta:
            await _call_maybe_async(cb.on_text_delta, response.text)

        # Exit condition: no more tool calls
        if response.stop_reason != "tool_use" or not response.tool_calls:
            logger.debug("agent_loop done after %d iterations, reason=%s", iteration + 1, response.stop_reason)
            return response

        # Execute each tool call
        tool_results: list[ToolResult] = []

        for tc in response.tool_calls:
            td = tools.get(tc.name)
            needs_confirm = td.requires_confirmation if td else False

            if cb.on_tool_call:
                await _call_maybe_async(cb.on_tool_call, tc, needs_confirm)

            # Confirmation gate
            approved = True
            if needs_confirm and cb.on_confirmation_needed:
                approved = bool(await _call_maybe_async(cb.on_confirmation_needed, tc))

            if not approved:
                logger.info("tool_call denied: %s", tc.name)
                tool_results.append(ToolResult(
                    tool_call_id=tc.id,
                    content=DENIED_CONTENT,
                    is_error=True,
                ))
                if cb.on_tool_result:
                    await _call_maybe_async(cb.on_tool_result, tc.id, DENIED_CONTENT, True)
                continue

            # Execute
            output = await tools.execute(tc.name, tc.input)
            is_error = _looks_like_error(output)

            tool_results.append(ToolResult(
                tool_call_id=tc.id,
                content=output,
                is_error=is_error,
            ))

            if cb.on_tool_result:
                await _call_maybe_async(cb.on_tool_result, tc.id, output, is_error)

        # Append tool results as a single message
        messages.append(Message(
            role="tool_result",
            tool_results=tool_results,
        ))

        # Micro-compress: truncate old tool results to save context
        truncate_old_tool_results(messages)

    # Safety: max iterations reached
    logger.warning("agent_loop hit max_iterations=%d", max_iterations)
    return ChatResponse(
        stop_reason="max_tokens",
        text="",
        tool_calls=[],
    )


def _looks_like_error(output: str) -> bool:
    """Heuristic: treat output as error if it's a JSON object with an 'error' key."""
    try:
        parsed = json.loads(output)
        return isinstance(parsed, dict) and "error" in parsed
    except (json.JSONDecodeError, ValueError):
        return False


async def _call_maybe_async(fn: Callable, *args):
    result = fn(*args)
    if inspect.isawaitable(result):
        return await result
    return result
