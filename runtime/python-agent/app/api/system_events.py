from __future__ import annotations

from fastapi import APIRouter, Depends

from app.deps import get_repo

router = APIRouter(prefix="/v1", tags=["system-events"])


@router.get("/system-events")
async def list_system_events(since_global_seq: int = 0, repo=Depends(get_repo)):
    events = await repo.list_system_events(since_global_seq)
    return {"events": events}
