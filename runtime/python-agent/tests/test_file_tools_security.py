from pathlib import Path

import pytest

from app.security.path_guard import PathGuardError
from app.tools.file_tools import write_file


def test_write_file_rejects_parent_traversal(tmp_path: Path):
    workspace = tmp_path / "workspace"
    workspace.mkdir()

    with pytest.raises(PathGuardError):
        write_file(str(workspace), "../escape.txt", "blocked")


def test_write_file_rejects_absolute_outside_workspace(tmp_path: Path):
    workspace = tmp_path / "workspace"
    workspace.mkdir()
    outside = tmp_path / "outside.txt"

    with pytest.raises(PathGuardError):
        write_file(str(workspace), str(outside), "blocked")
