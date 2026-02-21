from __future__ import annotations

from app.db.repositories import Repository
from app.services.execution_service import ExecutionService

_repo: Repository | None = None
_execution_service: ExecutionService | None = None


def set_dependencies(
    repo: Repository,
    execution_service: ExecutionService,
) -> None:
    global _repo, _execution_service
    _repo = repo
    _execution_service = execution_service


def get_repo() -> Repository:
    if _repo is None:
        raise RuntimeError("Repository not initialized")
    return _repo


def get_execution_service() -> ExecutionService:
    if _execution_service is None:
        raise RuntimeError("ExecutionService not initialized")
    return _execution_service
