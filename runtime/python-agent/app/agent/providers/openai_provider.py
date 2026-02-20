from __future__ import annotations

from openai import AsyncOpenAI

from app.agent.providers.base import ProviderAdapter, ProviderRequest


class OpenAIProvider(ProviderAdapter):
    def __init__(self, api_key: str, base_url: str | None = None):
        self.client = AsyncOpenAI(api_key=api_key, base_url=base_url)

    async def complete(self, request: ProviderRequest) -> str:
        payload = {
            "model": request.model,
            "input": request.input_text,
        }
        if request.system_prompt:
            payload["instructions"] = request.system_prompt
        if request.max_tokens is not None:
            payload["max_output_tokens"] = request.max_tokens
        if request.temperature is not None:
            payload["temperature"] = request.temperature

        response = await self.client.responses.create(**payload)
        return response.output_text
