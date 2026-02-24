from fastapi.testclient import TestClient

from app.main import app


client = TestClient(app)
INTERNAL_TOKEN = "goyais-internal-token"
AUTH_HEADERS = {"X-Internal-Token": INTERNAL_TOKEN}


def test_internal_executions_accepts_request_and_keeps_trace_consistency() -> None:
    response = client.post(
        "/internal/executions",
        headers={"X-Trace-Id": "tr_worker_user", **AUTH_HEADERS},
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
    assert body["execution"]["state"] == "pending"
    assert body["execution"]["trace_id"] == "tr_worker_user"
    assert response.headers["X-Trace-Id"] == "tr_worker_user"


def test_internal_events_generates_trace_when_missing() -> None:
    create_response = client.post(
        "/internal/executions",
        headers=AUTH_HEADERS,
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
        headers=AUTH_HEADERS,
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
        headers={"X-Trace-Id": "tr_missing_exec", **AUTH_HEADERS},
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


def test_internal_execution_confirm_and_stop_routes() -> None:
    create_response = client.post(
        "/internal/executions",
        headers=AUTH_HEADERS,
        json={
            "execution_id": "exec_ctrl",
            "workspace_id": "ws_local",
            "conversation_id": "conv_ctrl",
            "message_id": "msg_ctrl",
            "mode": "agent",
            "model_id": "gpt-4.1",
        },
    )
    assert create_response.status_code == 202

    confirm_response = client.post(
        "/internal/executions/exec_ctrl/confirm",
        headers=AUTH_HEADERS,
        json={"decision": "approve"},
    )
    assert confirm_response.status_code == 202
    confirm_body = confirm_response.json()
    assert confirm_body["accepted"] is True
    assert confirm_body["decision"] == "approve"

    stop_response = client.post(
        "/internal/executions/exec_ctrl/stop",
        headers=AUTH_HEADERS,
        json={},
    )
    assert stop_response.status_code == 202
    stop_body = stop_response.json()
    assert stop_body["accepted"] is True
    assert stop_body["execution_id"] == "exec_ctrl"


def test_internal_endpoints_require_internal_token() -> None:
    execution_response = client.post(
        "/internal/executions",
        json={
            "execution_id": "exec_unauth",
            "workspace_id": "ws_local",
            "conversation_id": "conv_unauth",
            "message_id": "msg_unauth",
            "mode": "agent",
            "model_id": "gpt-4.1",
        },
    )

    assert execution_response.status_code == 401
    execution_body = execution_response.json()
    assert execution_body["code"] == "AUTH_INTERNAL_TOKEN_REQUIRED"

    event_response = client.post(
        "/internal/events",
        headers={"Authorization": "Bearer wrong-token"},
        json={
            "event_id": "evt_unauth",
            "execution_id": "exec_02",
            "conversation_id": "conv_02",
            "type": "execution_done",
            "sequence": 1,
            "queue_index": 0,
        },
    )
    assert event_response.status_code == 401
    event_body = event_response.json()
    assert event_body["code"] == "AUTH_INVALID_INTERNAL_TOKEN"
