from __future__ import annotations

import json
import logging
from datetime import datetime, timezone
from typing import Any


class JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload: dict[str, Any] = {
            "level": record.levelname.lower(),
            "ts": datetime.now(tz=timezone.utc).isoformat(),
            "message": record.getMessage(),
        }

        for key in (
            "trace_id",
            "run_id",
            "event_id",
            "tool_name",
            "duration_ms",
            "outcome",
            "path",
            "status",
            "method",
        ):
            value = getattr(record, key, None)
            if value is not None:
                payload[key] = value

        return json.dumps(payload, ensure_ascii=False)


def get_runtime_logger() -> logging.Logger:
    logger = logging.getLogger("goyais.runtime")
    if logger.handlers:
        return logger

    handler = logging.StreamHandler()
    handler.setFormatter(JsonFormatter())
    logger.addHandler(handler)
    logger.setLevel(logging.INFO)
    logger.propagate = False
    return logger
