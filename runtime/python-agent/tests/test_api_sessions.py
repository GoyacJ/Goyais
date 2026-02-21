import uuid

from fastapi.testclient import TestClient

from app.main import app


def _create_project(client: TestClient, project_id: str) -> str:
    resp = client.post(
        "/v1/projects",
        json={
            "project_id": project_id,
            "name": "Session Project",
            "workspace_path": f"./{project_id}",
        },
    )
    assert resp.status_code == 200
    return project_id


def test_create_and_list_sessions():
    project_id = f"project-sessions-{uuid.uuid4().hex[:8]}"
    with TestClient(app) as client:
        _create_project(client, project_id)

        create_resp = client.post(
            "/v1/sessions",
            json={
                "project_id": project_id,
                "title": "First session",
            },
        )
        assert create_resp.status_code == 200
        session = create_resp.json()["session"]
        assert session["project_id"] == project_id
        assert session["title"] == "First session"

        list_resp = client.get("/v1/sessions", params={"project_id": project_id})
        assert list_resp.status_code == 200
        sessions = list_resp.json()["sessions"]
        assert len(sessions) >= 1
        assert sessions[0]["session_id"] == session["session_id"]


def test_rename_session():
    project_id = f"project-rename-{uuid.uuid4().hex[:8]}"
    with TestClient(app) as client:
        _create_project(client, project_id)

        create_resp = client.post(
            "/v1/sessions",
            json={
                "project_id": project_id,
                "title": "Old title",
            },
        )
        assert create_resp.status_code == 200
        session_id = create_resp.json()["session"]["session_id"]

        rename_resp = client.patch(
            f"/v1/sessions/{session_id}",
            json={"title": "New title"},
        )
        assert rename_resp.status_code == 200
        assert rename_resp.json()["session"]["title"] == "New title"


def test_sessions_validation_errors():
    with TestClient(app) as client:
        missing_project = client.post("/v1/sessions", json={"title": "x"})
        assert missing_project.status_code == 400

        missing_title = client.patch("/v1/sessions/s1", json={})
        assert missing_title.status_code == 400

        missing_query = client.get("/v1/sessions", params={"project_id": ""})
        assert missing_query.status_code == 400
