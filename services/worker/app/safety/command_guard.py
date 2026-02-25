from __future__ import annotations

import shlex


class CommandGuardError(ValueError):
    pass


_SHELL_METACHARACTERS = ("\n", ";", "&&", "||", "|", ">", "<", "$(", "`")
_BLOCKED_PATTERNS = ("rm -rf /", "shutdown", "reboot", "mkfs", ":(){:|:&};:")
_ALLOWED_COMMANDS = {"pwd", "ls", "cat", "rg", "git"}
_READ_ONLY_GIT_SUBCOMMANDS = {"status", "diff", "show", "log", "branch", "rev-parse"}
_ALLOWED_LS_FLAGS = {"-a", "-l", "-la", "-al", "--all", "--human-readable", "--color=never"}


def ensure_safe_command(command: str) -> list[str]:
    normalized = command.strip()
    if normalized == "":
        raise CommandGuardError("command is required")

    lowered = normalized.lower()
    for pattern in _BLOCKED_PATTERNS:
        if pattern in lowered:
            raise CommandGuardError(f"blocked command pattern: {pattern}")
    if any(token in normalized for token in _SHELL_METACHARACTERS):
        raise CommandGuardError("shell operators are not allowed")

    try:
        tokens = shlex.split(normalized)
    except ValueError as exc:
        raise CommandGuardError("command parsing failed") from exc
    if len(tokens) == 0:
        raise CommandGuardError("command is required")

    head = tokens[0].lower()
    if head not in _ALLOWED_COMMANDS:
        raise CommandGuardError(f"command is not allowed: {head}")
    if head == "pwd":
        if len(tokens) > 1:
            raise CommandGuardError("pwd does not accept arguments")
        return tokens
    if head == "ls":
        for arg in tokens[1:]:
            if arg.startswith("-") and arg not in _ALLOWED_LS_FLAGS:
                raise CommandGuardError(f"ls flag is not allowed: {arg}")
        return tokens
    if head == "cat":
        if len(tokens) < 2:
            raise CommandGuardError("cat requires a file path")
        for arg in tokens[1:]:
            if arg.startswith("-"):
                raise CommandGuardError(f"cat flag is not allowed: {arg}")
        return tokens
    if head == "rg":
        # ripgrep is allowed for read-only search workflows.
        return tokens

    # git
    if len(tokens) < 2:
        raise CommandGuardError("git requires a subcommand")
    subcommand = tokens[1].lower()
    if subcommand not in _READ_ONLY_GIT_SUBCOMMANDS:
        raise CommandGuardError(f"git subcommand is not allowed: {subcommand}")
    return tokens
