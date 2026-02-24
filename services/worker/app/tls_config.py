from __future__ import annotations

import atexit
import os
from pathlib import Path
import platform
import ssl
import subprocess
import tempfile
from typing import Any


class TLSConfigError(RuntimeError):
    def __init__(self, message: str, details: dict[str, Any] | None = None) -> None:
        super().__init__(message)
        self.details = details or {}


_cached_macos_bundle_path: str | None = None


def resolve_tls_context(url: str) -> ssl.SSLContext | None:
    if not url.lower().startswith("https://"):
        return None

    if _flag_env("WORKER_TLS_INSECURE_SKIP_VERIFY"):
        return ssl._create_unverified_context()

    cafile = _resolve_ca_file()
    if cafile is None:
        return ssl.create_default_context()
    try:
        return ssl.create_default_context(cafile=cafile)
    except (FileNotFoundError, OSError, ssl.SSLError, ValueError) as exc:
        raise TLSConfigError(
            f"tls ca file is invalid: {cafile}",
            {"ca_file": cafile, "error": str(exc)},
        ) from exc


def _resolve_ca_file() -> str | None:
    explicit = _first_non_empty_env(
        "WORKER_TLS_CA_FILE",
        "SSL_CERT_FILE",
        "REQUESTS_CA_BUNDLE",
        "CURL_CA_BUNDLE",
    )
    if explicit is not None:
        path = Path(explicit).expanduser()
        if not path.exists() or not path.is_file():
            raise TLSConfigError(
                f"tls ca file is invalid: {path}",
                {"ca_file": str(path), "error": "file_not_found"},
            )
        return str(path)

    if platform.system().lower() != "darwin":
        return None
    if not _has_proxy_env():
        return None
    return _resolve_macos_keychain_bundle()


def _resolve_macos_keychain_bundle() -> str | None:
    global _cached_macos_bundle_path
    if _cached_macos_bundle_path is not None:
        path = Path(_cached_macos_bundle_path)
        if path.exists() and path.is_file():
            return str(path)
        _cached_macos_bundle_path = None

    keychains = _macos_keychain_candidates()
    cmd = ["/usr/bin/security", "find-certificate", "-a", "-p", *keychains]
    completed = subprocess.run(cmd, check=False, capture_output=True)
    if completed.returncode != 0:
        return None
    pem = bytes(completed.stdout or b"")
    if pem.strip() == b"":
        return None

    fd, path = tempfile.mkstemp(prefix="goyais-ca-", suffix=".pem")
    os.close(fd)
    bundle_path = Path(path)
    bundle_path.write_bytes(pem)
    _cached_macos_bundle_path = str(bundle_path)
    atexit.register(_cleanup_cached_bundle)
    return _cached_macos_bundle_path


def _cleanup_cached_bundle() -> None:
    global _cached_macos_bundle_path
    path = _cached_macos_bundle_path
    _cached_macos_bundle_path = None
    if path is None:
        return
    try:
        Path(path).unlink(missing_ok=True)
    except OSError:
        return


def _macos_keychain_candidates() -> list[str]:
    candidates = [
        "/System/Library/Keychains/SystemRootCertificates.keychain",
        "/Library/Keychains/System.keychain",
        str(Path.home() / "Library/Keychains/login.keychain-db"),
    ]
    result: list[str] = []
    for item in candidates:
        if Path(item).exists():
            result.append(item)
    return result


def _has_proxy_env() -> bool:
    for key in ("HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"):
        if str(os.getenv(key, "")).strip() != "":
            return True
    return False


def _first_non_empty_env(*names: str) -> str | None:
    for name in names:
        value = str(os.getenv(name, "")).strip()
        if value != "":
            return value
    return None


def _flag_env(name: str) -> bool:
    return str(os.getenv(name, "")).strip().lower() in {"1", "true", "yes", "on"}
