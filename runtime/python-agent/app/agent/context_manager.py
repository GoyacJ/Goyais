"""Context window management: token estimation, truncation, and compaction."""
from __future__ import annotations

from typing import TYPE_CHECKING

from app.agent.messages import Message

if TYPE_CHECKING:
    from app.agent.providers.base import ProviderAdapter


def estimate_tokens(messages: list[Message]) -> int:
    """Rough token estimate: total chars / 3.5."""
    total_chars = 0
    for msg in messages:
        total_chars += len(msg.content)
        for tc in msg.tool_calls:
            total_chars += len(tc.name) + len(str(tc.input))
        for tr in msg.tool_results:
            total_chars += len(tr.content)
    return int(total_chars / 3.5)


def truncate_old_tool_results(
    messages: list[Message],
    *,
    max_chars: int = 2000,
    preserve_last_n: int = 6,
) -> None:
    """Truncate tool result content in older messages to save context space.

    Modifies messages in place. Preserves the last `preserve_last_n` messages
    and the first message (user task) untouched.
    """
    if len(messages) <= preserve_last_n + 1:
        return

    cutoff = len(messages) - preserve_last_n
    for msg in messages[1:cutoff]:
        if msg.role == "tool_result":
            for tr in msg.tool_results:
                if len(tr.content) > max_chars:
                    tr.content = tr.content[:max_chars] + "\n... [truncated]"


async def compact_context(
    messages: list[Message],
    *,
    provider: ProviderAdapter,
    model: str,
    token_limit: int,
) -> list[Message]:
    """Compress context when approaching token limits.

    Strategy: keep first message (task) and last 4 messages intact.
    Summarize everything in between using the LLM.
    Returns a new message list (does not mutate input).
    """
    from app.agent.providers.base import ChatRequest

    current_tokens = estimate_tokens(messages)
    if current_tokens < token_limit:
        return messages

    if len(messages) <= 6:
        return messages

    first = messages[0]
    middle = messages[1:-4]
    tail = messages[-4:]

    middle_text_parts: list[str] = []
    for msg in middle:
        if msg.content:
            middle_text_parts.append(f"[{msg.role}]: {msg.content[:300]}")
        for tc in msg.tool_calls:
            middle_text_parts.append(f"[tool_call]: {tc.name}({list(tc.input.keys())})")
        for tr in msg.tool_results:
            status = "error" if tr.is_error else "ok"
            middle_text_parts.append(f"[tool_result {status}]: {tr.content[:200]}")

    summary_input = "\n".join(middle_text_parts)

    summary_request = ChatRequest(
        model=model,
        system_prompt="Summarize the following conversation history concisely. Focus on key decisions, tool results, and current state.",
        messages=[Message(role="user", content=summary_input)],
        max_tokens=500,
        temperature=0.0,
    )
    summary_response = await provider.chat(summary_request)

    summary_msg = Message(
        role="user",
        content=f"[Conversation summary]: {summary_response.text}",
    )

    return [first, summary_msg, *tail]
