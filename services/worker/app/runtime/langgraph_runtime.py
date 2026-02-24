from __future__ import annotations

from app.runtime.base import EmitEventFn, IsCancelledFn
from app.runtime.vanilla import VanillaRuntime


class LangGraphRuntime:
    """P1 runtime placeholder. P0 falls back to vanilla runtime."""

    def __init__(self) -> None:
        self._fallback = VanillaRuntime()

    async def run(
        self,
        execution: dict[str, object],
        emit_event: EmitEventFn,
        is_cancelled: IsCancelledFn,
    ) -> None:
        await emit_event(
            execution,
            "thinking_delta",
            {"stage": "runtime_fallback", "runtime": "langgraph", "fallback": "vanilla"},
        )
        await self._fallback.run(execution, emit_event, is_cancelled)
