import asyncio
import os
import time
from pathlib import Path

from app.db.connection import open_connection
from app.db.migrations import apply_migrations
from app.db.repositories import Repository
from app.services.audit_service import AuditService
from app.services.confirmation_service import ConfirmationService
from app.services.run_service import RunService
from app.sse.event_bus import EventBus


async def _wait_for_apply_call_id(repo: Repository, run_id: str, timeout: float = 10.0) -> str:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        events = await repo.list_events_by_run(run_id)
        for event in events:
            if event["type"] == "tool_call" and event["payload"].get("tool_name") == "apply_patch":
                return str(event["payload"]["call_id"])
        await asyncio.sleep(0.05)
    raise AssertionError("timed out waiting for apply_patch tool_call")


async def _wait_for_tool_call_id(
    repo: Repository, run_id: str, tool_name: str, *, timeout: float = 10.0
) -> str:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        events = await repo.list_events_by_run(run_id)
        for event in events:
            if event["type"] == "tool_call" and event["payload"].get("tool_name") == tool_name:
                return str(event["payload"]["call_id"])
        await asyncio.sleep(0.05)
    raise AssertionError(f"timed out waiting for {tool_name} tool_call")


async def _run_with_confirmation(workspace: Path, db_path: Path, approved: bool):
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

        run_id = f"run-{'approve' if approved else 'deny'}"
        payload = {
            "project_id": "project-1",
            "session_id": "session-1",
            "input": "把 README 的标题改成 Flow Test",
            "model_config_id": "",
            "workspace_path": str(workspace),
            "options": {"use_worktree": False},
        }

        task = asyncio.create_task(service.start_run(run_id, payload, f"trace-{run_id}"))
        call_id = await _wait_for_apply_call_id(repo, run_id)
        await confirmation.resolve(run_id, call_id, approved)
        await task

        events = await repo.list_events_by_run(run_id)
        cursor = await conn.execute("SELECT COUNT(*) AS count FROM audit_logs WHERE run_id=?", (run_id,))
        row = await cursor.fetchone()
        audit_count = int(row["count"])
        return run_id, call_id, events, audit_count
    finally:
        await conn.close()


def test_apply_patch_deny_path(tmp_path: Path):
    workspace = tmp_path / "workspace-deny"
    workspace.mkdir()
    readme = workspace / "README.md"
    original = "# Original Title\nbody\n"
    readme.write_text(original, encoding="utf-8")

    db_path = tmp_path / "runtime-deny.db"
    _, call_id, events, audit_count = asyncio.run(_run_with_confirmation(workspace, db_path, approved=False))

    assert readme.read_text(encoding="utf-8") == original
    apply_results = [e for e in events if e["type"] == "tool_result" and e["payload"].get("call_id") == call_id]
    assert apply_results
    assert apply_results[-1]["payload"]["ok"] is False
    assert audit_count > 0


def test_apply_patch_approve_path(tmp_path: Path):
    workspace = tmp_path / "workspace-approve"
    workspace.mkdir()
    readme = workspace / "README.md"
    readme.write_text("# Original Title\nbody\n", encoding="utf-8")

    db_path = tmp_path / "runtime-approve.db"
    _, call_id, events, audit_count = asyncio.run(_run_with_confirmation(workspace, db_path, approved=True))

    updated = readme.read_text(encoding="utf-8")
    assert updated.startswith("# Flow Test")
    apply_results = [e for e in events if e["type"] == "tool_result" and e["payload"].get("call_id") == call_id]
    assert apply_results
    assert apply_results[-1]["payload"]["ok"] is True
    assert audit_count > 0


def test_run_command_allowlist_denial_after_approval(tmp_path: Path):
    workspace = tmp_path / "workspace-command"
    workspace.mkdir()
    readme = workspace / "README.md"
    readme.write_text("# Original Title\nbody\n", encoding="utf-8")
    db_path = tmp_path / "runtime-command.db"

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

            run_id = "run-command-deny"
            payload = {
                "project_id": "project-1",
                "session_id": "session-1",
                "input": "把 README 的标题改成 Command Test",
                "model_config_id": "",
                "workspace_path": str(workspace),
                "options": {"use_worktree": False, "run_tests": "ls -la"},
            }

            task = asyncio.create_task(service.start_run(run_id, payload, f"trace-{run_id}"))
            apply_call_id = await _wait_for_tool_call_id(repo, run_id, "apply_patch")
            await confirmation.resolve(run_id, apply_call_id, True)

            command_call_id = await _wait_for_tool_call_id(repo, run_id, "run_command")
            await confirmation.resolve(run_id, command_call_id, True)
            await task

            events = await repo.list_events_by_run(run_id)
            command_results = [
                e for e in events if e["type"] == "tool_result" and e["payload"].get("call_id") == command_call_id
            ]
            assert command_results
            assert command_results[-1]["payload"]["ok"] is False
            assert command_results[-1]["payload"]["error"]["code"] == "E_TOOL_DENIED"
            assert events[-1]["type"] == "done"
            assert events[-1]["payload"]["status"] == "failed"
        finally:
            await conn.close()

    asyncio.run(scenario())
