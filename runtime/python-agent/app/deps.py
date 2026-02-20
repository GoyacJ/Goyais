from __future__ import annotations

from app.db.repositories import Repository
from app.services.confirmation_service import ConfirmationService
from app.services.run_service import RunService
from app.services.sync_service import SyncService

_repo: Repository | None = None
_run_service: RunService | None = None
_confirmation_service: ConfirmationService | None = None
_sync_service: SyncService | None = None


def set_dependencies(
    repo: Repository,
    run_service: RunService,
    confirmation_service: ConfirmationService,
    sync_service: SyncService,
) -> None:
    global _repo, _run_service, _confirmation_service, _sync_service
    _repo = repo
    _run_service = run_service
    _confirmation_service = confirmation_service
    _sync_service = sync_service


def get_repo() -> Repository:
    if _repo is None:
        raise RuntimeError("Repository not initialized")
    return _repo


def get_run_service() -> RunService:
    if _run_service is None:
        raise RuntimeError("Run service not initialized")
    return _run_service


def get_confirmation_service() -> ConfirmationService:
    if _confirmation_service is None:
        raise RuntimeError("Confirmation service not initialized")
    return _confirmation_service


def get_sync_service() -> SyncService | None:
    return _sync_service
