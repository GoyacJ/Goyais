from __future__ import annotations

import uuid

from fastapi import APIRouter, Depends, Request

from app.deps import get_confirmation_service, get_repo
from app.trace import get_current_trace_id

router = APIRouter(prefix="/v1", tags=["tool-confirmations"])


@router.post("/tool-confirmations")
async def create_tool_confirmation(
    payload: dict,
    request: Request,
    confirmation_service=Depends(get_confirmation_service),
    repo=Depends(get_repo),
):
    run_id = payload["run_id"]
    call_id = payload["call_id"]
    approved = bool(payload["approved"])

    await confirmation_service.resolve(run_id, call_id, approved)
    await repo.insert_audit(
        audit_id=str(uuid.uuid4()),
        trace_id=str(getattr(request.state, "trace_id", get_current_trace_id())),
        run_id=run_id,
        event_id=None,
        call_id=call_id,
        action="tool_confirmation",
        tool_name=None,
        args=None,
        result={"approved": approved},
        requires_confirmation=True,
        user_decision="approve" if approved else "deny",
        outcome="approved" if approved else "denied",
    )
    return {"ok": True}
