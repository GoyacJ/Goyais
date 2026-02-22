from fastapi.testclient import TestClient

from app.main import app


client = TestClient(app)


def test_internal_executions_returns_standard_error_with_trace_consistency() -> None:
    response = client.post("/internal/executions", headers={"X-Trace-Id": "tr_worker_user"})

    assert response.status_code == 501
    body = response.json()
    assert body["code"] == "INTERNAL_NOT_IMPLEMENTED"
    assert body["details"]["method"] == "POST"
    assert body["details"]["path"] == "/internal/executions"
    assert response.headers["X-Trace-Id"] == "tr_worker_user"
    assert body["trace_id"] == response.headers["X-Trace-Id"]


def test_internal_events_generates_trace_when_missing() -> None:
    response = client.post("/internal/events")

    assert response.status_code == 501
    body = response.json()
    assert body["code"] == "INTERNAL_NOT_IMPLEMENTED"
    assert body["details"]["path"] == "/internal/events"
    assert response.headers["X-Trace-Id"].startswith("tr_")
    assert body["trace_id"] == response.headers["X-Trace-Id"]
