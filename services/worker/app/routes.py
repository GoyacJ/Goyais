from datetime import UTC, datetime
import os
from typing import Any

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

from app.errors import standard_error_response

router = APIRouter()

_executions: dict[str, dict[str, Any]] = {}
_events: list[dict[str, Any]] = []
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

    now = _now_iso()
    normalized = {
        "execution_id": execution_id,
        "workspace_id": str(payload["workspace_id"]).strip(),
        "conversation_id": str(payload["conversation_id"]).strip(),
        "message_id": str(payload["message_id"]).strip(),
        "mode": str(payload["mode"]).strip(),
        "model_id": str(payload["model_id"]).strip(),
        "state": str(payload.get("state") or "executing"),
        "queue_index": queue_index,
        "trace_id": str(payload.get("trace_id") or request.state.trace_id),
        "content": str(payload.get("content") or ""),
        "accepted_at": now,
        "updated_at": now,
    }
    _executions[execution_id] = normalized

    return JSONResponse(
        status_code=202,
        content={"accepted": True, "execution": normalized},
    )


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

    if event_type == "execution_done":
        execution["state"] = "completed"
    elif event_type == "execution_error":
        execution["state"] = "failed"
    elif event_type == "execution_stopped":
        execution["state"] = "cancelled"
    execution["updated_at"] = _now_iso()
    _executions[execution_id] = execution

    return JSONResponse(status_code=202, content={"accepted": True, "event": event})


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
