from __future__ import annotations

import uuid
from contextvars import ContextVar

TRACE_HEADER = "X-Trace-Id"

_trace_id_var: ContextVar[str | None] = ContextVar("trace_id", default=None)


def generate_trace_id() -> str:
    if hasattr(uuid, "uuid7"):
        return str(uuid.uuid7())  # type: ignore[attr-defined]
    return str(uuid.uuid4())


def normalize_trace_id(candidate: str | None) -> str:
    value = (candidate or "").strip()
    if value:
        return value
    return generate_trace_id()


def set_current_trace_id(trace_id: str) -> None:
    _trace_id_var.set(trace_id)


def get_current_trace_id() -> str:
    value = _trace_id_var.get()
    if value:
        return value
    return generate_trace_id()
