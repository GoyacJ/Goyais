from __future__ import annotations

from dataclasses import dataclass


@dataclass(slots=True)
class ProviderRequest:
    model: str
    input_text: str
    system_prompt: str | None = None
    max_tokens: int | None = None
    temperature: float | None = None


class ProviderAdapter:
    async def complete(self, request: ProviderRequest) -> str:  # pragma: no cover - interface
        raise NotImplementedError
