import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("withApiFallback", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.unstubAllEnvs();
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("strict 模式下不会触发 fallback", async () => {
    vi.stubEnv("VITE_API_MODE", "strict");
    vi.stubEnv("VITE_ENABLE_MOCK_FALLBACK", "true");

    const { withApiFallback } = await import("@/shared/services/fallback");
    const { ApiError } = await import("@/shared/services/http");
    const { resetRuntimeStore, runtimeStore } = await import("@/shared/stores/runtimeStore");

    resetRuntimeStore();
    const fallbackCall = vi.fn(() => ({ ok: true }));

    await expect(
      withApiFallback(
        "strict.case",
        () =>
          Promise.reject(
            new ApiError({
              status: 503,
              code: "SERVICE_UNAVAILABLE",
              message: "service unavailable",
              traceId: "tr_test",
              details: {}
            })
          ),
        fallbackCall
      )
    ).rejects.toBeInstanceOf(ApiError);

    expect(fallbackCall).not.toHaveBeenCalled();
    expect(runtimeStore.fallbackHits["strict.case"]).toBeUndefined();
  });

  it("hybrid 模式下 501 会触发 fallback", async () => {
    vi.stubEnv("VITE_API_MODE", "hybrid");
    vi.stubEnv("VITE_ENABLE_MOCK_FALLBACK", "true");

    const { withApiFallback } = await import("@/shared/services/fallback");
    const { ApiError } = await import("@/shared/services/http");
    const { resetRuntimeStore, runtimeStore } = await import("@/shared/stores/runtimeStore");

    resetRuntimeStore();
    const fallbackCall = vi.fn(() => ({ ok: true }));

    const result = await withApiFallback(
      "hybrid.case",
      () =>
        Promise.reject(
          new ApiError({
            status: 501,
            code: "INTERNAL_NOT_IMPLEMENTED",
            message: "not implemented",
            traceId: "tr_test",
            details: {}
          })
        ),
      fallbackCall
    );

    expect(result).toEqual({ ok: true });
    expect(fallbackCall).toHaveBeenCalledTimes(1);
    expect(runtimeStore.fallbackHits["hybrid.case"]?.status).toBe(501);
  });
});
