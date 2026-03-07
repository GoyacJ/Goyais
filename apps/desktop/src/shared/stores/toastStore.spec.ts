import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  clearToasts,
  dismissToastByKey,
  resetToastStore,
  showToast,
  toastStore
} from "@/shared/stores/toastStore";

describe("toast store", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    resetToastStore();
  });

  afterEach(() => {
    clearToasts();
    vi.useRealTimers();
  });

  it("auto dismisses non-persistent toast after default duration", () => {
    const toastId = showToast({
      tone: "info",
      message: "saved"
    });

    expect(toastId).not.toBe("");
    expect(toastStore.items).toHaveLength(1);

    vi.advanceTimersByTime(3199);
    expect(toastStore.items).toHaveLength(1);

    vi.advanceTimersByTime(1);
    expect(toastStore.items).toHaveLength(0);
  });

  it("replaces toast by key and keeps a single item", () => {
    const firstId = showToast({
      key: "workspace-model-test",
      tone: "info",
      message: "test success",
      persistent: true
    });

    const secondId = showToast({
      key: "workspace-model-test",
      tone: "error",
      message: "test failed"
    });

    expect(secondId).toBe(firstId);
    expect(toastStore.items).toHaveLength(1);
    expect(toastStore.items[0]?.tone).toBe("error");
    expect(toastStore.items[0]?.message).toBe("test failed");

    vi.advanceTimersByTime(3200);
    expect(toastStore.items).toHaveLength(0);
  });

  it("does not auto dismiss persistent toast", () => {
    showToast({
      tone: "retrying",
      message: "running",
      persistent: true
    });

    vi.advanceTimersByTime(10000);
    expect(toastStore.items).toHaveLength(1);
  });

  it("caps queue size at three items", () => {
    showToast({ tone: "info", message: "one", persistent: true });
    showToast({ tone: "info", message: "two", persistent: true });
    showToast({ tone: "info", message: "three", persistent: true });
    showToast({ tone: "info", message: "four", persistent: true });

    expect(toastStore.items).toHaveLength(3);
    expect(toastStore.items.map((item) => item.message)).toEqual(["two", "three", "four"]);
  });

  it("dismisses toasts by key", () => {
    showToast({
      key: "project-import-status",
      tone: "info",
      message: "imported",
      persistent: true
    });
    showToast({
      key: "workspace-auth-required",
      tone: "warning",
      message: "auth required",
      persistent: true
    });

    dismissToastByKey("project-import-status");

    expect(toastStore.items).toHaveLength(1);
    expect(toastStore.items[0]?.key).toBe("workspace-auth-required");
  });
});
