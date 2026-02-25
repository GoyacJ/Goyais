from app.version import DEFAULT_RUNTIME_VERSION, get_runtime_version


def test_runtime_version_defaults_to_dev(monkeypatch) -> None:
    monkeypatch.delenv("GOYAIS_VERSION", raising=False)
    assert get_runtime_version() == DEFAULT_RUNTIME_VERSION


def test_runtime_version_strips_prefix(monkeypatch) -> None:
    monkeypatch.setenv("GOYAIS_VERSION", "v1.2.3")
    assert get_runtime_version() == "1.2.3"
