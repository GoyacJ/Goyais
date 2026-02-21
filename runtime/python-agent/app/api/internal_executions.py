"""
internal_executions.py — Worker 内部 API

接收 Hub 的执行调度 (POST /internal/executions) 和
确认决策转发 (POST /internal/confirmations)。
提供 Git commit 代理 (POST /internal/executions/{id}/commit)。
"""
from __future__ import annotations

import asyncio

from fastapi import APIRouter, Depends, HTTPException, Request

from app.deps import get_execution_service
from app.errors import GoyaisApiError
from app.services.worktree_manager import WorktreeError, WorktreeManager

router = APIRouter(prefix="/internal", tags=["internal"])


@router.post("/executions", status_code=202)
async def dispatch_execution(
    payload: dict,
    request: Request,
    execution_service=Depends(get_execution_service),
):
    """Hub → Worker: start a new execution."""
    execution_id = payload.get("execution_id")
    if not execution_id:
        raise HTTPException(status_code=400, detail="execution_id required")

    # Fire-and-forget; Hub tracks progress via event reports
    asyncio.create_task(execution_service.execute(payload))
    return {"execution_id": execution_id, "status": "accepted"}


@router.post("/confirmations", status_code=200)
async def receive_confirmation(
    payload: dict,
    execution_service=Depends(get_execution_service),
):
    """Hub → Worker: forward a tool confirmation decision."""
    execution_id = payload.get("execution_id", "")
    call_id = payload.get("call_id", "")
    decision = payload.get("decision", "denied")
    approved = decision == "approved"
    await execution_service.receive_confirmation(execution_id, call_id, approved)
    return {"status": "ok"}


@router.post("/executions/{execution_id}/commit", status_code=200)
async def commit_execution(execution_id: str, payload: dict):
    """Hub → Worker: create a git commit in execution worktree."""
    worktree_root = str(payload.get("worktree_root") or "").strip()
    message = str(payload.get("message") or "").strip()
    git_name = str(payload.get("git_name") or "").strip()
    git_email = str(payload.get("git_email") or "").strip()

    if not execution_id or not worktree_root or not message or not git_name or not git_email:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="execution_id, worktree_root, message, git_name, git_email are required",
            retryable=False,
            status_code=400,
            cause="internal_commit_payload",
        )

    try:
        commit_sha = await WorktreeManager.commit(
            worktree_root=worktree_root,
            message=message,
            git_name=git_name,
            git_email=git_email,
        )
    except WorktreeError as exc:
        raise GoyaisApiError(
            code="E_WORKTREE",
            message=str(exc),
            retryable=False,
            status_code=422,
            cause="worktree_commit_failed",
        ) from exc

    return {"commit_sha": commit_sha}


@router.post("/executions/{execution_id}/discard", status_code=200)
async def discard_execution(execution_id: str, payload: dict):
    """Hub → Worker: discard worktree changes by removing execution worktree."""
    repo_root = str(payload.get("repo_root") or "").strip()
    if not execution_id or not repo_root:
        raise GoyaisApiError(
            code="E_SCHEMA_INVALID",
            message="execution_id and repo_root are required",
            retryable=False,
            status_code=400,
            cause="internal_discard_payload",
        )

    try:
        await WorktreeManager.remove(repo_root=repo_root, execution_id=execution_id, force=True)
    except WorktreeError as exc:
        raise GoyaisApiError(
            code="E_WORKTREE",
            message=str(exc),
            retryable=False,
            status_code=422,
            cause="worktree_discard_failed",
        ) from exc

    return {"status": "ok"}
