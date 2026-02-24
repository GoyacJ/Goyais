from __future__ import annotations

import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from app.model_adapters import ToolCall
from app.safety.command_guard import CommandGuardError, ensure_safe_command
from app.safety.path_guard import PathGuardError, resolve_guarded_path
from app.safety.risk_gate import classify_content_risk, classify_tool_risk
from app.tools.subagent_tools import subagent_tool_spec


@dataclass(slots=True)
class ToolExecutionResult:
    output: dict[str, Any]
    diff: dict[str, Any] | None = None


def default_tools() -> list[dict[str, Any]]:
    return [
        {
            "name": "read_file",
            "description": "Read file content from the current project.",
            "input_schema": {
                "type": "object",
                "properties": {"path": {"type": "string"}},
                "required": ["path"],
            },
        },
        {
            "name": "write_file",
            "description": "Write updated file content to the current project.",
            "input_schema": {
                "type": "object",
                "properties": {
                    "path": {"type": "string"},
                    "content": {"type": "string"},
                },
                "required": ["path", "content"],
            },
        },
        {
            "name": "edit_file",
            "description": "Replace exact text in a file.",
            "input_schema": {
                "type": "object",
                "properties": {
                    "path": {"type": "string"},
                    "old_text": {"type": "string"},
                    "new_text": {"type": "string"},
                },
                "required": ["path", "old_text", "new_text"],
            },
        },
        {
            "name": "run_command",
            "description": "Execute a terminal command in the current project.",
            "input_schema": {
                "type": "object",
                "properties": {"command": {"type": "string"}},
                "required": ["command"],
            },
        },
        subagent_tool_spec(),
    ]


def execute_tool_call(tool_call: ToolCall, working_directory: str) -> ToolExecutionResult:
    name = tool_call.name.strip()
    name_lower = name.lower()
    root = Path(working_directory).resolve()

    if name_lower == "read_file":
        return _read_file(tool_call.arguments, root)
    if name_lower == "write_file":
        return _write_file(tool_call.arguments, root, name)
    if name_lower == "edit_file":
        return _edit_file(tool_call.arguments, root, name)
    if name_lower == "run_command":
        return _run_command(tool_call.arguments, root, name)

    return ToolExecutionResult(output={"summary": f"Unsupported tool: {name}"})


def _read_file(arguments: dict[str, Any], root: Path) -> ToolExecutionResult:
    raw_path = str(arguments.get("path") or "").strip()
    if raw_path == "":
        return ToolExecutionResult(output={"error": "path is required"})
    try:
        path = resolve_guarded_path(root, raw_path)
        content = path.read_text(encoding="utf-8")
    except PathGuardError as exc:
        return ToolExecutionResult(output={"error": str(exc)})
    except FileNotFoundError:
        return ToolExecutionResult(output={"error": f"file not found: {raw_path}"})
    except Exception as exc:  # pragma: no cover - defensive branch
        return ToolExecutionResult(output={"error": str(exc)})
    return ToolExecutionResult(
        output={
            "path": raw_path,
            "summary": f"Read {raw_path}",
            "content_preview": content[:50000],
        }
    )


def _write_file(arguments: dict[str, Any], root: Path, tool_name: str) -> ToolExecutionResult:
    raw_path = str(arguments.get("path") or "").strip()
    content = str(arguments.get("content") or "")
    if raw_path == "":
        return ToolExecutionResult(output={"error": "path is required"})
    try:
        path = resolve_guarded_path(root, raw_path)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
    except PathGuardError as exc:
        return ToolExecutionResult(output={"error": str(exc)})
    except Exception as exc:  # pragma: no cover - defensive branch
        return ToolExecutionResult(output={"error": str(exc)})
    return ToolExecutionResult(
        output={"path": raw_path, "summary": f"Wrote {len(content)} bytes"},
        diff={
            "id": f"diff_{tool_name}_{raw_path}",
            "path": raw_path,
            "change_type": "modified",
            "summary": f"{tool_name} updated file",
        },
    )


def _edit_file(arguments: dict[str, Any], root: Path, tool_name: str) -> ToolExecutionResult:
    raw_path = str(arguments.get("path") or "").strip()
    old_text = str(arguments.get("old_text") or "")
    new_text = str(arguments.get("new_text") or "")
    if raw_path == "":
        return ToolExecutionResult(output={"error": "path is required"})
    try:
        path = resolve_guarded_path(root, raw_path)
        current = path.read_text(encoding="utf-8")
        if old_text not in current:
            return ToolExecutionResult(output={"error": f"text not found in {raw_path}"})
        path.write_text(current.replace(old_text, new_text, 1), encoding="utf-8")
    except PathGuardError as exc:
        return ToolExecutionResult(output={"error": str(exc)})
    except FileNotFoundError:
        return ToolExecutionResult(output={"error": f"file not found: {raw_path}"})
    except Exception as exc:  # pragma: no cover - defensive branch
        return ToolExecutionResult(output={"error": str(exc)})
    return ToolExecutionResult(
        output={"path": raw_path, "summary": f"Edited {raw_path}"},
        diff={
            "id": f"diff_{tool_name}_{raw_path}",
            "path": raw_path,
            "change_type": "modified",
            "summary": f"{tool_name} updated file",
        },
    )


def _run_command(arguments: dict[str, Any], root: Path, tool_name: str) -> ToolExecutionResult:
    raw_command = str(arguments.get("command") or "").strip()
    if raw_command == "":
        return ToolExecutionResult(output={"error": "command is required"})
    try:
        command = ensure_safe_command(raw_command)
    except CommandGuardError as exc:
        return ToolExecutionResult(output={"error": str(exc)})

    try:
        result = subprocess.run(
            command,
            shell=True,
            cwd=root,
            capture_output=True,
            text=True,
            timeout=120,
        )
        output = (result.stdout + result.stderr).strip()
    except subprocess.TimeoutExpired:
        return ToolExecutionResult(output={"error": "command timeout (120s)"})
    except Exception as exc:  # pragma: no cover - defensive branch
        return ToolExecutionResult(output={"error": str(exc)})

    return ToolExecutionResult(
        output={
            "summary": f"Command finished with code {result.returncode}",
            "exit_code": result.returncode,
            "output": output[:50000],
        },
        diff={
            "id": f"diff_{tool_name}_command",
            "path": ".",
            "change_type": "modified",
            "summary": "Command may have changed files",
        },
    )
