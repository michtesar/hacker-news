#!/usr/bin/env bash
set -euo pipefail

# Accept common FOSS license families.
allowed_regex='MIT|Apache|BSD|ISC|MPL|LGPL|GPL|Unlicense|CC0|Zlib|EPL|CDDL|Artistic|Python'

main_module="$(go list -m -f '{{.Path}}')"

fail=0
while IFS= read -r line; do
  [[ -z "$line" ]] && continue

  module_path="${line%%::*}"
  module_dir="${line##*::}"

  if [[ "$module_path" == "$main_module" ]]; then
    continue
  fi

  # If module source is not present locally, skip with warning; this can happen
  # for metadata-only entries and should not hard-fail the pipeline.
  if [[ -z "$module_dir" || "$module_dir" == "<nil>" || ! -d "$module_dir" ]]; then
    echo "[WARN] Skipping module without local dir: $module_path"
    continue
  fi

  license_file=""
  for candidate in LICENSE LICENSE.txt LICENSE.md COPYING COPYING.txt COPYING.md LICENCE LICENCE.txt; do
    if [[ -f "$module_dir/$candidate" ]]; then
      license_file="$module_dir/$candidate"
      break
    fi
  done

  if [[ -z "$license_file" ]]; then
    echo "[FAIL] No license file found for module: $module_path"
    fail=1
    continue
  fi

  head_text="$(head -n 120 "$license_file" || true)"
  if ! grep -Eiq "$allowed_regex" <<<"$head_text"; then
    echo "[FAIL] Unknown or unsupported license for module: $module_path (file: $license_file)"
    fail=1
    continue
  fi
done < <(go list -deps -f '{{with .Module}}{{.Path}}::{{.Dir}}{{end}}' ./... | sort -u)

if [[ "$fail" -ne 0 ]]; then
  echo "FOSS license compliance check failed."
  exit 1
fi

echo "FOSS license compliance check passed."
