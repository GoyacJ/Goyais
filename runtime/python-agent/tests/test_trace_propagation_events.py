import asyncio
import os
from pathlib import Path

from app.db.connection import open_connection
from app.db.migrations import apply_migrations
from app.db.repositories import Repository
from app.services.audit_service import AuditService
from app.services.confirmation_service import ConfirmationService
from app.services.run_service import RunService
from app.sse.event_bus import EventBus


def test_run_events_and_run_row_include_trace_id(tmp_path: Path):
    workspace = tmp_path / "workspace-trace"
    workspace.mkdir()
    (workspace / "README.md").write_text("# Trace Test\\n", encoding="utf-8")
    db_path = tmp_path / "runtime-trace.db"
    trace_id = "trace-test-run-1"

    async def scenario():
        os.environ["GOYAIS_DB_PATH"] = str(db_path)
        await apply_migrations()
        conn = await open_connection(db_path)
        try:
            repo = Repository(conn)
            confirmation = ConfirmationService(repo)
            service = RunService(
                repo=repo,
                bus=EventBus(),
                confirmation_service=confirmation,
                audit_service=AuditService(repo),
                agent_mode="mock",
            )

            run_id = "run-trace-id"
            payload = {
                "project_id": "project_trace",
                "session_id": "session_trace",
                "input": "update readme",
                "model_config_id": "",
                "workspace_path": str(workspace),
                "options": {"use_worktree": False},
            }

            task = asyncio.create_task(service.start_run(run_id, payload, trace_id))

            apply_call_id = None
            while apply_call_id is None:
                current = await repo.list_events_by_run(run_id)
                for event in current:
                    if event["type"] == "tool_call" and event["payload"].get("tool_name") == "apply_patch":
                        apply_call_id = str(event["payload"]["call_id"])
                        break
                if apply_call_id is None:
                    await asyncio.sleep(0.05)
            await confirmation.resolve(run_id, apply_call_id, True)

            while True:
                status = await repo.get_run_status(run_id)
                if status in {"completed", "failed"}:
                    break
                await asyncio.sleep(0.05)
            await task

            events = await repo.list_events_by_run(run_id)
            assert events
            for event in events:
                assert event["trace_id"] == trace_id
                assert event["payload"]["trace_id"] == trace_id

            cursor = await conn.execute("SELECT trace_id FROM runs WHERE run_id=?", (run_id,))
            row = await cursor.fetchone()
            assert row is not None
            assert row["trace_id"] == trace_id
        finally:
            await conn.close()

    asyncio.run(scenario())
