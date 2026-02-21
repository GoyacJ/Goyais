"""
execution_service.py — v0.2.0 受控 Worker 执行服务

替代旧执行编排实现。Worker 不再自治管理 session/execution，
而是接受 Hub 调度，执行 agent，通过 HubReporter 上报事件。

关键改变：
- 接收 ExecutionContext（来自 Hub POST /internal/executions）
- 通过 HubReporter 上报事件，不使用本地 EventBus
- 执行结束时发送 done 事件，由 Hub 负责清理 session mutex
"""
from __future__ import annotations

import asyncio
import json
import os
from datetime import datetime, timezone
from typing import Any

from app.agent.graph_agent import AgentState, build_graph, build_plan_with_deepagents
from app.agent.mock_agent import build_mock_plan, compute_readme_patch, new_call_id
from app.agent.prompts import (
    PATCH_SYSTEM_PROMPT,
    PLAN_SYSTEM_PROMPT,
    build_patch_prompt,
    build_plan_prompt,
)
from app.agent.provider_router import build_provider
from app.agent.providers.base import ProviderRequest
from app.db.repositories import Repository
from app.errors import build_goyais_error, error_from_exception
from app.observability.logging import get_runtime_logger
from app.services.hub_reporter import HubReporter
from app.services.secret_resolver import resolve_secret_via_hub
from app.services.worktree_manager import WorktreeManager
from app.tools.file_tools import read_file
from app.tools.patch_tools import apply_patch
from app.tools.policy import requires_confirmation
from unidiff import PatchSet

logger = get_runtime_logger()

PLAN_APPROVAL_CALL_ID = "plan-approval"


class PlanRejectedError(Exception):
    """Raised when the user rejects the agent's plan in plan mode."""


