from __future__ import annotations

import asyncio
import os
from typing import Any

from app.model_adapters import ModelAdapterError, ModelInvocation, run_model_turn

SUBAGENT_MAX_TASK_CHARS = 2_000
SUBAGENT_MAX_OUTPUT_CHARS = 4_000


def _resolve_subagent_limit() -> int:
    raw_limit = str(os.getenv("WORKER_MAX_SUBAGENTS", "3")).strip()
    try:
        limit = int(raw_limit)
    except ValueError:
        limit = 3
    return min(max(limit, 1), 3)


_SUBAGENT_SEMAPHORE = asyncio.Semaphore(_resolve_subagent_limit())


def subagent_tool_spec() -> dict[str, Any]:
    return {
        "name": "run_subagent",
        "description": "Delegate an independent sub-task to a constrained subagent. Max parallel subagents: 3.",
        "input_schema": {
            "type": "object",
            "properties": {
                "task": {"type": "string"},
                "goal": {"type": "string"},
            },
            "required": ["task"],
        },
    }


async def run_subagent(arguments: dict[str, Any], invocation: ModelInvocation) -> dict[str, Any]:
    task = str(arguments.get("task") or "").strip()
    goal = str(arguments.get("goal") or "").strip()
    if task == "":
        return {"ok": False, "error": "task is required"}

    normalized_task = task[:SUBAGENT_MAX_TASK_CHARS]
    if goal != "":
        normalized_task = f"{normalized_task}\n\nGoal: {goal[:SUBAGENT_MAX_TASK_CHARS]}"

    messages = [
        {
            "role": "system",
            "content": (
                "You are a constrained subagent. Return concise, deterministic analysis only. "
                "Do not request or execute tools."
            ),
        },
        {"role": "user", "content": normalized_task},
    ]

    async with _SUBAGENT_SEMAPHORE:
        try:
            turn = await run_model_turn(invocation, messages, [])
            summary = (turn.text or "").strip()
            if summary == "":
                summary = "Subagent finished without textual output."
            return {
                "ok": True,
                "summary": summary[:SUBAGENT_MAX_OUTPUT_CHARS],
                "vendor": invocation.vendor,
                "model_id": invocation.model_id,
            }
        except ModelAdapterError as exc:
            return {
                "ok": False,
                "error": exc.code,
                "message": str(exc),
                "details": exc.details,
            }
        except Exception as exc:  # pragma: no cover - defensive branch
            return {
                "ok": False,
                "error": "SUBAGENT_RUNTIME_ERROR",
                "message": str(exc),
            }

