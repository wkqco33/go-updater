#!/usr/bin/env bash
set -euo pipefail

profile=${1:?usage: check-coverage.sh coverage.out [minimum-percent]}
minimum=${2:-50}

actual=$(go tool cover -func="$profile" | awk '/^total:/ {gsub("%", "", $3); print $3}')
if [[ -z "$actual" ]]; then
  echo "coverage total was not found in $profile" >&2
  exit 1
fi

awk -v actual="$actual" -v minimum="$minimum" 'BEGIN {
  if (actual + 0 < minimum + 0) {
    printf "coverage %.1f%% is below required %.1f%%\n", actual, minimum > "/dev/stderr"
    exit 1
  }
  printf "coverage %.1f%% meets required %.1f%%\n", actual, minimum
}'
