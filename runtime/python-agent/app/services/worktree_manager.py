"""
worktree_manager.py — 管理 execution 级别的 git worktree

每次执行创建独立 worktree 目录，执行完成后按策略清理：
- 成功：默认清理
- 失败：默认保留（便于诊断）
"""
from __future__ import annotations

import asyncio
import logging
import os
from pathlib import Path

logger = logging.getLogger(__name__)

WORKTREE_BASE = ".goyais/worktrees"


class WorktreeError(Exception):
    pass


async def _run_git(args: list[str], cwd: str) -> str:
    """Run a git command and return stdout."""
    proc = await asyncio.create_subprocess_exec(
        "git",
        *args,
        cwd=cwd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    stdout, stderr = await proc.communicate()
    if proc.returncode != 0:
        raise WorktreeError(
            f"git {' '.join(args)} failed (rc={proc.returncode}): {stderr.decode().strip()}"
        )
    return stdout.decode().strip()


class WorktreeManager:
    """Lifecycle manager for per-execution git worktrees."""

    @staticmethod
    def worktree_root(repo_root: str, execution_id: str) -> str:
        return str(Path(repo_root) / WORKTREE_BASE / execution_id)

    @staticmethod
    def branch_name(execution_id: str) -> str:
        return f"goyais-exec-{execution_id}"

    @classmethod
    async def create(cls, repo_root: str, execution_id: str) -> str:
        """
        Create an isolated worktree for the given execution.
        Returns the worktree_root path.
        Raises WorktreeError on failure.
        """
        worktree_root = cls.worktree_root(repo_root, execution_id)
        branch = cls.branch_name(execution_id)

        # Ensure base dir exists
        Path(worktree_root).parent.mkdir(parents=True, exist_ok=True)

        await _run_git(
            ["worktree", "add", worktree_root, "-b", branch],
            cwd=repo_root,
        )
        logger.info(
            "worktree_created execution_id=%s path=%s branch=%s",
            execution_id,
            worktree_root,
            branch,
        )
        return worktree_root

    @classmethod
    async def remove(cls, repo_root: str, execution_id: str, *, force: bool = True) -> None:
        """
        Remove the worktree and delete the temporary branch.
        Idempotent — logs and returns if already gone.
        """
        worktree_root = cls.worktree_root(repo_root, execution_id)
        branch = cls.branch_name(execution_id)

        if not Path(worktree_root).exists():
            logger.debug("worktree_remove: path already gone execution_id=%s", execution_id)
            return

        try:
            force_flag = ["--force"] if force else []
            await _run_git(["worktree", "remove", worktree_root, *force_flag], cwd=repo_root)
        except WorktreeError as exc:
            logger.warning("worktree_remove failed: %s", exc)

        # Delete the temporary branch (best-effort)
        try:
            await _run_git(["branch", "-D", branch], cwd=repo_root)
        except WorktreeError:
            pass

        logger.info("worktree_removed execution_id=%s", execution_id)

    @classmethod
    async def commit(
        cls,
        worktree_root: str,
        message: str,
        git_name: str,
        git_email: str,
    ) -> str:
        """
        Stage all changes and create a local commit in the worktree.
        author/committer are set from git_name/git_email.
        Returns the commit SHA.
        """
        env = {
            **os.environ,
            "GIT_AUTHOR_NAME": git_name,
            "GIT_AUTHOR_EMAIL": git_email,
            "GIT_COMMITTER_NAME": git_name,
            "GIT_COMMITTER_EMAIL": git_email,
        }

        # Stage all changes
        proc_add = await asyncio.create_subprocess_exec(
            "git", "add", "-A",
            cwd=worktree_root,
            env=env,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        _, stderr_add = await proc_add.communicate()
        if proc_add.returncode != 0:
            raise WorktreeError(f"git add -A failed: {stderr_add.decode().strip()}")

        # Check if there is anything to commit
        proc_status = await asyncio.create_subprocess_exec(
            "git", "diff", "--cached", "--quiet",
            cwd=worktree_root,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        await proc_status.communicate()
        if proc_status.returncode == 0:
            raise WorktreeError("nothing to commit — working tree is clean")

        # Commit
        proc_commit = await asyncio.create_subprocess_exec(
            "git", "commit", "-m", message,
            cwd=worktree_root,
            env=env,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        _, stderr_commit = await proc_commit.communicate()
        if proc_commit.returncode != 0:
            raise WorktreeError(f"git commit failed: {stderr_commit.decode().strip()}")

        # Return commit SHA
        proc_rev = await asyncio.create_subprocess_exec(
            "git", "rev-parse", "HEAD",
            cwd=worktree_root,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout_rev, _ = await proc_rev.communicate()
        return stdout_rev.decode().strip()

    @classmethod
    async def diff(cls, worktree_root: str) -> str:
        """
        Return a unified diff of all uncommitted changes in the worktree
        (staged + unstaged vs HEAD).
        """
        proc = await asyncio.create_subprocess_exec(
            "git", "diff", "HEAD",
            cwd=worktree_root,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await proc.communicate()
        result = stdout.decode()
        if not result.strip():
            # Try staged-only diff (after commit)
            proc2 = await asyncio.create_subprocess_exec(
                "git", "show", "HEAD",
                cwd=worktree_root,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout2, _ = await proc2.communicate()
            result = stdout2.decode()
        return result
