from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def test_health_route() -> None:
    response = client.get("/health")
    assert response.status_code == 200
    payload = response.json()
    assert payload["ok"] is True
    assert payload["version"] == "0.4.0"


def test_legacy_internal_routes_removed() -> None:
    response = client.post("/internal/executions", json={})
    assert response.status_code == 404

