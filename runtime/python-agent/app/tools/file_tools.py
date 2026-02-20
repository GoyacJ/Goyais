from __future__ import annotations

from pathlib import Path
from typing import Any

from app.security.path_guard import resolve_in_workspace


def list_dir(workspace_path: str, path: str = ".") -> list[str]:
    target = resolve_in_workspace(workspace_path, path)
    return sorted([entry.name for entry in target.iterdir()])


def read_file(workspace_path: str, path: str) -> str:
    target = resolve_in_workspace(workspace_path, path)
    return target.read_text(encoding="utf-8")


def search_in_files(workspace_path: str, query: str, glob: str | None = None) -> list[dict[str, Any]]:
    workspace = Path(workspace_path).resolve()
    pattern = glob or "**/*"
    matches: list[dict[str, Any]] = []

    for file_path in workspace.glob(pattern):
        if not file_path.is_file():
            continue
        try:
            content = file_path.read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue

        for line_no, line in enumerate(content.splitlines(), start=1):
            if query in line:
                matches.append(
                    {
                        "path": str(file_path.relative_to(workspace)),
                        "line": line_no,
                        "text": line,
                    }
                )

    return matches


def write_file(workspace_path: str, path: str, content: str) -> str:
    target = resolve_in_workspace(workspace_path, path)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(content, encoding="utf-8")
    return f"wrote {target}"
