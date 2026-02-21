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

export async function deleteToken(profileId: string): Promise<void> {
  await invoke("delete_token", {
    profileId
  });
}

export async function setProviderSecret(provider: string, profile: string, value: string): Promise<void> {
  await invoke("secret_set", {
    provider,
    profile,
    value
  });
}

export async function getProviderSecret(provider: string, profile: string): Promise<string | null> {
  const value = await invoke<string | null>("secret_get", {
    provider,
    profile
  });
  return value ?? null;
}
