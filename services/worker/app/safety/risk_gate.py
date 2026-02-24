from __future__ import annotations

import json
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
