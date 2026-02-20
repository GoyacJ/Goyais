from __future__ import annotations

PLAN_SYSTEM_PROMPT = (
    "You are a senior coding planner. Return a concise plan with 3-5 steps. "
    "Be specific to the task and existing file context."
)

PATCH_SYSTEM_PROMPT = (
    "You are a coding agent that outputs only valid unified diff text. "
    "Do not include markdown fences. "
    "Do not modify files outside the requested target."
)


def build_plan_prompt(task_input: str, readme_preview: str) -> str:
    return (
        "Task:\n"
        f"{task_input}\n\n"
        "Current README excerpt:\n"
        f"{readme_preview}\n\n"
        "Return a short execution plan. Use one step per line."
    )


def build_patch_prompt(task_input: str, readme_content: str) -> str:
    return (
        "Task:\n"
        f"{task_input}\n\n"
        "Target file: README.md\n\n"
        "Current README.md content:\n"
        f"{readme_content}\n\n"
        "Return only a unified diff that updates README.md.\n"
        "The diff must use a/README.md and b/README.md paths.\n"
        "Do not return explanations."
    )
