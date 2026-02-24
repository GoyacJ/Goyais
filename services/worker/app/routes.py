from __future__ import annotations

import asyncio
from datetime import UTC, datetime
import json
import os
from typing import Any
import urllib.error
import urllib.request

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

from app.errors import standard_error_response
from app.trace import TRACE_HEADER

router = APIRouter()

_executions: dict[str, dict[str, Any]] = {}
_events: list[dict[str, Any]] = []
_execution_tasks: dict[str, asyncio.Task[None]] = {}
_execution_confirm_queues: dict[str, asyncio.Queue[str]] = {}
_execution_cancel_events: dict[str, asyncio.Event] = {}
_execution_sequence: dict[str, int] = {}

_INTERNAL_TOKEN_HEADER = "X-Internal-Token"
_AUTHORIZATION_HEADER = "Authorization"
_BEARER_PREFIX = "Bearer "
_DEFAULT_INTERNAL_TOKEN = "goyais-internal-token"


@router.get("/health")
def health() -> dict[str, object]:
    return {"ok": True, "version": "0.4.0"}


@router.post("/internal/executions")
async def internal_executions(request: Request):
    auth_err = _require_internal_token(request)
    if auth_err is not None:
        return auth_err

    payload, err = await _decode_json(request)
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
    missing = [field for field in required_fields if _is_blank(payload.get(field))]
    if missing:
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="Missing required fields",
            details={"missing_fields": missing},
        )

    execution_id = str(payload["execution_id"]).strip()
    queue_index = _parse_queue_index(payload.get("queue_index"), request)
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

    now = _now_iso()
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
        "project_revision_snapshot": _safe_int(payload.get("project_revision_snapshot"), 0),
        "state": "pending",
        "queue_index": queue_index,
        "trace_id": str(payload.get("trace_id") or request.state.trace_id),
        "content": str(payload.get("content") or ""),
        "accepted_at": now,
        "updated_at": now,
    }
    _executions[execution_id] = normalized
    if execution_id not in _execution_confirm_queues:
        _execution_confirm_queues[execution_id] = asyncio.Queue()
    if execution_id not in _execution_cancel_events:
        _execution_cancel_events[execution_id] = asyncio.Event()

    task = _execution_tasks.get(execution_id)
    if task is None or task.done():
        _execution_tasks[execution_id] = asyncio.create_task(_run_execution(execution_id))

    return JSONResponse(
        status_code=202,
        content={"accepted": True, "execution": normalized},
    )


@router.post("/internal/executions/{execution_id}/confirm")
async def internal_execution_confirm(execution_id: str, request: Request):
    auth_err = _require_internal_token(request)
    if auth_err is not None:
        return auth_err
    payload, err = await _decode_json(request)
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
    auth_err = _require_internal_token(request)
    if auth_err is not None:
        return auth_err

    normalized_execution_id = str(execution_id).strip()
    execution = _executions.get(normalized_execution_id)
    if execution is None:
        return standard_error_response(
            request=request,
            status_code=404,
            code="EXECUTION_NOT_FOUND",
            message="Execution does not exist",
            details={"execution_id": normalized_execution_id},
        )

    cancel_event = _execution_cancel_events.setdefault(normalized_execution_id, asyncio.Event())
    cancel_event.set()

    return JSONResponse(status_code=202, content={"accepted": True, "execution_id": normalized_execution_id})


@router.post("/internal/events")
async def internal_events(request: Request):
    auth_err = _require_internal_token(request)
    if auth_err is not None:
        return auth_err

    payload, err = await _decode_json(request)
    if err is not None:
        return err

    required_fields = ["event_id", "execution_id", "conversation_id", "type", "sequence", "queue_index"]
    missing = [field for field in required_fields if _is_blank(payload.get(field))]
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

    sequence = _parse_int(payload.get("sequence"), "sequence", request)
    if isinstance(sequence, JSONResponse):
        return sequence
    queue_index = _parse_int(payload.get("queue_index"), "queue_index", request)
    if isinstance(queue_index, JSONResponse):
        return queue_index

    timestamp = str(payload.get("timestamp") or _now_iso())
    event_type = str(payload["type"]).strip()
    trace_id = str(payload.get("trace_id") or request.state.trace_id)
    event = {
        "event_id": str(payload["event_id"]).strip(),
        "execution_id": execution_id,
        "conversation_id": str(payload["conversation_id"]).strip(),
        "trace_id": trace_id,
        "sequence": sequence,
        "queue_index": queue_index,
        "type": event_type,
        "timestamp": timestamp,
        "payload": payload.get("payload") if isinstance(payload.get("payload"), dict) else {},
    }
    _events.append(event)
    _apply_execution_state_by_event(execution, event_type, event["payload"])

    return JSONResponse(status_code=202, content={"accepted": True, "event": event})


