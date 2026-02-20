from fastapi.testclient import TestClient

from app.main import app


def test_create_run_returns_run_id():
    payload = {
        "project_id": "project_1",
        "session_id": "session_1",
        "input": "update readme",
        "model_config_id": "model_1",
        "workspace_path": ".",
        "options": {"use_worktree": False},
    }
    with TestClient(app) as client:
        resp = client.post("/v1/runs", json=payload)

    assert resp.status_code == 200
    assert "run_id" in resp.json()
