"""Context-aware system prompt builder replacing the old minimal prompts.py."""
from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from app.agent.tool_registry import ToolRegistry

ROLE_PREAMBLE = """\
You are Goyais, a professional software engineering assistant.
You help users analyze, modify, and maintain codebases by using the tools available to you.
You work inside a sandboxed workspace and can read files, write files, apply patches, \
search code, and run shell commands.

Key principles:
- Always read relevant files before modifying them.
- Prefer minimal, targeted changes over large rewrites.
- Explain your reasoning briefly before taking action.
- If a task is ambiguous, ask for clarification rather than guessing.
- Never modify files outside the workspace.
- Never run destructive commands (rm -rf, DROP TABLE, etc.) without explicit user approval.
"""

PLAN_MODE_ADDENDUM = """\

## Mode: PLAN

You are in **plan mode**. Your job is to analyze the task and produce a detailed execution plan.
- Use read-only tools (read_file, list_dir, search_in_files) to understand the codebase.
- Do NOT use write_file, apply_patch, or run_command.
- After analysis, output a concise plan with numbered steps.
- Each step should be specific and actionable.
- End with a summary of files to modify and expected changes.
"""

AUTO_MODE_ADDENDUM = """\

## Mode: AUTO

You are in **auto mode**. Execute the task directly using available tools.
- Read files first to understand context.
- Make changes using write_file or apply_patch.
- Verify your work by reading modified files.
- Run relevant tests if applicable.
- When done, provide a brief summary of what you changed and why.
"""


def _format_tool_docs(registry: ToolRegistry) -> str:
    lines = ["## Available Tools\n"]
    for schema in registry.to_schemas():
        lines.append(f"### {schema.name}")
        lines.append(schema.description)
        props = schema.input_schema.get("properties", {})
        required = set(schema.input_schema.get("required", []))
        if props:
            lines.append("Parameters:")
            for pname, pinfo in props.items():
                req_marker = " (required)" if pname in required else ""
                desc = pinfo.get("description", "")
                lines.append(f"  - {pname}: {desc}{req_marker}")
        lines.append("")
    return "\n".join(lines)


def build_system_prompt(
    *,
    mode: str,
    workspace_path: str,
    tool_registry: ToolRegistry,
    project_summary: str = "",
    skill_descriptions: list[str] | None = None,
) -> str:
    parts = [ROLE_PREAMBLE]

    if mode == "plan":
        parts.append(PLAN_MODE_ADDENDUM)
    else:
        parts.append(AUTO_MODE_ADDENDUM)

    parts.append(_format_tool_docs(tool_registry))

    parts.append(f"\n## Workspace\nPath: `{workspace_path}`\n")

    if project_summary:
        parts.append(f"## Project Context\n{project_summary}\n")

    if skill_descriptions:
        parts.append("## Loaded Skills\n")
        for desc in skill_descriptions:
            parts.append(f"- {desc}")
        parts.append("")

    return "\n".join(parts)
