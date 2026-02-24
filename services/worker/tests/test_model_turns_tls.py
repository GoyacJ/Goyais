from __future__ import annotations

from typing import Any
import urllib.request

import pytest

from app import model_turns
from app.model_adapters import ModelAdapterError
from app.model_turns import _post_json_sync


class _FakeHTTPResponse:
    def __enter__(self) -> "_FakeHTTPResponse":
        return self

    def __exit__(self, exc_type: Any, exc: Any, tb: Any) -> bool:
        return False

    def read(self) -> bytes:
        return b'{"ok": true}'


def test_post_json_sync_uses_unverified_context_when_insecure_tls_enabled(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("WORKER_TLS_INSECURE_SKIP_VERIFY", "1")
    monkeypatch.delenv("WORKER_TLS_CA_FILE", raising=False)
    captured: dict[str, Any] = {}

    def fake_urlopen(
        request: urllib.request.Request,
        timeout: float,
        context: ssl.SSLContext | None = None,
    ) -> _FakeHTTPResponse:
        captured["request"] = request
        captured["timeout"] = timeout
        captured["context"] = context
        return _FakeHTTPResponse()

    monkeypatch.setattr(urllib.request, "urlopen", fake_urlopen)

    result = _post_json_sync("https://example.com/v1/chat/completions", {}, {}, 1_000)

    assert result["ok"] is True
    context = captured.get("context")
    assert context is not None


def test_post_json_sync_fails_fast_when_custom_ca_file_invalid(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("WORKER_TLS_CA_FILE", "/path/not/exist/ca.pem")
    monkeypatch.delenv("WORKER_TLS_INSECURE_SKIP_VERIFY", raising=False)

    def fail_urlopen(*_: Any, **__: Any) -> _FakeHTTPResponse:
        pytest.fail("urlopen should not be called when WORKER_TLS_CA_FILE is invalid")

    monkeypatch.setattr(urllib.request, "urlopen", fail_urlopen)

    with pytest.raises(ModelAdapterError) as exc_info:
        _post_json_sync("https://example.com/v1/chat/completions", {}, {}, 1_000)

    assert exc_info.value.code == "MODEL_TLS_CONFIG_INVALID"


def test_post_json_sync_wraps_tls_config_errors(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("WORKER_TLS_CA_FILE", raising=False)
    monkeypatch.delenv("WORKER_TLS_INSECURE_SKIP_VERIFY", raising=False)

    class _FakeTLSError(Exception):
        details = {"error": "bad_tls"}

    def fake_resolve_tls_context(url: str):
        raise model_turns.TLSConfigError("bad tls config", {"url": url, "error": "bad_tls"})

    monkeypatch.setattr(model_turns, "resolve_tls_context", fake_resolve_tls_context)

    with pytest.raises(ModelAdapterError) as exc_info:
        _post_json_sync("https://example.com/v1/chat/completions", {}, {}, 1_000)

    assert exc_info.value.code == "MODEL_TLS_CONFIG_INVALID"
    assert exc_info.value.details["error"] == "bad_tls"
