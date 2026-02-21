from __future__ import annotations

import uuid

from fastapi import APIRouter, Depends

from app.deps import get_repo
from app.errors import GoyaisApiError

router = APIRouter(prefix="/v1", tags=["projects"])


@router.get("/projects")
async def list_projects(repo=Depends(get_repo)):
    return {"projects": await repo.list_projects()}


@router.post("/projects")
async def create_project(payload: dict, repo=Depends(get_repo)):
    project_id = payload.get("project_id", str(uuid.uuid4()))
    name = payload.get("name", project_id)
    workspace_path = payload["workspace_path"]
    await repo.create_project(project_id, name, workspace_path)
    event_id = await repo.insert_system_event(
        "project_upserted",
        {
            "entity": "project",
            "project_id": project_id,
            "name": name,
            "workspace_path": workspace_path,
        },
    )
    return {"project_id": project_id, "event_id": event_id}


@router.delete("/projects/{project_id}")
async def delete_project(project_id: str, repo=Depends(get_repo)):
    deleted = await repo.delete_project(project_id)
    if not deleted:
        raise GoyaisApiError(
            code="E_NOT_FOUND",
            message="project not found",
            retryable=False,
            status_code=404,
            cause="project_not_found",
        )

    event_id = await repo.insert_system_event(
        "project_deleted",
        {
            "entity": "project",
            "project_id": project_id,
        },
    )
    return {"ok": True, "event_id": event_id}
