<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# odh-cli

`odh` is an unofficial JSON-first command-line interface for public Open Data Hub APIs.

It is built for developers, scripts, demos, and AI agents that need stable command behavior instead of scraping web UI pages. It wraps known Open Data Hub API entrypoints, fetches OpenAPI specs, and provides small curated commands for common Tourism and Mobility API calls.

## Disclaimer

This is an unofficial community project. It is not affiliated with, endorsed by, or maintained by NOI Techpark or Open Data Hub unless ownership is explicitly transferred or adopted by those organizations.

## Data Scope

Open Data Hub is maintained by NOI Techpark and provides a machine-readable access point for datasets in domains such as mobility, tourism, environment, and more. In practical use, many of the public Tourism and Mobility datasets are centered on South Tyrol, also known as the Autonomous Province of Bolzano.

The official documentation also describes Open Data Hub as an international and European data platform that is expanding beyond its original regional scope. For that reason, `odh` does not hardcode a "South Tyrol only" assumption. When geographic precision matters, inspect the returned fields, coordinates, origins, provider metadata, and upstream API docs.

See [docs/data-scope.md](docs/data-scope.md) for the official-source summary used by this project.

## Status

This is an early v0.1 project. It intentionally focuses on a small working core:

- list known Open Data Hub APIs,
- report machine-readable build/version metadata,
- run a non-interactive doctor check for registry and upstream reachability,
- search a small curated dataset catalog for common entry points,
- fetch OpenAPI specs as JSON,
- call any registered API path with query parameters,
- query a random Tourism POI,
- discover Tourism taxonomies such as POI, event, accommodation, venue, article, and tag types,
- discover Mobility station, event, and edge types,
- list Mobility stations for a station type,
- summarize Mobility data types for a station type and origin,
- query latest Mobility time-series measurements,
- inspect A22 Mobility event and forecast feeds with explicit warnings when the data does not look like current traffic incidents.

It does not yet include MCP, generated clients, Homebrew packaging, Docker images, a full upstream metadata catalog, or authenticated write flows.

## Install

Install the latest GitHub release:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh
```

The installer detects macOS/Linux and `amd64`/`arm64`, downloads the matching release archive, verifies the published SHA-256 checksum, and installs `odh` to `~/.local/bin` by default.

Install a specific version or directory:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh -s -- --version v0.1.2 --dir "$HOME/bin"
```

Build from source instead:

```bash
go build -o odh ./cmd/odh
```

Run directly during development:

```bash
go run ./cmd/odh --help
```

Build with release metadata:

```bash
go build \
  -ldflags "-X github.com/galjos/odh-cli/internal/version.Version=0.1.2 -X github.com/galjos/odh-cli/internal/version.Commit=$(git rev-parse --short HEAD) -X github.com/galjos/odh-cli/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o odh ./cmd/odh
```

## Quickstart

Print version metadata:

```bash
odh version
```

Check the local CLI and upstream API reachability:

```bash
odh doctor
```

List known API surfaces:

```bash
odh apis
```

Search the curated dataset catalog:

```bash
odh datasets search parking
```

Fetch the Mobility OpenAPI spec as JSON:

```bash
odh openapi mobility
```

Call a Tourism API endpoint:

```bash
odh call tourism /v1/ODHActivityPoi \
  --param pagenumber=1 \
  --param pagesize=1 \
  --param seed=42 \
  --param fields=Detail.en.Title,GpsInfo
```

Use the curated Tourism POI command:

```bash
odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
```

Discover Tourism event taxonomy values:

```bash
odh tourism types --dataset event --limit 10
```

Use the curated Mobility latest-measurements command:

```bash
odh mobility latest \
  --station-type EChargingStation \
  --data-type number-available \
  --limit 5
```

Discover Mobility event origins:

```bash
odh mobility types --kind event
```

List Mobility parking stations:

```bash
odh mobility stations --station-type ParkingStation --limit 5
```

Find A22 traffic-sensor data types:

```bash
odh mobility datatypes \
  --station-type TrafficSensor \
  --origin A22 \
  --limit 100
```

Check the A22-specific diagnostic output:

```bash
odh a22 status --limit 10
```

## Automation Contract

`odh` is designed to be script and agent friendly:

- stdout is machine-readable output,
- JSON is the default output for data commands,
- stderr is for diagnostics,
- failures return nonzero exit codes,
- commands are non-interactive,
- examples use public endpoints only,
- commands avoid hidden browser state and web scraping.

See [docs/agent-usage.md](docs/agent-usage.md) for details.

## Open Data Hub Links

- Open Data Hub APIs: https://opendatahub.com/api/
- Data access overview: https://opendatahub.com/services/data-access/
- About Open Data Hub: https://opendatahub.com/about-us/
- Domains and datasets: https://docs.opendatahub.com/en/latest/datasets.html
- Mobility getting started: https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
- Tourism data browser: https://tourism.databrowser.opendatahub.com/
- Tourism Swagger UI: https://tourism.opendatahub.com/swagger/index.html
- Mobility OpenAPI spec: https://mobility.api.opendatahub.com/v2/apispec

## Development

```bash
go fmt ./...
go test ./...
go vet ./...
go build ./cmd/odh
```

Optional live smoke tests:

```bash
ODH_LIVE_TESTS=1 go test ./internal/commands -run Live
```

More details are in [docs/development.md](docs/development.md).

Installation details are in [docs/install.md](docs/install.md).

Release builds are documented in [docs/release.md](docs/release.md).

The agent skill bundle is in [skills/open-data-hub-cli/SKILL.md](skills/open-data-hub-cli/SKILL.md).
