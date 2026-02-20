from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Awaitable, Callable

from deepagents import create_deep_agent
from langgraph.graph import END, StateGraph

from app.agent.mock_agent import build_mock_plan, compute_readme_patch


@dataclass
class AgentState:
    task_input: str
    workspace_path: str
    readme_content: str = ""
    plan: dict | None = None
    patch: str | None = None


PlanBuilder = Callable[[AgentState], Awaitable[dict]]
PatchBuilder = Callable[[AgentState], Awaitable[str]]


def build_graph(
    *,
    plan_builder: PlanBuilder | None = None,
    patch_builder: PatchBuilder | None = None,
):
    graph = StateGraph(AgentState)

    async def plan_node(state: AgentState):
        if plan_builder is None:
            state.plan = build_mock_plan(state.task_input)
        else:
            state.plan = await plan_builder(state)
        return state

    async def patch_node(state: AgentState):
        if patch_builder is None:
            state.patch = compute_readme_patch(state.workspace_path, state.task_input)
        else:
            state.patch = await patch_builder(state)
        return state

    graph.add_node("plan", plan_node)
    graph.add_node("patch", patch_node)
    graph.set_entry_point("plan")
    graph.add_edge("plan", "patch")
    graph.add_edge("patch", END)

    return graph.compile()


async def build_plan_with_deepagents(
    task_input: str,
    *,
    provider: str,
    model: str,
    api_key: str,
) -> dict:
    """Best-effort Deep Agents plan generation.

    If provider credentials are unavailable, caller should fallback gracefully.
    """
    if provider == "openai":
        os.environ["OPENAI_API_KEY"] = api_key
        model_id = f"openai:{model}"
    elif provider == "anthropic":
        os.environ["ANTHROPIC_API_KEY"] = api_key
        model_id = f"anthropic:{model}"
    else:
        raise ValueError(f"Unsupported deepagents provider: {provider}")

    agent = create_deep_agent(
        model=model_id,
        tools=[],
        system_prompt=(
            "You are a coding planner. Return concise plain text with 3-5 steps. "
            "Do not execute tools."
        ),
    )
    result = await agent.ainvoke({"messages": [{"role": "user", "content": task_input}]})

    summary = f"DeepAgents plan for task: {task_input}"
    steps = [
        "Analyze task",
        "Inspect files",
        "Prepare patch",
        "Wait for user approval",
        "Apply patch",
    ]

    messages = result.get("messages", []) if isinstance(result, dict) else []
    if messages:
        last = messages[-1]
        content = getattr(last, "content", "")
        if isinstance(content, list):
            text_chunks = []
            for item in content:
                if isinstance(item, dict):
                    text_chunks.append(str(item.get("text", "")))
                else:
                    text_chunks.append(str(getattr(item, "text", item)))
            text = "\n".join(chunk for chunk in text_chunks if chunk).strip()
        else:
            text = str(content).strip()
        if text:
            summary = text.splitlines()[0][:400]
            parsed_steps = [line.strip("- ").strip() for line in text.splitlines()[1:] if line.strip()]
            if parsed_steps:
                steps = parsed_steps[:5]

    return {"summary": summary, "steps": steps}
