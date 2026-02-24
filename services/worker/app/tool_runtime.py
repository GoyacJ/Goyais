from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any

from app.model_adapters import ToolCall


@dataclass(slots=True)
class ToolExecutionResult:
    output: dict[str, Any]
    diff: dict[str, Any] | None = None


def classify_content_risk(content: str) -> str:
    normalized = content.lower()
    critical_keywords = [" delete ", " rm ", "remove file", "drop table", "删除"]
    high_keywords = [
        "write",
        "apply_patch",
        "run ",
        "command",
        "network",
        "edit ",
        "修改",
        "写入",
        "执行",
        "联网",
    ]

    wrapped = f" {normalized} "
    if any(keyword in wrapped for keyword in critical_keywords):
        return "critical"
    if any(keyword in normalized for keyword in high_keywords):
        return "high"
    return "low"


def classify_tool_risk(tool_name: str, arguments: dict[str, Any]) -> str:
    normalized = tool_name.lower()
    critical_keywords = ["delete", "remove", "rm", "drop"]
    high_keywords = ["write", "patch", "run", "command", "network", "edit", "create"]
    if any(keyword in normalized for keyword in critical_keywords):
        return "critical"
    if any(keyword in normalized for keyword in high_keywords):
        return "high"

    raw_arguments = json.dumps(arguments, ensure_ascii=False).lower()
    if any(keyword in raw_arguments for keyword in ["delete", "rm ", "remove", "drop table", "删除"]):
        return "critical"
    if any(keyword in raw_arguments for keyword in ["write", "apply_patch", "run_command", "network"]):
        return "high"
    return "low"


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
            "name": "run_command",
            "description": "Execute a terminal command in the current project.",
            "input_schema": {
                "type": "object",
                "properties": {"command": {"type": "string"}},
                "required": ["command"],
            },
        },
    ]


def execute_tool_call(tool_call: ToolCall) -> ToolExecutionResult:
    name = tool_call.name
    name_lower = name.lower()
    path = _resolve_tool_path(tool_call.arguments)

    if "read" in name_lower or "list" in name_lower or "search" in name_lower:
        return ToolExecutionResult(
            output={
                "path": path,
                "summary": f"Read completed for {path}",
                "content_preview": "simulated tool output",
            }
        )

    if "delete" in name_lower or "remove" in name_lower or "rm" in name_lower:
        return ToolExecutionResult(
            output={"path": path, "summary": f"Deleted {path}"},
            diff={
                "id": f"diff_{name}_{path}",
                "path": path,
                "change_type": "deleted",
                "summary": f"{name} removed file",
            },
        )

    if (
        "write" in name_lower
        or "patch" in name_lower
        or "edit" in name_lower
        or "run" in name_lower
        or "command" in name_lower
    ):
        return ToolExecutionResult(
            output={"path": path, "summary": f"Applied update via {name}"},
            diff={
                "id": f"diff_{name}_{path}",
                "path": path,
                "change_type": "modified",
                "summary": f"{name} updated file",
            },
        )

    return ToolExecutionResult(output={"summary": f"Tool {name} executed", "path": path})


def _resolve_tool_path(arguments: dict[str, Any]) -> str:
    for key in ("path", "file_path", "target", "target_path"):
        value = arguments.get(key)
        if isinstance(value, str) and value.strip() != "":
            return value.strip()
    return "src/main.ts"
