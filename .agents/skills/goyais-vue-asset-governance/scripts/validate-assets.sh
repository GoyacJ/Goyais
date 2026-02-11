#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
ASSET_ROOT="${REPO_ROOT}/vue_web/src/assets"
NOTICES_FILE="${ASSET_ROOT}/THIRD_PARTY_NOTICES.md"
CATALOG_FILE="${ASSET_ROOT}/RESOURCE_CATALOG.yaml"

fail() {
  echo "[validate-assets] ERROR: $*" >&2
  exit 1
}

[[ -d "$ASSET_ROOT" ]] || fail "missing assets root: ${ASSET_ROOT}"
[[ -f "$NOTICES_FILE" ]] || fail "missing notices file: ${NOTICES_FILE}"
[[ -f "$CATALOG_FILE" ]] || fail "missing catalog file: ${CATALOG_FILE}"

if rg -n 'https?://' "$ASSET_ROOT" --glob '*.svg' | rg -v 'www\.w3\.org/2000/svg|www\.w3\.org/1999/xlink' >/tmp/goyais_asset_links.txt; then
  cat /tmp/goyais_asset_links.txt
  fail "external URL detected in SVG files"
fi

rg -n 'Heroicons' "$NOTICES_FILE" >/dev/null || fail "Heroicons not found in notices"
rg -n 'unDraw' "$NOTICES_FILE" >/dev/null || fail "unDraw not found in notices"
rg -n 'MIT' "$NOTICES_FILE" >/dev/null || fail "MIT license not recorded"

entry_count="$(rg -c '^- asset_id:' "$CATALOG_FILE" || true)"
[[ "$entry_count" -gt 0 ]] || fail "catalog has no entries"

for key in type scenario local_path license source_url version_or_date token_constraints; do
  key_count="$(rg -c "^  ${key}:" "$CATALOG_FILE" || true)"
  [[ "$key_count" -eq "$entry_count" ]] || fail "catalog key '${key}' count (${key_count}) != entry count (${entry_count})"
done

while IFS= read -r path; do
  [[ -f "${REPO_ROOT}/${path}" ]] || fail "catalog path does not exist: ${path}"
done < <(awk '/^  local_path: /{print $2}' "$CATALOG_FILE")

if rg -n '#[0-9A-Fa-f]{3,8}' "$ASSET_ROOT/icons" "$ASSET_ROOT/illustrations/states" "$ASSET_ROOT/bg" >/tmp/goyais_asset_hex.txt; then
  cat /tmp/goyais_asset_hex.txt
  fail "hardcoded hex color found in token-aligned SVG directories"
fi

echo "[validate-assets] OK"
