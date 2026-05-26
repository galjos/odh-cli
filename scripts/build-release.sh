#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: MPL-2.0

set -euo pipefail

VERSION="${VERSION:-0.2.4-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
TARGET_OS="${GOOS:-$(go env GOOS)}"
TARGET_ARCH="${GOARCH:-$(go env GOARCH)}"
DIST_DIR="${DIST_DIR:-dist}"

if [[ -z "$VERSION" || -z "$COMMIT" || -z "$DATE" ]]; then
  echo "VERSION, COMMIT, and DATE are required" >&2
  exit 2
fi

package_name="odh_${VERSION}_${TARGET_OS}_${TARGET_ARCH}"
package_dir="${DIST_DIR}/${package_name}"
binary_name="odh"
if [[ "$TARGET_OS" == "windows" ]]; then
  binary_name="odh.exe"
fi

rm -rf "$package_dir"
mkdir -p "$package_dir"

CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build \
  -trimpath \
  -ldflags "-s -w -X github.com/galjos/odh-cli/internal/version.Version=${VERSION} -X github.com/galjos/odh-cli/internal/version.Commit=${COMMIT} -X github.com/galjos/odh-cli/internal/version.Date=${DATE}" \
  -o "${package_dir}/${binary_name}" ./cmd/odh

cp README.md LICENSE "$package_dir/"

mkdir -p "$DIST_DIR"
if [[ "$TARGET_OS" == "windows" ]]; then
  (cd "$DIST_DIR" && zip -qr "${package_name}.zip" "$package_name")
  (cd "$DIST_DIR" && shasum -a 256 "${package_name}.zip" >"${package_name}.zip.sha256")
else
  (cd "$DIST_DIR" && tar -czf "${package_name}.tar.gz" "$package_name")
  (cd "$DIST_DIR" && shasum -a 256 "${package_name}.tar.gz" >"${package_name}.tar.gz.sha256")
fi

rm -rf "$package_dir"
