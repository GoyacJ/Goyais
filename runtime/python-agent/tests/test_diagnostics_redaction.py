import asyncio
import tempfile
import uuid
from pathlib import Path

from app.deps import get_repo


async def _seed_execution_with_sensitive_audit(execution_id: str, trace_id: str, workspace_path: str) -> None:
    repo = get_repo()
    payload = {
        "project_id": f"diag-project-{execution_id}",
        "session_id": f"diag-session-{execution_id}",
        "input": "diagnostics",
        "model_config_id": "",
        "workspace_path": workspace_path,
    }
    await repo.ensure_project(payload["project_id"], payload["workspace_path"])
    await repo.ensure_session(payload["session_id"], payload["project_id"])
    await repo.create_execution(payload, execution_id, trace_id)
    await repo.insert_audit(
        audit_id=str(uuid.uuid4()),
        trace_id=trace_id,
        user_id="diag-user",
        execution_id=execution_id,
        event_id=None,
        call_id="call-redact",
        action="tool_call",
        tool_name="run_command",
        args={
            "Authorization": "Bearer super-secret-token",
            "secret_ref": "keychain:openai:default",
            "path": "/tmp/very/secret/path/file.txt",
        },
        result={"apiKey": "sk-real-value"},
        requires_confirmation=False,
        user_decision="n/a",
        outcome="requested",
    )


def test_diagnostics_endpoint_redacts_sensitive_values(isolated_client):
    execution_id = f"diag-execution-{uuid.uuid4().hex[:8]}"
    trace_id = "trace-diag-1"
    workspace = tempfile.mkdtemp(prefix="goyais-diag-")
    Path(workspace, "README.md").write_text("# Diagnostics\\n", encoding="utf-8")

    asyncio.run(_seed_execution_with_sensitive_audit(execution_id, trace_id, workspace))
    response = isolated_client.get(
        f"/v1/diagnostics/execution/{execution_id}",
        headers={"X-Runtime-Token": "dev-secret-token"},
    )

    assert response.status_code == 200
    body = response.json()
    serialized = str(body)
    assert "super-secret-token" not in serialized
    assert "keychain:openai:default" not in serialized
    assert "sk-real-value" not in serialized
    assert "/tmp/very/secret/path/file.txt" not in serialized
    assert "file.txt" in serialized
