"""
execution_service.py — v0.3.0 Agentic Worker

Replaces the old linear plan→patch pipeline with a true agentic loop.
The LLM calls tools autonomously, receives results, and loops until done.

Two backends switchable via GOYAIS_AGENT_MODE:
  - "vanilla"   (default): pure asyncio while-loop (app/agent/loop.py)
  - "langgraph"           : LangGraph ReAct StateGraph

Preserved from v0.2.0:
  - HubReporter event protocol
  - WorktreeManager git isolation
  - receive_confirmation() / _wait_for_confirmation() asyncio.Future flow
  - _await_plan_confirmation() plan approval
  - _resolve_provider() / _resolve_secret() key resolution
  - Error handling / done event shape
"""
from __future__ import annotations

import asyncio
import json
import os
import uuid
from typing import Any

from app.agent.loop import LoopCallbacks, agent_loop
from app.agent.messages import Message, ToolCall
from app.agent.provider_router import build_provider
from app.agent.system_prompts import build_system_prompt
from app.agent.tool_registry import ToolRegistry, build_builtin_tools
from app.db.repositories import Repository
from app.errors import error_from_exception
from app.observability.logging import get_runtime_logger
from app.services.hub_reporter import HubReporter
from app.services.secret_resolver import resolve_secret_via_hub
from app.services.tool_injector import ToolInjector
from app.services.worktree_manager import WorktreeManager

logger = get_runtime_logger()

PLAN_APPROVAL_CALL_ID = "plan-approval"


class PlanRejectedError(Exception):
    """Raised when the user rejects the agent's plan in plan mode."""


