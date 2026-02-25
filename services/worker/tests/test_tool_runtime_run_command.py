from __future__ import annotations

import subprocess
from pathlib import Path
from typing import Any

from app.model_adapters import ToolCall
from app.tool_runtime import execute_tool_call


def test_run_command_executes_without_shell(monkeypatch, tmp_path: Path) -> None:
    captured: dict[str, Any] = {}

    def fake_run(*args: Any, **kwargs: Any) -> subprocess.CompletedProcess[str]:
        captured["args"] = args
        captured["kwargs"] = kwargs
        return subprocess.CompletedProcess(args[0], 0, stdout="ok\n", stderr="")

    monkeypatch.setattr(subprocess, "run", fake_run)

    result = execute_tool_call(
        ToolCall(id="tc_1", name="run_command", arguments={"command": "pwd"}),
        str(tmp_path),
    )

    assert result.output["exit_code"] == 0
    assert captured["args"][0] == ["pwd"]
    assert captured["kwargs"]["shell"] is False
