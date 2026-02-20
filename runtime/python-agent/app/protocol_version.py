from __future__ import annotations

import json
import os
from functools import lru_cache
from pathlib import Path


def _version_order(version_dir_name: str) -> int | None:
    if not version_dir_name.startswith("v"):
        return None
    suffix = version_dir_name[1:]
    if not suffix.isdigit():
        return None
    return int(suffix)


def _candidate_schema_roots() -> list[Path]:
    env_path = os.getenv("GOYAIS_PROTOCOL_SCHEMAS_DIR")
    if env_path:
        return [Path(env_path)]

    current_file = Path(__file__).resolve()
    # app/protocol_version.py -> repo root at parents[3]
    from_repo_root = current_file.parents[3] / "packages" / "protocol" / "schemas"
    from_cwd = Path.cwd().resolve() / "packages" / "protocol" / "schemas"
    return [from_repo_root, from_cwd]


@lru_cache(maxsize=1)
def load_protocol_version() -> str:
    for root in _candidate_schema_roots():
        if not root.exists() or not root.is_dir():
            continue

        version_dirs = []
        for child in root.iterdir():
            if not child.is_dir():
                continue
            order = _version_order(child.name)
            if order is None:
                continue
            version_dirs.append((order, child))

        for _, version_dir in sorted(version_dirs, key=lambda item: item[0], reverse=True):
            version_file = version_dir / "protocol-version.json"
            if not version_file.exists():
                continue
            raw = json.loads(version_file.read_text(encoding="utf-8"))
            version = raw.get("version")
            if isinstance(version, str) and version.strip():
                return version.strip()
            raise RuntimeError(f"Invalid protocol version file: {version_file}")

    raise RuntimeError("Protocol version schema file was not found.")
