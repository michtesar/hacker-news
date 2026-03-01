#!/usr/bin/env bash
set -euo pipefail

# Allowed FOSS license families. Keep this list explicit.
allowed_regex='MIT|Apache|BSD|ISC|MPL|LGPL|GPL|Unlicense|CC0|Zlib|EPL|CDDL|Artistic|Python'

unknown=0
module_lines="$(go list -mod=readonly -m -f '{{.Path}}::{{.Dir}}' all)"

while IFS= read -r line; do
  module_path="${line%%::*}"
  module_dir="${line##*::}"

  # Skip the main module.
  if [[ "$module_path" == "github.com/michael/hacker-news" ]]; then
    continue
  fi

  license_file=""
  for candidate in LICENSE LICENSE.txt LICENSE.md COPYING COPYING.txt; do
    if [[ -f "$module_dir/$candidate" ]]; then
      license_file="$module_dir/$candidate"
      break
    fi
  done

  if [[ -z "$module_dir" || "$module_dir" == "<nil>" ]]; then
    echo "[FAIL] Module directory missing for: $module_path"
    unknown=1
    continue
  fi

  if [[ -z "$license_file" ]]; then
    echo "[FAIL] No license file found for module: $module_path"
    unknown=1
    continue
  fi

  head_text="$(head -n 80 "$license_file" || true)"
  if ! grep -Eiq "$allowed_regex" <<<"$head_text"; then
    echo "[FAIL] Non-FOSS or unknown license for module: $module_path (file: $license_file)"
    unknown=1
    continue
  fi

  if grep -Eiq 'all rights reserved|proprietary|commercial license' <<<"$head_text"; then
    echo "[FAIL] Potential proprietary wording for module: $module_path (file: $license_file)"
    unknown=1
  fi
done <<<"$module_lines"

if [[ "$unknown" -ne 0 ]]; then
  echo "FOSS license compliance check failed."
  exit 1
fi

echo "FOSS license compliance check passed."
