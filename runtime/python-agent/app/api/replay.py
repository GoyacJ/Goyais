from __future__ import annotations

from fastapi import APIRouter

router = APIRouter(prefix="/v1", tags=["replay"])


@router.get("/health")
async def health():
    return {"ok": True}
