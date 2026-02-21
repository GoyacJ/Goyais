"""Tool registry: definitions, schema generation, and dispatch."""
from __future__ import annotations

import asyncio
import inspect
import json
import traceback
from dataclasses import dataclass, field
from typing import Any, Callable

from app.agent.providers.base import ToolSchema


@dataclass(slots=True)
class ToolDef:
    name: str
    description: str
    input_schema: dict
    handler: Callable
    requires_confirmation: bool = False


class ToolRegistry:
    def __init__(self) -> None:
        self._tools: dict[str, ToolDef] = {}

    def register(self, tool: ToolDef) -> None:
        self._tools[tool.name] = tool

    def get(self, name: str) -> ToolDef | None:
        return self._tools.get(name)

    def to_schemas(self) -> list[ToolSchema]:
        return [
            ToolSchema(
                name=td.name,
                description=td.description,
                input_schema=td.input_schema,
            )
            for td in self._tools.values()
        ]

    async def execute(self, name: str, args: dict) -> str:
        td = self._tools.get(name)
        if td is None:
            return json.dumps({"error": f"Unknown tool: {name}"})
        try:
            if inspect.iscoroutinefunction(td.handler):
                result = await td.handler(**args)
            else:
                result = await asyncio.to_thread(td.handler, **args)
            if isinstance(result, str):
                return result
            return json.dumps(result, ensure_ascii=False, default=str)
        except Exception as exc:
            return json.dumps({
                "error": str(exc),
                "traceback": traceback.format_exc()[-500:],
            })


def build_builtin_tools(workspace_path: str) -> list[ToolDef]:
    """Build the default set of built-in tools bound to a workspace."""
    from app.tools.command_tools import run_command
    from app.tools.file_tools import list_dir, read_file, search_in_files, write_file
    from app.tools.patch_tools import apply_patch

    return [
        ToolDef(
            name="read_file",
            description="Read a file's content. Returns the full text of the file.",
            input_schema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Relative file path within the workspace"},
                },
                "required": ["path"],
            },
            handler=lambda path: read_file(workspace_path, path),
        ),
        ToolDef(
            name="write_file",
            description="Write content to a file. Creates parent directories if needed.",
            input_schema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Relative file path"},
                    "content": {"type": "string", "description": "File content to write"},
                },
                "required": ["path", "content"],
            },
            handler=lambda path, content: write_file(workspace_path, path, content),
            requires_confirmation=True,
        ),
        ToolDef(
            name="list_dir",
            description="List directory contents. Returns a list of file/directory names.",
            input_schema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Relative directory path (default: '.')", "default": "."},
                },
            },
            handler=lambda path=".": list_dir(workspace_path, path),
        ),
        ToolDef(
            name="search_in_files",
            description="Search for a text pattern in files. Returns matching lines with file paths and line numbers.",
            input_schema={
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "Text to search for"},
                    "glob": {"type": "string", "description": "Glob pattern to filter files (e.g. '**/*.py')"},
                },
                "required": ["query"],
            },
            handler=lambda query, glob=None: search_in_files(workspace_path, query, glob),
        ),
        ToolDef(
            name="apply_patch",
            description="Apply a unified diff patch to files in the workspace.",
            input_schema={
                "type": "object",
                "properties": {
                    "unified_diff": {"type": "string", "description": "Unified diff text"},
                },
                "required": ["unified_diff"],
            },
            handler=lambda unified_diff: apply_patch(workspace_path, unified_diff),
            requires_confirmation=True,
        ),
        ToolDef(
            name="run_command",
            description="Run a shell command in the workspace. Returns stdout, stderr, and return code.",
            input_schema={
                "type": "object",
                "properties": {
                    "cmd": {"type": "string", "description": "Shell command to execute"},
                    "cwd": {"type": "string", "description": "Working directory relative to workspace (optional)"},
                },
                "required": ["cmd"],
            },
            handler=lambda cmd, cwd=None: run_command(workspace_path, cmd, cwd),
            requires_confirmation=True,
        ),
    ]
