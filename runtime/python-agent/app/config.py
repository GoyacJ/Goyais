from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


@dataclass(slots=True)
class Settings:
    db_path: Path
    runtime_host: str
    runtime_port: int
    agent_mode: str
    sync_server_url: str
    sync_token: str
    sync_device_id: str
    runtime_secret_token: str
    runtime_require_hub_auth: bool
    runtime_shared_secret: str
    runtime_workspace_id: str
    runtime_workspace_root: Path
    hub_base_url: str


def _parse_bool(value: str | None, default: bool = False) -> bool:
    if value is None:
        return default
    normalized = value.strip().lower()
    if normalized in {"1", "true", "yes", "on"}:
        return True
    if normalized in {"0", "false", "no", "off"}:
        return False
    return default


def load_settings() -> Settings:
    db_path = Path(os.getenv("GOYAIS_DB_PATH", ".goyais/runtime.db"))
    db_path.parent.mkdir(parents=True, exist_ok=True)
    runtime_require_hub_auth = _parse_bool(os.getenv("GOYAIS_RUNTIME_REQUIRE_HUB_AUTH"), False)
    runtime_shared_secret = os.getenv("GOYAIS_RUNTIME_SHARED_SECRET", "").strip()
    runtime_workspace_id = os.getenv("GOYAIS_RUNTIME_WORKSPACE_ID", "local").strip()
    runtime_workspace_root = Path(
        os.getenv("GOYAIS_RUNTIME_WORKSPACE_ROOT", str(Path.cwd().resolve()))
    ).resolve()

    if runtime_require_hub_auth and not runtime_shared_secret:
        raise RuntimeError("GOYAIS_RUNTIME_SHARED_SECRET is required when GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true")
    if runtime_require_hub_auth and not runtime_workspace_id:
        raise RuntimeError("GOYAIS_RUNTIME_WORKSPACE_ID is required when GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true")

    return Settings(
        db_path=db_path,
        runtime_host=os.getenv("GOYAIS_RUNTIME_HOST", "127.0.0.1"),
        runtime_port=int(os.getenv("GOYAIS_RUNTIME_PORT", "8040")),
        agent_mode=os.getenv("GOYAIS_AGENT_MODE", "vanilla"),
        sync_server_url=os.getenv("GOYAIS_SYNC_SERVER_URL", "http://127.0.0.1:8140"),
        sync_token=os.getenv("GOYAIS_SYNC_TOKEN", "change-me"),
        sync_device_id=os.getenv("GOYAIS_SYNC_DEVICE_ID", "local-device"),
        runtime_secret_token=os.getenv("GOYAIS_RUNTIME_SECRET_TOKEN", "dev-secret-token"),
        runtime_require_hub_auth=runtime_require_hub_auth,
        runtime_shared_secret=runtime_shared_secret,
        runtime_workspace_id=runtime_workspace_id,
        runtime_workspace_root=runtime_workspace_root,
        hub_base_url=os.getenv("GOYAIS_HUB_BASE_URL", "http://127.0.0.1:8787").strip(),
    )
