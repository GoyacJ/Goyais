from __future__ import annotations

import uuid

from fastapi import APIRouter, Depends

from app.deps import get_repo

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
