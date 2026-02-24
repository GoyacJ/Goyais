from __future__ import annotations

import asyncio
import json
import os
from typing import Any
import urllib.error
import urllib.request

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

from app.execution_engine import run_execution_loop
from app.errors import standard_error_response
from app.internal_api import (
    DEFAULT_INTERNAL_TOKEN,
    INTERNAL_TOKEN_HEADER,
    decode_json,
    is_blank,
    now_iso,
    parse_int,
    parse_queue_index,
    require_internal_token,
    safe_int,
)
from app.trace import TRACE_HEADER

router = APIRouter()

_executions: dict[str, dict[str, Any]] = {}
_events: list[dict[str, Any]] = []
_execution_tasks: dict[str, asyncio.Task[None]] = {}
_execution_confirm_queues: dict[str, asyncio.Queue[str]] = {}
_execution_cancel_events: dict[str, asyncio.Event] = {}
_execution_sequence: dict[str, int] = {}


@router.get("/health")
def health() -> dict[str, object]:
    return {"ok": True, "version": "0.4.0"}


@router.post("/internal/executions")
async def internal_executions(request: Request):
    auth_err = require_internal_token(request)
    if auth_err is not None:
        return auth_err

    payload, err = await decode_json(request)
    if err is not None:
        return err

    required_fields = [
        "execution_id",
        "workspace_id",
        "conversation_id",
        "message_id",
        "mode",
        "model_id",
    ]
    missing = [field for field in required_fields if is_blank(payload.get(field))]
    if missing:
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="Missing required fields",
            details={"missing_fields": missing},
        )

    execution_id = str(payload["execution_id"]).strip()
    queue_index = parse_queue_index(payload.get("queue_index"), request)
    if isinstance(queue_index, JSONResponse):
        return queue_index

    mode = str(payload.get("mode") or "agent").strip().lower()
    if mode not in {"agent", "plan"}:
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="mode must be agent or plan",
            details={"field": "mode"},
        )

    now = now_iso()
    normalized = {
        "execution_id": execution_id,
        "workspace_id": str(payload["workspace_id"]).strip(),
        "conversation_id": str(payload["conversation_id"]).strip(),
        "message_id": str(payload["message_id"]).strip(),
        "mode": mode,
        "mode_snapshot": str(payload.get("mode_snapshot") or mode).strip().lower(),
        "model_id": str(payload["model_id"]).strip(),
        "model_snapshot": payload.get("model_snapshot")
        if isinstance(payload.get("model_snapshot"), dict)
        else {"model_id": str(payload["model_id"]).strip()},
        "project_revision_snapshot": safe_int(payload.get("project_revision_snapshot"), 0),
        "state": "pending",
        "queue_index": queue_index,
        "trace_id": str(payload.get("trace_id") or request.state.trace_id),
        "content": str(payload.get("content") or ""),
        "accepted_at": now,
        "updated_at": now,
    }
    _executions[execution_id] = normalized
    _execution_confirm_queues.setdefault(execution_id, asyncio.Queue())
    _execution_cancel_events.setdefault(execution_id, asyncio.Event())

    task = _execution_tasks.get(execution_id)
    if task is None or task.done():
        _execution_tasks[execution_id] = asyncio.create_task(_run_execution(execution_id))

    return JSONResponse(status_code=202, content={"accepted": True, "execution": normalized})


@router.post("/internal/executions/{execution_id}/confirm")
async def internal_execution_confirm(execution_id: str, request: Request):
    auth_err = require_internal_token(request)
    if auth_err is not None:
        return auth_err
    payload, err = await decode_json(request)
    if err is not None:
        return err

    normalized_execution_id = str(execution_id).strip()
    if normalized_execution_id not in _executions:
        return standard_error_response(
            request=request,
            status_code=404,
            code="EXECUTION_NOT_FOUND",
            message="Execution does not exist",
            details={"execution_id": normalized_execution_id},
        )

    decision = str(payload.get("decision") or "").strip().lower()
    if decision not in {"approve", "deny"}:
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="decision must be approve or deny",
            details={"field": "decision"},
        )

    queue = _execution_confirm_queues.setdefault(normalized_execution_id, asyncio.Queue())
    queue.put_nowait(decision)
    return JSONResponse(status_code=202, content={"accepted": True, "decision": decision})


@router.post("/internal/executions/{execution_id}/stop")
async def internal_execution_stop(execution_id: str, request: Request):
    auth_err = require_internal_token(request)
    if auth_err is not None:
        return auth_err

    normalized_execution_id = str(execution_id).strip()
    if normalized_execution_id not in _executions:
        return standard_error_response(
            request=request,
            status_code=404,
            code="EXECUTION_NOT_FOUND",
            message="Execution does not exist",
            details={"execution_id": normalized_execution_id},
        )

    _execution_cancel_events.setdefault(normalized_execution_id, asyncio.Event()).set()
    return JSONResponse(
        status_code=202,
        content={"accepted": True, "execution_id": normalized_execution_id},
    )