async def _run_execution(execution_id: str) -> None:
    execution = _executions.get(execution_id)
    if execution is None:
        return

    try:
        await _emit_event(
            execution,
            event_type="execution_started",
            payload={"mode": execution.get("mode_snapshot") or execution.get("mode")},
        )
        if _is_cancelled(execution_id):
            await _emit_event(execution, event_type="execution_stopped", payload={"reason": "stop_requested"})
            return

        content = str(execution.get("content") or "")
        risk_level = _classify_risk(content)

        mode_snapshot = str(execution.get("mode_snapshot") or execution.get("mode") or "agent").strip().lower()
        if mode_snapshot == "plan" and risk_level in {"high", "critical"}:
            await _emit_event(
                execution,
                event_type="execution_error",
                payload={
                    "reason": "PLAN_MODE_REJECTED",
                    "message": "Plan mode rejects high/critical operations.",
                    "risk_level": risk_level,
                },
            )
            return

        if mode_snapshot == "agent" and risk_level in {"high", "critical"}:
            await _emit_event(
                execution,
                event_type="confirmation_required",
                payload={
                    "risk_level": risk_level,
                    "summary": "Operation requires approval.",
                    "preview": content[:400],
                },
            )
            decision = await _wait_for_confirmation_or_cancel(execution_id, timeout_seconds=900)
            if decision == "cancelled":
                await _emit_event(execution, event_type="execution_stopped", payload={"reason": "stop_requested"})
                return
            await _emit_event(
                execution,
                event_type="confirmation_resolved",
                payload={"decision": decision},
            )
            if decision != "approve":
                await _emit_event(
                    execution,
                    event_type="execution_error",
                    payload={"reason": "USER_DENIED", "message": "Operation denied by user."},
                )
                return

        if _is_cancelled(execution_id):
            await _emit_event(execution, event_type="execution_stopped", payload={"reason": "stop_requested"})
            return

        await _emit_event(
            execution,
            event_type="thinking_delta",
            payload={"delta": "Analyzing project context and planning patch..."},
        )
        await asyncio.sleep(0.05)
        if _is_cancelled(execution_id):
            await _emit_event(execution, event_type="execution_stopped", payload={"reason": "stop_requested"})
            return

        tool_name = "write_file" if risk_level in {"high", "critical"} else "read_file"
        await _emit_event(
            execution,
            event_type="tool_call",
            payload={"name": tool_name, "risk_level": risk_level},
        )
        await asyncio.sleep(0.03)
        await _emit_event(
            execution,
            event_type="tool_result",
            payload={"name": tool_name, "ok": True},
        )

        diff = [
            {
                "id": f"diff_{execution_id[-4:]}",
                "path": "src/main.ts",
                "change_type": "modified",
                "summary": "Apply worker generated patch",
            }
        ]
        await _emit_event(
            execution,
            event_type="diff_generated",
            payload={"files": len(diff), "diff": diff},
        )

        await _emit_event(
            execution,
            event_type="execution_done",
            payload={
                "content": f"Execution {execution_id} completed via worker loop.",
                "result": "ok",
            },
        )
    except Exception as exc:  # pragma: no cover - protection path
        await _emit_event(
            execution,
            event_type="execution_error",
            payload={"reason": "WORKER_RUNTIME_ERROR", "message": str(exc)},
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

    next_sequence = _execution_sequence.get(execution_id, 0) + 1
    _execution_sequence[execution_id] = next_sequence

    event = {
        "event_id": f"evt_{execution_id}_{next_sequence}",
        "execution_id": execution_id,
        "conversation_id": str(execution.get("conversation_id") or "").strip(),
        "trace_id": str(execution.get("trace_id") or "").strip(),
        "sequence": next_sequence,
        "queue_index": _safe_int(execution.get("queue_index"), 0),
        "type": event_type,
        "timestamp": _now_iso(),
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
    execution["updated_at"] = _now_iso()
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


def _classify_risk(content: str) -> str:
    normalized = content.lower()
    critical_keywords = [" delete ", " rm ", "删除", "remove file", "drop table"]
    high_keywords = [
        "write",
        "apply_patch",
        "run ",
        "command",
        "network",
        "edit ",
        "修改",
        "写入",
        "执行",
        "联网",
    ]

    wrapped = f" {normalized} "
    if any(keyword in wrapped for keyword in critical_keywords):
        return "critical"
    if any(keyword in normalized for keyword in high_keywords):
        return "high"
    return "low"


def _post_hub_event(event: dict[str, Any]) -> None:
    hub_base_url = os.getenv("HUB_BASE_URL", "").strip().rstrip("/")
    if hub_base_url == "":
        return

    token = os.getenv("HUB_INTERNAL_TOKEN", _DEFAULT_INTERNAL_TOKEN).strip()
    body = json.dumps(event).encode("utf-8")
    request = urllib.request.Request(
        f"{hub_base_url}/internal/events",
        data=body,
        method="POST",
        headers={
            "Content-Type": "application/json",
            _INTERNAL_TOKEN_HEADER: token,
            TRACE_HEADER: str(event.get("trace_id") or "").strip(),
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=5):
            pass
    except (urllib.error.HTTPError, urllib.error.URLError, TimeoutError):
        # Best effort callback: keep local state if hub callback fails.
        return


async def _decode_json(request: Request) -> tuple[dict[str, Any], JSONResponse | None]:
    try:
        payload = await request.json()
    except Exception:
        return {}, standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="Invalid JSON request body",
            details={},
        )

    if not isinstance(payload, dict):
        return {}, standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message="Request body must be a JSON object",
            details={},
        )

    return payload, None


def _parse_queue_index(raw_value: Any, request: Request) -> int | JSONResponse:
    if raw_value is None:
        return 0
    return _parse_int(raw_value, "queue_index", request)


def _parse_int(raw_value: Any, field: str, request: Request) -> int | JSONResponse:
    try:
        value = int(raw_value)
    except (TypeError, ValueError):
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message=f"{field} must be an integer",
            details={"field": field},
        )

    if value < 0:
        return standard_error_response(
            request=request,
            status_code=400,
            code="VALIDATION_ERROR",
            message=f"{field} must be >= 0",
            details={"field": field},
        )
    return value


