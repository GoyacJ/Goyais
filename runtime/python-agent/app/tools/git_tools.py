from __future__ import annotations

import subprocess
from pathlib import Path

from app.security.path_guard import resolve_in_workspace


def git_worktree_create(workspace_path: str, task_id: str) -> str:
    workspace = Path(workspace_path).resolve()
    worktrees_dir = workspace / ".worktrees"
    worktrees_dir.mkdir(exist_ok=True)
    target = worktrees_dir / task_id
    branch = f"goya/{task_id}"

    subprocess.run(["git", "worktree", "add", str(target), "-b", branch], cwd=workspace, check=True)
    return str(target)


def git_worktree_cleanup(workspace_path: str, task_id: str) -> str:
    target = resolve_in_workspace(workspace_path, f".worktrees/{task_id}")
    subprocess.run(["git", "worktree", "remove", "--force", str(target)], check=True)
    return str(target)
