import { describe, expect, it } from "vitest";

import {
  enforceHubSecurityPolicy,
  resolveControlHubBaseUrl,
  resolveRuntimeCapabilities,
  resolveRuntimeTarget
} from "@/shared/runtime";

describe("runtime", () => {
  it("defaults runtime target to desktop", () => {
    expect(resolveRuntimeTarget(undefined)).toBe("desktop");
  });

  it("resolves runtime target from env value", () => {
    expect(resolveRuntimeTarget("mobile")).toBe("mobile");
    expect(resolveRuntimeTarget("web")).toBe("web");
  });

  it("exposes mobile capabilities without sidecar features", () => {
    const capabilities = resolveRuntimeCapabilities("mobile");
    expect(capabilities.supportsSidecar).toBe(false);
    expect(capabilities.supportsWindowControls).toBe(false);
    expect(capabilities.supportsDirectoryImport).toBe(false);
    expect(capabilities.supportsAutostart).toBe(false);
    expect(capabilities.supportsLocalWorkspace).toBe(false);
  });

  it("requires explicit hub base url on mobile when sidecar is unavailable", () => {
    expect(() =>
      resolveControlHubBaseUrl({
        runtimeTarget: "mobile",
        capabilities: resolveRuntimeCapabilities("mobile"),
        hubBaseUrl: "",
        requireHttpsHub: true,
        allowInsecureHub: false,
        isDev: false
      })
    ).toThrowError(/VITE_HUB_BASE_URL/i);
  });

  it("rejects insecure http hub for mobile release policy", () => {
    expect(() =>
      enforceHubSecurityPolicy("http://hub.example.com:8787", {
        runtimeTarget: "mobile",
        requireHttpsHub: true,
        allowInsecureHub: false,
        isDev: false
      })
    ).toThrowError(/https/i);
  });

  it("allows insecure hub in mobile dev only when explicitly enabled", () => {
    expect(() =>
      enforceHubSecurityPolicy("http://hub.example.com:8787", {
        runtimeTarget: "mobile",
        requireHttpsHub: true,
        allowInsecureHub: true,
        isDev: true
      })
    ).not.toThrow();
  });
});
