from __future__ import annotations

import os
import re
from datetime import datetime, timezone
from typing import Any

import httpx

from app.services.secret_resolver import resolve_secret_via_hub

ProviderKey = str

SUPPORTED_PROVIDERS: set[ProviderKey] = {
    "deepseek",
    "minimax_cn",
    "minimax_intl",
    "zhipu",
    "qwen",
    "doubao",
    "openai",
    "anthropic",
    "google",
    "custom",
}

OPENAI_COMPATIBLE_PROVIDERS: set[ProviderKey] = {
    "deepseek",
    "minimax_cn",
    "minimax_intl",
    "zhipu",
    "qwen",
    "doubao",
    "openai",
    "custom",
}

PROVIDER_DEFAULT_BASE_URL: dict[ProviderKey, str] = {
    "deepseek": "https://api.deepseek.com",
    "minimax_cn": "https://api.minimaxi.com/v1",
    "minimax_intl": "https://api.minimax.io/v1",
    "zhipu": "https://open.bigmodel.cn/api/paas/v4",
    "qwen": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "doubao": "https://ark.cn-beijing.volces.com/api/v3",
    "openai": "https://api.openai.com/v1",
    "anthropic": "https://api.anthropic.com/v1",
    "google": "https://generativelanguage.googleapis.com/v1beta",
    "custom": "",
}

FALLBACK_MODELS: dict[ProviderKey, list[dict[str, Any]]] = {
    "deepseek": [
        {"model_id": "deepseek-chat", "display_name": "DeepSeek Chat", "is_latest": True},
        {"model_id": "deepseek-reasoner", "display_name": "DeepSeek Reasoner", "is_latest": False},
    ],
    "minimax_cn": [
        {"model_id": "MiniMax-M1", "display_name": "MiniMax M1", "is_latest": True},
        {"model_id": "abab6.5-chat", "display_name": "abab6.5 Chat", "is_latest": False},
    ],
    "minimax_intl": [
        {"model_id": "MiniMax-M1", "display_name": "MiniMax M1", "is_latest": True},
        {"model_id": "abab6.5-chat", "display_name": "abab6.5 Chat", "is_latest": False},
    ],
    "zhipu": [
        {"model_id": "glm-4-plus", "display_name": "GLM-4 Plus", "is_latest": True},
        {"model_id": "glm-4-air", "display_name": "GLM-4 Air", "is_latest": False},
    ],
    "qwen": [
        {"model_id": "qwen-plus-latest", "display_name": "Qwen Plus Latest", "is_latest": True},
        {"model_id": "qwen-turbo-latest", "display_name": "Qwen Turbo Latest", "is_latest": False},
    ],
    "doubao": [
        {"model_id": "doubao-1.5-pro-32k", "display_name": "Doubao 1.5 Pro 32k", "is_latest": True},
        {"model_id": "doubao-1.5-lite-32k", "display_name": "Doubao 1.5 Lite 32k", "is_latest": False},
    ],
    "openai": [
        {"model_id": "gpt-5", "display_name": "GPT-5", "is_latest": True},
        {"model_id": "gpt-5-mini", "display_name": "GPT-5 Mini", "is_latest": False},
        {"model_id": "gpt-4.1", "display_name": "GPT-4.1", "is_latest": False},
    ],
    "anthropic": [
        {"model_id": "claude-sonnet-4-5", "display_name": "Claude Sonnet 4.5", "is_latest": True},
        {"model_id": "claude-opus-4-1", "display_name": "Claude Opus 4.1", "is_latest": False},
    ],
    "google": [
        {"model_id": "gemini-2.5-pro", "display_name": "Gemini 2.5 Pro", "is_latest": True},
        {"model_id": "gemini-2.5-flash", "display_name": "Gemini 2.5 Flash", "is_latest": False},
    ],
    "custom": [],
}


LATEST_HINT_PATTERN = re.compile(r"(latest|\bpro\b|\b4\.1\b|\b4\.5\b|\b5\b)", flags=re.IGNORECASE)


def is_supported_provider(provider: str) -> bool:
    return provider in SUPPORTED_PROVIDERS


def _now_iso() -> str:
    return datetime.now(tz=timezone.utc).isoformat()


def _is_latest_model(model_id: str, display_name: str) -> bool:
    candidate = f"{model_id} {display_name}".lower()
    if "latest" in candidate:
        return True
    return bool(LATEST_HINT_PATTERN.search(candidate))


