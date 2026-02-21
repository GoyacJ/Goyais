from __future__ import annotations

import uuid

from fastapi import APIRouter, Depends

from app.deps import get_repo
from app.errors import GoyaisApiError

router = APIRouter(prefix="/v1", tags=["sessions"])


@router.get("/sessions")
async def list_sessions(project_id: str, repo=Depends(get_repo)):
    if not project_id.strip():
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="project_id query parameter is required",
            retryable=False,
            status_code=400,
            cause="project_id_missing",
        )

    sessions = await repo.list_sessions_by_project(project_id.strip())
    return {"sessions": sessions}


@router.post("/sessions")
async def create_session(payload: dict, repo=Depends(get_repo)):
    project_id = str(payload.get("project_id", "")).strip()
    if not project_id:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="project_id is required",
            retryable=False,
            status_code=400,
            cause="project_id_missing",
        )

    session_id = str(payload.get("session_id") or uuid.uuid4())
    title = str(payload.get("title") or "").strip() or session_id
    try:
        session = await repo.create_session(session_id, project_id, title)
    except Exception as exc:  # noqa: BLE001
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message=str(exc),
            retryable=False,
            status_code=400,
            cause="session_create_invalid",
        ) from exc

    return {"session": session}


@router.patch("/sessions/{session_id}")
async def rename_session(session_id: str, payload: dict, repo=Depends(get_repo)):
    title = str(payload.get("title") or "").strip()
    if not title:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="title is required",
            retryable=False,
            status_code=400,
            cause="session_title_missing",
        )

    session = await repo.rename_session(session_id, title)
    if session is None:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="session not found",
            retryable=False,
            status_code=404,
            cause="session_not_found",
        )

    return {"session": session}
