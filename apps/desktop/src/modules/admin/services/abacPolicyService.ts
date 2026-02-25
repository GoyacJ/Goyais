import { getControlClient } from "@/shared/services/clients";
import type { ABACPolicy } from "@/shared/types/api";

export async function listABACPolicies(workspaceId: string): Promise<ABACPolicy[]> {
  return getControlClient().get<ABACPolicy[]>(`/v1/admin/abac-policies?workspace_id=${encodeURIComponent(workspaceId)}`);
}

export async function upsertABACPolicy(workspaceId: string, input: ABACPolicy): Promise<ABACPolicy> {
  const payload: ABACPolicy = { ...input, workspace_id: workspaceId };
  return getControlClient().post<ABACPolicy>(`/v1/admin/abac-policies?workspace_id=${encodeURIComponent(workspaceId)}`, payload);
}

export async function patchABACPolicy(
  workspaceId: string,
  policyId: string,
  patch: Partial<ABACPolicy>
): Promise<ABACPolicy> {
  return getControlClient().request<ABACPolicy>(
    `/v1/admin/abac-policies/${encodeURIComponent(policyId)}?workspace_id=${encodeURIComponent(workspaceId)}`,
    {
      method: "PATCH",
      body: patch
    }
  );
}

export async function deleteABACPolicy(workspaceId: string, policyId: string): Promise<void> {
  await getControlClient().request<void>(
    `/v1/admin/abac-policies/${encodeURIComponent(policyId)}?workspace_id=${encodeURIComponent(workspaceId)}`,
    { method: "DELETE" }
  );
}
