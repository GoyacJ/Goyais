from fastapi.testclient import TestClient

from app.main import app


client = TestClient(app)


def test_internal_executions_accepts_request_and_keeps_trace_consistency() -> None:
    response = client.post(
        "/internal/executions",
        headers={"X-Trace-Id": "tr_worker_user"},
        json={
            "execution_id": "exec_01",
            "workspace_id": "ws_local",
            "conversation_id": "conv_01",
            "message_id": "msg_01",
            "mode": "agent",
            "model_id": "gpt-4.1",
            "queue_index": 0,
        },
    )

    assert response.status_code == 202
    body = response.json()
    assert body["accepted"] is True
    assert body["execution"]["execution_id"] == "exec_01"
    assert body["execution"]["state"] == "executing"
    assert body["execution"]["trace_id"] == "tr_worker_user"
    assert response.headers["X-Trace-Id"] == "tr_worker_user"


def test_internal_events_generates_trace_when_missing() -> None:
    create_response = client.post(
        "/internal/executions",
        json={
            "execution_id": "exec_02",
            "workspace_id": "ws_local",
            "conversation_id": "conv_02",
            "message_id": "msg_02",
            "mode": "agent",
            "model_id": "gpt-4.1",
        },
    )
    assert create_response.status_code == 202

    response = client.post(
        "/internal/events",
        json={
            "event_id": "evt_01",
            "execution_id": "exec_02",
            "conversation_id": "conv_02",
            "type": "execution_done",
            "sequence": 1,
            "queue_index": 0,
            "payload": {"summary": "done"},
        },
    )

    assert response.status_code == 202
    body = response.json()
    assert body["accepted"] is True
    assert body["event"]["type"] == "execution_done"
    assert response.headers["X-Trace-Id"].startswith("tr_")
    assert body["event"]["trace_id"] == response.headers["X-Trace-Id"]


def test_internal_events_returns_404_for_unknown_execution() -> None:
    response = client.post(
        "/internal/events",
        headers={"X-Trace-Id": "tr_missing_exec"},
        json={
            "event_id": "evt_missing",
            "execution_id": "exec_missing",
            "conversation_id": "conv_missing",
            "type": "execution_started",
            "sequence": 1,
            "queue_index": 0,
            "payload": {},
        },
    )

    assert response.status_code == 404
    body = response.json()
    assert body["code"] == "EXECUTION_NOT_FOUND"
    assert body["details"]["execution_id"] == "exec_missing"
    assert body["trace_id"] == "tr_missing_exec"
