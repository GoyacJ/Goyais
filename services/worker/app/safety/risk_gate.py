from __future__ import annotations

import json
import shlex
from typing import Any


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
    if normalized == "run_subagent":
        return "low"
    if normalized == "run_command":
        if _is_read_only_run_command(arguments):
            return "low"
        command = f" {str(arguments.get('command') or '').strip().lower()} "
        if any(keyword in command for keyword in [" delete ", " rm ", " remove ", " drop "]):
            return "critical"
        return "high"
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


def _is_read_only_run_command(arguments: dict[str, Any]) -> bool:
    command = str(arguments.get("command") or "").strip()
    if command == "":
        return False
    if any(operator in command for operator in ["\n", ";", "&&", "||", "|", ">", "<", "$(", "`"]):
        return False

    try:
        tokens = shlex.split(command)
    except ValueError:
        return False
    if len(tokens) == 0:
        return False

    head = tokens[0].lower()
    if head in {"pwd", "ls"}:
        return True
    if head == "cat":
        return len(tokens) >= 2
    if head == "rg":
        return len(tokens) >= 2
    if head == "git":
        return len(tokens) >= 2 and tokens[1].lower() == "status"
    return False
