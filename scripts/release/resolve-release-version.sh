#!/usr/bin/env bash
set -euo pipefail

raw_version="${RELEASE_VERSION_INPUT:-}"
event_name="${GITHUB_EVENT_NAME:-}"
ref_name="${GITHUB_REF_NAME:-}"

if [[ -z "$raw_version" && "$event_name" == "push" ]]; then
  raw_version="$ref_name"
fi

raw_version="$(echo "$raw_version" | tr -d '[:space:]')"
if [[ -z "$raw_version" ]]; then
  echo "[release-version] missing release version input" >&2
  exit 1
fi

normalized_version="${raw_version#v}"
normalized_version="${normalized_version#V}"
semver_regex='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
if [[ ! "$normalized_version" =~ $semver_regex ]]; then
  echo "[release-version] invalid semver version: ${raw_version}" >&2
  exit 1
fi

release_tag="v${normalized_version}"

if [[ -n "${GITHUB_ENV:-}" ]]; then
  {
    echo "GOYAIS_VERSION=${normalized_version}"
    echo "RELEASE_TAG=${release_tag}"
  } >>"$GITHUB_ENV"
fi

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  {
    echo "goyais_version=${normalized_version}"
    echo "release_tag=${release_tag}"
  } >>"$GITHUB_OUTPUT"
fi

echo "[release-version] version=${normalized_version} tag=${release_tag}"
