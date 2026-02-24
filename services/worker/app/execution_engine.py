from __future__ import annotations

import asyncio
import json
import os
from typing import Any, Awaitable, Callable

from app.model_adapters import ModelAdapterError, resolve_model_invocation, run_model_turn
from app.tool_runtime import (
    classify_content_risk,
    classify_tool_risk,
    default_tools,
    execute_tool_call,
)
from app.tools.subagent_tools import run_subagent

EmitEventFn = Callable[[dict[str, Any], str, dict[str, Any]], Awaitable[None]]
IsCancelledFn = Callable[[str], bool]

DEFAULT_MAX_TURNS = 24
MIN_MAX_TURNS = 4
MAX_MAX_TURNS = 64
SYSTEM_PROMPT = (
    "You are Goyais worker. Prefer deterministic code edits and concise explanations. "
    "Use available tools only when necessary."
)


async def run_execution_loop(
    execution: dict[str, Any],
    emit_event: EmitEventFn,
    is_cancelled: IsCancelledFn,
) -> None:
    execution_id = str(execution.get("execution_id") or "").strip()
    if execution_id == "":
        return

    mode_snapshot = str(execution.get("mode_snapshot") or execution.get("mode") or "agent").strip().lower()
    content = str(execution.get("content") or "").strip()
    working_directory = str(execution.get("working_directory") or ".").strip()

    try:
        await emit_event(
            execution,
            "execution_started",
            {"mode": mode_snapshot, "model_id": str(execution.get("model_id") or "")},
        )
        if is_cancelled(execution_id):
            await emit_event(execution, "execution_stopped", {"reason": "stop_requested"})
            return

        content_risk = classify_content_risk(content)
        if mode_snapshot == "plan" and content_risk in {"high", "critical"}:
            await emit_event(
                execution,
                "execution_error",
                {
                    "reason": "PLAN_MODE_REJECTED",
                    "message": "Plan mode rejects high/critical operations.",
                    "risk_level": content_risk,
                },
            )
            return

        invocation = resolve_model_invocation(execution)
        messages = [
            {"role": "system", "content": _build_system_prompt(execution)},
            {"role": "user", "content": content},
        ]
        diffs: list[dict[str, Any]] = []
        final_text = ""
        usage_totals = {"input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
        max_turns = _resolve_max_turns(execution)

        for turn in range(max_turns):
            if is_cancelled(execution_id):
                await emit_event(execution, "execution_stopped", {"reason": "stop_requested"})
                return

            await emit_event(
                execution,
                "thinking_delta",
                {
                    "stage": "model_call",
                    "turn": turn + 1,
                    "vendor": invocation.vendor,
                    "model_id": invocation.model_id,
                },
            )
            turn_result = await run_model_turn(invocation, messages, default_tools())
            usage_totals = _accumulate_usage(usage_totals, turn_result.usage)

            if turn_result.text:
                final_text = turn_result.text
                await emit_event(
                    execution,
                    "thinking_delta",
                    {
                        "stage": "assistant_output",
                        "turn": turn + 1,
                        "delta": turn_result.text[:1000],
                        "usage": usage_totals,
                    },
                )

            if not turn_result.tool_calls:
                if diffs:
                    await emit_event(
                        execution,
                        "diff_generated",
                        {"files": len(diffs), "diff": diffs},
                    )
                await emit_event(
                    execution,
                    "execution_done",
                    {
                        "content": final_text or f"Execution {execution_id} completed.",
                        "result": "ok",
                        "turns": turn + 1,
                        "max_turns": max_turns,
                        "usage": usage_totals,
                    },
                )
                return

            messages.append(
                {
                    "role": "assistant",
                    "content": turn_result.text,
                    "tool_calls": [
                        {"id": call.id, "name": call.name, "arguments": call.arguments}
                        for call in turn_result.tool_calls
                    ],
                }
            )

            subagent_calls: list[tuple[str, str, asyncio.Task[dict[str, Any]]]] = []
            for tool_call in turn_result.tool_calls:
                if is_cancelled(execution_id):
                    await emit_event(execution, "execution_stopped", {"reason": "stop_requested"})
                    return

                risk_level = classify_tool_risk(tool_call.name, tool_call.arguments)
                if mode_snapshot == "plan" and risk_level in {"high", "critical"}:
                    await emit_event(
                        execution,
                        "execution_error",
                        {
                            "reason": "PLAN_MODE_REJECTED",
                            "message": "Plan mode rejects high/critical tool usage.",
                            "tool_name": tool_call.name,
                            "risk_level": risk_level,
                        },
                    )
                    return

                await emit_event(
                    execution,
                    "tool_call",
                    {
                        "call_id": tool_call.id,
                        "name": tool_call.name,
                        "input": tool_call.arguments,
                        "risk_level": risk_level,
                    },
                )

                if tool_call.name.strip().lower() == "run_subagent":
                    task = asyncio.create_task(run_subagent(tool_call.arguments, invocation))
                    subagent_calls.append((tool_call.id, tool_call.name, task))
                    continue

                tool_result = execute_tool_call(tool_call, working_directory)
                await emit_event(
                    execution,
                    "tool_result",
                    {
                        "call_id": tool_call.id,
                        "name": tool_call.name,
                        "ok": True,
                        "output": tool_result.output,
                    },
                )
                if tool_result.diff is not None:
                    diffs.append(tool_result.diff)

                messages.append(
                    {
                        "role": "tool",
                        "tool_call_id": tool_call.id,
                        "name": tool_call.name,
                        "content": json.dumps(tool_result.output, ensure_ascii=False),
                    }
                )

            if subagent_calls:
                subagent_results = await asyncio.gather(
                    *(item[2] for item in subagent_calls), return_exceptions=True
                )
                for (tool_call_id, tool_name, _), subagent_result in zip(
                    subagent_calls, subagent_results, strict=False
                ):
                    if isinstance(subagent_result, Exception):
                        output: dict[str, Any] = {
                            "ok": False,
                            "error": "SUBAGENT_RUNTIME_ERROR",
                            "message": str(subagent_result),
                        }
                    else:
                        output = subagent_result

                    await emit_event(
                        execution,
                        "tool_result",
                        {
                            "call_id": tool_call_id,
                            "name": tool_name,
                            "ok": bool(output.get("ok", False)),
                            "output": output,
                        },
                    )
                    messages.append(
                        {
                            "role": "tool",
                            "tool_call_id": tool_call_id,
                            "name": tool_name,
                            "content": json.dumps(output, ensure_ascii=False),
                        }
                    )

        await _emit_turn_limit_summary(
            execution=execution,
            invocation=invocation,
            messages=messages,
            emit_event=emit_event,
            execution_id=execution_id,
            max_turns=max_turns,
            final_text=final_text,
            usage_totals=usage_totals,
        )
    except ModelAdapterError as exc:
        await emit_event(
            execution,
            "execution_error",
            {"reason": exc.code, "message": str(exc), "details": exc.details},
        )
    except Exception as exc:  # pragma: no cover - protection path
        await emit_event(
            execution,
            "execution_error",
            {"reason": "WORKER_RUNTIME_ERROR", "message": str(exc)},
        )


def _build_system_prompt(execution: dict[str, Any]) -> str:
    project_name = str(execution.get("project_name") or "").strip()
    project_path = str(execution.get("project_path") or execution.get("working_directory") or "").strip()

    context_parts: list[str] = []
    if project_name != "":
        context_parts.append(f"Current project name: {project_name}.")
    if project_path != "":
        context_parts.append(f"Current project path: {project_path}.")
    if len(context_parts) == 0:
        return SYSTEM_PROMPT
    return f"{SYSTEM_PROMPT} {' '.join(context_parts)} Use this context when answering project-scoped questions."


def _resolve_max_turns(execution: dict[str, Any]) -> int:
    snapshot = execution.get("agent_config_snapshot")
    candidate_values: list[Any] = []
    if isinstance(snapshot, dict):
        candidate_values.append(snapshot.get("max_model_turns"))
        nested_execution = snapshot.get("execution")
        if isinstance(nested_execution, dict):
            candidate_values.append(nested_execution.get("max_model_turns"))
    candidate_values.append(os.getenv("WORKER_MAX_MODEL_TURNS"))
    candidate_values.append(DEFAULT_MAX_TURNS)

    for candidate in candidate_values:
        try:
            value = int(candidate)
        except (TypeError, ValueError):
            continue
        if value < MIN_MAX_TURNS:
            return MIN_MAX_TURNS
        if value > MAX_MAX_TURNS:
            return MAX_MAX_TURNS
        return value
    return DEFAULT_MAX_TURNS


async def _emit_turn_limit_summary(
    execution: dict[str, Any],
    invocation: Any,
    messages: list[dict[str, Any]],
    emit_event: EmitEventFn,
    execution_id: str,
    max_turns: int,
    final_text: str,
    usage_totals: dict[str, int],
) -> None:
    await emit_event(
        execution,
        "thinking_delta",
        {
            "stage": "turn_limit_reached",
            "max_turns": max_turns,
        },
    )
    summary_messages = list(messages)
    summary_messages.append(
        {
            "role": "user",
            "content": (
                "Tool-call turn limit reached. Do not call tools. "
                "Provide a concise final answer based on the current context."
            ),
        }
    )
    try:
        summary_result = await run_model_turn(invocation, summary_messages, [])
    except Exception as exc:
        await emit_event(
            execution,
            "execution_error",
            {
                "reason": "MAX_TURNS_EXCEEDED",
                "message": "Execution exceeded the max model turns.",
                "max_turns": max_turns,
                "details": {"summary_error": str(exc)},
            },
        )
        return
    usage_totals = _accumulate_usage(usage_totals, summary_result.usage)

    summary_text = str(summary_result.text or "").strip()
    if summary_text != "":
        await emit_event(
            execution,
            "thinking_delta",
            {
                "stage": "assistant_output",
                "turn": max_turns + 1,
                "delta": summary_text[:1000],
                "usage": usage_totals,
            },
        )

    await emit_event(
        execution,
        "execution_done",
        {
            "content": summary_text or final_text or f"Execution {execution_id} completed.",
            "result": "ok",
            "turns": max_turns,
            "truncated": True,
            "reason": "MAX_TURNS_REACHED",
            "max_turns": max_turns,
            "usage": usage_totals,
        },
    )


def _accumulate_usage(current: dict[str, int], incoming: dict[str, Any] | None) -> dict[str, int]:
    incoming_usage = incoming if isinstance(incoming, dict) else {}
    input_tokens = _to_non_negative_int(incoming_usage.get("input_tokens"))
    output_tokens = _to_non_negative_int(incoming_usage.get("output_tokens"))
    total_tokens = _to_non_negative_int(incoming_usage.get("total_tokens"))
    if total_tokens == 0:
        total_tokens = input_tokens + output_tokens
    return {
        "input_tokens": current["input_tokens"] + input_tokens,
        "output_tokens": current["output_tokens"] + output_tokens,
        "total_tokens": current["total_tokens"] + total_tokens,
    }


def _to_non_negative_int(value: Any) -> int:
    try:
        parsed = int(value or 0)
    except (TypeError, ValueError):
        return 0
    return parsed if parsed >= 0 else 0
