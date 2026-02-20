from __future__ import annotations

import re


class CommandGuardError(Exception):
    pass


ALLOW_PATTERNS = [
    r"^git\s+status$",
    r"^git\s+diff(\s+.*)?$",
    r"^npm\s+test(\s+.*)?$",
    r"^pnpm\s+test(\s+.*)?$",
    r"^pytest(\s+.*)?$",
    r"^python\s+-m\s+pytest(\s+.*)?$",
    r"^python3\s+-m\s+pytest(\s+.*)?$",
]

DENY_PATTERNS = [
    r"\brm\b",
    r"curl\s+.+\|\s*(sh|bash|zsh)",
    r"\bdd\b",
    r":\s*\(\)\s*\{\s*:\|:\s*&\s*\}\s*;",
]


def validate_command(cmd: str) -> None:
    for pattern in DENY_PATTERNS:
        if re.search(pattern, cmd):
            raise CommandGuardError(f"Command blocked by denylist pattern: {pattern}")

    for pattern in ALLOW_PATTERNS:
        if re.search(pattern, cmd):
            return

    raise CommandGuardError("Command is not in allowlist")
