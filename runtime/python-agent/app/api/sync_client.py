from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException

from app.deps import get_sync_service

router = APIRouter(prefix="/v1", tags=["sync-client"])


@router.post("/sync/now")
async def sync_now(sync_service=Depends(get_sync_service)):
    if sync_service is None:
        raise HTTPException(status_code=503, detail="sync service not configured")
    return await sync_service.sync_now()
