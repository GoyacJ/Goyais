from __future__ import annotations

from typing import Any, Awaitable, Callable, Protocol

EmitEventFn = Callable[[dict[str, Any], str, dict[str, Any]], Awaitable[None]]
IsCancelledFn = Callable[[str], bool]


class ExecutionRuntime(Protocol):
    async def run(
        self,
        execution: dict[str, Any],
        emit_event: EmitEventFn,
        is_cancelled: IsCancelledFn,
    ) -> None: ...
