def test_dispatch_internal_execution_returns_accepted(isolated_client):
    payload = {
        "execution_id": "exec-1",
        "trace_id": "trace-1",
        "workspace_id": "ws-local",
        "project_id": "project-1",
        "session_id": "session-1",
        "user_message": "update readme",
        "repo_root": ".",
        "use_worktree": False,
    }
    resp = isolated_client.post("/internal/executions", json=payload)

    assert resp.status_code == 202
    body = resp.json()
    assert body["execution_id"] == "exec-1"
    assert body["status"] == "accepted"


def test_internal_confirmation_returns_ok(isolated_client):
    resp = isolated_client.post(
        "/internal/confirmations",
        json={
            "execution_id": "exec-1",
            "call_id": "call-1",
            "decision": "approved",
        },
    )

    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"
