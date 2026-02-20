from __future__ import annotations

from contextvars import ContextVar

_current_user_id: ContextVar[str] = ContextVar("goyais_user_id", default="user")


def set_current_user_id(user_id: str) -> None:
    normalized = user_id.strip() if isinstance(user_id, str) else ""
    _current_user_id.set(normalized or "user")


def get_current_user_id() -> str:
    value = _current_user_id.get()
    return value.strip() or "user"
