"""
tool_injector.py — Skills/MCP tool injection

Loads skill_sets and mcp_connectors from Hub execution context and injects
them into the ToolRegistry as ToolDef objects (with callable wrappers).
"""
from __future__ import annotations

import json
import logging
from typing import Any

import httpx

from app.agent.tool_registry import ToolDef

logger = logging.getLogger(__name__)


class ToolInjector:
    """
    Resolves Skills and MCP tools from the Hub and returns ToolDef objects
    ready for registration in a ToolRegistry.

    Usage::

        injector = ToolInjector(hub_base_url, hub_internal_secret)
        tool_defs = await injector.resolve_tools(context)
        for td in tool_defs:
            registry.register(td)
    """

    def __init__(self, hub_base_url: str, hub_internal_secret: str) -> None:
        self._hub_base_url = hub_base_url.rstrip("/")
        self._secret = hub_internal_secret
        self._headers = {
            "X-Hub-Auth": hub_internal_secret,
            "Content-Type": "application/json",
        }

    async def resolve_tools(self, context: dict[str, Any]) -> list[ToolDef]:
        """
        Resolve all tools from the execution context's skill_set_ids and
        mcp_connector_ids. Returns a list of ToolDef objects.
        """
        tools: list[ToolDef] = []
        workspace_id = context.get("workspace_id", "")

        skill_set_ids = _parse_id_list(context.get("skill_set_ids", "[]"))
        for skill_set_id in skill_set_ids:
            try:
                skill_tools = await self._load_skill_tools(workspace_id, skill_set_id)
                tools.extend(skill_tools)
            except Exception as exc:
                logger.warning("Failed to load skill set %s: %s", skill_set_id, exc)

        mcp_connector_ids = _parse_id_list(context.get("mcp_connector_ids", "[]"))
        for connector_id in mcp_connector_ids:
            try:
                mcp_tools = await self._load_mcp_tools(workspace_id, connector_id)
                tools.extend(mcp_tools)
            except Exception as exc:
                logger.warning("Failed to load MCP connector %s: %s", connector_id, exc)

        return tools

    async def _load_skill_tools(self, workspace_id: str, skill_set_id: str) -> list[ToolDef]:
        """Fetch skills for a skill set from Hub and return ToolDef objects."""
        url = f"{self._hub_base_url}/v1/skill-sets/{skill_set_id}/skills?workspace_id={workspace_id}"
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.get(url, headers=self._headers)
            resp.raise_for_status()
            data = resp.json()

        result: list[ToolDef] = []
        for skill in data.get("skills", []):
            config: dict = {}
            try:
                config = json.loads(skill.get("config_json", "{}"))
            except json.JSONDecodeError:
                pass

            skill_name: str = skill["name"]
            skill_type: str = skill.get("type", "unknown")

            # Capture loop variables for the closure
            captured_name = skill_name
            captured_config = dict(config)

            async def _skill_handler(captured_name=captured_name, captured_config=captured_config, **kwargs: Any) -> str:
                # Phase 6b: actual skill execution via transport
                return json.dumps({
                    "status": "not_implemented",
                    "skill": captured_name,
                    "note": "Skill transport (Phase 6b) not yet implemented.",
                })

            result.append(ToolDef(
                name=skill_name,
                description=f"Skill: {skill_name} (type={skill_type})",
                input_schema={
                    "type": "object",
                    "properties": {
                        "input": {"type": "string", "description": "Input for the skill"},
                    },
                },
                handler=_skill_handler,
                requires_confirmation=False,
            ))
            logger.debug("Loaded skill tool: %s", skill_name)

        return result

    async def _load_mcp_tools(self, workspace_id: str, connector_id: str) -> list[ToolDef]:
        """
        Fetch MCP connector config from Hub.
        Phase 6b: actual transport (stdio/sse/streamable_http) not yet implemented.
        """
        url = f"{self._hub_base_url}/v1/mcp-connectors/{connector_id}?workspace_id={workspace_id}"
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.get(url, headers=self._headers)
            resp.raise_for_status()
            data = resp.json()

        connector = data.get("mcp_connector", {})
        if not connector.get("enabled", False):
            logger.info("MCP connector %s is disabled, skipping", connector_id)
            return []

        connector_name: str = connector.get("name", connector_id)
        transport: str = connector.get("transport", "unknown")

        logger.info(
            "MCP connector %s (%s) loaded (transport=%s) — tool enumeration not yet implemented",
            connector_name, connector_id, transport,
        )

        captured_name = connector_name

        async def _mcp_handler(captured_name=captured_name, **kwargs: Any) -> str:
            return json.dumps({
                "status": "not_implemented",
                "connector": captured_name,
                "note": "MCP transport (Phase 6b) not yet implemented.",
            })

        return [ToolDef(
            name=f"mcp_{connector_name}",
            description=f"MCP connector: {connector_name} ({transport})",
            input_schema={
                "type": "object",
                "properties": {
                    "input": {"type": "string", "description": "Input for the MCP connector"},
                },
            },
            handler=_mcp_handler,
            requires_confirmation=False,
        )]


def _parse_id_list(value: Any) -> list[str]:
    if isinstance(value, list):
        return [str(v) for v in value if v]
    if isinstance(value, str) and value.strip():
        try:
            parsed = json.loads(value)
            if isinstance(parsed, list):
                return [str(v) for v in parsed if v]
        except json.JSONDecodeError:
            pass
    return []
