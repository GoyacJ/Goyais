from __future__ import annotations

import uuid
from urllib.parse import urlparse

from fastapi import APIRouter, Depends, Header, Request

from app.deps import get_repo
from app.errors import GoyaisApiError
from app.services.model_catalog_service import is_supported_provider, list_models_for_model_config

router = APIRouter(prefix="/v1", tags=["model-configs"])


def _validate_provider(value: str) -> str:
    provider = value.strip()
    if not provider:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="provider is required",
            retryable=False,
            status_code=400,
            cause="provider_missing",
        )
    if not is_supported_provider(provider):
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message=f"unsupported provider: {provider}",
            retryable=False,
            status_code=400,
            cause="provider_unsupported",
        )
    return provider


def _validate_model(value: str) -> str:
    model = value.strip()
    if not model:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="model is required",
            retryable=False,
            status_code=400,
            cause="model_missing",
        )
    return model


def _validate_base_url(value: str | None) -> str | None:
    if value is None:
        return None
    normalized = str(value).strip()
    if not normalized:
        return None
    parsed = urlparse(normalized)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="base_url must be a valid http/https URL",
            retryable=False,
            status_code=400,
            cause="base_url_invalid",
        )
    return normalized.rstrip("/")


def _validate_temperature(value: object | None, default: float) -> float:
    if value is None:
        return default
    try:
        parsed = float(value)
    except Exception as exc:  # noqa: BLE001
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="temperature must be a number",
            retryable=False,
            status_code=400,
            cause="temperature_invalid",
        ) from exc
    if parsed < 0 or parsed > 2:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="temperature must be between 0 and 2",
            retryable=False,
            status_code=400,
            cause="temperature_out_of_range",
        )
    return parsed


def _validate_max_tokens(value: object | None) -> int | None:
    if value is None:
        return None
    if isinstance(value, str) and not value.strip():
        return None

    try:
        parsed = int(value)
    except Exception as exc:  # noqa: BLE001
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="max_tokens must be an integer",
            retryable=False,
            status_code=400,
            cause="max_tokens_invalid",
        ) from exc

    if parsed <= 0:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="max_tokens must be positive",
            retryable=False,
            status_code=400,
            cause="max_tokens_not_positive",
        )
    return parsed


def _validate_secret_ref(value: str) -> str:
    secret_ref = value.strip()
    if not secret_ref:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="secret_ref is required",
            retryable=False,
            status_code=400,
            cause="secret_ref_missing",
        )
    return secret_ref


@router.get("/model-configs")
async def list_model_configs(repo=Depends(get_repo)):
    return {"model_configs": await repo.list_model_configs()}


@router.post("/model-configs")
async def create_model_config(payload: dict, repo=Depends(get_repo)):
    model_config_id = str(payload.get("model_config_id") or uuid.uuid4())
    provider = _validate_provider(str(payload.get("provider") or ""))
    model = _validate_model(str(payload.get("model") or ""))
    base_url = _validate_base_url(payload.get("base_url"))
    temperature = _validate_temperature(payload.get("temperature"), default=0)
    max_tokens = _validate_max_tokens(payload.get("max_tokens"))
    secret_ref = _validate_secret_ref(str(payload.get("secret_ref") or ""))

    await repo.upsert_model_config(
        {
            "model_config_id": model_config_id,
            "provider": provider,
            "model": model,
            "base_url": base_url,
            "temperature": temperature,
            "max_tokens": max_tokens,
            "secret_ref": secret_ref,
        }
    )
    event_id = await repo.insert_system_event(
        "model_config_upserted",
        {
            "entity": "model_config",
            "model_config_id": model_config_id,
            "provider": provider,
            "model": model,
            "secret_ref": secret_ref,
        },
    )
    model_config = await repo.get_model_config(model_config_id)
    return {"model_config": model_config, "event_id": event_id}


@router.patch("/model-configs/{model_config_id}")
async def update_model_config(model_config_id: str, payload: dict, repo=Depends(get_repo)):
    existing = await repo.get_model_config(model_config_id)
    if existing is None:
        raise GoyaisApiError(
            code="E_NOT_FOUND",
            message="model config not found",
            retryable=False,
            status_code=404,
            cause="model_config_not_found",
        )

    allowed_keys = {"provider", "model", "base_url", "temperature", "max_tokens", "secret_ref"}
    has_any_key = any(key in payload for key in allowed_keys)
    if not has_any_key:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="at least one updatable field is required",
            retryable=False,
            status_code=400,
            cause="model_config_patch_empty",
        )

    provider = _validate_provider(str(payload.get("provider") or existing["provider"]))
    model = _validate_model(str(payload.get("model") or existing["model"]))
    base_url = _validate_base_url(payload.get("base_url", existing.get("base_url")))
    temperature = _validate_temperature(payload.get("temperature"), default=float(existing.get("temperature") or 0))
    max_tokens = _validate_max_tokens(payload.get("max_tokens", existing.get("max_tokens")))
    secret_ref = _validate_secret_ref(str(payload.get("secret_ref") or existing.get("secret_ref") or ""))

    await repo.upsert_model_config(
        {
            "model_config_id": model_config_id,
            "provider": provider,
            "model": model,
            "base_url": base_url,
            "temperature": temperature,
            "max_tokens": max_tokens,
            "secret_ref": secret_ref,
        }
    )
    event_id = await repo.insert_system_event(
        "model_config_upserted",
        {
            "entity": "model_config",
            "model_config_id": model_config_id,
            "provider": provider,
            "model": model,
            "secret_ref": secret_ref,
        },
    )
    model_config = await repo.get_model_config(model_config_id)
    return {"model_config": model_config, "event_id": event_id}


@router.delete("/model-configs/{model_config_id}")
async def delete_model_config(model_config_id: str, repo=Depends(get_repo)):
    deleted = await repo.delete_model_config(model_config_id)
    if not deleted:
        raise GoyaisApiError(
            code="E_NOT_FOUND",
            message="model config not found",
            retryable=False,
            status_code=404,
            cause="model_config_not_found",
        )

    event_id = await repo.insert_system_event(
        "model_config_deleted",
        {
            "entity": "model_config",
            "model_config_id": model_config_id,
        },
    )
    return {"ok": True, "event_id": event_id}


@router.get("/model-configs/{model_config_id}/models")
async def list_provider_models(
    model_config_id: str,
    request: Request,
    x_api_key_override: str = Header(default=""),
    repo=Depends(get_repo),
):
    model_config = await repo.get_model_config(model_config_id)
    if model_config is None:
        raise GoyaisApiError(
            code="E_NOT_FOUND",
            message="model config not found",
            retryable=False,
            status_code=404,
            cause="model_config_not_found",
        )

    trace_id = str(getattr(request.state, "trace_id", ""))
    api_key_override = x_api_key_override.strip() or None
    return await list_models_for_model_config(
        model_config,
        trace_id=trace_id,
        api_key_override=api_key_override,
    )
