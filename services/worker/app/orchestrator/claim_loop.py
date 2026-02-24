from __future__ import annotations

import asyncio
import logging
import os
import time
import uuid
from typing import Any

from app.hub_client import HubClient, HubRequestError
from app.runtime.langgraph_runtime import LangGraphRuntime
from app.runtime.vanilla import VanillaRuntime
from app.worktree.manager import WorktreeManager

logger = logging.getLogger("goyais.worker")


class ClaimLoopService:
    def __init__(self) -> None:
        self.hub = HubClient()
        self.worker_id = os.getenv("WORKER_ID", f"worker-{uuid.uuid4().hex[:8]}")
        self.max_concurrency = max(1, int(os.getenv("WORKER_MAX_CONCURRENCY", "3")))
        self.lease_seconds = max(10, int(os.getenv("WORKER_LEASE_SECONDS", "30")))
        self.claim_interval_ms = max(100, int(os.getenv("WORKER_CLAIM_INTERVAL_MS", "500")))
        self.heartbeat_interval_seconds = max(3, int(os.getenv("WORKER_HEARTBEAT_SECONDS", "10")))
        self._running = False
        self._claim_task: asyncio.Task[None] | None = None
        self._heartbeat_task: asyncio.Task[None] | None = None
        self._active_tasks: set[asyncio.Task[None]] = set()
        self._runtime = self._resolve_runtime()
        self._worktree = WorktreeManager()

    async def start(self) -> None:
        if self._running:
            return
        self._running = True
        try:
            await self.hub.register_worker(
                self.worker_id,
                {
                    "runtime": os.getenv("WORKER_RUNTIME", "vanilla").strip().lower(),
                    "max_concurrency": self.max_concurrency,
                },
            )
        except Exception as exc:
            logger.warning("worker register failed: %s", exc)

        self._heartbeat_task = asyncio.create_task(self._heartbeat_loop())
        self._claim_task = asyncio.create_task(self._claim_loop())

    async def stop(self) -> None:
        self._running = False
        for task in (self._claim_task, self._heartbeat_task):
            if task is not None:
                task.cancel()
        for task in list(self._active_tasks):
            task.cancel()
        await asyncio.gather(*(t for t in [self._claim_task, self._heartbeat_task] if t is not None), return_exceptions=True)
        await asyncio.gather(*self._active_tasks, return_exceptions=True)
        self._active_tasks.clear()

    async def _heartbeat_loop(self) -> None:
        while self._running:
            try:
                await self.hub.heartbeat(self.worker_id, "active")
            except Exception as exc:
                logger.warning("worker heartbeat failed: %s", exc)
            await asyncio.sleep(self.heartbeat_interval_seconds)

    async def _claim_loop(self) -> None:
        while self._running:
            if len(self._active_tasks) >= self.max_concurrency:
                await asyncio.sleep(self.claim_interval_ms / 1000.0)
                continue
            try:
                response = await self.hub.claim_execution(self.worker_id, self.lease_seconds)
            except Exception as exc:
                logger.warning("execution claim failed: %s", exc)
                await asyncio.sleep(self.claim_interval_ms / 1000.0)
                continue

            claimed = bool(response.get("claimed"))
            envelope = response.get("execution")
            if not claimed or not isinstance(envelope, dict):
                await asyncio.sleep(self.claim_interval_ms / 1000.0)
                continue

            task = asyncio.create_task(self._run_claimed_execution(envelope))
            self._active_tasks.add(task)
            task.add_done_callback(lambda done: self._active_tasks.discard(done))

    async def _run_claimed_execution(self, envelope: dict[str, Any]) -> None:
        execution = envelope.get("execution")
        if not isinstance(execution, dict):
            return
        execution_id = str(execution.get("execution_id") or execution.get("id") or "").strip()
        if execution_id == "":
            return

        execution = dict(execution)
        execution["execution_id"] = execution_id
        if str(execution.get("id") or "").strip() == "":
            execution["id"] = execution_id
        execution["content"] = str(envelope.get("content") or execution.get("content") or "")
        execution["queue_index"] = int(execution.get("queue_index") or 0)
        execution.setdefault("trace_id", f"tr_worker_{execution_id}")

        project_path = str(envelope.get("project_path") or "").strip()
        project_name = str(envelope.get("project_name") or "").strip()
        if project_name == "" and project_path != "":
            project_name = os.path.basename(project_path.rstrip("/")) or ""
        project_is_git = bool(envelope.get("project_is_git"))
        worktree = self._worktree.prepare(execution_id, project_path, project_is_git)
        execution["working_directory"] = worktree.path
        execution["project_path"] = project_path
        execution["project_name"] = project_name

        controls = _ExecutionControls(self.hub, execution_id)
        await controls.start()
        emitter = _ExecutionEventEmitter(self.hub, execution)
        try:
            await self._runtime.run(
                execution=execution,
                emit_event=emitter.emit,
                is_cancelled=controls.is_cancelled,
            )
        except Exception as exc:
            await emitter.emit(
                execution,
                "execution_error",
                {"reason": "WORKER_ORCHESTRATOR_ERROR", "message": str(exc)},
            )
        finally:
            await controls.stop()
            self._worktree.cleanup(worktree, project_path, project_is_git)

    def _resolve_runtime(self):
        runtime_mode = os.getenv("WORKER_RUNTIME", "vanilla").strip().lower()
        if runtime_mode == "langgraph":
            return LangGraphRuntime()
        return VanillaRuntime()


