import { describe, expect, it } from "vitest";

import { parseUnifiedDiff, reduceDiffSelection } from "@/lib/diff";

const sampleDiff = `--- a/README.md
+++ b/README.md
@@ -1,2 +1,2 @@
-# Old
+# New
 line\n`;

describe("parseUnifiedDiff", () => {
  it("extracts old/new text and hunk metadata", () => {
    const parsed = parseUnifiedDiff(sampleDiff);
    expect(parsed.oldText).toContain("# Old");
    expect(parsed.newText).toContain("# New");
    expect(parsed.hunks.length).toBe(1);
  });
});

describe("reduceDiffSelection", () => {
  it("toggles hunk selection", () => {
    const next = reduceDiffSelection({}, { type: "toggle", hunkId: "h1" });
    expect(next.h1).toBe(true);
  });
});
