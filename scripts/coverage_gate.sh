#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <coverage_profile> <min_total_percent>"
  exit 1
fi

profile="$1"
min_coverage="$2"

if [[ ! -f "$profile" ]]; then
  echo "coverage profile not found: $profile"
  exit 1
fi

total_line="$(go tool cover -func="$profile" | tail -n 1)"
# expected format: total: (statements) 72.3%
total_percent="$(awk '{print $3}' <<<"$total_line" | tr -d '%')"

if [[ -z "$total_percent" ]]; then
  echo "failed to parse total coverage from: $total_line"
  exit 1
fi

awk -v total="$total_percent" -v min="$min_coverage" 'BEGIN {
  if (total + 0 < min + 0) {
    printf("Coverage gate failed: total=%.1f%%, required>=%.1f%%\n", total, min)
    exit 1
  }
  printf("Coverage gate passed: total=%.1f%%, required>=%.1f%%\n", total, min)
}'
