from __future__ import annotations


from unidiff import PatchSet

from app.security.path_guard import resolve_in_workspace


class PatchApplyError(Exception):
    pass


def _normalize_path(file_path: str) -> str:
    normalized = file_path
    if normalized.startswith("a/") or normalized.startswith("b/"):
        normalized = normalized[2:]
    return normalized


def apply_patch(workspace_path: str, unified_diff: str) -> str:
    patch = PatchSet(unified_diff)
    if not patch:
        raise PatchApplyError("Empty patch")

    for patched_file in patch:
        relative_path = _normalize_path(patched_file.path)
        target = resolve_in_workspace(workspace_path, relative_path)

        if target.exists():
            original_text = target.read_text(encoding="utf-8")
            original_had_newline = original_text.endswith("\n")
            original_lines = original_text.splitlines()
        else:
            original_had_newline = False
            original_lines = []

        result_lines: list[str] = []
        cursor = 0

        for hunk in patched_file:
            source_start = max(hunk.source_start - 1, 0)
            result_lines.extend(original_lines[cursor:source_start])
            cursor = source_start

            for line in hunk:
                value = line.value.rstrip("\n")
                if line.is_context:
                    if cursor >= len(original_lines) or original_lines[cursor] != value:
                        raise PatchApplyError(f"Context mismatch while applying patch to {relative_path}")
                    result_lines.append(value)
                    cursor += 1
                elif line.is_removed:
                    if cursor >= len(original_lines) or original_lines[cursor] != value:
                        raise PatchApplyError(f"Remove mismatch while applying patch to {relative_path}")
                    cursor += 1
                elif line.is_added:
                    result_lines.append(value)

        result_lines.extend(original_lines[cursor:])
        target.parent.mkdir(parents=True, exist_ok=True)
        output = "\n".join(result_lines)
        if result_lines and original_had_newline:
            output += "\n"
        target.write_text(output, encoding="utf-8")

    return "patch applied"
