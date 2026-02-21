from __future__ import annotations

from dataclasses import dataclass, field
from typing import Dict


@dataclass(slots=True)
class RuntimeMetrics:
    runs_total: int = 0
    runs_failed_total: int = 0
    tool_calls_total: Dict[str, int] = field(default_factory=dict)
    confirmations_pending: int = 0
    sync_push_total: int = 0
    sync_pull_total: int = 0

    def increment_tool_call(self, tool_name: str) -> None:
        self.tool_calls_total[tool_name] = self.tool_calls_total.get(tool_name, 0) + 1

    def snapshot(self) -> dict:
        return {
            "runs_total": self.runs_total,
            "runs_failed_total": self.runs_failed_total,
            "tool_calls_total": dict(self.tool_calls_total),
            "confirmations_pending": self.confirmations_pending,
            "sync_push_total": self.sync_push_total,
            "sync_pull_total": self.sync_pull_total,
        }


_runtime_metrics = RuntimeMetrics()


def get_runtime_metrics() -> RuntimeMetrics:
    return _runtime_metrics
