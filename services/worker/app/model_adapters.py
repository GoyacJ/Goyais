from __future__ import annotations

from dataclasses import dataclass, field
import os
from typing import Any, Mapping


DEFAULT_TIMEOUT_MS = 30_000
DEFAULT_BASE_URLS = {
    "openai": "https://api.openai.com/v1",
    "google": "https://generativelanguage.googleapis.com/v1beta",
    "qwen": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "doubao": "https://ark.cn-beijing.volces.com/api/v3",
    "zhipu": "https://open.bigmodel.cn/api/paas/v4",
    "minimax": "https://api.minimax.chat/v1",
    "local": "http://127.0.0.1:11434/v1",
}
API_KEY_ENV_BY_VENDOR = {
    "openai": "OPENAI_API_KEY",
    "google": "GOOGLE_API_KEY",
    "qwen": "QWEN_API_KEY",
    "doubao": "DOUBAO_API_KEY",
    "zhipu": "ZHIPU_API_KEY",
    "minimax": "MINIMAX_API_KEY",
    "local": "",
}
SUPPORTED_PARAM_KEYS = {
    "temperature",
    "top_p",
    "max_tokens",
    "presence_penalty",
    "frequency_penalty",
}


@dataclass(slots=True)
class ModelInvocation:
    vendor: str
    model_id: str
    base_url: str
    api_key: str
    timeout_ms: int
    params: dict[str, Any] = field(default_factory=dict)


@dataclass(slots=True)
class ToolCall:
    id: str
    name: str
    arguments: dict[str, Any]


@dataclass(slots=True)
class ModelTurnResult:
    text: str
    tool_calls: list[ToolCall]
    raw_response: dict[str, Any]


class ModelAdapterError(RuntimeError):
    def __init__(self, code: str, message: str, details: dict[str, Any] | None = None) -> None:
        super().__init__(message)
        self.code = code
        self.details = details or {}


def resolve_model_invocation(
    execution: Mapping[str, Any], env: Mapping[str, str] | None = None
) -> ModelInvocation:
    environ = os.environ if env is None else env
    snapshot = execution.get("model_snapshot")
    snapshot_map = snapshot if isinstance(snapshot, dict) else {}
    params = snapshot_map.get("params")
    params_map = params if isinstance(params, dict) else {}

    model_id = str(snapshot_map.get("model_id") or execution.get("model_id") or "").strip()
    if model_id == "":
        raise ModelAdapterError("MODEL_ID_REQUIRED", "model_id is required for model invocation")

    raw_vendor = (
        snapshot_map.get("vendor")
        or params_map.get("vendor")
        or execution.get("vendor")
        or _infer_vendor_from_model_id(model_id)
    )
    vendor = _normalize_vendor(raw_vendor)

    base_url = str(
        snapshot_map.get("base_url")
        or params_map.get("base_url")
        or DEFAULT_BASE_URLS.get(vendor, DEFAULT_BASE_URLS["openai"])
    ).strip()
    if base_url == "":
        raise ModelAdapterError("MODEL_BASE_URL_REQUIRED", "base_url is required for model invocation")

    timeout_ms = _resolve_timeout_ms(snapshot_map, params_map, environ)
    api_key = _resolve_api_key(vendor, params_map, environ)
    if vendor != "local" and api_key == "":
        raise ModelAdapterError(
            "MODEL_API_KEY_MISSING",
            f"api_key is required for vendor={vendor}",
            {"vendor": vendor},
        )

    return ModelInvocation(
        vendor=vendor,
        model_id=model_id,
        base_url=base_url.rstrip("/"),
        api_key=api_key,
        timeout_ms=timeout_ms,
        params={k: v for k, v in params_map.items() if k != "api_key"},
    )


async def run_model_turn(
    invocation: ModelInvocation, messages: list[dict[str, Any]], tools: list[dict[str, Any]]
) -> ModelTurnResult:
    if invocation.vendor == "google":
        from app.model_turns import run_google_turn

        return await run_google_turn(invocation, messages, tools)
    from app.model_turns import run_openai_compatible_turn

    return await run_openai_compatible_turn(invocation, messages, tools)


def _resolve_api_key(vendor: str, params: Mapping[str, Any], env: Mapping[str, str]) -> str:
    from_params = str(params.get("api_key") or "").strip()
    if from_params:
        return from_params

    env_key = API_KEY_ENV_BY_VENDOR.get(vendor, "")
    if env_key:
        from_vendor_env = str(env.get(env_key, "")).strip()
        if from_vendor_env:
            return from_vendor_env
    return str(env.get("MODEL_API_KEY", "")).strip()


def _resolve_timeout_ms(
    snapshot: Mapping[str, Any], params: Mapping[str, Any], env: Mapping[str, str]
) -> int:
    raw_timeout = snapshot.get("timeout_ms")
    if raw_timeout is None:
        raw_timeout = params.get("timeout_ms")
    if raw_timeout is None:
        raw_timeout = env.get("WORKER_MODEL_TIMEOUT_MS")
    try:
        timeout_ms = int(raw_timeout)
    except (TypeError, ValueError):
        timeout_ms = DEFAULT_TIMEOUT_MS
    return min(max(timeout_ms, 1_000), 120_000)


def _infer_vendor_from_model_id(model_id: str) -> str:
    normalized = model_id.lower()
    if normalized.startswith("gemini"):
        return "google"
    if "qwen" in normalized:
        return "qwen"
    if "doubao" in normalized or "ark" in normalized:
        return "doubao"
    if normalized.startswith("glm") or "zhipu" in normalized:
        return "zhipu"
    if "minimax" in normalized:
        return "minimax"
    if ":" in normalized:
        return "local"
    return "openai"


def _normalize_vendor(raw_vendor: Any) -> str:
    normalized = str(raw_vendor or "").strip().lower()
    mapping = {
        "openai": "openai",
        "google": "google",
        "qwen": "qwen",
        "doubao": "doubao",
        "zhipu": "zhipu",
        "minimax": "minimax",
        "local": "local",
    }
    return mapping.get(normalized, _infer_vendor_from_model_id(normalized or "gpt-4.1"))
