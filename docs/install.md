<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Install

The released package name is `odh-cli` / `open-data-hub-cli`, while the installed binary is `odh`.

## Release Installer

Install the latest GitHub release:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh
```

The installer:

- detects `darwin` or `linux`,
- detects `amd64` or `arm64`,
- downloads the matching archive from GitHub Releases,
- downloads the matching `.sha256` file,
- verifies the archive checksum,
- installs `odh` into `~/.local/bin` unless another directory is configured.

Install a specific release:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh -s -- --version v0.1.9
```

Install into a different directory:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh -s -- --dir "$HOME/bin"
```

For audited installs, download the script first and read it before running:

```bash
curl -fsSLo install-odh.sh https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh
sh install-odh.sh --version v0.1.9 --dir "$HOME/bin"
```

## Environment

- `ODH_VERSION` - release tag to install, or `latest`.
- `ODH_INSTALL_DIR` - install directory.
- `ODH_REPO` - GitHub repository, default `galjos/odh-cli`.

Example:

```bash
ODH_VERSION=v0.1.9 ODH_INSTALL_DIR="$HOME/bin" sh scripts/install.sh
```

## Verify

```bash
odh version
odh doctor --timeout 10s
```

## Go Install

Agents or developer machines with Go installed can also use:

```bash
go install github.com/galjos/odh-cli/cmd/odh@v0.1.9
```

This is the installer path declared in the OpenClaw skill metadata.
