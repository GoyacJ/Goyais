import { describe, expect, it } from "vitest";

import enUS from "@/i18n/locales/en-US/common.json";
import zhCN from "@/i18n/locales/zh-CN/common.json";

type JsonObject = Record<string, unknown>;

function flattenKeys(value: JsonObject, prefix = ""): string[] {
  const keys: string[] = [];

  for (const [key, child] of Object.entries(value)) {
    const fullKey = prefix ? `${prefix}.${key}` : key;
    keys.push(fullKey);

    if (child && typeof child === "object" && !Array.isArray(child)) {
      keys.push(...flattenKeys(child as JsonObject, fullKey));
    }
  }

  return keys;
}

describe("i18n locale resources", () => {
  it("keeps zh-CN and en-US keysets aligned", () => {
    const enKeys = flattenKeys(enUS as JsonObject).sort();
    const zhKeys = flattenKeys(zhCN as JsonObject).sort();

    expect(zhKeys).toEqual(enKeys);
  });
});
