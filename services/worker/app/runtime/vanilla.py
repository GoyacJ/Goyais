from __future__ import annotations

from typing import Any

from app.execution_engine import run_execution_loop
from app.runtime.base import EmitEventFn, IsCancelledFn, WaitConfirmationFn


class VanillaRuntime:
    async def run(
        self,
        execution: dict[str, Any],
        emit_event: EmitEventFn,
        wait_confirmation: WaitConfirmationFn,
        is_cancelled: IsCancelledFn,
    ) -> None:
        await run_execution_loop(
            execution=execution,
            emit_event=emit_event,
            wait_confirmation=wait_confirmation,
            is_cancelled=is_cancelled,
        )

