from __future__ import annotations

import asyncio
import json
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
WaitConfirmationFn = Callable[[str, int], Awaitable[str]]
IsCancelledFn = Callable[[str], bool]

MAX_TURNS = 6
CONFIRMATION_TIMEOUT_SECONDS = 900
SYSTEM_PROMPT = (
    "You are Goyais worker. Prefer deterministic code edits and concise explanations. "
    "Use available tools only when necessary."
)


async def run_execution_loop(
    execution: dict[str, Any],
    emit_event: EmitEventFn,
    wait_confirmation: WaitConfirmationFn,
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
        if mode_snapshot == "agent" and content_risk in {"high", "critical"}:
            decision = await _resolve_risk_confirmation(
                execution=execution,
                emit_event=emit_event,
                wait_confirmation=wait_confirmation,
                is_cancelled=is_cancelled,
                risk_level=content_risk,
                summary="Operation requires approval.",
                preview=content[:400],
            )
            if decision != "approve":
                return

        invocation = resolve_model_invocation(execution)
        messages = [
            {"role": "system", "content": _build_system_prompt(execution)},
            {"role": "user", "content": content},
        ]
        diffs: list[dict[str, Any]] = []
        final_text = ""

        for turn in range(MAX_TURNS):
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

            if turn_result.text:
                final_text = turn_result.text
                await emit_event(
                    execution,
                    "thinking_delta",
                    {"stage": "assistant_output", "turn": turn + 1, "delta": turn_result.text[:1000]},
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
                if mode_snapshot == "agent" and risk_level in {"high", "critical"}:
                    preview = json.dumps(tool_call.arguments, ensure_ascii=False)[:400]
                    decision = await _resolve_risk_confirmation(
                        execution=execution,
                        emit_event=emit_event,
                        wait_confirmation=wait_confirmation,
                        is_cancelled=is_cancelled,
                        risk_level=risk_level,
                        summary=f"Tool requires approval: {tool_call.name}",
                        preview=preview,
                    )
                    if decision != "approve":
                        return

                await emit_event(
                    execution,
                    "tool_call",
                    {
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

        await emit_event(
            execution,
            "execution_error",
            {
                "reason": "MAX_TURNS_EXCEEDED",
                "message": "Execution exceeded the max model turns.",
                "max_turns": MAX_TURNS,
            },
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


async def _resolve_risk_confirmation(
    execution: dict[str, Any],
    emit_event: EmitEventFn,
    wait_confirmation: WaitConfirmationFn,
    is_cancelled: IsCancelledFn,
    risk_level: str,
    summary: str,
    preview: str,
) -> str:
    execution_id = str(execution.get("execution_id") or "").strip()
    await emit_event(
        execution,
        "confirmation_required",
        {"risk_level": risk_level, "summary": summary, "preview": preview},
    )
    decision = await wait_confirmation(execution_id, timeout_seconds=CONFIRMATION_TIMEOUT_SECONDS)
    if decision == "cancelled" or is_cancelled(execution_id):
        await emit_event(execution, "execution_stopped", {"reason": "stop_requested"})
        return "cancelled"

    await emit_event(execution, "confirmation_resolved", {"decision": decision})
    if decision != "approve":
        await emit_event(
            execution,
            "execution_error",
            {"reason": "USER_DENIED", "message": "Operation denied by user."},
        )
        return decision
    return "approve"


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
