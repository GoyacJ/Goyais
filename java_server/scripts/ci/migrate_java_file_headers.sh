#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

MIGRATION_SINCE="${MIGRATION_SINCE:-$(date '+%Y-%m-%d %H:%M:%S')}"

collect_files() {
  rg --files java_server \
    -g '*.java' \
    -g '!**/target/**' \
    -g '!**/build/**'
}

updated=0
total=0

while IFS= read -r file; do
  [[ -n "${file}" ]] || continue
  total=$((total + 1))

  before_hash="$(shasum "${file}" | awk '{print $1}')"

  MIGRATION_SINCE="${MIGRATION_SINCE}" perl -0777 -i -pe '
    my $since = $ENV{MIGRATION_SINCE};
    my $spdx = q{SPDX-License-Identifier: Apache-2.0};

    my $desc = q{Goyais source file.};
    if (/\A\s*\/\*\*(.*?)\*\//s) {
      my $block = $1;
      if ($block =~ /Description:\s*(.+?)\s*(?:\r?\n|\*\/)/s) {
        $desc = $1;
      } elsif ($block =~ /<p>(.+?)<\/p>/s) {
        $desc = $1;
      }
      $desc =~ s/^\s+|\s+$//g;
      $desc =~ s/\s+/ /g;
      $desc = q{Goyais source file.} if $desc eq q{};

      my $header = "/**\n"
        . " * ${spdx}\n"
        . " * <p>${desc}</p>\n"
        . " * \@author Goya\n"
        . " * \@since ${since}\n"
        . " */\n\n";

      s/\A\s*\/\*\*.*?\*\/\s*/$header/s;
    } else {
      my $header = "/**\n"
        . " * ${spdx}\n"
        . " * <p>${desc}</p>\n"
        . " * \@author Goya\n"
        . " * \@since ${since}\n"
        . " */\n\n";
      s/\A/$header/s;
    }
  ' "${file}"

  after_hash="$(shasum "${file}" | awk '{print $1}')"
  if [[ "${before_hash}" != "${after_hash}" ]]; then
    updated=$((updated + 1))
  fi
done < <(collect_files)

echo "[migrate_java_file_headers] total=${total} updated=${updated} since=${MIGRATION_SINCE}"