def _sort_items(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    return sorted(
        items,
        key=lambda item: (
            1 if item.get("is_latest") else 0,
            str(item.get("released_at") or ""),
            str(item.get("model_id") or ""),
        ),
        reverse=True,
    )


def _normalize_base_url(provider: ProviderKey, base_url: str | None) -> str:
    if base_url and base_url.strip():
        return base_url.strip().rstrip("/")
    return PROVIDER_DEFAULT_BASE_URL.get(provider, "").rstrip("/")


def _fallback_response(provider: ProviderKey, warning: str | None = None) -> dict[str, Any]:
    fallback_items = [
        {
            "model_id": item["model_id"],
            "display_name": item["display_name"],
            "provider": provider,
            "released_at": item.get("released_at"),
            "is_latest": bool(item.get("is_latest", False)),
            "source": "snapshot",
            "capabilities": ["chat"],
        }
        for item in FALLBACK_MODELS.get(provider, [])
    ]
    return {
        "provider": provider,
        "items": _sort_items(fallback_items),
        "fetched_at": _now_iso(),
        "fallback_used": True,
        "warning": warning,
    }


async def resolve_api_key(secret_ref: str, *, trace_id: str) -> str:
    if not secret_ref:
        raise RuntimeError("missing secret_ref in model config")

    if secret_ref.startswith("secret:"):
        return await resolve_secret_via_hub(secret_ref, trace_id)

    if secret_ref.startswith("env:"):
        env_key = secret_ref.split(":", 1)[1]
    elif secret_ref.startswith("keychain:"):
        parts = secret_ref.split(":")
        if len(parts) != 3:
            raise RuntimeError(f"invalid secret_ref format: {secret_ref}")
        provider, profile = parts[1], parts[2]
        env_key = f"GOYAIS_SECRET_{provider.upper()}_{profile.upper()}"
    else:
        env_key = secret_ref

    value = os.getenv(env_key)
    if not value:
        raise RuntimeError(
            f"API key not found for secret_ref '{secret_ref}'. "
            f"Set environment variable '{env_key}'."
        )
    return value


async def _fetch_openai_compatible_models(provider: ProviderKey, base_url: str, api_key: str) -> list[dict[str, Any]]:
    if not base_url:
        raise RuntimeError("base_url is required for OpenAI-compatible model list")

    async with httpx.AsyncClient(timeout=10.0) as client:
        response = await client.get(
            f"{base_url}/models",
            headers={"Authorization": f"Bearer {api_key}"},
        )
        response.raise_for_status()
        payload = response.json()

    data = payload.get("data") if isinstance(payload, dict) else None
    if not isinstance(data, list):
        return []

    items: list[dict[str, Any]] = []
    for entry in data:
        if not isinstance(entry, dict):
            continue
        model_id = str(entry.get("id") or "").strip()
        if not model_id:
            continue
        display_name = str(entry.get("display_name") or entry.get("name") or model_id)
        items.append(
            {
                "model_id": model_id,
                "display_name": display_name,
                "provider": provider,
                "released_at": entry.get("created") or entry.get("created_at"),
                "is_latest": _is_latest_model(model_id, display_name),
                "source": "live",
                "capabilities": ["chat"],
            }
        )
    return _sort_items(items)


async def _fetch_anthropic_models(provider: ProviderKey, base_url: str, api_key: str) -> list[dict[str, Any]]:
    endpoint = (base_url or PROVIDER_DEFAULT_BASE_URL["anthropic"]).rstrip("/") + "/models"
    async with httpx.AsyncClient(timeout=10.0) as client:
        response = await client.get(
            endpoint,
            headers={
                "x-api-key": api_key,
                "anthropic-version": "2023-06-01",
            },
        )
        response.raise_for_status()
        payload = response.json()

    data = payload.get("data") if isinstance(payload, dict) else None
    if not isinstance(data, list):
        return []

    items: list[dict[str, Any]] = []
    for entry in data:
        if not isinstance(entry, dict):
            continue
        model_id = str(entry.get("id") or "").strip()
        if not model_id:
            continue
        display_name = str(entry.get("display_name") or model_id)
        items.append(
            {
                "model_id": model_id,
                "display_name": display_name,
                "provider": provider,
                "released_at": entry.get("created_at"),
                "is_latest": _is_latest_model(model_id, display_name),
                "source": "live",
                "capabilities": ["chat"],
            }
        )
    return _sort_items(items)


async def _fetch_google_models(provider: ProviderKey, base_url: str, api_key: str) -> list[dict[str, Any]]:
    endpoint = (base_url or PROVIDER_DEFAULT_BASE_URL["google"]).rstrip("/") + "/models"
    async with httpx.AsyncClient(timeout=10.0) as client:
        response = await client.get(
            endpoint,
            params={"key": api_key},
        )
        response.raise_for_status()
        payload = response.json()

    data = payload.get("models") if isinstance(payload, dict) else None
    if not isinstance(data, list):
        return []

    items: list[dict[str, Any]] = []
    for entry in data:
        if not isinstance(entry, dict):
            continue
        model_name = str(entry.get("name") or "").strip()
        if not model_name:
            continue
        model_id = model_name.split("/")[-1]
        display_name = str(entry.get("displayName") or model_id)
        capabilities = entry.get("supportedGenerationMethods")
        items.append(
            {
                "model_id": model_id,
                "display_name": display_name,
                "provider": provider,
                "released_at": entry.get("version"),
                "is_latest": _is_latest_model(model_id, display_name),
                "source": "live",
                "capabilities": capabilities if isinstance(capabilities, list) else ["chat"],
            }
        )
    return _sort_items(items)


async def list_models_for_model_config(model_config: dict[str, Any], *, trace_id: str) -> dict[str, Any]:
    provider = str(model_config.get("provider") or "").strip()
    if not is_supported_provider(provider):
        return _fallback_response("custom", f"unsupported provider: {provider}")

    provider_key: ProviderKey = provider
    base_url = _normalize_base_url(provider_key, model_config.get("base_url"))

    try:
        api_key = await resolve_api_key(str(model_config.get("secret_ref") or ""), trace_id=trace_id)
    except Exception as exc:  # noqa: BLE001
        return _fallback_response(provider_key, str(exc))

    try:
        if provider_key in OPENAI_COMPATIBLE_PROVIDERS:
            live_items = await _fetch_openai_compatible_models(provider_key, base_url, api_key)
        elif provider_key == "anthropic":
            live_items = await _fetch_anthropic_models(provider_key, base_url, api_key)
        elif provider_key == "google":
            live_items = await _fetch_google_models(provider_key, base_url, api_key)
        else:
            live_items = []

        if not live_items:
            return _fallback_response(provider_key, "provider returned empty model list")

        return {
            "provider": provider_key,
            "items": live_items,
            "fetched_at": _now_iso(),
            "fallback_used": False,
        }
    except Exception as exc:  # noqa: BLE001
        return _fallback_response(provider_key, str(exc))
