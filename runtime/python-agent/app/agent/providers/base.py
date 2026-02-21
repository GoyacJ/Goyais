"""Provider base types: ChatRequest / ChatResponse / ProviderAdapter."""
from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import AsyncIterator

from app.agent.messages import ChatResponse, Message


@dataclass(slots=True)
class ToolSchema:
    name: str
    description: str
    input_schema: dict


@dataclass(slots=True)
class ChatRequest:
    model: str
    system_prompt: str
    messages: list[Message]
    tools: list[ToolSchema] = field(default_factory=list)
    max_tokens: int = 4096
    temperature: float = 0.0


class ProviderAdapter(ABC):
    @abstractmethod
    async def chat(self, request: ChatRequest) -> ChatResponse: ...

    async def chat_stream(self, request: ChatRequest) -> AsyncIterator[dict]:  # pragma: no cover
        raise NotImplementedError("streaming not implemented for this provider")
        yield  # noqa: RET503 — make it an async generator
