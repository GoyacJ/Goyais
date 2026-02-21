"""
tool_injector.py — Phase 6: Skills/MCP tool injection

Loads skill_sets and mcp_connectors from the Hub execution context
and injects their tools into the agent before execution begins.

Current implementation: loads skill/MCP definitions from Hub and
registers them as no-op stubs. Full MCP transport integration
(stdio/sse/streamable_http) is a follow-up in Phase 6b.
"""
from __future__ import annotations

import json
import logging
from typing import Any

import httpx

logger = logging.getLogger(__name__)


class ToolInjector:
    """
    Injects Skills and MCP tools into an agent based on ExecutionContext.

    Usage::

        injector = ToolInjector(hub_base_url, hub_internal_secret)
        tools = await injector.resolve_tools(context)
        # tools is a list of callables or tool descriptors to add to the agent
    """

    def __init__(self, hub_base_url: str, hub_internal_secret: str) -> None:
        self._hub_base_url = hub_base_url.rstrip("/")
        self._secret = hub_internal_secret
        self._headers = {
            "X-Hub-Auth": hub_internal_secret,
            "Content-Type": "application/json",
        }

    async def resolve_tools(self, context: dict[str, Any]) -> list[dict[str, Any]]:
        """
        Resolve all tools from the execution context's skill_set_ids and
        mcp_connector_ids. Returns a list of tool descriptors.

        Args:
            context: ExecutionContext dict from Hub scheduler. Expected keys:
                - skill_set_ids: JSON array string or list of skill set IDs
                - mcp_connector_ids: JSON array string or list of connector IDs
                - workspace_id: workspace identifier

        Returns:
            List of tool descriptor dicts (name, description, type, config).
        """
        tools: list[dict[str, Any]] = []

        workspace_id = context.get("workspace_id", "")

        # Resolve skill set tools
        skill_set_ids = _parse_id_list(context.get("skill_set_ids", "[]"))
        for skill_set_id in skill_set_ids:
            try:
                skill_tools = await self._load_skill_tools(workspace_id, skill_set_id)
                tools.extend(skill_tools)
            except Exception as exc:
                logger.warning(
                    "Failed to load skill set %s: %s", skill_set_id, exc
                )

        # Resolve MCP connector tools
        mcp_connector_ids = _parse_id_list(context.get("mcp_connector_ids", "[]"))
        for connector_id in mcp_connector_ids:
            try:
                mcp_tools = await self._load_mcp_tools(workspace_id, connector_id)
                tools.extend(mcp_tools)
            except Exception as exc:
                logger.warning(
                    "Failed to load MCP connector %s: %s", connector_id, exc
                )

        return tools

    async def _load_skill_tools(
        self, workspace_id: str, skill_set_id: str
    ) -> list[dict[str, Any]]:
        """Fetch skills for a skill set from the Hub and convert to tool descriptors."""
        url = (
            f"{self._hub_base_url}/v1/skill-sets/{skill_set_id}/skills"
            f"?workspace_id={workspace_id}"
        )
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.get(url, headers=self._headers)
            resp.raise_for_status()
            data = resp.json()

        skill_tools = []
        for skill in data.get("skills", []):
            config = {}
            try:
                config = json.loads(skill.get("config_json", "{}"))
            except json.JSONDecodeError:
                pass

            skill_tools.append(
                {
                    "name": skill["name"],
                    "description": f"Skill: {skill['name']} (type={skill['type']})",
                    "type": "skill",
                    "skill_type": skill["type"],
                    "config": config,
                    "skill_id": skill["skill_id"],
                    "skill_set_id": skill_set_id,
                }
            )
            logger.debug("Loaded skill tool: %s", skill["name"])

        return skill_tools

    async def _load_mcp_tools(
        self, workspace_id: str, connector_id: str
    ) -> list[dict[str, Any]]:
        """
        Fetch MCP connector config from Hub and resolve available tools.

        For stdio transport: spawns subprocess (not yet implemented).
        For sse/streamable_http: connects via HTTP SSE (not yet implemented).

        Current: returns a single placeholder descriptor for the connector.
        Full transport implementation is Phase 6b.
        """
        url = (
            f"{self._hub_base_url}/v1/mcp-connectors/{connector_id}"
            f"?workspace_id={workspace_id}"
        )
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.get(url, headers=self._headers)
            resp.raise_for_status()
            data = resp.json()

        connector = data.get("mcp_connector", {})
        if not connector.get("enabled", False):
            logger.info(
                "MCP connector %s is disabled, skipping", connector_id
            )
            return []

        # Placeholder: in Phase 6b, actually connect via transport and list_tools()
        logger.info(
            "MCP connector %s (%s) loaded (transport=%s) — "
            "tool enumeration not yet implemented",
            connector.get("name"),
            connector_id,
            connector.get("transport"),
        )
        return [
            {
                "name": f"mcp_{connector.get('name', connector_id)}",
                "description": f"MCP connector: {connector.get('name')} ({connector.get('transport')})",
                "type": "mcp_connector",
                "connector_id": connector_id,
                "transport": connector.get("transport"),
                "endpoint": connector.get("endpoint"),
            }
        ]


def _parse_id_list(value: Any) -> list[str]:
    """Parse a skill_set_ids / mcp_connector_ids value that may be a JSON
    array string or already a Python list."""
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
