from __future__ import annotations

from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api import model_configs, projects, replay, runs, secrets, sync_client, system_events, tool_confirmations
from app.config import load_settings
from app.deps import set_dependencies
from app.db.connection import open_connection
from app.db.migrations import apply_migrations
from app.db.repositories import Repository
from app.services.audit_service import AuditService
from app.services.confirmation_service import ConfirmationService
from app.services.run_service import RunService
from app.services.sync_service import SyncService
from app.sse.event_bus import EventBus

settings = load_settings()


@asynccontextmanager
async def lifespan(app: FastAPI):
    await apply_migrations()
    conn = await open_connection(settings.db_path)
    repo = Repository(conn)
    bus = EventBus()
    confirmation_service = ConfirmationService(repo)
    audit_service = AuditService(repo)
    run_service = RunService(
        repo=repo,
        bus=bus,
        confirmation_service=confirmation_service,
        audit_service=audit_service,
        agent_mode=settings.agent_mode,
    )
    sync_service = SyncService(repo, settings.sync_server_url, settings.sync_token, settings.sync_device_id)
    set_dependencies(repo, run_service, confirmation_service, sync_service)
    await run_service.recover_pending_confirmations_after_restart()

    yield

    await conn.close()


app = FastAPI(title="Goyais Python Runtime", version="0.1.0", lifespan=lifespan)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(runs.router)
app.include_router(tool_confirmations.router)
app.include_router(projects.router)
app.include_router(model_configs.router)
app.include_router(replay.router)
app.include_router(sync_client.router)
app.include_router(secrets.router)
app.include_router(system_events.router)