class _ExecutionEventEmitter:
    def __init__(self, hub: HubClient, execution: dict[str, Any]) -> None:
        self.hub = hub
        self.execution = execution
        self.execution_id = str(execution.get("execution_id") or execution.get("id") or "").strip()
        self.conversation_id = str(execution.get("conversation_id") or "").strip()
        self.trace_id = str(execution.get("trace_id") or "").strip()
        self.queue_index = int(execution.get("queue_index") or 0)
        self.sequence = 0

    async def emit(self, execution: dict[str, Any], event_type: str, payload: dict[str, Any]) -> None:
        self.sequence += 1
        event = {
            "event_id": f"evt_{self.execution_id}_{self.sequence}",
            "execution_id": self.execution_id,
            "conversation_id": self.conversation_id,
            "trace_id": self.trace_id,
            "sequence": self.sequence,
            "queue_index": self.queue_index,
            "type": event_type,
            "timestamp": _now_iso(),
            "payload": payload,
        }
        try:
            await self.hub.send_events_batch(self.execution_id, [event])
        except Exception as exc:
            logger.warning("event batch send failed execution=%s: %s", self.execution_id, exc)


class _ExecutionControls:
    def __init__(self, hub: HubClient, execution_id: str) -> None:
        self.hub = hub
        self.execution_id = execution_id
        self.after_seq = 0
        self._running = False
        self._task: asyncio.Task[None] | None = None
        self._cancelled = False

    async def start(self) -> None:
        self._running = True
        self._task = asyncio.create_task(self._poll_loop())

    async def stop(self) -> None:
        self._running = False
        if self._task is not None:
            self._task.cancel()
            await asyncio.gather(self._task, return_exceptions=True)

    def is_cancelled(self, _: str) -> bool:
        return self._cancelled

    async def _poll_loop(self) -> None:
        while self._running:
            try:
                response = await self.hub.poll_control(self.execution_id, self.after_seq, 2000)
            except HubRequestError as exc:
                if exc.status_code == 404 and "EXECUTION_NOT_FOUND" in exc.body_text:
                    self._cancelled = True
                    self._running = False
                    logger.info(
                        "control poll closed execution=%s because execution no longer exists",
                        self.execution_id,
                    )
                    return
                logger.warning("control poll failed execution=%s: %s", self.execution_id, exc)
                await asyncio.sleep(0.5)
                continue
            except Exception as exc:
                logger.warning("control poll failed execution=%s: %s", self.execution_id, exc)
                await asyncio.sleep(0.5)
                continue

            last_seq = int(response.get("last_seq") or self.after_seq)
            if last_seq > self.after_seq:
                self.after_seq = last_seq
            commands = response.get("commands")
            if not isinstance(commands, list):
                continue
            for command in commands:
                if not isinstance(command, dict):
                    continue
                command_type = str(command.get("type") or "").strip().lower()
                if command_type == "stop":
                    self._cancelled = True


def _now_iso() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
