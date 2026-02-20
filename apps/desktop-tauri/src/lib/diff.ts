import type { DiffHunkSelectionState } from "@/types/ui";

export interface ParsedDiffHunk {
  id: string;
  header: string;
  lines: string[];
}

export interface ParsedUnifiedDiff {
  oldText: string;
  newText: string;
  hunks: ParsedDiffHunk[];
}

export type DiffSelectionAction =
  | { type: "toggle"; hunkId: string }
  | { type: "clear" }
  | { type: "selectAll"; hunkIds: string[] };

export function reduceDiffSelection(
  state: DiffHunkSelectionState,
  action: DiffSelectionAction
): DiffHunkSelectionState {
  if (action.type === "clear") return {};
  if (action.type === "selectAll") {
    return action.hunkIds.reduce<DiffHunkSelectionState>((acc, id) => {
      acc[id] = true;
      return acc;
    }, {});
  }

  return {
    ...state,
    [action.hunkId]: !state[action.hunkId]
  };
}

export function parseUnifiedDiff(unifiedDiff: string): ParsedUnifiedDiff {
  const oldLines: string[] = [];
  const newLines: string[] = [];
  const hunks: ParsedDiffHunk[] = [];

  let currentHunk: ParsedDiffHunk | null = null;
  let hunkIndex = 0;

  for (const line of unifiedDiff.split("\n")) {
    if (line.startsWith("@@")) {
      if (currentHunk) {
        hunks.push(currentHunk);
      }
      hunkIndex += 1;
      currentHunk = {
        id: `hunk-${hunkIndex}`,
        header: line,
        lines: []
      };
      continue;
    }

    if (currentHunk) {
      currentHunk.lines.push(line);
    }

    if (line.startsWith("---") || line.startsWith("+++")) {
      continue;
    }
    if (line.startsWith("+")) {
      newLines.push(line.slice(1));
      continue;
    }
    if (line.startsWith("-")) {
      oldLines.push(line.slice(1));
      continue;
    }
    if (line.startsWith(" ")) {
      oldLines.push(line.slice(1));
      newLines.push(line.slice(1));
    }
  }

  if (currentHunk) {
    hunks.push(currentHunk);
  }

  return {
    oldText: oldLines.join("\n"),
    newText: newLines.join("\n"),
    hunks
  };
}
