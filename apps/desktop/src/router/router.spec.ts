import { describe, expect, it } from "vitest";

import { routes } from "@/router";

describe("desktop routes", () => {
  it("contains required placeholder routes", () => {
    const routePaths = routes.map((route) => route.path);

    expect(routePaths).toContain("/workspace");
    expect(routePaths).toContain("/project");
    expect(routePaths).toContain("/conversation");
    expect(routePaths).toContain("/resource");
    expect(routePaths).toContain("/admin");
  });
});
