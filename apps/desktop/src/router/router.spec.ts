import { beforeEach, describe, expect, it } from "vitest";
import { createMemoryHistory } from "vue-router";

import { createAppRouter, routes } from "@/router";
import { authStore, resetAuthStore } from "@/shared/stores/authStore";

describe("desktop routes", () => {
  beforeEach(() => {
    resetAuthStore();
  });

  it("contains required routes", () => {
    const routePaths = routes.map((route) => route.path);

    expect(routePaths).toContain("/workspace");
    expect(routePaths).toContain("/project");
    expect(routePaths).toContain("/conversation");
    expect(routePaths).toContain("/resource");
    expect(routePaths).toContain("/admin");
  });

  it("redirects /admin to /workspace when admin capability is missing", async () => {
    authStore.capabilities = {
      admin_console: false,
      resource_write: false,
      execution_control: false
    };

    const router = createAppRouter(createMemoryHistory());
    await router.push("/admin");
    await router.isReady();

    expect(router.currentRoute.value.fullPath).toBe("/workspace?reason=admin_forbidden");
  });

  it("allows /admin when admin capability is present", async () => {
    authStore.capabilities = {
      admin_console: true,
      resource_write: true,
      execution_control: true
    };

    const router = createAppRouter(createMemoryHistory());
    await router.push("/admin");
    await router.isReady();

    expect(router.currentRoute.value.path).toBe("/admin");
  });
});
