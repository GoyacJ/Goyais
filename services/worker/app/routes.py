from __future__ import annotations

from fastapi import APIRouter

from app.version import get_runtime_version

router = APIRouter()


@router.get("/health")
def health() -> dict[str, object]:
    return {"ok": True, "version": get_runtime_version()}
