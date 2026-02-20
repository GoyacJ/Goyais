import DiffViewer from "react-diff-viewer-continued";

function parseUnifiedDiff(unifiedDiff: string): { oldText: string; newText: string } {
  const oldLines: string[] = [];
  const newLines: string[] = [];

  for (const line of unifiedDiff.split("\n")) {
    if (line.startsWith("---") || line.startsWith("+++") || line.startsWith("@@") || line.length === 0) {
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

  return {
    oldText: oldLines.join("\n"),
    newText: newLines.join("\n")
  };
}

export function DiffPanel({ unifiedDiff }: { unifiedDiff?: string }) {
  if (!unifiedDiff) {
    return (
      <section className="panel">
        <h3>Diff</h3>
        <p>No patch yet.</p>
      </section>
    );
  }

  const { oldText, newText } = parseUnifiedDiff(unifiedDiff);

  return (
    <section className="panel">
      <h3>Diff</h3>
      <DiffViewer oldValue={oldText} newValue={newText} splitView showDiffOnly={false} />
      <details>
        <summary>Raw unified diff</summary>
        <pre>{unifiedDiff}</pre>
      </details>
    </section>
  );
}
