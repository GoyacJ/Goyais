from __future__ import annotations

import asyncio
from typing import Dict, Tuple

from app.db.repositories import Repository
from app.errors import GoyaisApiError


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

    async def resolve(self, run_id: str, call_id: str, approved: bool, *, decided_by: str = "user") -> None:
        status = "approved" if approved else "denied"
        updated = await self.repo.resolve_pending_tool_confirmation(run_id, call_id, status, decided_by=decided_by)
        if not updated:
            existing_status = await self.repo.get_tool_confirmation_status(run_id, call_id)
            if existing_status in {"approved", "denied"}:
                raise GoyaisApiError(
                    code="E_CONFIRMATION_ALREADY_DECIDED",
                    message="Tool confirmation has already been decided.",
                    retryable=False,
                    status_code=409,
                    details={
                        "run_id": run_id,
                        "call_id": call_id,
                        "status": existing_status,
                    },
                    cause="confirmation_conflict",
                )
            raise GoyaisApiError(
                code="E_SCHEMA_INVALID",
                message="Tool confirmation request is invalid.",
                retryable=False,
                status_code=400,
                details={
                    "run_id": run_id,
                    "call_id": call_id,
                },
                cause="confirmation_missing",
            )
        key = (run_id, call_id)
        waiter = self._waiters.get(key)
        if waiter and not waiter.done():
            waiter.set_result(approved)