class ExecutionService:
    """
    Controlled Worker: receives Hub-scheduled ExecutionContext, runs the
    agentic loop, and streams events back via HubReporter.
    """

    def __init__(self, *, repo: Repository, agent_mode: str) -> None:
        self.repo = repo
        self.agent_mode = agent_mode  # "vanilla" | "langgraph"
        self._reporters: dict[str, HubReporter] = {}
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

            await self._execute_agent(execution_id, context, workspace_path, reporter)
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

    # ------------------------------------------------------------------ core

    async def _execute_agent(
        self,
        execution_id: str,
        context: dict[str, Any],
        workspace_path: str,
        reporter: HubReporter,
    ) -> None:
        provider, model_config, _ = await self._resolve_provider(context)
        model = str(model_config["model"])
        mode = context.get("mode", "auto")

        # Build tool registry
        registry = ToolRegistry()
        for td in build_builtin_tools(workspace_path):
            registry.register(td)

        # Inject Hub Skills / MCP tools
        try:
            injector = ToolInjector(self._hub_base_url(), self._hub_secret())
            hub_tools = await injector.resolve_tools(context)
            for td in hub_tools:
                registry.register(td)
        except Exception as exc:  # noqa: BLE001
            logger.warning("tool injection failed, continuing without hub tools: %s", exc)

        # Build system prompt
        system_prompt = build_system_prompt(
            mode=mode,
            workspace_path=workspace_path,
            tool_registry=registry,
        )

        user_msg = context["user_message"]

        if self.agent_mode == "langgraph":
            await self._run_langgraph(
                execution_id, context, provider, model,
                registry, system_prompt, user_msg, reporter,
            )
        else:
            await self._run_vanilla(
                execution_id, context, provider, model,
                registry, system_prompt, user_msg, reporter,
            )

    # ------------------------------------------------------------------ vanilla

    async def _run_vanilla(
        self,
        execution_id: str,
        context: dict[str, Any],
        provider: Any,
        model: str,
        registry: ToolRegistry,
        system_prompt: str,
        user_msg: str,
        reporter: HubReporter,
    ) -> None:
        messages: list[Message] = [Message(role="user", content=user_msg)]
        callbacks = self._build_callbacks(execution_id, reporter)

        if context.get("mode") == "plan":
            plan_response = await agent_loop(
                provider=provider,
                model=model,
                system_prompt=system_prompt,
                messages=messages,
                tools=registry,
                callbacks=callbacks,
            )
            plan_payload = {
                "summary": plan_response.text[:400],
                "full_text": plan_response.text,
            }
            await reporter.report("plan", plan_payload)
            await self._await_plan_confirmation(execution_id, context, plan_payload, reporter)
            messages.append(Message(role="user", content="Plan approved. Now implement it."))

        await agent_loop(
            provider=provider,
            model=model,
            system_prompt=system_prompt,
            messages=messages,
            tools=registry,
            callbacks=callbacks,
        )

    # ------------------------------------------------------------------ langgraph

    async def _run_langgraph(
        self,
        execution_id: str,
        context: dict[str, Any],
        provider: Any,
        model: str,
        registry: ToolRegistry,
        system_prompt: str,
        user_msg: str,
        reporter: HubReporter,
    ) -> None:
        from langchain_core.messages import HumanMessage

        from app.agent.langgraph_graph import build_react_graph

        graph = build_react_graph()

        lg_config: dict[str, Any] = {"configurable": {
            "provider": provider,
            "model": model,
            "system_prompt": system_prompt,
            "tool_registry": registry,
            "reporter": reporter,
            "execution_id": execution_id,
            "confirmation_handler": lambda tc: self._wait_for_tool_confirmation(execution_id, tc),
        }}

        initial_state: dict[str, Any] = {
            "messages": [HumanMessage(content=user_msg)],
            "workspace_path": context.get("repo_root", ""),
            "mode": context.get("mode", "auto"),
            "iteration_count": 0,
        }

        if context.get("mode") == "plan":
            result = await graph.ainvoke(initial_state, config=lg_config)
            last_msgs = result.get("messages", [])
            plan_text = last_msgs[-1].content if last_msgs else ""
            plan_payload = {"summary": str(plan_text)[:400], "full_text": str(plan_text)}
            await reporter.report("plan", plan_payload)
            await self._await_plan_confirmation(execution_id, context, plan_payload, reporter)
            # Append approval and re-run for execution
            result["messages"].append(HumanMessage(content="Plan approved. Now implement it."))
            await graph.ainvoke(result, config=lg_config)
        else:
            await graph.ainvoke(initial_state, config=lg_config)

    # ------------------------------------------------------------------ callbacks

    def _build_callbacks(self, execution_id: str, reporter: HubReporter) -> LoopCallbacks:
        return LoopCallbacks(
            on_tool_call=lambda tc, needs_confirm: self._emit_tool_call(
                execution_id, tc, needs_confirm, reporter
            ),
            on_tool_result=lambda call_id, output, is_err: self._emit_tool_result(
                execution_id, call_id, output, is_err, reporter
            ),
            on_confirmation_needed=lambda tc: self._wait_for_tool_confirmation(execution_id, tc),
        )

    # ------------------------------------------------------------------ plan confirmation

    async def _await_plan_confirmation(
        self,
        execution_id: str,
        context: dict[str, Any],
        plan_payload: dict[str, Any],
        reporter: HubReporter,
    ) -> None:
        await reporter.report("confirmation_request", {
            "call_id": PLAN_APPROVAL_CALL_ID,
            "tool_name": "plan_approval",
            "risk_level": "medium",
            "parameters_summary": json.dumps(plan_payload)[:500],
        })
        approved = await self._wait_for_confirmation(execution_id, PLAN_APPROVAL_CALL_ID, timeout=600.0)
        if not approved:
            raise PlanRejectedError("User rejected the execution plan.")

    # ------------------------------------------------------------------ tool events

    async def _emit_tool_call(
        self,
        execution_id: str,
        tc: ToolCall,
        needs_confirm: bool,
        reporter: HubReporter,
    ) -> None:
        if needs_confirm:
            await reporter.report("confirmation_request", {
                "call_id": tc.id,
                "tool_name": tc.name,
                "risk_level": "high",
                "parameters_summary": json.dumps(tc.input)[:200],
            })
        await reporter.report("tool_call", {
            "call_id": tc.id,
            "tool_name": tc.name,
            "args": tc.input,
            "requires_confirmation": needs_confirm,
        })

    async def _emit_tool_result(
        self,
        execution_id: str,
        call_id: str,
        output: str,
        is_error: bool,
        reporter: HubReporter,
    ) -> None:
        payload: dict[str, Any] = {"call_id": call_id, "ok": not is_error}
        if is_error:
            payload["error"] = {"code": "E_TOOL_FAILED", "message": output[:500]}
        else:
            payload["output"] = {"result": output[:2000]}
        await reporter.report("tool_result", payload)

    # ------------------------------------------------------------------ confirmation futures

    async def _wait_for_tool_confirmation(self, execution_id: str, tc: ToolCall) -> bool:
        return await self._wait_for_confirmation(execution_id, tc.id, timeout=300.0)

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

    # ------------------------------------------------------------------ provider resolution

    async def _resolve_provider(self, context: dict[str, Any]):
        model_config_id = (context.get("model_config_id") or "").strip()
        if not model_config_id:
            raise RuntimeError("model_config_id is required")
        model_config = await self.repo.get_model_config(model_config_id)
        if model_config is None:
            raise RuntimeError(f"model config not found: {model_config_id}")
        api_key = await self._resolve_secret(
            str(model_config.get("secret_ref", "")), context.get("trace_id", "")
        )
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
            prov, profile = parts[1], parts[2]
            env_key = f"GOYAIS_SECRET_{prov.upper()}_{profile.upper()}"
        else:
            env_key = secret_ref
        value = os.getenv(env_key)
        if not value:
            raise RuntimeError(f"API key not found for '{secret_ref}', set env '{env_key}'")
        return value

    # ------------------------------------------------------------------ statics

    @staticmethod
    def _hub_base_url() -> str:
        return os.getenv("GOYAIS_HUB_BASE_URL", "http://127.0.0.1:8080").rstrip("/")

    @staticmethod
    def _hub_secret() -> str:
        return os.getenv("GOYAIS_HUB_INTERNAL_SECRET", "")
