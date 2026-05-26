<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Install

The released package name is `odh-cli` / `open-data-hub-cli`, while the installed binary is `odh`.

## Homebrew

Install with Homebrew:

```bash
brew install galjos/odh/odh
```

Or tap first:

```bash
brew tap galjos/odh
brew install odh
```

Verify:

```bash
odh version
odh doctor --network=false
```

The tap repository is <https://github.com/galjos/homebrew-odh>.

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
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh -s -- --version v0.2.1
```

Install into a different directory:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh -s -- --dir "$HOME/bin"
```

For audited installs, download the script first and read it before running:

```bash
curl -fsSLo install-odh.sh https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh
sh install-odh.sh --version v0.2.1 --dir "$HOME/bin"
```

## Debian Package

GitHub Releases include `.deb` packages for Linux `amd64` and `arm64`.

```bash
curl -LO https://github.com/galjos/odh-cli/releases/download/v0.2.1/odh_v0.2.1_linux_amd64.deb
curl -LO https://github.com/galjos/odh-cli/releases/download/v0.2.1/odh_v0.2.1_linux_amd64.deb.sha256
shasum -a 256 -c odh_v0.2.1_linux_amd64.deb.sha256
sudo apt install ./odh_v0.2.1_linux_amd64.deb
```

Use `odh_v0.2.1_linux_arm64.deb` on ARM64 Linux.

This is a direct Debian package install, not an APT repository. It will not
auto-upgrade through `apt upgrade` until a signed APT repository exists.

## Environment

- `ODH_VERSION` - release tag to install, or `latest`.
- `ODH_INSTALL_DIR` - install directory.
- `ODH_REPO` - GitHub repository, default `galjos/odh-cli`.

Example:

```bash
ODH_VERSION=v0.2.1 ODH_INSTALL_DIR="$HOME/bin" sh scripts/install.sh
```

## Verify

```bash
odh version
odh doctor --timeout 10s
```

## Go Install

Agents or developer machines with Go installed can also use:

```bash
go install github.com/galjos/odh-cli/cmd/odh@v0.2.1
```

This is the installer path declared in the OpenClaw skill metadata.

## Homebrew Tap Maintenance

The Homebrew formula lives in <https://github.com/galjos/homebrew-odh>. For a
new release, update the tap formula and verify it by name:

```bash
brew audit --strict --online galjos/odh/odh
brew install --build-from-source --skip-link galjos/odh/odh
brew test galjos/odh/odh
```
