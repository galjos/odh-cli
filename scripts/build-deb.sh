#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: MPL-2.0

set -euo pipefail

VERSION="${VERSION:-0.2.2-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
TARGET_OS="${GOOS:-linux}"
TARGET_ARCH="${GOARCH:-$(go env GOARCH)}"
DIST_DIR="${DIST_DIR:-dist}"

if [[ "$TARGET_OS" != "linux" ]]; then
  echo "Debian packages can only be built for linux targets, got GOOS=$TARGET_OS" >&2
  exit 2
fi

case "$TARGET_ARCH" in
  amd64 | arm64)
    deb_arch="$TARGET_ARCH"
    ;;
  *)
    echo "unsupported Debian package architecture: $TARGET_ARCH" >&2
    exit 2
    ;;
esac

if ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "dpkg-deb is required to build Debian packages" >&2
  exit 2
fi

deb_version="${VERSION#v}"
deb_version="${deb_version/-dev/~dev}"

package_name="odh_${VERSION}_linux_${TARGET_ARCH}"
package_root="${DIST_DIR}/${package_name}.debroot"
deb_path="${DIST_DIR}/${package_name}.deb"

rm -rf "$package_root" "$deb_path"
mkdir -p \
  "$package_root/DEBIAN" \
  "$package_root/usr/bin" \
  "$package_root/usr/share/doc/odh"

CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_ARCH" go build \
  -trimpath \
  -ldflags "-s -w -X github.com/galjos/odh-cli/internal/version.Version=${VERSION} -X github.com/galjos/odh-cli/internal/version.Commit=${COMMIT} -X github.com/galjos/odh-cli/internal/version.Date=${DATE}" \
  -o "$package_root/usr/bin/odh" ./cmd/odh

install -m 0644 README.md "$package_root/usr/share/doc/odh/README.md"
install -m 0644 LICENSE "$package_root/usr/share/doc/odh/copyright"

installed_size="$(du -sk "$package_root/usr" | awk '{print $1}')"
cat >"$package_root/DEBIAN/control" <<CONTROL
Package: odh
Version: ${deb_version}
Section: utils
Priority: optional
Architecture: ${deb_arch}
Maintainer: Josef M. Gallmetzer <64498081+galjos@users.noreply.github.com>
Homepage: https://github.com/galjos/odh-cli
Installed-Size: ${installed_size}
Description: Agent-friendly CLI for public Open Data Hub APIs
 odh is an unofficial JSON-first command-line interface for public Open Data
 Hub APIs. It is built for developers, scripts, demos, and AI agents that need
 stable command behavior instead of scraping web UI pages.
CONTROL

dpkg-deb --build --root-owner-group "$package_root" "$deb_path"
(cd "$DIST_DIR" && shasum -a 256 "$(basename "$deb_path")" >"$(basename "$deb_path").sha256")
rm -rf "$package_root"
