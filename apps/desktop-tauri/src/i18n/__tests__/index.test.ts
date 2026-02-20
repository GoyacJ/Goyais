import { describe, expect, it } from "vitest";

import { DEFAULT_LOCALE, detectInitialLocale } from "@/i18n";

describe("detectInitialLocale", () => {
  it("prefers stored locale", () => {
    const locale = detectInitialLocale({
      storedLocale: "en-US",
      preferredLanguages: ["zh-CN", "en-US"]
    });

    expect(locale).toBe("en-US");
  });

  it("falls back to preferred language when storage missing", () => {
    const locale = detectInitialLocale({
      preferredLanguages: ["en-US"]
    });

    expect(locale).toBe("en-US");
  });

  it("falls back to default locale", () => {
    const locale = detectInitialLocale({
      preferredLanguages: ["fr-FR"]
    });

    expect(locale).toBe(DEFAULT_LOCALE);
  });
});