class ExecutionService:
    """
    受控 Worker：接收 Hub 调度的 ExecutionContext，执行 agent，
    通过 HubReporter 上报事件流。

    每次 execute() 调用对应 Hub 侧一个 execution 记录。
    """

    def __init__(self, *, repo: Repository, agent_mode: str) -> None:
        self.repo = repo
        self.agent_mode = agent_mode
        # execution_id → HubReporter
        self._reporters: dict[str, HubReporter] = {}
        # execution_id → confirmation futures
        self._confirmation_futures: dict[str, asyncio.Future] = {}

    async def execute(self, context: dict[str, Any]) -> None:
        """
        Main entry point. context = ExecutionContext JSON from Hub.
        Runs in an asyncio.Task (fire-and-forget from the internal API handler).
        """
        execution_id: str = context["execution_id"]
        trace_id: str = context["trace_id"]
        repo_root: str = context.get("repo_root", os.getcwd())
        use_worktree = bool(context.get("use_worktree"))
        workspace_path = repo_root

        reporter = HubReporter(
            hub_base_url=self._hub_base_url(),
            hub_internal_secret=self._hub_secret(),
            execution_id=execution_id,
        )
        self._reporters[execution_id] = reporter
        reporter.start()

        try:
            if use_worktree:
                workspace_path = await WorktreeManager.create(repo_root, execution_id)

            if self.agent_mode in {"graph", "deepagents"}:
                await self._execute_graph(execution_id, context, workspace_path, reporter)
            else:
                await self._execute_mock(execution_id, context, workspace_path, reporter)

            await reporter.report("done", {"status": "completed", "message": "execution finished"})

        except PlanRejectedError:
            await reporter.report("done", {"status": "cancelled", "message": "Plan rejected by user."})

        except Exception as exc:  # noqa: BLE001
            _, error_payload = error_from_exception(exc, trace_id)
            await reporter.report("error", {"error": error_payload["error"]})
            await reporter.report("done", {"status": "failed", "message": str(exc)})

        finally:
            await reporter.stop()
            self._reporters.pop(execution_id, None)
            self._confirmation_futures.pop(execution_id, None)

    async def receive_confirmation(self, execution_id: str, call_id: str, approved: bool) -> None:
        """Called by the internal confirmations API when Hub forwards a decision."""
        key = f"{execution_id}:{call_id}"
        fut = self._confirmation_futures.get(key)
        if fut and not fut.done():
            fut.set_result(approved)

    # ------------------------------------------------------------------ graph

    async def _execute_graph(
        self,
        execution_id: str,
        context: dict[str, Any],
        workspace_path: str,
        reporter: HubReporter,
    ) -> None:
        provider, model_config, api_key = await self._resolve_provider(context)

        read_call_id = new_call_id()
        await self._emit_tool_call(execution_id, read_call_id, "read_file", {"path": "README.md"}, False, reporter)
        try:
            readme_content = read_file(workspace_path, "README.md")
            await self._emit_tool_result(execution_id, read_call_id, ok=True, output={"content_preview": readme_content[:200]}, reporter=reporter)
        except Exception as exc:  # noqa: BLE001
            await self._emit_tool_result(execution_id, read_call_id, ok=False, exc=exc, reporter=reporter)
            raise

        async def plan_builder(state: AgentState) -> dict[str, Any]:
            return await self._generate_plan(
                task_input=state.task_input,
                readme_content=state.readme_content,
                model_config=model_config,
                provider=provider,
                api_key=api_key,
            )

        async def patch_builder(state: AgentState) -> str:
            return await self._generate_patch(
                workspace_path=state.workspace_path,
                task_input=state.task_input,
                readme_content=state.readme_content,
                model_config=model_config,
                provider=provider,
            )

        compiled = build_graph(plan_builder=plan_builder, patch_builder=patch_builder)
        state = AgentState(
            task_input=context["user_message"],
            workspace_path=workspace_path,
            readme_content=readme_content,
        )
        result = await compiled.ainvoke(state)
        plan_payload = result.plan or build_mock_plan(context["user_message"])
        patch = result.patch or compute_readme_patch(workspace_path, context["user_message"])

        await reporter.report("plan", plan_payload)

        if context.get("mode") == "plan":
            await self._await_plan_confirmation(execution_id, context, plan_payload, reporter)

        await self._emit_patch_flow(execution_id, context, workspace_path, patch, reporter)

    # ------------------------------------------------------------------ mock

    async def _execute_mock(
        self,
        execution_id: str,
        context: dict[str, Any],
        workspace_path: str,
        reporter: HubReporter,
    ) -> None:
        plan_payload = build_mock_plan(context["user_message"])
        await reporter.report("plan", plan_payload)

        if context.get("mode") == "plan":
            await self._await_plan_confirmation(execution_id, context, plan_payload, reporter)

        read_call_id = new_call_id()
        await self._emit_tool_call(execution_id, read_call_id, "read_file", {"path": "README.md"}, False, reporter)
        try:
            content = read_file(workspace_path, "README.md")
            await self._emit_tool_result(execution_id, read_call_id, ok=True, output={"content_preview": content[:200]}, reporter=reporter)
        except Exception as exc:  # noqa: BLE001
            await self._emit_tool_result(execution_id, read_call_id, ok=False, exc=exc, reporter=reporter)
            raise

        patch = compute_readme_patch(workspace_path, context["user_message"])
        await self._emit_patch_flow(execution_id, context, workspace_path, patch, reporter)

    # ------------------------------------------------------------------ patch flow

    async def _emit_patch_flow(
        self,
        execution_id: str,
        context: dict[str, Any],
        workspace_path: str,
        patch: str,
        reporter: HubReporter,
    ) -> None:
        await reporter.report("patch", {"unified_diff": patch})

        apply_call_id = new_call_id()
        must_confirm = requires_confirmation("apply_patch")
        await self._emit_tool_call(execution_id, apply_call_id, "apply_patch", {"unified_diff": patch[:200]}, must_confirm, reporter)

        approved = True
        if must_confirm:
            approved = await self._wait_for_confirmation(execution_id, apply_call_id)

        if not approved:
            await self._emit_tool_result(
                execution_id, apply_call_id, ok=False,
                error=build_goyais_error(
                    code="E_TOOL_DENIED",
                    message="User denied apply_patch confirmation.",
                    trace_id=context.get("trace_id", ""),
                    retryable=False,
                    cause="user_denied",
                ),
                reporter=reporter,
            )
            return

        try:
            output = apply_patch(workspace_path, patch)
            await self._emit_tool_result(execution_id, apply_call_id, ok=True, output={"message": output}, reporter=reporter)
        except Exception as exc:  # noqa: BLE001
            await self._emit_tool_result(execution_id, apply_call_id, ok=False, exc=exc, reporter=reporter)
            raise

    # ------------------------------------------------------------------ helpers

    async def _await_plan_confirmation(
        self,
        execution_id: str,
        context: dict[str, Any],
        plan_payload: dict[str, Any],
        reporter: HubReporter,
    ) -> None:
        """Send a plan confirmation_request and wait for the user decision.

        Raises PlanRejectedError if the user denies the plan.
        Only called when context["mode"] == "plan".
        """
        await reporter.report("confirmation_request", {
            "call_id": PLAN_APPROVAL_CALL_ID,
            "tool_name": "plan_approval",
            "risk_level": "medium",
            "parameters_summary": json.dumps(plan_payload)[:500],
        })
        # Wait up to 10 minutes for the user to approve / reject the plan
        approved = await self._wait_for_confirmation(execution_id, PLAN_APPROVAL_CALL_ID, timeout=600.0)
        if not approved:
            raise PlanRejectedError("User rejected the execution plan.")

    async def _wait_for_confirmation(self, execution_id: str, call_id: str, timeout: float = 300.0) -> bool:
        key = f"{execution_id}:{call_id}"
        loop = asyncio.get_event_loop()
        fut: asyncio.Future = loop.create_future()
        self._confirmation_futures[key] = fut
        try:
            return await asyncio.wait_for(fut, timeout=timeout)
        except asyncio.TimeoutError:
            return False
        finally:
            self._confirmation_futures.pop(key, None)

    async def _emit_tool_call(
        self,
        execution_id: str,
        call_id: str,
        tool_name: str,
        args: dict[str, Any],
        must_confirm: bool,
        reporter: HubReporter,
    ) -> None:
        payload: dict[str, Any] = {
            "call_id": call_id,
            "tool_name": tool_name,
            "args": args,
            "requires_confirmation": must_confirm,
        }
        if must_confirm:
            # Signal Hub to create execution_confirmations record
            await reporter.report("confirmation_request", {
                "call_id": call_id,
                "tool_name": tool_name,
                "risk_level": "high",
                "parameters_summary": json.dumps(args)[:200],
            })
        await reporter.report("tool_call", payload)

    async def _emit_tool_result(
        self,
        execution_id: str,
        call_id: str,
        *,
        ok: bool,
        output: Any | None = None,
        error: dict[str, Any] | None = None,
        exc: Exception | None = None,
        reporter: HubReporter,
    ) -> None:
        payload: dict[str, Any] = {"call_id": call_id, "ok": ok}
        if ok:
            payload["output"] = output
        else:
            if error is None and exc is not None:
                _, err_payload = error_from_exception(exc, "")
                error = err_payload["error"]
            payload["error"] = error or {"code": "E_INTERNAL", "message": "Tool failed"}
        await reporter.report("tool_result", payload)

    async def _resolve_provider(self, context: dict[str, Any]):
        model_config_id = (context.get("model_config_id") or "").strip()
        if not model_config_id:
            raise RuntimeError("model_config_id is required for graph/deepagents mode")
        model_config = await self.repo.get_model_config(model_config_id)
        if model_config is None:
            raise RuntimeError(f"model config not found: {model_config_id}")
        api_key = await self._resolve_secret(str(model_config.get("secret_ref", "")), context.get("trace_id", ""))
        provider = build_provider(str(model_config["provider"]), api_key, model_config.get("base_url"))
        return provider, model_config, api_key

    async def _resolve_secret(self, secret_ref: str, trace_id: str) -> str:
        if not secret_ref:
            raise RuntimeError("missing secret_ref")
        if secret_ref.startswith("secret:"):
            return await resolve_secret_via_hub(secret_ref, trace_id)
        if secret_ref.startswith("env:"):
            env_key = secret_ref.split(":", 1)[1]
        elif secret_ref.startswith("keychain:"):
            parts = secret_ref.split(":")
            if len(parts) != 3:
                raise RuntimeError(f"invalid secret_ref: {secret_ref}")
            provider, profile = parts[1], parts[2]
            env_key = f"GOYAIS_SECRET_{provider.upper()}_{profile.upper()}"
        else:
            env_key = secret_ref
        value = os.getenv(env_key)
        if not value:
            raise RuntimeError(f"API key not found for '{secret_ref}', set env '{env_key}'")
        return value

    async def _generate_plan(self, *, task_input, readme_content, model_config, provider, api_key) -> dict[str, Any]:
        if self.agent_mode == "deepagents":
            try:
                return await build_plan_with_deepagents(
                    task_input,
                    provider=str(model_config["provider"]),
                    model=str(model_config["model"]),
                    api_key=api_key,
                )
            except Exception:  # noqa: BLE001
                pass
        plan_text = await provider.complete(
            ProviderRequest(
                model=str(model_config["model"]),
                input_text=build_plan_prompt(task_input, readme_content[:1500]),
                system_prompt=PLAN_SYSTEM_PROMPT,
                max_tokens=model_config.get("max_tokens"),
                temperature=float(model_config.get("temperature", 0)),
            )
        )
        lines = [l.strip("- ").strip() for l in plan_text.splitlines() if l.strip()]
        if not lines:
            return build_mock_plan(task_input)
        return {"summary": lines[0][:400], "steps": lines[1:6] or build_mock_plan(task_input)["steps"]}

    async def _generate_patch(self, *, workspace_path, task_input, readme_content, model_config, provider) -> str:
        patch_text = await provider.complete(
            ProviderRequest(
                model=str(model_config["model"]),
                input_text=build_patch_prompt(task_input, readme_content),
                system_prompt=PATCH_SYSTEM_PROMPT,
                max_tokens=model_config.get("max_tokens"),
                temperature=float(model_config.get("temperature", 0)),
            )
        )
        candidate = self._extract_unified_diff(patch_text)
        try:
            if PatchSet(candidate):
                return candidate
        except Exception:  # noqa: BLE001
            pass
        return compute_readme_patch(workspace_path, task_input)

    def _extract_unified_diff(self, text: str) -> str:
        candidate = text.strip()
        if "```" in candidate:
            for segment in candidate.split("```"):
                stripped = segment.strip()
                if stripped.startswith("diff"):
                    stripped = stripped[4:].lstrip()
                if stripped.startswith("--- "):
                    candidate = stripped
                    break
        if not candidate.startswith("--- "):
            for i, line in enumerate(candidate.splitlines()):
                if line.startswith("--- "):
                    candidate = "\n".join(candidate.splitlines()[i:])
                    break
        if not candidate.endswith("\n"):
            candidate += "\n"
        return candidate

    @staticmethod
    def _hub_base_url() -> str:
        return os.getenv("GOYAIS_HUB_BASE_URL", "http://127.0.0.1:8080").rstrip("/")

    @staticmethod
    def _hub_secret() -> str:
        return os.getenv("GOYAIS_HUB_INTERNAL_SECRET", "")
