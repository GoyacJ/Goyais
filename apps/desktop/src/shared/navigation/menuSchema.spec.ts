import { describe, expect, it } from "vitest";

import { getMenuKeysForContext, menuSchema } from "@/shared/navigation/menuSchema";

describe("menu schema", () => {
  it("keeps menu keys unique in a single source of truth", () => {
    const keys = menuSchema.map((item) => item.key);
    expect(new Set(keys).size).toBe(keys.length);
  });

  it("builds account menu keys from remote+workspace groups", () => {
    expect(getMenuKeysForContext("account")).toEqual([
      "remote_account",
      "remote_members_roles",
      "remote_permissions_audit",
      "workspace_project_config",
      "workspace_agent",
      "workspace_model",
      "workspace_rules",
      "workspace_skills",
      "workspace_mcp"
    ]);
  });

  it("builds settings menu keys from workspace+settings groups", () => {
    expect(getMenuKeysForContext("settings")).toEqual([
      "workspace_agent",
      "workspace_model",
      "workspace_rules",
      "workspace_skills",
      "workspace_mcp",
      "workspace_project_config",
      "settings_theme",
      "settings_i18n",
      "settings_updates_diagnostics",
      "settings_general"
    ]);
  });
});
