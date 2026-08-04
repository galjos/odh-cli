<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# odh-cli

[![CI](https://github.com/galjos/odh-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/galjos/odh-cli/actions/workflows/ci.yml)
[![Release](https://github.com/galjos/odh-cli/actions/workflows/release.yml/badge.svg)](https://github.com/galjos/odh-cli/actions/workflows/release.yml)
[![Upstream smoke](https://github.com/galjos/odh-cli/actions/workflows/upstream-smoke.yml/badge.svg)](https://github.com/galjos/odh-cli/actions/workflows/upstream-smoke.yml)

`odh` is an unofficial command-line interface for public Open Data Hub APIs, with compact human output for curated workflows and JSON output for scripts and agents.

This is a community project, not affiliated with or endorsed by NOI Techpark or Open Data Hub. Most practical Tourism and Mobility data is centered on South Tyrol; verify record-level geography from returned coordinates and origins rather than assuming it.

## Features

- Discovery: known APIs, curated dataset catalog, guided dataset/source paths, OpenAPI specs, Tourism taxonomies, Mobility station/event/edge types, origins, and data types.
- Raw access: call any registered API path with query parameters.
- Mobility: stations, latest time-series measurements with active/freshness/sorting filters for availability questions.
- Traffic: deduplicated South Tyrol roadworks, closures, and road events from `PROVINCE_BZ` with zone discovery, date filtering, text search, and stale-record warnings; `--source content` reads the Content API road bulletin instead, where `--area`/`--zone-id` match by geographic inference from historical zone coordinates and the filters that feed cannot answer are rejected; A22 event/forecast inspection with explicit caveats.
- Transit: GTFS datasets, live GTFS-RT feeds, stop search, departures, direct trips, and static transfer-journey planning with optional realtime annotations.
- Data quality: diagnostics verdicts for EV charging availability, parking forecasts, and Tourism events before answering from those areas.
- Robustness: bounded retries, 24h caching of static-ish discovery responses, global `--timeout`, provenance and warning fields on curated JSON output.
- MCP server mode exposing the curated commands as Model Context Protocol tools.

Not included: authenticated write flows, live transit rerouting, historical GTFS-RT delay archives, and historical A22 incident reconstruction. Unsupported capabilities are reported explicitly instead of guessed.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh
```

The installer detects macOS/Linux and `amd64`/`arm64`, verifies the published SHA-256 checksum, and installs to `~/.local/bin`. Pass `--version v0.5.1 --dir "$HOME/bin"` to pin a version or directory.

Alternatives:

```bash
brew install galjos/odh/odh
sudo apt install ./odh_v0.5.1_linux_amd64.deb   # from GitHub Releases
go build -o odh ./cmd/odh                       # from source
```

Releases include per-asset checksums, a `SHA256SUMS` manifest, and GitHub artifact attestations (`gh attestation verify <asset> --repo galjos/odh-cli`). Shell completions: `odh completion bash|zsh|fish|powershell`.

## Quickstart

```bash
odh doctor
odh apis
odh datasets guide parking --format json
odh datasets search parking
odh call tourism /v1/ODHActivityPoi --param pagesize=1 --param fields=Detail.en.Title,GpsInfo
odh mobility origins --station-type ParkingStation
odh mobility latest --station-type EChargingStation --data-type number-available --active --fresh-within 24h --sort newest --limit 5
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic search "road closed badia" --today --json
odh a22 status --limit 10
odh transit stops search merano --limit 10
odh transit journey --from merano --to ora --time 16:40 --max-transfers 3 --with-realtime --json
```

Every command documents flags and examples in `odh <command> --help`.

## Automation And MCP

`odh` is script and agent friendly: stdout is data (JSON via `--json` or `--format json`), stderr is diagnostics, exit code `2` means bad invocation and `1` runtime failure, and commands are non-interactive. The contract, caveats, and stable JSON fields are documented in [docs/agent-usage.md](docs/agent-usage.md) and [docs/json-contracts.md](docs/json-contracts.md).

Run the same surface as MCP tools (see [docs/mcp.md](docs/mcp.md)):

```bash
claude mcp add odh -- odh mcp serve
```

Machine-readable command recipes are in [evals/agent/recipes.json](evals/agent/recipes.json); the agent skill bundle is in [skills/open-data-hub-cli/SKILL.md](skills/open-data-hub-cli/SKILL.md).

## Development

```bash
go fmt ./... && go vet ./... && go test ./... && go build ./cmd/odh
```

See [docs/development.md](docs/development.md), [docs/release.md](docs/release.md), and [docs/evaluation.md](docs/evaluation.md). Release notes are in [CHANGELOG.md](CHANGELOG.md); planned work is tracked in [GitHub issues](https://github.com/galjos/odh-cli/issues).

## Open Data Hub Links

- APIs and access: https://opendatahub.com/api/
- Datasets: https://docs.opendatahub.com/en/latest/datasets.html
- Tourism Swagger: https://tourism.opendatahub.com/swagger/index.html
- Mobility spec: https://mobility.api.opendatahub.com/v2/apispec
