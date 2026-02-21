"""
hub_reporter.py — 向 Hub 批量上报执行事件

Worker 产出事件后调用此模块，替代原来的本地 EventBus 发布。
"""
from __future__ import annotations

import asyncio
import logging
from collections import deque
from datetime import datetime, timezone
from typing import Any

import httpx

logger = logging.getLogger(__name__)

_RING_BUFFER_MAX = 1000
_BATCH_INTERVAL_MS = 100  # ms between flush cycles


class HubReporter:
    """
    上报执行事件到 Hub 的 /internal/executions/{id}/events 接口。

    - 维护 ring buffer（最多 1000 条）用于失败重试
    - 每 100ms flush 一次，或 buffer 满时立即 flush
    - Worker 崩溃前应调用 flush_all() 确保上报完成
    """

    def __init__(
        self,
        hub_base_url: str,
        hub_internal_secret: str,
        execution_id: str,
    ) -> None:
        self._hub_url = hub_base_url.rstrip("/")
        self._secret = hub_internal_secret
        self._execution_id = execution_id
        self._buffer: deque[dict[str, Any]] = deque(maxlen=_RING_BUFFER_MAX)
        self._seq = 0
        self._lock = asyncio.Lock()
        self._flush_task: asyncio.Task | None = None

    def start(self) -> None:
        """Start background flush loop."""
        self._flush_task = asyncio.create_task(self._flush_loop())

    async def stop(self) -> None:
        """Stop flush loop and drain remaining buffer."""
        if self._flush_task:
            self._flush_task.cancel()
            try:
                await self._flush_task
            except asyncio.CancelledError:
                pass
        await self.flush_all()

    async def report(self, event_type: str, payload: dict[str, Any]) -> dict[str, Any]:
        """Enqueue one event for reporting. Returns the event dict."""
        async with self._lock:
            self._seq += 1
            seq = self._seq

        event = {
            "seq": seq,
            "ts": datetime.now(tz=timezone.utc).isoformat(),
            "type": event_type,
            "payload": payload,
        }
        self._buffer.append(event)

        # Flush immediately if buffer is large
        if len(self._buffer) >= 50:
            await self._do_flush()

        return event

    async def flush_all(self) -> None:
        """Drain the entire buffer, retrying once on failure."""
        for _ in range(2):
            if not self._buffer:
                return
            await self._do_flush()
            await asyncio.sleep(0.05)

    async def _flush_loop(self) -> None:
        while True:
            await asyncio.sleep(_BATCH_INTERVAL_MS / 1000)
            if self._buffer:
                await self._do_flush()

    async def _do_flush(self) -> None:
        if not self._buffer:
            return

        # Drain buffer atomically
        batch = list(self._buffer)
        self._buffer.clear()

        url = f"{self._hub_url}/internal/executions/{self._execution_id}/events"
        headers = {
            "X-Hub-Auth": self._secret,
            "Content-Type": "application/json",
        }
        try:
            async with httpx.AsyncClient(timeout=10.0) as client:
                resp = await client.post(url, json={"events": batch}, headers=headers)
                if resp.status_code >= 400:
                    logger.warning(
                        "hub_reporter: flush failed status=%d execution_id=%s",
                        resp.status_code,
                        self._execution_id,
                    )
                    # Re-enqueue on failure (ring buffer will drop oldest if full)
                    for ev in batch:
                        self._buffer.appendleft(ev)
        except Exception as exc:  # noqa: BLE001
            logger.warning(
                "hub_reporter: flush error execution_id=%s error=%s",
                self._execution_id,
                exc,
            )
            for ev in batch:
                self._buffer.appendleft(ev)
