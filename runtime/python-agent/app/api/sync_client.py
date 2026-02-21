from __future__ import annotations

from fastapi import APIRouter, Depends

from app.deps import get_sync_service
from app.errors import GoyaisApiError

router = APIRouter(prefix="/v1", tags=["sync-client"])


@router.post("/sync/now")
async def sync_now(sync_service=Depends(get_sync_service)):
    if sync_service is None:
        raise GoyaisApiError(
            code="E_INTERNAL",
            message="Sync service not configured.",
            retryable=False,
            status_code=503,
            cause="sync_service_missing",
        )
    return await sync_service.sync_now()
