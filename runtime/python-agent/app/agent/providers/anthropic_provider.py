from __future__ import annotations

from anthropic import AsyncAnthropic

from app.agent.providers.base import ProviderAdapter, ProviderRequest


class AnthropicProvider(ProviderAdapter):
    def __init__(self, api_key: str):
        self.client = AsyncAnthropic(api_key=api_key)

    async def complete(self, request: ProviderRequest) -> str:
        payload = {
            "model": request.model,
            "max_tokens": request.max_tokens or 1024,
            "messages": [{"role": "user", "content": request.input_text}],
        }
        if request.system_prompt:
            payload["system"] = request.system_prompt

        response = await self.client.messages.create(**payload)
        return "\n".join(block.text for block in response.content if getattr(block, "type", "") == "text")
