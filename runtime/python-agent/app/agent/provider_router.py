from __future__ import annotations

from app.agent.providers.anthropic_provider import AnthropicProvider
from app.agent.providers.base import ProviderAdapter
from app.agent.providers.openai_provider import OpenAIProvider

OPENAI_COMPATIBLE_PROVIDERS = {
    "deepseek",
    "minimax_cn",
    "minimax_intl",
    "zhipu",
    "qwen",
    "doubao",
    "openai",
    "custom",
}


def build_provider(provider: str, api_key: str, base_url: str | None = None) -> ProviderAdapter:
    if provider in OPENAI_COMPATIBLE_PROVIDERS:
        return OpenAIProvider(api_key=api_key, base_url=base_url)
    if provider == "anthropic":
        return AnthropicProvider(api_key=api_key)
    if provider == "google":
        raise ValueError("Unsupported provider for execution: google")
    raise ValueError(f"Unsupported provider: {provider}")
