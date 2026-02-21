import { invoke } from "@tauri-apps/api/core";

const LOCAL_HUB_AUTH_PROVIDER = "local_hub_auth";
const LOCAL_HUB_AUTH_PROFILE = "default";

export interface LocalHubCredentials {
  email: string;
  password: string;
  displayName: string;
}

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

export async function storeLocalHubCredentials(credentials: LocalHubCredentials): Promise<void> {
  await setProviderSecret(
    LOCAL_HUB_AUTH_PROVIDER,
    LOCAL_HUB_AUTH_PROFILE,
    JSON.stringify({
      email: credentials.email,
      password: credentials.password,
      displayName: credentials.displayName
    })
  );
}

export async function loadLocalHubCredentials(): Promise<LocalHubCredentials | null> {
  const raw = await getProviderSecret(LOCAL_HUB_AUTH_PROVIDER, LOCAL_HUB_AUTH_PROFILE);
  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as Partial<LocalHubCredentials>;
    if (
      typeof parsed.email !== "string"
      || typeof parsed.password !== "string"
      || typeof parsed.displayName !== "string"
      || !parsed.email.trim()
      || !parsed.password.trim()
      || !parsed.displayName.trim()
    ) {
      return null;
    }

    return {
      email: parsed.email.trim(),
      password: parsed.password,
      displayName: parsed.displayName.trim()
    };
  } catch {
    return null;
  }
}
