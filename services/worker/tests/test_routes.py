from fastapi.testclient import TestClient

from app.main import app
from app.version import DEFAULT_RUNTIME_VERSION

client = TestClient(app)


def test_health_route(monkeypatch) -> None:
    monkeypatch.delenv("GOYAIS_VERSION", raising=False)
    response = client.get("/health")
    assert response.status_code == 200
    payload = response.json()
    assert payload["ok"] is True
    assert payload["version"] == DEFAULT_RUNTIME_VERSION


def test_health_route_uses_environment_version(monkeypatch) -> None:
    monkeypatch.setenv("GOYAIS_VERSION", "0.6.0")
    response = client.get("/health")
    assert response.status_code == 200
    payload = response.json()
    assert payload["ok"] is True
    assert payload["version"] == "0.6.0"


def test_legacy_internal_routes_removed() -> None:
    response = client.post("/internal/executions", json={})
    assert response.status_code == 404
