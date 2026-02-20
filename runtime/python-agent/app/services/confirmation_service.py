from __future__ import annotations

import asyncio
from typing import Dict, Tuple

from app.db.repositories import Repository


class ConfirmationService:
    def __init__(self, repo: Repository):
        self.repo = repo
        self._waiters: Dict[Tuple[str, str], asyncio.Future[bool]] = {}

    async def wait_for(self, run_id: str, call_id: str, timeout_seconds: int = 600) -> bool:
        existing_status = await self.repo.get_tool_confirmation_status(run_id, call_id)
        if existing_status == "approved":
            return True
        if existing_status == "denied":
            return False

        key = (run_id, call_id)
        loop = asyncio.get_running_loop()
        fut = loop.create_future()
        self._waiters[key] = fut
        try:
            return await asyncio.wait_for(fut, timeout=timeout_seconds)
        finally:
            self._waiters.pop(key, None)

    async def resolve(self, run_id: str, call_id: str, approved: bool) -> None:
        existing_status = await self.repo.get_tool_confirmation_status(run_id, call_id)
        if existing_status in {"approved", "denied"}:
            return

        status = "approved" if approved else "denied"
        await self.repo.upsert_tool_confirmation_status(run_id, call_id, status)
        key = (run_id, call_id)
        waiter = self._waiters.get(key)
        if waiter and not waiter.done():
            waiter.set_result(approved)
