from __future__ import annotations

from pathlib import Path
import ssl
import subprocess

import pytest

from app import tls_config


def test_resolve_tls_context_uses_unverified_context_when_enabled(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("WORKER_TLS_INSECURE_SKIP_VERIFY", "1")
    monkeypatch.delenv("WORKER_TLS_CA_FILE", raising=False)

    context = tls_config.resolve_tls_context("https://example.com/v1")

    assert isinstance(context, ssl.SSLContext)
    assert context.verify_mode == ssl.CERT_NONE
    assert context.check_hostname is False


def test_resolve_tls_context_uses_macos_keychain_bundle_when_proxy_enabled(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setattr(tls_config.platform, "system", lambda: "Darwin")
    monkeypatch.setenv("HTTPS_PROXY", "http://127.0.0.1:7890")
    monkeypatch.delenv("WORKER_TLS_CA_FILE", raising=False)
    monkeypatch.delenv("WORKER_TLS_INSECURE_SKIP_VERIFY", raising=False)

    pem = b"""-----BEGIN CERTIFICATE-----\nMIIBszCCAVmgAwIBAgIUdRr8YwQ2C0j4\n-----END CERTIFICATE-----\n"""
    completed = subprocess.CompletedProcess(
        args=["security", "find-certificate"],
        returncode=0,
        stdout=pem,
        stderr=b"",
    )
    monkeypatch.setattr(tls_config.subprocess, "run", lambda *args, **kwargs: completed)

    captured: dict[str, str | None] = {}
    real_create_default_context = ssl.create_default_context

    def fake_create_default_context(*, cafile: str | None = None) -> ssl.SSLContext:
        captured["cafile"] = cafile
        if cafile is None:
            return real_create_default_context()
        return ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)

    monkeypatch.setattr(tls_config.ssl, "create_default_context", fake_create_default_context)

    context = tls_config.resolve_tls_context("https://example.com/v1")

    assert isinstance(context, ssl.SSLContext)
    bundle_path = captured.get("cafile")
    assert bundle_path is not None and bundle_path != ""
    assert Path(bundle_path).exists()


def test_resolve_tls_context_rejects_invalid_explicit_ca_file(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("WORKER_TLS_CA_FILE", "/path/not/exist/ca.pem")
    monkeypatch.delenv("WORKER_TLS_INSECURE_SKIP_VERIFY", raising=False)

    with pytest.raises(tls_config.TLSConfigError) as exc_info:
        tls_config.resolve_tls_context("https://example.com/v1")

    assert "invalid" in str(exc_info.value).lower()
