from __future__ import annotations

from app.agent.providers.anthropic_provider import AnthropicProvider
from app.agent.providers.base import ProviderAdapter
from app.agent.providers.openai_provider import OpenAIProvider


def build_provider(provider: str, api_key: str, base_url: str | None = None) -> ProviderAdapter:
    if provider == "openai":
        return OpenAIProvider(api_key=api_key, base_url=base_url)
    if provider == "anthropic":
        return AnthropicProvider(api_key=api_key)
    raise ValueError(f"Unsupported provider: {provider}")
