from __future__ import annotations

from contextlib import asynccontextmanager
from datetime import datetime, timezone

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.api import diagnostics, model_configs, ops, projects, replay, runs, secrets, sync_client, system_events, tool_confirmations
from app.config import load_settings
from app.deps import set_dependencies
from app.db.connection import open_connection
from app.db.migrations import apply_migrations
from app.db.repositories import Repository
from app.errors import error_from_exception
from app.observability.logging import get_runtime_logger
from app.services.audit_service import AuditService
from app.services.confirmation_service import ConfirmationService
from app.services.run_service import RunService
from app.services.sync_service import SyncService
from app.sse.event_bus import EventBus
from app.trace import TRACE_HEADER, get_current_trace_id, normalize_trace_id, set_current_trace_id

settings = load_settings()
logger = get_runtime_logger()


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


@app.middleware("http")
async def trace_middleware(request: Request, call_next):
    trace_id = normalize_trace_id(request.headers.get(TRACE_HEADER))
    request.state.trace_id = trace_id
    set_current_trace_id(trace_id)
    started = datetime.now(tz=timezone.utc)
    try:
        response = await call_next(request)
    except Exception as exc:  # noqa: BLE001
        status_code, payload = error_from_exception(exc, trace_id)
        response = JSONResponse(status_code=status_code, content=payload)
    duration_ms = int((datetime.now(tz=timezone.utc) - started).total_seconds() * 1000)
    logger.info(
        "http_request",
        extra={
            "trace_id": trace_id,
            "path": request.url.path,
            "method": request.method,
            "status": response.status_code,
            "duration_ms": duration_ms,
            "outcome": "ok" if response.status_code < 400 else "error",
        },
    )
    response.headers[TRACE_HEADER] = trace_id
    return response


@app.exception_handler(Exception)
async def exception_handler(request: Request, exc: Exception):
    trace_id = str(getattr(request.state, "trace_id", get_current_trace_id()))
    status_code, payload = error_from_exception(exc, trace_id)
    response = JSONResponse(status_code=status_code, content=payload)
    response.headers[TRACE_HEADER] = trace_id
    return response


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    return await exception_handler(request, exc)


app.include_router(runs.router)
app.include_router(tool_confirmations.router)
app.include_router(projects.router)
app.include_router(model_configs.router)
app.include_router(replay.router)
app.include_router(ops.router)
app.include_router(diagnostics.router)
app.include_router(sync_client.router)
app.include_router(secrets.router)
app.include_router(system_events.router)
