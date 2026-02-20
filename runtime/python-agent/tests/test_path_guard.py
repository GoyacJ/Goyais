from pathlib import Path

import pytest

from app.security.path_guard import PathGuardError, resolve_in_workspace


def test_rejects_outside_workspace(tmp_path: Path):
    workspace = tmp_path / "workspace"
    workspace.mkdir()

    with pytest.raises(PathGuardError):
        resolve_in_workspace(str(workspace), "../escape.txt")


def test_accepts_inside_workspace(tmp_path: Path):
    workspace = tmp_path / "workspace"
    workspace.mkdir()

    target = resolve_in_workspace(str(workspace), "src/file.txt")
    assert str(target).startswith(str(workspace))


def test_rejects_absolute_outside_workspace(tmp_path: Path):
    workspace = tmp_path / "workspace"
    workspace.mkdir()

    outside = tmp_path / "outside.txt"
    with pytest.raises(PathGuardError):
        resolve_in_workspace(str(workspace), str(outside))
