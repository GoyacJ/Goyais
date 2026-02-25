from __future__ import annotations

import pytest

from app.safety.command_guard import CommandGuardError, ensure_safe_command


def test_allow_read_only_commands() -> None:
    assert ensure_safe_command("pwd") == ["pwd"]
    assert ensure_safe_command("ls -la") == ["ls", "-la"]
    assert ensure_safe_command("cat README.md") == ["cat", "README.md"]
    assert ensure_safe_command("rg --files") == ["rg", "--files"]
    assert ensure_safe_command("git status") == ["git", "status"]


def test_reject_shell_metacharacters() -> None:
    with pytest.raises(CommandGuardError, match="shell operators are not allowed"):
        ensure_safe_command("ls; rm -rf .")


def test_reject_non_allowlisted_command() -> None:
    with pytest.raises(CommandGuardError, match="command is not allowed"):
        ensure_safe_command("python scripts/sync.py")


def test_reject_git_write_subcommand() -> None:
    with pytest.raises(CommandGuardError, match="git subcommand is not allowed"):
        ensure_safe_command("git commit -m test")
