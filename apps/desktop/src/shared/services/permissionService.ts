import { getControlClient } from "@/shared/services/clients";
import type { PermissionSnapshot } from "@/shared/types/api";

export async function getPermissionSnapshot(token?: string): Promise<PermissionSnapshot> {
  return getControlClient().get<PermissionSnapshot>("/v1/me/permissions", { token });
}
