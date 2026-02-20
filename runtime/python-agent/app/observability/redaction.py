from __future__ import annotations

import os
import re
from pathlib import Path
from typing import Any

_SENSITIVE_KEYWORDS = ("authorization", "api_key", "apikey", "token", "secret", "secret_ref")
_BEARER_PATTERN = re.compile(r"(?i)bearer\\s+[a-z0-9_\\-\\.]+")


def _is_sensitive_key(key: str) -> bool:
    normalized = key.lower()
    return any(word in normalized for word in _SENSITIVE_KEYWORDS)


def _redact_string(key: str, value: str) -> str:
    if _is_sensitive_key(key):
        return "<redacted>"

    if "path" in key.lower():
        return Path(value).name

    return _BEARER_PATTERN.sub("Bearer <redacted>", value)


def redact(value: Any, key: str = "") -> Any:
    if isinstance(value, dict):
        return {k: redact(v, k) for k, v in value.items()}
    if isinstance(value, list):
        return [redact(item, key) for item in value]
    if isinstance(value, str):
        if value.startswith(os.sep) and "path" in key.lower():
            return Path(value).name
        return _redact_string(key, value)
    return value
