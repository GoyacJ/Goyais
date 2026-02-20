from __future__ import annotations

import asyncio
from collections import defaultdict
from typing import AsyncIterator


class EventBus:
    def __init__(self) -> None:
        self._subscribers: dict[str, set[asyncio.Queue[dict]]] = defaultdict(set)

    async def publish(self, run_id: str, event: dict) -> None:
        for queue in list(self._subscribers.get(run_id, set())):
            await queue.put(event)

    async def subscribe(self, run_id: str) -> AsyncIterator[dict]:
        queue: asyncio.Queue[dict] = asyncio.Queue()
        self._subscribers[run_id].add(queue)
        try:
            while True:
                event = await queue.get()
                yield event
                if event.get("type") == "done":
                    break
        finally:
            self._subscribers[run_id].discard(queue)
