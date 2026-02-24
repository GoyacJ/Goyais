from __future__ import annotations


class CommandGuardError(ValueError):
    pass


_BLOCKED_PATTERNS = (
    "rm -rf /",
    "shutdown",
    "reboot",
    "mkfs",
    ":(){:|:&};:",
)


def ensure_safe_command(command: str) -> str:
    normalized = command.strip()
    lowered = normalized.lower()
    for pattern in _BLOCKED_PATTERNS:
        if pattern in lowered:
            raise CommandGuardError(f"blocked command pattern: {pattern}")
    return normalized

