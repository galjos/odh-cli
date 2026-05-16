#!/usr/bin/env sh
# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: MPL-2.0

set -eu

repo="${ODH_REPO:-galjos/odh-cli}"
version="${ODH_VERSION:-latest}"
install_dir="${ODH_INSTALL_DIR:-$HOME/.local/bin}"
verify_checksum=1
dry_run=0

usage() {
  cat <<'EOF'
Install odh from GitHub Releases.

Usage:
  scripts/install.sh [--version v0.1.3] [--dir ~/.local/bin] [--repo owner/repo]

Environment:
  ODH_VERSION      Release tag to install, or "latest" (default: latest)
  ODH_INSTALL_DIR  Install directory (default: ~/.local/bin)
  ODH_REPO         GitHub repository (default: galjos/odh-cli)

Options:
  --version TAG    Release tag, with or without leading "v"
  --dir DIR        Install directory
  --repo OWNER/REPO
  --no-verify      Skip SHA-256 verification
  --dry-run        Print planned install without downloading
  -h, --help       Show this help
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || {
        echo "missing value for --version" >&2
        exit 2
      }
      version="$2"
      shift 2
      ;;
    --dir)
      [ "$#" -ge 2 ] || {
        echo "missing value for --dir" >&2
        exit 2
      }
      install_dir="$2"
      shift 2
      ;;
    --repo)
      [ "$#" -ge 2 ] || {
        echo "missing value for --repo" >&2
        exit 2
      }
      repo="$2"
      shift 2
      ;;
    --no-verify)
      verify_checksum=0
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

need curl
need tar

case "$(uname -s)" in
  Darwin)
    os="darwin"
    ;;
  Linux)
    os="linux"
    ;;
  *)
    echo "unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64 | amd64)
    arch="amd64"
    ;;
  arm64 | aarch64)
    arch="arm64"
    ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

resolve_latest_version() {
  latest_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")"
  latest_tag="${latest_url##*/}"
  if [ -z "$latest_tag" ] || [ "$latest_tag" = "latest" ]; then
    echo "could not resolve latest release for $repo" >&2
    exit 1
  fi
  printf '%s\n' "$latest_tag"
}

if [ "$version" = "latest" ]; then
  version="$(resolve_latest_version)"
fi

case "$version" in
  v*)
    tag="$version"
    ;;
  *)
    tag="v$version"
    ;;
esac

asset="odh_${tag}_${os}_${arch}.tar.gz"
base_url="https://github.com/$repo/releases/download/$tag"
archive_url="$base_url/$asset"
checksum_url="$archive_url.sha256"

if [ "$dry_run" -eq 1 ]; then
  cat <<EOF
repo=$repo
version=$tag
os=$os
arch=$arch
asset=$asset
install_dir=$install_dir
archive_url=$archive_url
checksum_url=$checksum_url
EOF
  exit 0
fi

tmpdir="$(mktemp -d 2>/dev/null || mktemp -d -t odh-install)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

echo "Downloading $archive_url" >&2
curl -fsSLo "$tmpdir/$asset" "$archive_url"

if [ "$verify_checksum" -eq 1 ]; then
  echo "Verifying SHA-256 checksum" >&2
  curl -fsSLo "$tmpdir/$asset.sha256" "$checksum_url"
  if command -v shasum >/dev/null 2>&1; then
    (cd "$tmpdir" && shasum -a 256 -c "$asset.sha256")
  elif command -v sha256sum >/dev/null 2>&1; then
    (cd "$tmpdir" && sha256sum -c "$asset.sha256")
  else
    echo "neither shasum nor sha256sum is available; rerun with --no-verify to skip checksum verification" >&2
    exit 1
  fi
fi

tar -xzf "$tmpdir/$asset" -C "$tmpdir"
binary="$tmpdir/odh_${tag}_${os}_${arch}/odh"
if [ ! -x "$binary" ]; then
  echo "archive did not contain executable odh binary" >&2
  exit 1
fi

mkdir -p "$install_dir"
cp "$binary" "$install_dir/odh"
chmod 0755 "$install_dir/odh"

echo "Installed odh to $install_dir/odh" >&2
"$install_dir/odh" version --format text

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    echo "Warning: $install_dir is not on PATH" >&2
    ;;
esac
