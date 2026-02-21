from __future__ import annotations

from fastapi import APIRouter, Depends, Header

from app.config import load_settings
from app.deps import get_repo
from app.errors import GoyaisApiError
from app.services.diagnostics_service import DiagnosticsService

router = APIRouter(prefix="/v1", tags=["diagnostics"])
settings = load_settings()


@router.get("/diagnostics/execution/{execution_id}")
async def diagnostics_by_execution(
    execution_id: str,
    limit: int = 200,
    x_runtime_token: str = Header(default=""),
    repo=Depends(get_repo),
):
    if x_runtime_token != settings.runtime_secret_token:
        raise GoyaisApiError(
            code="E_SYNC_AUTH",
            message="Invalid runtime token.",
            retryable=False,
            status_code=401,
            cause="runtime_token",
        )

    bounded_limit = max(1, min(limit, 1000))
    service = DiagnosticsService(repo)
    return await service.export_execution(execution_id, bounded_limit)
