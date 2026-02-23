import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId } from "@/shared/services/mockData";
import type { ABACPolicy } from "@/shared/types/api";

const mockPoliciesByWorkspace = new Map<string, ABACPolicy[]>();

export async function listABACPolicies(workspaceId: string): Promise<ABACPolicy[]> {
  return withApiFallback(
    "admin.listABACPolicies",
    () => getControlClient().get<ABACPolicy[]>(`/v1/admin/abac-policies?workspace_id=${encodeURIComponent(workspaceId)}`),
    () => cloneABACPolicies(ensureMockPolicies(workspaceId))
  );
}

export async function upsertABACPolicy(workspaceId: string, input: ABACPolicy): Promise<ABACPolicy> {
  const payload: ABACPolicy = { ...input, workspace_id: workspaceId };
  return withApiFallback(
    "admin.upsertABACPolicy",
    () => getControlClient().post<ABACPolicy>(`/v1/admin/abac-policies?workspace_id=${encodeURIComponent(workspaceId)}`, payload),
    () => {
      const items = ensureMockPolicies(workspaceId);
      const target: ABACPolicy = {
        ...payload,
        id: payload.id.trim() || createMockId("abac"),
        created_at: payload.created_at ?? new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      const index = items.findIndex((item) => item.id === target.id);
      if (index >= 0) {
        items[index] = target;
      } else {
        items.push(target);
      }
      return target;
    }
  );
}

export async function patchABACPolicy(
  workspaceId: string,
  policyId: string,
  patch: Partial<ABACPolicy>
): Promise<ABACPolicy> {
  return withApiFallback(
    "admin.patchABACPolicy",
    () =>
      getControlClient().request<ABACPolicy>(`/v1/admin/abac-policies/${encodeURIComponent(policyId)}`, {
        method: "PATCH",
        body: patch
      }),
    () => {
      const items = ensureMockPolicies(workspaceId);
      const target = items.find((item) => item.id === policyId);
      if (!target) {
        throw new Error("Policy not found");
      }
      Object.assign(target, patch, { updated_at: new Date().toISOString() });
      return { ...target };
    }
  );
}

export async function deleteABACPolicy(workspaceId: string, policyId: string): Promise<void> {
  return withApiFallback(
    "admin.removeABACPolicy",
    async () => {
      await getControlClient().request<void>(
        `/v1/admin/abac-policies/${encodeURIComponent(policyId)}?workspace_id=${encodeURIComponent(workspaceId)}`,
        { method: "DELETE" }
      );
    },
    () => {
      const items = ensureMockPolicies(workspaceId).filter((item) => item.id !== policyId);
      mockPoliciesByWorkspace.set(workspaceId, items);
    }
  );
}

function ensureMockPolicies(workspaceId: string): ABACPolicy[] {
  const cached = mockPoliciesByWorkspace.get(workspaceId);
  if (cached) {
    return cached;
  }

  const created: ABACPolicy[] = [
    {
      id: createMockId("abac"),
      workspace_id: workspaceId,
      name: "allow self workspace",
      effect: "allow",
      priority: 100,
      enabled: true,
      subject_expr: { roles: { in: ["developer", "approver", "admin"] } },
      resource_expr: { workspace_id: { eq: "$subject.workspace_id" } },
      action_expr: {},
      context_expr: {},
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    }
  ];
  mockPoliciesByWorkspace.set(workspaceId, created);
  return created;
}

function cloneABACPolicies(items: ABACPolicy[]): ABACPolicy[] {
  return items.map((item) => ({
    ...item,
    subject_expr: { ...item.subject_expr },
    resource_expr: { ...item.resource_expr },
    action_expr: { ...item.action_expr },
    context_expr: { ...item.context_expr }
  }));
}
