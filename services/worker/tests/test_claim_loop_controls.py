from __future__ import annotations

import asyncio
from typing import Any

from app.hub_client import HubRequestError
from app.orchestrator.claim_loop import ClaimLoopService, _ExecutionControls


class _NotFoundHub:
    def __init__(self) -> None:
        self.calls = 0

    async def poll_control(
        self, execution_id: str, after_seq: int, wait_ms: int
    ) -> dict[str, Any]:
        del execution_id, after_seq, wait_ms
        self.calls += 1
        raise HubRequestError(
            404,
            '{"code":"EXECUTION_NOT_FOUND","message":"Execution does not exist"}',
        )


def test_execution_controls_stop_polling_when_execution_not_found() -> None:
    hub = _NotFoundHub()
    controls = _ExecutionControls(hub, "exec_missing")

    async def run_once() -> None:
        await controls.start()
        await asyncio.sleep(0.05)
        await controls.stop()

    asyncio.run(run_once())

    assert hub.calls == 1
    assert controls.is_cancelled("") is True


def test_claim_loop_default_max_concurrency_is_three(monkeypatch) -> None:
    monkeypatch.delenv("WORKER_MAX_CONCURRENCY", raising=False)
    service = ClaimLoopService()
    assert service.max_concurrency == 3


def test_claim_loop_max_concurrency_can_be_overridden(monkeypatch) -> None:
    monkeypatch.setenv("WORKER_MAX_CONCURRENCY", "5")
    service = ClaimLoopService()
    assert service.max_concurrency == 5
