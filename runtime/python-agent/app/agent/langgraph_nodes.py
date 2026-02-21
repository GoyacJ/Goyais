"""
langgraph_nodes.py — agent_node and tools_node for the LangGraph ReAct graph.

Confirmation flow: uses config-injected confirmation_handler (asyncio.Future-based),
NOT LangGraph interrupt(), to stay compatible with the existing Hub confirmation API.
"""
from __future__ import annotations

import json
import logging

from langchain_core.messages import AIMessage, ToolMessage

from app.agent.langgraph_compat import chat_response_to_ai_message
from app.agent.langgraph_state import AgentState
from app.agent.messages import ChatResponse, Message, ToolCall
from app.agent.providers.base import ChatRequest

logger = logging.getLogger(__name__)

_DENIED_CONTENT = json.dumps({"error": "Tool call denied by user."})


async def agent_node(state: AgentState, config: dict) -> dict:
    """Call LLM and return the AI message to append to state."""
    provider = config["configurable"]["provider"]
    model: str = config["configurable"]["model"]
    system_prompt: str = config["configurable"]["system_prompt"]
    tool_registry = config["configurable"]["tool_registry"]

    # Convert LangGraph messages → internal Message format for provider
    from app.agent.langgraph_compat import langgraph_to_internal
    internal_msgs: list[Message] = [langgraph_to_internal(m) for m in state["messages"]]

    request = ChatRequest(
        model=model,
        system_prompt=system_prompt,
        messages=internal_msgs,
        tools=tool_registry.to_schemas(),
        max_tokens=4096,
        temperature=0.0,
    )

    response: ChatResponse = await provider.chat(request)
    ai_msg = chat_response_to_ai_message(response)

    logger.debug(
        "langgraph agent_node: stop_reason=%s tool_calls=%d",
        response.stop_reason,
        len(response.tool_calls),
    )

    return {
        "messages": [ai_msg],
        "iteration_count": state.get("iteration_count", 0) + 1,
    }


async def tools_node(state: AgentState, config: dict) -> dict:
    """Execute tool calls from the last AI message, return ToolMessages."""
    tool_registry = config["configurable"]["tool_registry"]
    reporter = config["configurable"].get("reporter")
    confirmation_handler = config["configurable"].get("confirmation_handler")
    execution_id: str = config["configurable"].get("execution_id", "")

    # Get last AI message
    messages = state["messages"]
    last_msg = messages[-1]
    if not isinstance(last_msg, AIMessage) or not last_msg.tool_calls:
        return {"messages": []}

    tool_messages: list[ToolMessage] = []

    for tc_dict in last_msg.tool_calls:
        tc = ToolCall(
            id=tc_dict.get("id", ""),
            name=tc_dict.get("name", ""),
            input=tc_dict.get("args", {}),
        )

        td = tool_registry.get(tc.name)
        needs_confirm = td.requires_confirmation if td else False

        # Emit tool_call event to Hub
        if reporter:
            await _emit_tool_call_event(reporter, execution_id, tc, needs_confirm)

        # Confirmation gate
        approved = True
        if needs_confirm and confirmation_handler:
            approved = await confirmation_handler(tc)

        if not approved:
            logger.info("langgraph tools_node: tool denied: %s", tc.name)
            tool_messages.append(ToolMessage(
                content=_DENIED_CONTENT,
                tool_call_id=tc.id,
                status="error",
            ))
            if reporter:
                await _emit_tool_result_event(reporter, execution_id, tc.id, _DENIED_CONTENT, is_error=True)
            continue

        output = await tool_registry.execute(tc.name, tc.input)
        is_error = _looks_like_error(output)

        tool_messages.append(ToolMessage(
            content=output,
            tool_call_id=tc.id,
            status="error" if is_error else "success",
        ))

        if reporter:
            await _emit_tool_result_event(reporter, execution_id, tc.id, output, is_error=is_error)

    return {"messages": tool_messages}


def should_continue(state: AgentState) -> str:
    """Conditional edge: route to 'tools' if last message has tool calls, else END."""
    messages = state["messages"]
    if not messages:
        return "end"
    last = messages[-1]
    if isinstance(last, AIMessage) and last.tool_calls:
        return "tools"
    return "end"


# ─── helpers ──────────────────────────────────────────────────────────────────

async def _emit_tool_call_event(reporter, execution_id: str, tc: ToolCall, needs_confirm: bool) -> None:
    if needs_confirm:
        await reporter.report("confirmation_request", {
            "call_id": tc.id,
            "tool_name": tc.name,
            "risk_level": "high",
            "parameters_summary": json.dumps(tc.input)[:200],
        })
    await reporter.report("tool_call", {
        "call_id": tc.id,
        "tool_name": tc.name,
        "args": tc.input,
        "requires_confirmation": needs_confirm,
    })


async def _emit_tool_result_event(
    reporter, execution_id: str, call_id: str, output: str, *, is_error: bool
) -> None:
    payload: dict = {"call_id": call_id, "ok": not is_error}
    if is_error:
        payload["error"] = {"code": "E_TOOL_FAILED", "message": output[:500]}
    else:
        payload["output"] = {"result": output[:2000]}
    await reporter.report("tool_result", payload)


def _looks_like_error(output: str) -> bool:
    try:
        parsed = json.loads(output)
        return isinstance(parsed, dict) and "error" in parsed
    except (json.JSONDecodeError, ValueError):
        return False
