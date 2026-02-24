from __future__ import annotations

from pathlib import Path


class PathGuardError(ValueError):
    pass


def resolve_guarded_path(root_dir: str | Path, raw_path: str) -> Path:
    root = Path(root_dir).resolve()
    candidate = (root / raw_path).resolve()
    if not candidate.is_relative_to(root):
        raise PathGuardError(f"path escapes workspace: {raw_path}")
    return candidate

