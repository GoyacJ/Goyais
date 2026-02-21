"""
langgraph_graph.py — Build the ReAct StateGraph.

Graph topology:
  START → agent_node → [tool_use?] → tools_node → agent_node → ...
                      ↘ [end_turn] → END
"""
from __future__ import annotations

from langgraph.graph import END, StateGraph

from app.agent.langgraph_nodes import agent_node, should_continue, tools_node
from app.agent.langgraph_state import AgentState


def build_react_graph():
    """Construct and compile the ReAct agent graph."""
    graph = StateGraph(AgentState)

    graph.add_node("agent", agent_node)
    graph.add_node("tools", tools_node)

    graph.set_entry_point("agent")

    graph.add_conditional_edges(
        "agent",
        should_continue,
        {
            "tools": "tools",
            "end": END,
        },
    )
    graph.add_edge("tools", "agent")

    return graph.compile()
