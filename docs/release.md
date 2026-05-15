<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Release

Releases are built from Git tags named `v*`, for example `v0.1.0`.

## Local Archive Build

Build a stamped archive for the current platform:

```bash
VERSION=v0.1.0 scripts/build-release.sh
```

Cross-compile by setting `GOOS` and `GOARCH`:

```bash
VERSION=v0.1.0 GOOS=linux GOARCH=amd64 scripts/build-release.sh
```

Artifacts are written to `dist/` as archives plus SHA-256 checksum files.

## GitHub Release

Push a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds Linux and macOS binaries for `amd64` and `arm64`, stamps `odh version`, uploads archives, and creates the GitHub Release.

The workflow can also be dispatched manually for an existing tag.
