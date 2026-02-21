def test_secret_endpoint_returns_goyais_error_shape(isolated_client):
    response = isolated_client.get("/v1/secrets/openai/default")

    assert response.status_code == 401
    payload = response.json()
    assert "error" in payload
    error = payload["error"]
    assert isinstance(error.get("code"), str)
    assert isinstance(error.get("message"), str)
    assert isinstance(error.get("trace_id"), str)
    assert isinstance(error.get("retryable"), bool)
    assert response.headers.get("X-Trace-Id") == error["trace_id"]
