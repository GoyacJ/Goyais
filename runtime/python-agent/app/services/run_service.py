from __future__ import annotations

import asyncio
import json
import os
import uuid
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
from app.observability.metrics import get_runtime_metrics
from app.services.audit_service import AuditService
from app.services.confirmation_service import ConfirmationService
from app.sse.event_bus import EventBus
from app.tools.command_tools import run_command
from app.tools.file_tools import read_file
from app.tools.patch_tools import apply_patch
from app.tools.policy import requires_confirmation
from app.trace import generate_trace_id
from unidiff import PatchSet

PROTOCOL_VERSION = "2.0.0"
logger = get_runtime_logger()
metrics = get_runtime_metrics()


class RunService:
    def __init__(
        self,
        *,
        repo: Repository,
        bus: EventBus,
        confirmation_service: ConfirmationService,
        audit_service: AuditService,
        agent_mode: str,
    ):
        self.repo = repo
        self.bus = bus
        self.confirmation_service = confirmation_service
        self.audit_service = audit_service
        self.agent_mode = agent_mode
        self._run_traces: dict[str, str] = {}

    async def recover_pending_confirmations_after_restart(self) -> None:
        pending = await self.repo.list_pending_confirmations()
        for item in pending:
            run_id = str(item["run_id"])
            call_id = str(item["call_id"])
            trace_id = await self._trace_id_for_run(run_id)
            await self.repo.upsert_tool_confirmation_status(run_id, call_id, "denied", decided_by="system")
            await self.repo.update_run_status(run_id, "failed")

            runtime_error = build_goyais_error(
                code="E_INTERNAL",
                message="Runtime restarted while waiting for confirmation.",
                trace_id=trace_id,
                retryable=False,
                cause="confirmation_recovery",
            )
            error_event = await self.emit_event(
                run_id,
                "error",
                {
                    "error": runtime_error,
                },
            )
            await self.audit_service.record(
                trace_id=trace_id,
                run_id=run_id,
                event_id=error_event["event_id"],
                call_id=call_id,
                action="confirmation_recovery",
                tool_name=None,
                args={"reason": "runtime_restart"},
                result={"status": "denied"},
                requires_confirmation=True,
                user_decision="deny",
                outcome="failed_after_restart",
            )
            await self.emit_event(
                run_id,
                "done",
                {
                    "status": "failed",
                    "message": "Runtime restarted while waiting for confirmation.",
                },
            )

    async def start_run(self, run_id: str, payload: dict[str, Any], trace_id: str) -> None:
        metrics.runs_total += 1
        self._run_traces[run_id] = trace_id
        try:
            await self.repo.ensure_project(payload["project_id"], payload["workspace_path"])
            await self.repo.ensure_session(payload["session_id"], payload["project_id"])
            await self.repo.create_run(payload, run_id, trace_id)

            if self.agent_mode in {"graph", "deepagents"}:
                await self._execute_graph(run_id, payload)
            else:
                await self._execute_mock(run_id, payload)

            await self.repo.update_run_status(run_id, "completed")
            await self.emit_event(
                run_id,
                "done",
                {"status": "completed", "message": "run finished"},
            )
        except Exception as exc:  # noqa: BLE001
            metrics.runs_failed_total += 1
            run_status = await self.repo.get_run_status(run_id)
            if run_status is None:
                return
            await self.repo.update_run_status(run_id, "failed")
            _, error_payload = error_from_exception(exc, trace_id)
            await self.emit_event(
                run_id,
                "error",
                {
                    "error": error_payload["error"],
                },
            )
            await self.emit_event(
                run_id,
                "done",
                {
                    "status": "failed",
                    "message": str(error_payload["error"]["message"]),
                },
            )

    async def _execute_graph(self, run_id: str, payload: dict[str, Any]) -> None:
        provider, model_config, api_key = await self._resolve_provider(payload)

        read_call_id = new_call_id()
        await self._emit_tool_call(run_id, read_call_id, "read_file", {"path": "README.md"}, requires_confirmation("read_file"))
        try:
            readme_content = read_file(payload["workspace_path"], "README.md")
            await self._emit_tool_result(run_id, read_call_id, ok=True, output={"content_preview": readme_content[:200]})
        except Exception as exc:  # noqa: BLE001
            await self._emit_tool_result(run_id, read_call_id, ok=False, exc=exc)
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
            task_input=payload["input"],
            workspace_path=payload["workspace_path"],
            readme_content=readme_content,
        )
        result = await compiled.ainvoke(state)
        plan_payload = result.plan or build_mock_plan(payload["input"])
        patch = result.patch or compute_readme_patch(payload["workspace_path"], payload["input"])
        await self.emit_event(run_id, "plan", plan_payload)
        await self._emit_patch_flow(run_id, payload, patch)

    async def _resolve_provider(self, payload: dict[str, Any]):
        model_config_id = (payload.get("model_config_id") or "").strip()
        if not model_config_id:
            raise RuntimeError("model_config_id is required when GOYAIS_AGENT_MODE is graph/deepagents")

        model_config = await self.repo.get_model_config(model_config_id)
        if model_config is None:
            raise RuntimeError(f"model config not found: {model_config_id}")

        secret_ref = str(model_config.get("secret_ref", "")).strip()
        api_key = self._resolve_secret_ref(secret_ref)
        provider = build_provider(
            str(model_config["provider"]),
            api_key,
            model_config.get("base_url"),
        )
        return provider, model_config, api_key

    def _resolve_secret_ref(self, secret_ref: str) -> str:
        if not secret_ref:
            raise RuntimeError("missing secret_ref in model config")

        if secret_ref.startswith("env:"):
            env_key = secret_ref.split(":", 1)[1]
        elif secret_ref.startswith("keychain:"):
            parts = secret_ref.split(":")
            if len(parts) != 3:
                raise RuntimeError(f"invalid secret_ref format: {secret_ref}")
            provider, profile = parts[1], parts[2]
            env_key = f"GOYAIS_SECRET_{provider.upper()}_{profile.upper()}"
        else:
            env_key = secret_ref

        value = os.getenv(env_key)
        if not value:
            raise RuntimeError(
                f"API key not found for secret_ref '{secret_ref}'. "
                f"Set environment variable '{env_key}'."
            )
        return value

    async def _generate_plan(
        self,
        *,
        task_input: str,
        readme_content: str,
        model_config: dict[str, Any],
        provider,
        api_key: str,
    ) -> dict[str, Any]:
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
        lines = [line.strip("- ").strip() for line in plan_text.splitlines() if line.strip()]
        if not lines:
            return build_mock_plan(task_input)
        summary = lines[0][:400]
        steps = lines[1:6] if len(lines) > 1 else build_mock_plan(task_input)["steps"]
        return {"summary": summary, "steps": steps}

    async def _generate_patch(
        self,
        *,
        workspace_path: str,
        task_input: str,
        readme_content: str,
        model_config: dict[str, Any],
        provider,
    ) -> str:
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
            parsed = PatchSet(candidate)
            if parsed:
                return candidate
        except Exception:  # noqa: BLE001
            pass
        return compute_readme_patch(workspace_path, task_input)

    def _extract_unified_diff(self, text: str) -> str:
        candidate = text.strip()
        if "```" in candidate:
            segments = candidate.split("```")
            for segment in segments:
                stripped = segment.strip()
                if stripped.startswith("diff"):
                    stripped = stripped[4:].lstrip()
                if stripped.startswith("--- "):
                    candidate = stripped
                    break
        if not candidate.startswith("--- "):
            lines = candidate.splitlines()
            for index, line in enumerate(lines):
                if line.startswith("--- "):
                    candidate = "\n".join(lines[index:])
                    break
        if not candidate.endswith("\n"):
            candidate += "\n"
        return candidate

    async def _execute_mock(self, run_id: str, payload: dict[str, Any]) -> None:
        plan_payload = build_mock_plan(payload["input"])
        await self.emit_event(run_id, "plan", plan_payload)

        read_call_id = new_call_id()
        await self._emit_tool_call(run_id, read_call_id, "read_file", {"path": "README.md"}, requires_confirmation("read_file"))
        try:
            content = read_file(payload["workspace_path"], "README.md")
            await self._emit_tool_result(run_id, read_call_id, ok=True, output={"content_preview": content[:200]})
        except Exception as exc:  # noqa: BLE001
            await self._emit_tool_result(run_id, read_call_id, ok=False, exc=exc)
            raise

        patch = compute_readme_patch(payload["workspace_path"], payload["input"])
        await self._emit_patch_flow(run_id, payload, patch)

    async def _emit_patch_flow(self, run_id: str, payload: dict[str, Any], patch: str) -> None:
        await self.emit_event(run_id, "patch", {"unified_diff": patch})

        apply_call_id = new_call_id()
        await self._emit_tool_call(
            run_id,
            apply_call_id,
            "apply_patch",
            {"unified_diff": patch},
            requires_confirmation("apply_patch"),
        )

        try:
            approved = await self.confirmation_service.wait_for(run_id, apply_call_id)
        except asyncio.TimeoutError as exc:
            await self.repo.upsert_tool_confirmation_status(run_id, apply_call_id, "denied", decided_by="system")
            await self._emit_tool_result(run_id, apply_call_id, ok=False, exc=exc)
            raise RuntimeError("Timed out waiting for apply_patch confirmation") from exc

        if not approved:
            await self._emit_tool_result(
                run_id,
                apply_call_id,
                ok=False,
                error=build_goyais_error(
                    code="E_TOOL_DENIED",
                    message="User denied apply_patch confirmation.",
                    trace_id=await self._trace_id_for_run(run_id),
                    retryable=False,
                    cause="user_denied",
                ),
            )
            return

        try:
            output = apply_patch(payload["workspace_path"], patch)
            await self._emit_tool_result(run_id, apply_call_id, ok=True, output={"message": output})
        except Exception as exc:  # noqa: BLE001
            await self._emit_tool_result(run_id, apply_call_id, ok=False, exc=exc)
            raise

        if payload.get("options", {}).get("run_tests"):
            command_call_id = new_call_id()
            cmd = payload["options"]["run_tests"]
            await self._emit_tool_call(run_id, command_call_id, "run_command", {"cmd": cmd, "cwd": "."}, True)
            try:
                command_approved = await self.confirmation_service.wait_for(run_id, command_call_id)
            except asyncio.TimeoutError as exc:
                await self.repo.upsert_tool_confirmation_status(run_id, command_call_id, "denied", decided_by="system")
                await self._emit_tool_result(run_id, command_call_id, ok=False, exc=exc)
                raise RuntimeError("Timed out waiting for run_command confirmation") from exc
            if command_approved:
                try:
                    result = run_command(payload["workspace_path"], cmd, ".")
                    await self._emit_tool_result(run_id, command_call_id, ok=True, output=result)
                except Exception as exc:  # noqa: BLE001
                    await self._emit_tool_result(run_id, command_call_id, ok=False, exc=exc)
                    raise
            else:
                await self._emit_tool_result(
                    run_id,
                    command_call_id,
                    ok=False,
                    error=build_goyais_error(
                        code="E_TOOL_DENIED",
                        message="User denied run_command confirmation.",
                        trace_id=await self._trace_id_for_run(run_id),
                        retryable=False,
                        cause="user_denied",
                    ),
                )

    async def _emit_tool_call(
        self, run_id: str, call_id: str, tool_name: str, args: dict[str, Any], must_confirm: bool
    ) -> None:
        trace_id = await self._trace_id_for_run(run_id)
        metrics.increment_tool_call(tool_name)
        if must_confirm:
            await self.repo.upsert_tool_confirmation_status(run_id, call_id, "pending", decided_by="system")
            await self.repo.update_run_status(run_id, "waiting_confirmation")
            metrics.confirmations_pending += 1

        event = await self.emit_event(
            run_id,
            "tool_call",
            {
                "call_id": call_id,
                "tool_name": tool_name,
                "args": args,
                "requires_confirmation": must_confirm,
            },
        )
        await self.audit_service.record(
            trace_id=trace_id,
            run_id=run_id,
            event_id=event["event_id"],
            call_id=call_id,
            action="tool_call",
            tool_name=tool_name,
            args=args,
            result={"requires_confirmation": must_confirm},
            requires_confirmation=must_confirm,
            user_decision="n/a",
            outcome="requested",
        )
        logger.info(
            "tool_call",
            extra={
                "trace_id": trace_id,
                "run_id": run_id,
                "event_id": event["event_id"],
                "tool_name": tool_name,
                "outcome": "requested",
            },
        )

    async def _emit_tool_result(
        self,
        run_id: str,
        call_id: str,
        *,
        ok: bool,
        output: Any | None = None,
        error: dict[str, Any] | None = None,
        exc: Exception | None = None,
    ) -> None:
        trace_id = await self._trace_id_for_run(run_id)
        current_status = await self.repo.get_run_status(run_id)
        if current_status == "waiting_confirmation":
            await self.repo.update_run_status(run_id, "running")
            if metrics.confirmations_pending > 0:
                metrics.confirmations_pending -= 1

        payload: dict[str, Any] = {
            "call_id": call_id,
            "ok": ok,
        }

        audit_result: Any
        if ok:
            payload["output"] = output
            audit_result = output
        else:
            if error is None:
                if exc is not None:
                    _, payload_error = error_from_exception(exc, trace_id)
                    error = payload_error["error"]
                else:
                    error = build_goyais_error(
                        code="E_INTERNAL",
                        message="Tool execution failed.",
                        trace_id=trace_id,
                        retryable=False,
                    )
            payload["error"] = error
            audit_result = error

        event = await self.emit_event(run_id, "tool_result", payload)
        await self.audit_service.record(
            trace_id=trace_id,
            run_id=run_id,
            event_id=event["event_id"],
            call_id=call_id,
            action="tool_result",
            tool_name=None,
            args=None,
            result=audit_result,
            requires_confirmation=False,
            user_decision="n/a",
            outcome="ok" if ok else "error",
        )
        logger.info(
            "tool_result",
            extra={
                "trace_id": trace_id,
                "run_id": run_id,
                "event_id": event["event_id"],
                "outcome": "ok" if ok else "error",
            },
        )

    async def emit_event(self, run_id: str, event_type: str, payload: dict[str, Any]) -> dict[str, Any]:
        trace_id = await self._trace_id_for_run(run_id)
        seq = await self.repo.next_seq(run_id)

        payload_with_trace = dict(payload)
        payload_with_trace["trace_id"] = trace_id

        event = {
            "protocol_version": PROTOCOL_VERSION,
            "trace_id": trace_id,
            "event_id": str(uuid.uuid4()),
            "run_id": run_id,
            "seq": seq,
            "ts": datetime.now(tz=timezone.utc).isoformat(),
            "type": event_type,
            "payload": payload_with_trace,
        }
        await self.repo.insert_event(event)
        await self.bus.publish(run_id, event)
        return event

    async def _trace_id_for_run(self, run_id: str) -> str:
        cached = self._run_traces.get(run_id)
        if cached:
            return cached

        trace_id = await self.repo.get_run_trace_id(run_id)
        if trace_id:
            self._run_traces[run_id] = trace_id
            return trace_id

        generated = generate_trace_id()
        self._run_traces[run_id] = generated
        return generated


async def stream_as_sse(event: dict[str, Any]) -> dict[str, str]:
    return {
        "event": event["type"],
        "data": json.dumps(event, ensure_ascii=False),
    }