@router.post("/internal/events")
async def internal_events(request: Request):
    auth_err = require_internal_token(request)
    if auth_err is not None:
        return auth_err

    payload, err = await decode_json(request)
    if err is not None:
        return err

    required_fields = ["event_id", "execution_id", "conversation_id", "type", "sequence", "queue_index"]
    missing = [field for field in required_fields if is_blank(payload.get(field))]
    if missing:
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="Missing required fields",
            details={"missing_fields": missing},
        )

    execution_id = str(payload["execution_id"]).strip()
    execution = _executions.get(execution_id)
    if execution is None:
        return standard_error_response(
            request=request,
            status_code=404,
            code="EXECUTION_NOT_FOUND",
            message="Execution does not exist",
            details={"execution_id": execution_id},
        )

    sequence = parse_int(payload.get("sequence"), "sequence", request)
    if isinstance(sequence, JSONResponse):
        return sequence
    queue_index = parse_int(payload.get("queue_index"), "queue_index", request)
    if isinstance(queue_index, JSONResponse):
        return queue_index

    event = {
        "event_id": str(payload["event_id"]).strip(),
        "execution_id": execution_id,
        "conversation_id": str(payload["conversation_id"]).strip(),
        "trace_id": str(payload.get("trace_id") or request.state.trace_id),
        "sequence": sequence,
        "queue_index": queue_index,
        "type": str(payload["type"]).strip(),
        "timestamp": str(payload.get("timestamp") or now_iso()),
        "payload": payload.get("payload") if isinstance(payload.get("payload"), dict) else {},
    }
    _events.append(event)
    _apply_execution_state_by_event(execution, event["type"], event["payload"])
    return JSONResponse(status_code=202, content={"accepted": True, "event": event})


async def _run_execution(execution_id: str) -> None:
    execution = _executions.get(execution_id)
    if execution is None:
        return
    try:
        await run_execution_loop(
            execution=execution,
            emit_event=_emit_event,
            wait_confirmation=_wait_for_confirmation_or_cancel,
            is_cancelled=_is_cancelled,
        )
    finally:
        _execution_tasks.pop(execution_id, None)
        _execution_confirm_queues.pop(execution_id, None)
        _execution_cancel_events.pop(execution_id, None)
        _execution_sequence.pop(execution_id, None)


async def _emit_event(execution: dict[str, Any], event_type: str, payload: dict[str, Any]) -> None:
    execution_id = str(execution.get("execution_id") or "").strip()
    if execution_id == "":
        return

    sequence = _execution_sequence.get(execution_id, 0) + 1
    _execution_sequence[execution_id] = sequence
    event = {
        "event_id": f"evt_{execution_id}_{sequence}",
        "execution_id": execution_id,
        "conversation_id": str(execution.get("conversation_id") or "").strip(),
        "trace_id": str(execution.get("trace_id") or "").strip(),
        "sequence": sequence,
        "queue_index": safe_int(execution.get("queue_index"), 0),
        "type": event_type,
        "timestamp": now_iso(),
        "payload": payload,
    }
    _events.append(event)
    _apply_execution_state_by_event(execution, event_type, payload)
    _post_hub_event(event)


def _apply_execution_state_by_event(execution: dict[str, Any], event_type: str, payload: dict[str, Any]) -> None:
    if event_type == "execution_started":
        execution["state"] = "executing"
    elif event_type == "confirmation_required":
        execution["state"] = "confirming"
    elif event_type == "confirmation_resolved":
        decision = str(payload.get("decision") or "").strip().lower()
        execution["state"] = "cancelled" if decision == "deny" else "executing"
    elif event_type == "execution_done":
        execution["state"] = "completed"
    elif event_type == "execution_error":
        execution["state"] = "failed"
    elif event_type == "execution_stopped":
        execution["state"] = "cancelled"
    execution["updated_at"] = now_iso()
    _executions[str(execution.get("execution_id") or "").strip()] = execution


async def _wait_for_confirmation_or_cancel(execution_id: str, timeout_seconds: int) -> str:
    queue = _execution_confirm_queues.setdefault(execution_id, asyncio.Queue())
    cancel_event = _execution_cancel_events.setdefault(execution_id, asyncio.Event())
    start = asyncio.get_running_loop().time()

    while True:
        if cancel_event.is_set():
            return "cancelled"

        elapsed = asyncio.get_running_loop().time() - start
        if elapsed >= timeout_seconds:
            return "deny"

        try:
            decision = await asyncio.wait_for(queue.get(), timeout=0.2)
            normalized = str(decision).strip().lower()
            if normalized in {"approve", "deny"}:
                return normalized
        except TimeoutError:
            continue


def _is_cancelled(execution_id: str) -> bool:
    cancel_event = _execution_cancel_events.get(execution_id)
    return bool(cancel_event and cancel_event.is_set())


def _post_hub_event(event: dict[str, Any]) -> None:
    hub_base_url = os.getenv("HUB_BASE_URL", "").strip().rstrip("/")
    if hub_base_url == "":
        return

    token = os.getenv("HUB_INTERNAL_TOKEN", DEFAULT_INTERNAL_TOKEN).strip()
    request = urllib.request.Request(
        f"{hub_base_url}/internal/events",
        data=json.dumps(event).encode("utf-8"),
        method="POST",
        headers={
            "Content-Type": "application/json",
            INTERNAL_TOKEN_HEADER: token,
            TRACE_HEADER: str(event.get("trace_id") or "").strip(),
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=5):
            pass
    except (urllib.error.HTTPError, urllib.error.URLError, TimeoutError):
        return
