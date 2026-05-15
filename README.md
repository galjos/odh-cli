<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# odh-cli

`odh` is a JSON-first command-line interface for public Open Data Hub APIs.

It is built for developers, scripts, demos, and AI agents that need stable command behavior instead of scraping web UI pages. It wraps known Open Data Hub API entrypoints, fetches OpenAPI specs, and provides small curated commands for common Tourism and Mobility API calls.

## Status

This is an early v0.1 project. It intentionally focuses on a small working core:

- list known Open Data Hub APIs,
- report machine-readable build/version metadata,
- run a non-interactive doctor check for registry and upstream reachability,
- fetch OpenAPI specs as JSON,
- call any registered API path with query parameters,
- query a random Tourism POI,
- discover Mobility station, event, and edge types,
- summarize Mobility data types for a station type and origin,
- query latest Mobility time-series measurements,
- inspect A22 Mobility event and forecast feeds with explicit warnings when the data does not look like current traffic incidents.

It does not yet include MCP, generated clients, Homebrew packaging, Docker images, or authenticated write flows.

## Install

Build from source:

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
  -ldflags "-X github.com/galjos/odh-cli/internal/version.Version=0.1.0 -X github.com/galjos/odh-cli/internal/version.Commit=$(git rev-parse --short HEAD) -X github.com/galjos/odh-cli/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o odh ./cmd/odh
```

## Quickstart

Print version metadata:

```bash
./odh version
```

Check the local CLI and upstream API reachability:

```bash
./odh doctor
```

List known API surfaces:

```bash
./odh apis
```

Fetch the Mobility OpenAPI spec as JSON:

```bash
./odh openapi mobility
```

Call a Tourism API endpoint:

```bash
./odh call tourism /v1/ODHActivityPoi \
  --param pagenumber=1 \
  --param pagesize=1 \
  --param seed=42 \
  --param fields=Detail.en.Title,GpsInfo
```

Use the curated Tourism POI command:

```bash
./odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
```

Use the curated Mobility latest-measurements command:

```bash
./odh mobility latest \
  --station-type EChargingStation \
  --data-type number-available \
  --limit 5
```

Discover Mobility event origins:

```bash
./odh mobility types --kind event
```

Find A22 traffic-sensor data types:

```bash
./odh mobility datatypes \
  --station-type TrafficSensor \
  --origin A22 \
  --limit 100
```

Check the A22-specific diagnostic output:

```bash
./odh a22 status --limit 10
```

## Automation Contract

`odh` is designed to be script and agent friendly:

- stdout is machine-readable output,
- JSON is the default output for data commands,
- stderr is for diagnostics,
- failures return nonzero exit codes,
- commands are non-interactive,
- examples use public unauthenticated endpoints only.

See [docs/agent-usage.md](docs/agent-usage.md) for details.

## Open Data Hub Links

- Open Data Hub APIs: https://opendatahub.com/api/
- Data access overview: https://opendatahub.com/services/data-access/
- Mobility getting started: https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
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

Release builds are documented in [docs/release.md](docs/release.md).

The agent skill bundle is in [skills/open-data-hub-cli/SKILL.md](skills/open-data-hub-cli/SKILL.md).
