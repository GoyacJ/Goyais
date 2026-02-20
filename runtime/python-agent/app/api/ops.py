from __future__ import annotations

import os

from fastapi import APIRouter

from app.observability.metrics import get_runtime_metrics
from app.services.run_service import PROTOCOL_VERSION

router = APIRouter(prefix="/v1", tags=["ops"])


@router.get("/health")
async def health():
    return {
        "ok": True,
        "version": "0.1.0",
        "protocol_version": PROTOCOL_VERSION,
        "runtime_status": "ok",
    }


@router.get("/version")
async def version():
    return {
        "protocol_version": PROTOCOL_VERSION,
        "runtime_version": "0.1.0",
        "build": os.getenv("GOYAIS_BUILD"),
        "commit": os.getenv("GOYAIS_COMMIT"),
    }


@router.get("/metrics")
async def metrics():
    return get_runtime_metrics().snapshot()
