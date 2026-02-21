"""LangGraph AgentState for the ReAct loop."""
from __future__ import annotations

from typing import Annotated, Any

from langgraph.graph.message import add_messages
from typing_extensions import TypedDict


class AgentState(TypedDict):
    messages: Annotated[list[Any], add_messages]
    workspace_path: str
    mode: str          # "plan" | "auto"
    iteration_count: int
