from __future__ import annotations

import os

DEFAULT_RUNTIME_VERSION = "0.0.0-dev"


def get_runtime_version() -> str:
    raw_version = os.getenv("GOYAIS_VERSION", "").strip()
    if raw_version == "":
        return DEFAULT_RUNTIME_VERSION

    normalized = raw_version.removeprefix("v").removeprefix("V").strip()
    if normalized == "":
        return DEFAULT_RUNTIME_VERSION
    return normalized
