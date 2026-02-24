from __future__ import annotations

import shutil
import subprocess
from dataclasses import dataclass
from pathlib import Path


@dataclass(slots=True)
class WorktreeContext:
    path: str
    created: bool


class WorktreeManager:
    def prepare(self, execution_id: str, project_path: str, project_is_git: bool) -> WorktreeContext:
        root = Path(project_path).resolve() if project_path else Path.cwd()
        if not project_is_git or not root.exists():
            return WorktreeContext(path=str(root), created=False)

        worktree_root = root / ".goyais-worktrees"
        worktree_root.mkdir(parents=True, exist_ok=True)
        lane = worktree_root / execution_id
        if lane.exists():
            return WorktreeContext(path=str(lane), created=True)

        branch = f"goyais-{execution_id[:10]}"
        command = ["git", "worktree", "add", "-b", branch, str(lane), "HEAD"]
        try:
            run = subprocess.run(
                command,
                cwd=root,
                capture_output=True,
                text=True,
                timeout=60,
            )
        except Exception:
            return WorktreeContext(path=str(root), created=False)
        if run.returncode != 0:
            return WorktreeContext(path=str(root), created=False)
        return WorktreeContext(path=str(lane), created=True)

    def cleanup(self, context: WorktreeContext, project_path: str, project_is_git: bool) -> None:
        if not context.created or not project_is_git:
            return
        root = Path(project_path).resolve() if project_path else Path.cwd()
        lane = Path(context.path)
        if not lane.exists():
            return
        try:
            subprocess.run(
                ["git", "worktree", "remove", "--force", str(lane)],
                cwd=root,
                capture_output=True,
                text=True,
                timeout=60,
                check=False,
            )
        except Exception:
            # Best-effort cleanup fallback.
            shutil.rmtree(lane, ignore_errors=True)

