from __future__ import annotations

from app.services.worktree_manager import WorktreeError, WorktreeManager


def test_internal_execution_commit_success(isolated_client, monkeypatch):
    async def fake_commit(cls, worktree_root: str, message: str, git_name: str, git_email: str) -> str:
        assert worktree_root == "/tmp/wt-e1"
        assert message == "feat: test"
        assert git_name == "Alice"
        assert git_email == "alice@example.com"
        return "abc123def"

    monkeypatch.setattr(WorktreeManager, "commit", classmethod(fake_commit))

    resp = isolated_client.post(
        "/internal/executions/e1/commit",
        json={
            "worktree_root": "/tmp/wt-e1",
            "message": "feat: test",
            "git_name": "Alice",
            "git_email": "alice@example.com",
        },
    )

    assert resp.status_code == 200
    assert resp.json() == {"commit_sha": "abc123def"}


def test_internal_execution_commit_validation_error(isolated_client):
    resp = isolated_client.post(
        "/internal/executions/e1/commit",
        json={
            "worktree_root": "",
            "message": "",
            "git_name": "",
            "git_email": "",
        },
    )

    assert resp.status_code == 400
    assert resp.json()["error"]["code"] == "E_SCHEMA_INVALID"


def test_internal_execution_commit_worktree_error(isolated_client, monkeypatch):
    async def fake_commit(cls, worktree_root: str, message: str, git_name: str, git_email: str) -> str:
        raise WorktreeError("git commit failed")

    monkeypatch.setattr(WorktreeManager, "commit", classmethod(fake_commit))

    resp = isolated_client.post(
        "/internal/executions/e2/commit",
        json={
            "worktree_root": "/tmp/wt-e2",
            "message": "feat: test",
            "git_name": "Bob",
            "git_email": "bob@example.com",
        },
    )

    assert resp.status_code == 422
    assert resp.json()["error"]["code"] == "E_WORKTREE"


def test_internal_execution_discard_success(isolated_client, monkeypatch):
    async def fake_remove(cls, repo_root: str, execution_id: str, *, force: bool = True) -> None:
        assert repo_root == "/repo/main"
        assert execution_id == "e3"
        assert force is True

    monkeypatch.setattr(WorktreeManager, "remove", classmethod(fake_remove))

    resp = isolated_client.post(
        "/internal/executions/e3/discard",
        json={"repo_root": "/repo/main"},
    )
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


def test_internal_execution_discard_validation_error(isolated_client):
    resp = isolated_client.post(
        "/internal/executions/e3/discard",
        json={"repo_root": ""},
    )
    assert resp.status_code == 400
    assert resp.json()["error"]["code"] == "E_SCHEMA_INVALID"


def test_internal_execution_discard_worktree_error(isolated_client, monkeypatch):
    async def fake_remove(cls, repo_root: str, execution_id: str, *, force: bool = True) -> None:
        raise WorktreeError("worktree remove failed")

    monkeypatch.setattr(WorktreeManager, "remove", classmethod(fake_remove))

    resp = isolated_client.post(
        "/internal/executions/e3/discard",
        json={"repo_root": "/repo/main"},
    )
    assert resp.status_code == 422
    assert resp.json()["error"]["code"] == "E_WORKTREE"
