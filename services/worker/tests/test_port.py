from app.main import get_port


def test_get_port_uses_default_when_missing(monkeypatch) -> None:
    monkeypatch.delenv("PORT", raising=False)
    assert get_port() == 8788


def test_get_port_uses_port_env(monkeypatch) -> None:
    monkeypatch.setenv("PORT", "9012")
    assert get_port() == 9012


def test_get_port_falls_back_when_invalid(monkeypatch) -> None:
    monkeypatch.setenv("PORT", "abc")
    assert get_port() == 8788
