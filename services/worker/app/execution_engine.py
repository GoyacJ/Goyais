from __future__ import annotations

import json
from typing import Any, Awaitable, Callable

from app.model_adapters import ModelAdapterError, resolve_model_invocation, run_model_turn
from app.tool_runtime import (
    classify_content_risk,
    classify_tool_risk,
    default_tools,
    execute_tool_call,
)

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
            {"role": "system", "content": SYSTEM_PROMPT},
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

                tool_result = execute_tool_call(tool_call)
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
