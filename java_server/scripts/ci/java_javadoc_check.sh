#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

collect_files() {
  rg --files java_server \
    -g '*.java' \
    -g '!**/target/**' \
    -g '!**/build/**'
}

line_trimmed() {
  local file="$1"
  local lineno="$2"
  sed -n "${lineno}p" "${file}" | sed -E 's/^[[:space:]]+//;s/[[:space:]]+$//'
}

javadoc_bounds_before() {
  local file="$1"
  local line="$2"
  local cursor=$(( line - 1 ))
  local text

  while (( cursor > 0 )); do
    text="$(line_trimmed "${file}" "${cursor}")"
    if [[ -z "${text}" || "${text}" =~ ^@ ]]; then
      cursor=$(( cursor - 1 ))
      continue
    fi
    break
  done

  if (( cursor <= 0 )); then
    return 1
  fi

  text="$(line_trimmed "${file}" "${cursor}")"
  if [[ "${text}" != *"*/"* ]]; then
    return 1
  fi

  local end_line="${cursor}"

  while (( cursor > 0 )); do
    text="$(line_trimmed "${file}" "${cursor}")"
    if [[ "${text}" == "/**"* ]]; then
      printf '%s:%s\n' "${cursor}" "${end_line}"
      return 0
    fi
    if [[ "${text}" == "/*"* && "${text}" != "/**"* ]]; then
      return 1
    fi
    cursor=$(( cursor - 1 ))
  done

  return 1
}

