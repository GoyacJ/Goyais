from fastapi.testclient import TestClient

from app.main import app
from app.version import DEFAULT_RUNTIME_VERSION


client = TestClient(app)


def test_health(monkeypatch) -> None:
    monkeypatch.delenv("GOYAIS_VERSION", raising=False)
    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"ok": True, "version": DEFAULT_RUNTIME_VERSION}
    assert response.headers.get("X-Trace-Id", "").startswith("tr_")


def test_health_uses_environment_version(monkeypatch) -> None:
    monkeypatch.setenv("GOYAIS_VERSION", "v0.5.1")
    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"ok": True, "version": "0.5.1"}
