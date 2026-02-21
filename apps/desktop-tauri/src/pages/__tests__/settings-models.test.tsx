import { describe, expect, it } from "vitest";

import type { DataModelConfig } from "@/api/dataSource";
import {
  buildModelSuggestions,
  isModelAvailableInCatalog,
  SETTINGS_MODEL_LIST_COLUMNS,
  SETTINGS_MODEL_VISIBLE_FIELDS
} from "@/pages/SettingsPage";

describe("settings model fields", () => {
  it("uses editable model/api key fields and hides model_config_id/secret_ref", () => {
    expect(SETTINGS_MODEL_VISIBLE_FIELDS).toContain("model");
    expect(SETTINGS_MODEL_VISIBLE_FIELDS).toContain("api_key");
    expect(SETTINGS_MODEL_VISIBLE_FIELDS).not.toContain("model_config_id");
    expect(SETTINGS_MODEL_VISIBLE_FIELDS).not.toContain("secret_ref");
  });

  it("uses compact list columns for model table", () => {
    expect(SETTINGS_MODEL_LIST_COLUMNS).toEqual([
      "provider",
      "model",
      "base_url",
      "temperature",
      "max_tokens",
      "actions"
    ]);
  });

  it("builds provider-first model suggestions with fallback", () => {
    const configs = [
      { provider: "openai", model: "gpt-5" },
      { provider: "openai", model: "gpt-5-mini" },
      { provider: "openai", model: "gpt-5" },
      { provider: "anthropic", model: "claude-sonnet-4-20250514" }
    ] as DataModelConfig[];

    expect(buildModelSuggestions(configs, "openai")).toEqual(["gpt-5", "gpt-5-mini"]);
    expect(buildModelSuggestions(configs, "google")).toEqual([
      "claude-sonnet-4-20250514",
      "gpt-5",
      "gpt-5-mini"
    ]);
  });

  it("matches configured model against catalog model_id case-insensitively", () => {
    expect(
      isModelAvailableInCatalog("GPT-5-mini ", [
        { model_id: "gpt-5" },
        { model_id: "gpt-5-mini" }
      ])
    ).toBe(true);
    expect(
      isModelAvailableInCatalog("gpt-4.1", [
        { model_id: "gpt-5" },
        { model_id: "gpt-5-mini" }
      ])
    ).toBe(false);
  });
});
