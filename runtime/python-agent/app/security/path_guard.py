from __future__ import annotations

from pathlib import Path


class PathGuardError(Exception):
    pass


def resolve_in_workspace(workspace_path: str, candidate_path: str) -> Path:
    workspace = Path(workspace_path).resolve()
    candidate = (workspace / candidate_path).resolve() if not Path(candidate_path).is_absolute() else Path(candidate_path).resolve()

    try:
        candidate.relative_to(workspace)
    except ValueError as exc:
        raise PathGuardError(f"Path '{candidate}' is outside workspace '{workspace}'") from exc

    return candidate
