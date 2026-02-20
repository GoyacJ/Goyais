import { describe, expect, it } from "vitest";

import { classifyToolRisk } from "@/lib/risk";

describe("classifyToolRisk", () => {
  it("returns exec risk for run_command", () => {
    const result = classifyToolRisk("run_command", { cmd: "npm test", cwd: "." });
    expect(result.primary).toBe("exec");
    expect(result.hasRisk).toBe(true);
    expect(result.details.command).toContain("npm test");
  });

  it("extracts path list and marks outside-workspace path", () => {
    const result = classifyToolRisk("write_file", { path: "../escape.txt" });
    expect(result.primary).toBe("write");
    expect(result.details.paths).toContain("../escape.txt");
    expect(result.details.pathOutsideWorkspace).toBe(true);
  });
});
