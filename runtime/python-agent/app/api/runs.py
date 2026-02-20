from __future__ import annotations

import asyncio
import uuid

from fastapi import APIRouter, Depends, Request
from sse_starlette.sse import EventSourceResponse

from app.config import load_settings
from app.deps import get_repo, get_run_service
from app.services.run_service import stream_as_sse
from app.trace import get_current_trace_id

router = APIRouter(prefix="/v1", tags=["runs"])
settings = load_settings()


@router.post("/runs")
async def create_run(payload: dict, request: Request, repo=Depends(get_repo), run_service=Depends(get_run_service)):
    run_id = str(uuid.uuid4())
    trace_id = str(getattr(request.state, "trace_id", get_current_trace_id()))
    run_payload = dict(payload)
    run_payload["user_id"] = str(getattr(request.state, "user_id", "user"))
    if settings.runtime_require_hub_auth:
        run_payload["workspace_path"] = str(settings.runtime_workspace_root)
        run_payload["workspace_id"] = settings.runtime_workspace_id
    asyncio.create_task(run_service.start_run(run_id, run_payload, trace_id))
    return {"run_id": run_id}


@router.get("/runs/{run_id}/events")
async def stream_events(run_id: str, repo=Depends(get_repo), run_service=Depends(get_run_service)):
    historical_events = await repo.list_events_by_run(run_id)

    async def event_generator():
        for event in historical_events:
            yield await stream_as_sse(event)
        async for event in run_service.bus.subscribe(run_id):
            yield await stream_as_sse(event)

    return EventSourceResponse(event_generator())


@router.get("/runs")
async def list_runs(session_id: str, repo=Depends(get_repo)):
    runs = await repo.list_runs_by_session(session_id)
    return {"runs": runs}


@router.get("/runs/{run_id}/events/replay")
async def replay_events(run_id: str, repo=Depends(get_repo)):
    events = await repo.list_events_by_run(run_id)
    return {"events": events}
