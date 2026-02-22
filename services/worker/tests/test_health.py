from fastapi.testclient import TestClient

from app.main import app


client = TestClient(app)


def test_health() -> None:
    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"ok": True, "version": "0.4.0"}
    assert response.headers.get("X-Trace-Id", "").startswith("tr_")
