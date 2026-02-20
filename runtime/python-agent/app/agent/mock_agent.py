from __future__ import annotations

import difflib
import uuid

from app.tools.file_tools import read_file


def build_mock_plan(task_input: str) -> dict:
    return {
        "summary": f"Handle task: {task_input}",
        "steps": [
            "Read target file",
            "Propose patch",
            "Wait for approval",
            "Apply patch",
        ],
    }


def compute_readme_patch(workspace_path: str, task_input: str) -> str:
    target_path = "README.md"
    original = read_file(workspace_path, target_path)

    desired_title = "# Updated by Goyais"
    if "改成" in task_input:
        desired_title = f"# {task_input.split('改成', 1)[1].strip() or 'Updated by Goyais'}"
    elif "to" in task_input.lower():
        desired_title = f"# {task_input.split('to', 1)[1].strip() or 'Updated by Goyais'}"

    lines = original.splitlines()
    if lines and lines[0].startswith("# "):
        lines[0] = desired_title
    else:
        lines.insert(0, desired_title)

    updated = "\n".join(lines)
    patch_lines = difflib.unified_diff(
        original.splitlines(),
        updated.splitlines(),
        fromfile=f"a/{target_path}",
        tofile=f"b/{target_path}",
        lineterm="",
    )
    return "\n".join(patch_lines) + "\n"


def new_call_id() -> str:
    return f"call_{uuid.uuid4().hex[:8]}"
