import asyncio

from app.services.audit_service import AuditService
from app.services.confirmation_service import ConfirmationService
from app.services.run_service import RunService
from app.sse.event_bus import EventBus


class FakeRepo:
    def __init__(self):
        self.events = []
        self.status_updates = []
        self.seq = 0

    async def ensure_project(self, project_id, workspace_path):
        return None

    async def ensure_session(self, session_id, project_id):
        return None

    async def create_run(self, payload, run_id):
        return None

    async def update_run_status(self, run_id, status):
        self.status_updates.append((run_id, status))

    async def next_seq(self, run_id):
        self.seq += 1
        return self.seq

    async def insert_event(self, event):
        self.events.append(event)

    async def get_model_config(self, model_config_id):
        return None

    async def insert_audit(self, **kwargs):
        return None


def test_graph_mode_emits_error_when_model_config_missing():
    repo = FakeRepo()
    service = RunService(
        repo=repo,
        bus=EventBus(),
        confirmation_service=ConfirmationService(repo),  # not used in this test
        audit_service=AuditService(repo),
        agent_mode="graph",
    )

    payload = {
        "project_id": "project-1",
        "session_id": "session-1",
        "input": "update readme",
        "model_config_id": "",
        "workspace_path": ".",
        "options": {"use_worktree": False},
    }

    asyncio.run(service.start_run("run-1", payload))

    assert repo.status_updates[-1] == ("run-1", "failed")
    assert repo.events[-2]["type"] == "error"
    assert "model_config_id is required" in repo.events[-2]["payload"]["message"]
    assert repo.events[-1]["type"] == "done"
    assert repo.events[-1]["payload"]["status"] == "failed"