extract_signature() {
  local file="$1"
  local line="$2"
  awk -v start="${line}" '
    NR < start { next }
    {
      print
      if (index($0, ")") > 0) {
        seen_paren = 1
      }
      if (seen_paren && ($0 ~ /\{/ || $0 ~ /;/)) {
        exit
      }
      if ((NR - start) > 60) {
        exit
      }
    }
  ' "${file}" | tr '\n' ' ' | sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//'
}

method_name_from_signature() {
  local signature="$1"
  perl -e '
    my $sig = shift // q{};
    my $name = q{};
    while ($sig =~ /([A-Za-z_][A-Za-z0-9_]*)\s*\(/g) {
      $name = $1;
    }
    print $name;
  ' "${signature}"
}

param_names_from_signature() {
  local signature="$1"
  perl -e '
    my $sig = shift // q{};
    my $start = index($sig, q{(});
    if ($start < 0) { exit 0; }

    my $depth = 0;
    my $in = 0;
    my $params = q{};
    for (my $i = $start; $i < length($sig); $i++) {
      my $ch = substr($sig, $i, 1);
      if ($ch eq q{(}) {
        if ($in) { $depth++; $params .= $ch; }
        else { $in = 1; }
        next;
      }
      if ($ch eq q{)}) {
        if ($depth == 0) { last; }
        $depth--;
        $params .= $ch;
        next;
      }
      if ($in) { $params .= $ch; }
    }

    my @parts;
    my $buf = q{};
    my $angle = 0;
    for (my $i = 0; $i < length($params); $i++) {
      my $ch = substr($params, $i, 1);
      if ($ch eq q{<}) { $angle++; $buf .= $ch; next; }
      if ($ch eq q{>}) { $angle-- if $angle > 0; $buf .= $ch; next; }
      if ($ch eq q{,} && $angle == 0) {
        push @parts, $buf;
        $buf = q{};
        next;
      }
      $buf .= $ch;
    }
    push @parts, $buf if $buf ne q{};

    for my $raw (@parts) {
      my $p = $raw;
      $p =~ s/\@\w+(?:\s*\([^()]*\))?\s*//g;
      $p =~ s/\b(final|volatile|transient)\b\s*//g;
      $p =~ s/\s+/ /g;
      $p =~ s/^\s+|\s+$//g;
      next if $p eq q{};
      if ($p =~ /([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[\s*\])*\s*$/) {
        print "$1\n";
      }
    }
  ' "${signature}"
}

throws_from_signature() {
  local signature="$1"
  perl -e '
    my $sig = shift // q{};
    if ($sig !~ /\)\s*throws\s+([^\{;]+)/) {
      exit 0;
    }
    my $raw = $1;
    my @parts;
    my $buf = q{};
    my $angle = 0;
    for (my $i = 0; $i < length($raw); $i++) {
      my $ch = substr($raw, $i, 1);
      if ($ch eq q{<}) { $angle++; $buf .= $ch; next; }
      if ($ch eq q{>}) { $angle-- if $angle > 0; $buf .= $ch; next; }
      if ($ch eq q{,} && $angle == 0) {
        push @parts, $buf;
        $buf = q{};
        next;
      }
      $buf .= $ch;
    }
    push @parts, $buf if $buf ne q{};

    for my $item (@parts) {
      $item =~ s/^\s+|\s+$//g;
      next if $item eq q{};
      $item =~ s/\s+/ /g;
      print "$item\n";
    }
  ' "${signature}"
}

class_names_for_file() {
  local file="$1"
  rg -n '^[[:space:]]*(public|protected|private|static|final|abstract|sealed|non-sealed)[[:space:]]+.*(class|interface|enum|record)[[:space:]]+[A-Za-z_][A-Za-z0-9_]*|^[[:space:]]*(class|interface|enum|record)[[:space:]]+[A-Za-z_][A-Za-z0-9_]*' "${file}" \
    | sed -E 's/.*(class|interface|enum|record)[[:space:]]+([A-Za-z_][A-Za-z0-9_]*).*/\2/' \
    | sort -u
}

contains_exact_line() {
  local needle="$1"
  local haystack="$2"
  printf '%s\n' "${haystack}" | rg -Fx -- "${needle}" >/dev/null 2>&1
}

escape_regex() {
  sed -E 's/[][(){}.^$*+?|\\]/\\&/g' <<<"$1"
}

main() {
  local failed=0
  local total=0
  local file

  while IFS= read -r file; do
    [[ -n "${file}" ]] || continue
    total=$((total + 1))

    local class_names
    class_names="$(class_names_for_file "${file}")"

    local lineno text

    while IFS=: read -r lineno text; do
      [[ -n "${lineno}" ]] || continue
      if ! javadoc_bounds_before "${file}" "${lineno}" >/dev/null; then
        echo "[java_javadoc_check] missing_type_javadoc: ${file}:${lineno}"
        failed=$((failed + 1))
      fi
    done < <(rg -n '^[[:space:]]*(public|protected)[[:space:]]+(class|interface|enum|record)\b' "${file}" || true)

    while IFS=: read -r lineno text; do
      [[ -n "${lineno}" ]] || continue
      if printf '%s\n' "${text}" | rg -q '\b(class|interface|enum|record)\b'; then
        continue
      fi

      local bounds
      if ! bounds="$(javadoc_bounds_before "${file}" "${lineno}")"; then
        echo "[java_javadoc_check] missing_method_javadoc: ${file}:${lineno}"
        failed=$((failed + 1))
        continue
      fi

      local start_line end_line doc_text signature method_name
      start_line="${bounds%%:*}"
      end_line="${bounds##*:}"
      doc_text="$(sed -n "${start_line},${end_line}p" "${file}")"
      signature="$(extract_signature "${file}" "${lineno}")"
      method_name="$(method_name_from_signature "${signature}")"

      if [[ -z "${method_name}" ]]; then
        echo "[java_javadoc_check] parse_method_name_failed: ${file}:${lineno}"
        failed=$((failed + 1))
        continue
      fi

      if printf '%s\n' "${doc_text}" | rg -n '\{\@inheritDoc\}' >/dev/null 2>&1; then
        continue
      fi

      while IFS= read -r param_name; do
        [[ -n "${param_name}" ]] || continue
        if ! printf '%s\n' "${doc_text}" | rg -n "@param[[:space:]]+${param_name}\b" >/dev/null 2>&1; then
          echo "[java_javadoc_check] missing_param_tag(${param_name}): ${file}:${lineno}"
          failed=$((failed + 1))
        fi
      done < <(param_names_from_signature "${signature}")

      local is_ctor=0
      if contains_exact_line "${method_name}" "${class_names}"; then
        is_ctor=1
      fi

      if [[ "${is_ctor}" -eq 0 ]] && ! printf '%s\n' "${signature}" | rg -n "\bvoid[[:space:]]+${method_name}[[:space:]]*\(" >/dev/null 2>&1; then
        if ! printf '%s\n' "${doc_text}" | rg -n '@return\b' >/dev/null 2>&1; then
          echo "[java_javadoc_check] missing_return_tag: ${file}:${lineno}"
          failed=$((failed + 1))
        fi
      fi

      while IFS= read -r throws_name; do
        [[ -n "${throws_name}" ]] || continue
        local simple_name escaped_full escaped_simple
        simple_name="${throws_name##*.}"
        escaped_full="$(escape_regex "${throws_name}")"
        escaped_simple="$(escape_regex "${simple_name}")"
        if ! printf '%s\n' "${doc_text}" | rg -n "@throws[[:space:]]+(${escaped_full}|${escaped_simple})\b" >/dev/null 2>&1; then
          echo "[java_javadoc_check] missing_throws_tag(${throws_name}): ${file}:${lineno}"
          failed=$((failed + 1))
        fi
      done < <(throws_from_signature "${signature}")
    done < <(rg -n '^[[:space:]]*(public|protected)[[:space:]]+[^=;{}]*\(' "${file}" || true)

    while IFS=: read -r lineno text; do
      [[ -n "${lineno}" ]] || continue
      if ! javadoc_bounds_before "${file}" "${lineno}" >/dev/null; then
        echo "[java_javadoc_check] missing_field_javadoc: ${file}:${lineno}"
        failed=$((failed + 1))
      fi
    done < <(rg -n '^[[:space:]]*(public|protected)[[:space:]]+((static|final|transient|volatile|synchronized|strictfp)[[:space:]]+)*[^=;(){}]+[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*(=|;)' "${file}" || true)

  done < <(collect_files)

  if (( failed > 0 )); then
    echo "[java_javadoc_check] failed=${failed} files=${total}"
    exit 1
  fi

  echo "[java_javadoc_check] passed files=${total}"
}

main "$@"
