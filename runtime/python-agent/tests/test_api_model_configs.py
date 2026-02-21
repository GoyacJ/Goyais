from __future__ import annotations

import os
import uuid

from fastapi.testclient import TestClient

from app.main import app


def test_model_config_patch_and_delete_flow():
    model_config_id = f"mc-{uuid.uuid4().hex[:8]}"
    with TestClient(app) as client:
        create_resp = client.post(
            "/v1/model-configs",
            json={
                "model_config_id": model_config_id,
                "provider": "openai",
                "model": "gpt-4.1-mini",
                "secret_ref": "env:UNIT_TEST_MODEL_KEY",
                "temperature": 0.2,
            },
        )
        assert create_resp.status_code == 200
        assert create_resp.json()["model_config"]["provider"] == "openai"

        patch_resp = client.patch(
            f"/v1/model-configs/{model_config_id}",
            json={
                "provider": "deepseek",
                "model": "deepseek-chat",
                "temperature": 1.2,
                "max_tokens": 2048,
            },
        )
        assert patch_resp.status_code == 200
        patched = patch_resp.json()["model_config"]
        assert patched["provider"] == "deepseek"
        assert patched["model"] == "deepseek-chat"
        assert patched["max_tokens"] == 2048

        delete_resp = client.delete(f"/v1/model-configs/{model_config_id}")
        assert delete_resp.status_code == 200
        assert delete_resp.json()["ok"] is True

        list_resp = client.get("/v1/model-configs")
        assert list_resp.status_code == 200
        ids = [item["model_config_id"] for item in list_resp.json()["model_configs"]]
        assert model_config_id not in ids


def test_model_config_validation_errors():
    with TestClient(app) as client:
        invalid_provider = client.post(
            "/v1/model-configs",
            json={
                "provider": "unsupported_provider",
                "model": "foo",
                "secret_ref": "env:UNIT_TEST_MODEL_KEY",
            },
        )
        assert invalid_provider.status_code == 400

        invalid_temperature = client.post(
            "/v1/model-configs",
            json={
                "provider": "openai",
                "model": "gpt",
                "secret_ref": "env:UNIT_TEST_MODEL_KEY",
                "temperature": 9,
            },
        )
        assert invalid_temperature.status_code == 400

        missing_fields = client.patch(f"/v1/model-configs/{uuid.uuid4().hex[:8]}", json={})
        assert missing_fields.status_code in {400, 404}


def test_model_catalog_fallback_when_live_fetch_fails():
    previous = os.environ.get("UNIT_TEST_MODEL_KEY")
    os.environ["UNIT_TEST_MODEL_KEY"] = "dummy-key"

    model_config_id = f"mc-catalog-{uuid.uuid4().hex[:8]}"
    try:
        with TestClient(app) as client:
            create_resp = client.post(
                "/v1/model-configs",
                json={
                    "model_config_id": model_config_id,
                    "provider": "openai",
                    "model": "gpt-5-mini",
                    "base_url": "http://127.0.0.1:9/v1",
                    "secret_ref": "env:UNIT_TEST_MODEL_KEY",
                },
            )
            assert create_resp.status_code == 200

            catalog_resp = client.get(f"/v1/model-configs/{model_config_id}/models")
            assert catalog_resp.status_code == 200
            payload = catalog_resp.json()
            assert payload["provider"] == "openai"
            assert payload["fallback_used"] is True
            assert len(payload["items"]) > 0
    finally:
        if previous is None:
            os.environ.pop("UNIT_TEST_MODEL_KEY", None)
        else:
            os.environ["UNIT_TEST_MODEL_KEY"] = previous
