from __future__ import annotations

import asyncio
import json
import os
import time
from typing import Any
import urllib.error
import urllib.request

from app.internal_api import DEFAULT_INTERNAL_TOKEN, INTERNAL_TOKEN_HEADER
from app.trace import TRACE_HEADER


class HubClient:
    def __init__(self) -> None:
        self.base_url = os.getenv("HUB_BASE_URL", "http://127.0.0.1:8787").strip().rstrip("/")
        self.internal_token = os.getenv("HUB_INTERNAL_TOKEN", DEFAULT_INTERNAL_TOKEN).strip()
        self.timeout_seconds = 8

    async def register_worker(self, worker_id: str, capabilities: dict[str, Any]) -> dict[str, Any]:
        return await self._request(
            "POST",
            "/internal/workers/register",
            {"worker_id": worker_id, "capabilities": capabilities},
        )

    async def heartbeat(self, worker_id: str, status: str) -> dict[str, Any]:
        return await self._request(
            "POST",
            f"/internal/workers/{worker_id}/heartbeat",
            {"status": status},
        )

    async def claim_execution(self, worker_id: str, lease_seconds: int) -> dict[str, Any]:
        return await self._request(
            "POST",
            "/internal/executions/claim",
            {"worker_id": worker_id, "lease_seconds": lease_seconds},
        )

    async def send_events_batch(self, execution_id: str, events: list[dict[str, Any]]) -> dict[str, Any]:
        return await self._request(
            "POST",
            f"/internal/executions/{execution_id}/events/batch",
            {"events": events},
        )

    async def poll_control(
        self, execution_id: str, after_seq: int, wait_ms: int
    ) -> dict[str, Any]:
        return await self._request(
            "GET",
            f"/internal/executions/{execution_id}/control?after_seq={after_seq}&wait_ms={wait_ms}",
            None,
        )

    async def _request(
        self, method: str, path: str, payload: dict[str, Any] | None
    ) -> dict[str, Any]:
        return await asyncio.to_thread(self._request_sync, method, path, payload)

    def _request_sync(
        self, method: str, path: str, payload: dict[str, Any] | None
    ) -> dict[str, Any]:
        body: bytes | None = None
        if payload is not None:
            body = json.dumps(payload).encode("utf-8")

        request = urllib.request.Request(
            f"{self.base_url}{path}",
            data=body,
            method=method,
            headers={
                "Content-Type": "application/json",
                INTERNAL_TOKEN_HEADER: self.internal_token,
                TRACE_HEADER: f"tr_worker_{int(time.time() * 1000)}",
            },
        )
        try:
            with urllib.request.urlopen(request, timeout=self.timeout_seconds) as response:
                raw = response.read()
        except urllib.error.HTTPError as exc:
            body_text = exc.read().decode("utf-8", errors="ignore")
            raise RuntimeError(f"hub http error status={exc.code} body={body_text[:300]}") from exc
        except urllib.error.URLError as exc:
            raise RuntimeError(f"hub network error: {exc.reason}") from exc

        if not raw:
            return {}
        parsed = json.loads(raw.decode("utf-8"))
        if not isinstance(parsed, dict):
            raise RuntimeError("hub response must be a JSON object")
        return parsed
