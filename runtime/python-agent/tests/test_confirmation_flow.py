import asyncio

import pytest

from app.errors import GoyaisApiError
from app.services.audit_service import AuditService
from app.services.confirmation_service import ConfirmationService
from app.services.run_service import RunService
from app.sse.event_bus import EventBus


class FakeRepo:
    def __init__(self):
        self.values = {}

    async def get_tool_confirmation_status(self, run_id, call_id):
        return self.values.get((run_id, call_id))

    async def upsert_tool_confirmation_status(self, run_id, call_id, status, decided_by="user"):
        self.values[(run_id, call_id)] = status

    async def resolve_pending_tool_confirmation(self, run_id, call_id, status, decided_by="user"):
        key = (run_id, call_id)
        if self.values.get(key) not in {None, "pending"}:
            return False
        self.values[key] = status
        return True


def test_confirmation_wait_and_resolve():
    repo = FakeRepo()
    repo.values[("run1", "call1")] = "pending"
    service = ConfirmationService(repo)

    async def waiter():
        return await service.wait_for("run1", "call1", timeout_seconds=1)

    async def run():
        task = asyncio.create_task(waiter())
        await asyncio.sleep(0.05)
        await service.resolve("run1", "call1", True)
        return await task

    assert asyncio.run(run()) is True


def test_confirmation_conflict_raises_error():
    repo = FakeRepo()
    repo.values[("run1", "call1")] = "approved"
    service = ConfirmationService(repo)

    async def run():
        with pytest.raises(GoyaisApiError) as exc:
            await service.resolve("run1", "call1", True)
        assert exc.value.code == "E_CONFIRMATION_ALREADY_DECIDED"
        assert exc.value.status_code == 409

    asyncio.run(run())


class _RecoveryRepo:
    def __init__(self):
        self.values = {}
        self.pending = [{"run_id": "run-1", "call_id": "call-1"}]
        self.events = []
        self.run_updates = []
        self.audit_records = []
        self._seq = 0

    async def list_pending_confirmations(self):
        return self.pending

    async def upsert_tool_confirmation_status(self, run_id, call_id, status, decided_by="user"):
        self.values[(run_id, call_id)] = status

    async def resolve_pending_tool_confirmation(self, run_id, call_id, status, decided_by="user"):
        self.values[(run_id, call_id)] = status
        return True

    async def update_run_status(self, run_id, status):
        self.run_updates.append((run_id, status))

    async def next_seq(self, run_id):
        self._seq += 1
        return self._seq

    async def get_run_trace_id(self, run_id):
        return "trace-recovery"

    async def insert_event(self, event):
        self.events.append(event)

    async def insert_audit(self, **kwargs):
        self.audit_records.append(kwargs)


class _NoopConfirmationService:
    async def wait_for(self, run_id, call_id, timeout_seconds=600):  # pragma: no cover
        return False


def test_recover_pending_confirmation_emits_error_and_done():
    repo = _RecoveryRepo()
    service = RunService(
        repo=repo,
        bus=EventBus(),
        confirmation_service=_NoopConfirmationService(),
        audit_service=AuditService(repo),
        agent_mode="mock",
    )

    asyncio.run(service.recover_pending_confirmations_after_restart())

    assert repo.values[("run-1", "call-1")] == "denied"
    assert repo.run_updates == [("run-1", "failed")]
    assert [event["type"] for event in repo.events] == ["error", "done"]
    assert repo.audit_records[0]["action"] == "confirmation_recovery"