def _safe_int(raw_value: Any, default: int) -> int:
    try:
        value = int(raw_value)
        if value < 0:
            return default
        return value
    except (TypeError, ValueError):
        return default


def _is_blank(value: Any) -> bool:
    if value is None:
        return True
    if isinstance(value, str):
        return value.strip() == ""
    return False


def _now_iso() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def _require_internal_token(request: Request) -> JSONResponse | None:
    expected_token = os.getenv("WORKER_INTERNAL_TOKEN", _DEFAULT_INTERNAL_TOKEN).strip()
    if expected_token == "":
        return None

    provided_token = _extract_internal_token(request)
    if provided_token == "":
        return standard_error_response(
            request=request,
            status_code=401,
            code="AUTH_INTERNAL_TOKEN_REQUIRED",
            message="Internal token is required",
            details={"header": _INTERNAL_TOKEN_HEADER},
        )
    if provided_token != expected_token:
        return standard_error_response(
            request=request,
            status_code=401,
            code="AUTH_INVALID_INTERNAL_TOKEN",
            message="Internal token is invalid",
            details={},
        )
    return None


def _extract_internal_token(request: Request) -> str:
    token = request.headers.get(_INTERNAL_TOKEN_HEADER, "").strip()
    if token != "":
        return token

    authorization = request.headers.get(_AUTHORIZATION_HEADER, "").strip()
    if not authorization.startswith(_BEARER_PREFIX):
        return ""
    return authorization[len(_BEARER_PREFIX) :].strip()
