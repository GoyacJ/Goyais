// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Maintainers
// SPDX-License-Identifier: MIT

import { describe, expect, it } from "vitest";

import { enUSMessages } from "@/shared/i18n/messages.en-US";
import { zhCNMessages } from "@/shared/i18n/messages.zh-CN";

describe("inspector run i18n keys", () => {
  it("exposes run-based inspector keys in both locales", () => {
    expect(enUSMessages["session.inspector.run.latestRun"]).toBeTypeOf("string");
    expect(enUSMessages["session.inspector.trace.run"]).toBeTypeOf("string");
    expect(enUSMessages["session.inspector.trace.runShort"]).toBeTypeOf("string");
    expect(enUSMessages["session.inspector.trace.runCount"]).toBeTypeOf("string");

    expect(zhCNMessages["session.inspector.run.latestRun"]).toBeTypeOf("string");
    expect(zhCNMessages["session.inspector.trace.run"]).toBeTypeOf("string");
    expect(zhCNMessages["session.inspector.trace.runShort"]).toBeTypeOf("string");
    expect(zhCNMessages["session.inspector.trace.runCount"]).toBeTypeOf("string");
  });
});
