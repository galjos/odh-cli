<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Release

Releases are built from Git tags named `v*`, for example `v0.2.4`.

Before tagging, update `CHANGELOG.md` with the release date and the user-facing milestone summary.

## Local Archive Build

Build a stamped archive for the current platform:

```bash
VERSION=v0.2.4 scripts/build-release.sh
```

Cross-compile by setting `GOOS` and `GOARCH`:

```bash
VERSION=v0.2.4 GOOS=linux GOARCH=amd64 scripts/build-release.sh
```

Artifacts are written to `dist/` as archives plus per-asset SHA-256 checksum
files. Linux targets can also produce Debian packages:

```bash
VERSION=v0.2.4 GOOS=linux GOARCH=amd64 scripts/build-deb.sh
```

Generate the aggregate release manifest locally when testing release assets:

```bash
scripts/build-checksums.sh
```

## GitHub Release

Push a version tag:

```bash
git tag v0.2.4
git push origin v0.2.4
```

The release workflow builds Linux and macOS binaries for `amd64` and `arm64`,
stamps `odh version`, uploads archives, adds `.deb` packages for Linux targets,
generates `SHA256SUMS`, creates GitHub artifact attestations for every release
artifact in that manifest, and creates the GitHub Release.

The workflow can also be dispatched manually for an existing tag.

Release archives are named with this pattern:

```text
odh_<tag>_<goos>_<goarch>.tar.gz
odh_<tag>_linux_<goarch>.deb
```

The installer in `scripts/install.sh` depends on that asset naming and the adjacent `.sha256` files.

## Verify A Published Release

```bash
gh release view v0.2.4 --repo galjos/odh-cli
gh release download v0.2.4 --repo galjos/odh-cli --pattern SHA256SUMS --pattern 'odh_v0.2.4_darwin_arm64.tar.gz'
grep 'odh_v0.2.4_darwin_arm64.tar.gz' SHA256SUMS
shasum -a 256 odh_v0.2.4_darwin_arm64.tar.gz
gh attestation verify odh_v0.2.4_darwin_arm64.tar.gz --repo galjos/odh-cli
```

The checksum printed by `shasum` must match `SHA256SUMS`. The attestation check
must verify against `galjos/odh-cli`.
