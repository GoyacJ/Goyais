import { describe, expect, it } from "vitest";

import { assertToken } from "../src/auth";

describe("auth", () => {
  it("throws for missing header", () => {
    expect(() => assertToken({ headers: {} } as never, "token")).toThrow();
  });

  it("throws for invalid token", () => {
    expect(() =>
      assertToken({ headers: { authorization: "Bearer wrong" } } as never, "token")
    ).toThrow();
  });

  it("passes for expected token", () => {
    expect(() =>
      assertToken({ headers: { authorization: "Bearer token" } } as never, "token")
    ).not.toThrow();
  });
});
