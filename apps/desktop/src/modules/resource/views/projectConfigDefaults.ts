import { projectStore } from "@/modules/project/store";
import type { ProjectConfig } from "@/shared/types/api";

export function getProjectConfig(projectId: string): ProjectConfig {
  const config = projectStore.projectConfigsByProjectId[projectId];
  if (config) {
    return config;
  }
  return {
    project_id: projectId,
    model_config_ids: [],
    default_model_config_id: null,
    rule_ids: [],
    skill_ids: [],
    mcp_ids: [],
    updated_at: new Date().toISOString()
  };
}
