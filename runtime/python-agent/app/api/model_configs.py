from __future__ import annotations

import uuid

from fastapi import APIRouter, Depends

from app.deps import get_repo

router = APIRouter(prefix="/v1", tags=["model-configs"])


@router.get("/model-configs")
async def list_model_configs(repo=Depends(get_repo)):
    return {"model_configs": await repo.list_model_configs()}


@router.post("/model-configs")
async def create_model_config(payload: dict, repo=Depends(get_repo)):
    model_config_id = payload.get("model_config_id", str(uuid.uuid4()))
    await repo.upsert_model_config(
        {
            "model_config_id": model_config_id,
            "provider": payload["provider"],
            "model": payload["model"],
            "base_url": payload.get("base_url"),
            "temperature": payload.get("temperature", 0),
            "max_tokens": payload.get("max_tokens"),
            "secret_ref": payload["secret_ref"],
        }
    )
    event_id = await repo.insert_system_event(
        "model_config_upserted",
        {
            "entity": "model_config",
            "model_config_id": model_config_id,
            "provider": payload["provider"],
            "model": payload["model"],
            "secret_ref": payload["secret_ref"],
        },
    )
    return {"model_config_id": model_config_id, "event_id": event_id}
