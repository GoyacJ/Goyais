from __future__ import annotations

from collections import Counter
from typing import Any

from app.db.repositories import Repository
from app.errors import GoyaisApiError
from app.observability.redaction import redact


class DiagnosticsService:
    def __init__(self, repo: Repository):
        self.repo = repo

    async def export_run(self, run_id: str, limit: int = 200) -> dict[str, Any]:
        run = await self.repo.get_run(run_id)
        if run is None:
            raise GoyaisApiError(
                code="E_SCHEMA_INVALID",
                message="Run not found.",
                retryable=False,
                status_code=404,
                cause="run_not_found",
            )

        events = await self.repo.list_events_by_run(run_id)
        trimmed_events = events[-limit:]
        audits = await self.repo.list_audit_logs_by_run(run_id, limit=limit)

        error_events = [
            event
            for event in trimmed_events
            if event["type"] == "error"
            or (event["type"] == "tool_result" and event["payload"].get("ok") is False)
        ]

        action_counter = Counter(str(item.get("action", "unknown")) for item in audits)
        outcome_counter = Counter(str(item.get("outcome", "unknown")) for item in audits)

        diagnostics = {
            "run": run,
            "events": trimmed_events,
            "audit_summary": {
                "count": len(audits),
                "actions": dict(action_counter),
                "outcomes": dict(outcome_counter),
            },
            "audit_samples": audits[:20],
            "key_errors": error_events,
        }
        return redact(diagnostics)
