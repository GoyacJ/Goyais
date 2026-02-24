import type { ModelCatalogVendor } from "@/shared/types/api";

export function resolveDefaultEndpointKey(vendor: ModelCatalogVendor | null | undefined): string {
  if (!vendor || vendor.name === "Local") {
    return "";
  }
  const entries = Object.entries(vendor.base_urls ?? {})
    .map(([key, value]) => [key.trim(), value.trim()] as const)
    .filter(([key, value]) => key !== "" && value !== "");
  if (entries.length === 0) {
    return "";
  }
  const normalizedBaseURL = normalizeURL(vendor.base_url);
  if (normalizedBaseURL !== "") {
    const matched = entries.find(([, value]) => normalizeURL(value) === normalizedBaseURL);
    if (matched) {
      return matched[0];
    }
  }
  entries.sort(([a], [b]) => a.localeCompare(b));
  return entries[0][0];
}

function normalizeURL(raw: string): string {
  return raw.trim().replace(/\/+$/, "");
}
