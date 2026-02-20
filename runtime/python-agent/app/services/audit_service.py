from __future__ import annotations

import uuid
from typing import Any

from app.db.repositories import Repository


class AuditService:
    def __init__(self, repo: Repository):
        self.repo = repo

    async def record(
        self,
        *,
        trace_id: str,
        user_id: str = "user",
        run_id: str | None,
        event_id: str | None,
        call_id: str | None,
        action: str,
        tool_name: str | None,
        args: dict[str, Any] | None,
        result: Any,
        requires_confirmation: bool,
        user_decision: str,
        outcome: str,
    ) -> None:
        await self.repo.insert_audit(
            audit_id=str(uuid.uuid4()),
            trace_id=trace_id,
            user_id=user_id,
            run_id=run_id,
            event_id=event_id,
            call_id=call_id,
            action=action,
            tool_name=tool_name,
            args=args,
            result=result,
            requires_confirmation=requires_confirmation,
            user_decision=user_decision,
            outcome=outcome,
        )
