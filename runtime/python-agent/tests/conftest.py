from __future__ import annotations

import importlib

import pytest
from fastapi.testclient import TestClient

import app.main as main_module


@pytest.fixture
def isolated_client(tmp_path, monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setenv("GOYAIS_DB_PATH", str(tmp_path / "runtime-test.db"))
    module = importlib.reload(main_module)
    with TestClient(module.app) as client:
        yield client
