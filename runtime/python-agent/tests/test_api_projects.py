import sqlite3
import uuid
from datetime import datetime, timezone
import os


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _create_project(client, project_id: str) -> None:
    resp = client.post(
        "/v1/projects",
        json={
            "project_id": project_id,
            "name": "Project For Delete",
            "workspace_path": f"./{project_id}",
        },
    )
    assert resp.status_code == 200


def _create_session(client, project_id: str) -> str:
    resp = client.post(
        "/v1/sessions",
        json={
            "project_id": project_id,
            "title": "Session For Delete",
        },
    )
    assert resp.status_code == 200
    return resp.json()["session"]["session_id"]


def _create_model_config(client, model_config_id: str) -> None:
    resp = client.post(
        "/v1/model-configs",
        json={
            "model_config_id": model_config_id,
            "provider": "openai",
            "model": "gpt-5.2",
            "base_url": "https://api.openai.com/v1",
            "temperature": 0,
            "secret_ref": f"keychain:openai:{model_config_id}",
        },
    )
    assert resp.status_code == 200


def test_delete_project_removes_tracking_record(isolated_client):
    project_id = f"project-delete-{uuid.uuid4().hex[:8]}"
    _create_project(isolated_client, project_id)

    delete_resp = isolated_client.delete(f"/v1/projects/{project_id}")
    assert delete_resp.status_code == 200
    assert delete_resp.json()["ok"] is True

    list_resp = isolated_client.get("/v1/projects")
    assert list_resp.status_code == 200
    assert all(item["project_id"] != project_id for item in list_resp.json()["projects"])


def test_delete_project_returns_not_found(isolated_client):
    delete_resp = isolated_client.delete("/v1/projects/not-exists")
    assert delete_resp.status_code == 404
    assert delete_resp.json()["error"]["code"] == "E_NOT_FOUND"


def test_delete_model_config_detaches_runs_before_delete(isolated_client):
    project_id = f"project-model-delete-{uuid.uuid4().hex[:8]}"
    model_config_id = f"model-delete-{uuid.uuid4().hex[:8]}"
    _create_project(isolated_client, project_id)
    session_id = _create_session(isolated_client, project_id)
    _create_model_config(isolated_client, model_config_id)

    db_path = os.environ["GOYAIS_DB_PATH"]
    run_id = f"run-{uuid.uuid4().hex[:8]}"
    with sqlite3.connect(db_path) as conn:
        conn.execute(
            """
            CREATE TABLE IF NOT EXISTS runs (
              run_id TEXT PRIMARY KEY,
              project_id TEXT NOT NULL,
              session_id TEXT NOT NULL,
              model_config_id TEXT REFERENCES model_configs(model_config_id),
              input TEXT NOT NULL,
              workspace_path TEXT NOT NULL,
              status TEXT NOT NULL,
              created_at TEXT NOT NULL
            )
            """
        )
        conn.execute(
            """
            INSERT INTO runs(run_id, project_id, session_id, model_config_id, input, workspace_path, status, created_at)
            VALUES(?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                run_id,
                project_id,
                session_id,
                model_config_id,
                "hello",
                f"./{project_id}",
                "failed",
                _now_iso(),
            ),
        )
        conn.commit()

    delete_resp = isolated_client.delete(f"/v1/model-configs/{model_config_id}")
    assert delete_resp.status_code == 200
    assert delete_resp.json()["ok"] is True

    with sqlite3.connect(db_path) as conn:
        row = conn.execute("SELECT model_config_id FROM runs WHERE run_id=?", (run_id,)).fetchone()
    assert row is not None
    assert row[0] is None
