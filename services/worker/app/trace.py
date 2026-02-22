from contextvars import ContextVar, Token
import secrets

TRACE_HEADER = "X-Trace-Id"
_trace_id_ctx: ContextVar[str] = ContextVar("trace_id", default="")


def generate_trace_id() -> str:
    return f"tr_{secrets.token_hex(16)}"


def set_trace_id(trace_id: str) -> Token[str]:
    return _trace_id_ctx.set(trace_id)


def reset_trace_id(token: Token[str]) -> None:
    _trace_id_ctx.reset(token)


def get_trace_id() -> str:
    return _trace_id_ctx.get()
