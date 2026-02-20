from __future__ import annotations


SENSITIVE_TOOLS = {
    "write_file",
    "apply_patch",
    "run_command",
    "network_request",
}


def requires_confirmation(tool_name: str) -> bool:
    return tool_name in SENSITIVE_TOOLS
