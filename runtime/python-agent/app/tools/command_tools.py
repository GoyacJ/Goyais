from __future__ import annotations

import subprocess
from pathlib import Path

from app.security.command_guard import validate_command
from app.security.path_guard import resolve_in_workspace


def run_command(workspace_path: str, cmd: str, cwd: str | None = None) -> dict:
    validate_command(cmd)

    if cwd:
        safe_cwd = resolve_in_workspace(workspace_path, cwd)
    else:
        safe_cwd = Path(workspace_path).resolve()

    proc = subprocess.run(
        cmd,
        cwd=safe_cwd,
        shell=True,
        capture_output=True,
        text=True,
        timeout=120,
    )

    return {
        "returncode": proc.returncode,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
    }
