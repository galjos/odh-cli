#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: MPL-2.0

set -euo pipefail

DIST_DIR="${DIST_DIR:-dist}"

if [[ ! -d "$DIST_DIR" ]]; then
  echo "dist directory does not exist: $DIST_DIR" >&2
  exit 2
fi

artifacts=()
while IFS= read -r -d '' artifact; do
  artifacts+=("$(basename "$artifact")")
done < <(
  find "$DIST_DIR" -maxdepth 1 -type f \
    \( -name 'odh_*.tar.gz' -o -name 'odh_*.zip' -o -name 'odh_*.deb' \) \
    -print0 | sort -z
)

if [[ "${#artifacts[@]}" -eq 0 ]]; then
  echo "no release artifacts found in $DIST_DIR" >&2
  exit 2
fi

(
  cd "$DIST_DIR"
  for artifact in "${artifacts[@]}"; do
    shasum -a 256 "$artifact"
  done >SHA256SUMS
  shasum -a 256 SHA256SUMS >SHA256SUMS.sha256
)
