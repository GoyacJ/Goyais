import { invoke } from "@tauri-apps/api/core";

export async function storeToken(profileId: string, token: string): Promise<void> {
  await invoke("store_token", {
    profileId,
    token
  });
}

export async function loadToken(profileId: string): Promise<string | null> {
  const token = await invoke<string | null>("load_token", {
    profileId
  });
  return token ?? null;
}
