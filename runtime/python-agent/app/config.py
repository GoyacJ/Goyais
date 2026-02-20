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


def load_settings() -> Settings:
    db_path = Path(os.getenv("GOYAIS_DB_PATH", ".goyais/runtime.db"))
    db_path.parent.mkdir(parents=True, exist_ok=True)

    return Settings(
        db_path=db_path,
        runtime_host=os.getenv("GOYAIS_RUNTIME_HOST", "127.0.0.1"),
        runtime_port=int(os.getenv("GOYAIS_RUNTIME_PORT", "8040")),
        agent_mode=os.getenv("GOYAIS_AGENT_MODE", "mock"),
        sync_server_url=os.getenv("GOYAIS_SYNC_SERVER_URL", "http://127.0.0.1:8140"),
        sync_token=os.getenv("GOYAIS_SYNC_TOKEN", "change-me"),
        sync_device_id=os.getenv("GOYAIS_SYNC_DEVICE_ID", "local-device"),
        runtime_secret_token=os.getenv("GOYAIS_RUNTIME_SECRET_TOKEN", "dev-secret-token"),
    )
