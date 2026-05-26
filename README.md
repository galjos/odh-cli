<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# odh-cli

`odh` is an unofficial command-line interface for public Open Data Hub APIs, with compact human output for curated workflows and JSON output for scripts and agents.

It is built for developers, scripts, demos, and AI agents that need stable command behavior instead of scraping web UI pages. It wraps known Open Data Hub API entrypoints, fetches OpenAPI specs, and provides small curated commands for common Tourism and Mobility API calls, including an opinionated traffic layer over Open Data Hub `PROVINCE_BZ` events.

## Disclaimer

This is an unofficial community project. It is not affiliated with, endorsed by, or maintained by NOI Techpark or Open Data Hub unless ownership is explicitly transferred or adopted by those organizations.

## Data Scope

Open Data Hub is maintained by NOI Techpark and provides a machine-readable access point for datasets in domains such as mobility, tourism, environment, and more. In practical use, many of the public Tourism and Mobility datasets are centered on South Tyrol, also known as the Autonomous Province of Bolzano.

The official documentation also describes Open Data Hub as an international and European data platform that is expanding beyond its original regional scope. For that reason, `odh` does not hardcode a "South Tyrol only" assumption. When geographic precision matters, inspect the returned fields, coordinates, origins, provider metadata, and upstream API docs.

See [docs/data-scope.md](docs/data-scope.md) for the official-source summary used by this project.

## Status

This is a v0.2 project with a small, working, agent-friendly core:

- list known Open Data Hub APIs,
- report machine-readable build/version metadata,
- generate shell completion scripts for `bash`, `zsh`, `fish`, and `powershell`,
- run a non-interactive doctor check for registry and upstream reachability,
- bound any command with an opt-in global `--timeout` for scripts and agents,
- search a small curated dataset catalog for common entry points,
- fetch OpenAPI specs as JSON,
- call any registered API path with query parameters,
- retry transient HTTP `429` and `5xx` failures with bounded exponential backoff,
- cache low-risk discovery responses such as OpenAPI specs, taxonomies, and station metadata for 24 hours,
- include source, endpoint, warning, and data-kind metadata on curated JSON outputs where provenance matters,
- query a random Tourism POI,
- discover Tourism taxonomies such as POI, event, accommodation, venue, article, and tag types,
- discover Mobility station, event, and edge types,
- list Mobility stations for a station type,
- summarize Mobility data types for a station type and origin,
- query latest Mobility time-series measurements, with optional local origin, active-station, freshness, and sorting filters for agent-friendly availability checks,
- run data-quality diagnostics for EV charging availability, parking forecasts, and Tourism event caveats,
- list GTFS datasets and inspect GTFS-RT trip updates, vehicle positions, and service alerts,
- search STA timetable stops, inspect static GTFS departures and direct trip matches, and plan static transfer journeys,
- annotate static transit journeys with matching current GTFS-RT trip updates and service alerts,
- summarize South Tyrol roadworks, closures, events, and traffic notices from Open Data Hub `PROVINCE_BZ` with zone discovery, date filtering, text search, deduplication, and stale-record warnings,
- inspect A22 Mobility event and forecast feeds with explicit warnings when the data does not look like current traffic incidents.

It does not yet include MCP, generated clients, Docker images, a full upstream metadata catalog, authenticated write flows, full live public-transport rerouting, historical GTFS-RT delay archives, or historical A22 incident reconstruction.

See [CHANGELOG.md](CHANGELOG.md) for release notes.

## Install

Install the latest GitHub release:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh
```

The installer detects macOS/Linux and `amd64`/`arm64`, downloads the matching release archive, verifies the published SHA-256 checksum, and installs `odh` to `~/.local/bin` by default. Releases also include a `SHA256SUMS` manifest and GitHub artifact attestations for independent verification.

Install with Homebrew:

```bash
brew install galjos/odh/odh
```

Install a Linux Debian package from GitHub Releases:

```bash
curl -LO https://github.com/galjos/odh-cli/releases/download/v0.2.5/odh_v0.2.5_linux_amd64.deb
sudo apt install ./odh_v0.2.5_linux_amd64.deb
```

Install a specific version or directory:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh -s -- --version v0.2.5 --dir "$HOME/bin"
```

Build from source instead:

```bash
go build -o odh ./cmd/odh
```

Run directly during development:

```bash
go run ./cmd/odh --help
go run ./cmd/odh version
```

When running from a source checkout, pass the `odh` subcommand directly after `./cmd/odh`; do not insert an extra `--` before the subcommand.

Install shell completions:

```bash
mkdir -p ~/.zfunc
odh completion zsh > ~/.zfunc/_odh
mkdir -p ~/.local/share/bash-completion/completions
odh completion bash > ~/.local/share/bash-completion/completions/odh
mkdir -p ~/.config/fish/completions
odh completion fish > ~/.config/fish/completions/odh.fish
```

Build with release metadata:

```bash
go build \
  -ldflags "-X github.com/galjos/odh-cli/internal/version.Version=0.2.5 -X github.com/galjos/odh-cli/internal/version.Commit=$(git rev-parse --short HEAD) -X github.com/galjos/odh-cli/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
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

Bound a command in automation:

```bash
odh --timeout 15s traffic today --area ueberetsch-unterland --format table
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

`--param` is repeatable and preserves comma-separated API values such as Tourism `fields`.

Use the curated Tourism POI command:

```bash
odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
```

Discover Tourism event taxonomy values:

```bash
odh tourism types --dataset event --limit 10
odh tourism types --dataset event --limit 10 --json
```

Use the curated Mobility latest-measurements command:

```bash
odh mobility latest \
  --station-type EChargingStation \
  --data-type number-available \
  --origin ALPERIA \
  --active \
  --fresh-within 24h \
  --sort newest \
  --request-limit 1000 \
  --limit 5 \
  --format table
```

Check whether a data area is currently reliable enough to answer from:

```bash
odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h
odh diagnostics tourism-events --date 2026-05-18
```

Discover Mobility station types and station origins:

```bash
odh mobility types --kind station
odh mobility origins --station-type TrafficSensor
```

List Open Data Hub GTFS datasets and inspect live GTFS-RT trip updates:

```bash
odh gtfs datasets --format table
odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5
```

Search public-transport stops and check timetable data:

```bash
odh transit stops search auer
odh transit departures --stop "Ora, Stazione di Ora" --date 2026-05-16 --around 14:05
odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train
odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train --json
odh transit departures --stop-id <stop_id-from-search> --date 2026-05-16 --around 14:05 --mode train
odh transit trip --from-stop-id <origin-stop-id> --to-stop-id <destination-stop-id> --date 2026-05-16 --time 14:05 --mode train
odh transit journey --from-stop-id <origin-stop-id> --to-stop-id <destination-stop-id> --date 2026-05-16 --time 14:05 --max-transfers 3
odh transit journey --from-stop-id <origin-stop-id> --to-stop-id <destination-stop-id> --date 2026-05-16 --time 14:05 --max-transfers 3 --with-realtime --json
odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday
```

`odh transit journey` plans static GTFS transfer itineraries. Add `--with-realtime` to annotate the returned static journeys with matching current GTFS-RT trip delays, service alerts, adjusted leg times, and transfer-risk hints. It still does not reroute live. `odh transit delay-stats` is intentionally explicit: it reports that historical delay probability is unsupported until a GTFS-RT archive collector exists. It does not guess probabilities from the live feed.

Use `odh transit stops search` first and switch to stop IDs when a place name matches many platforms or bus bays. Parent station IDs are valid and expand to their child platform stops.

Summarize roadworks and closures for South Tyrol traffic zones and event text:

```bash
odh traffic zones
odh traffic categories
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic search "road closed badia" --today --zone-id 6 --json
odh traffic events --area bozen-unterland --from 2026-05-16 --to 2026-05-16 --format json
```

List Mobility parking stations:

```bash
odh mobility stations --station-type ParkingStation --limit 5
```

Find A22 traffic-sensor data types:

```bash
odh mobility origins --station-type TrafficSensor
odh mobility datatypes \
  --station-type TrafficSensor \
  --origin A22 \
  --limit 1000
odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 1000 --json
```

Check the A22-specific diagnostic output:

```bash
odh a22 status --limit 10
```

## Automation Contract

`odh` is designed to be script and agent friendly:

- stdout is command output,
- JSON is available on data commands through `--json` or `--format json`,
- curated answer/discovery commands such as `traffic`, `a22 status`, `transit`, `tourism types`, and `mobility types`/`datatypes` default to compact table output,
- stderr is for diagnostics,
- failures return nonzero exit codes,
- commands are non-interactive,
- usage errors return exit code `2`, while runtime failures return exit code `1`,
- current-data commands are not cached by the generic HTTP cache,
- examples use public endpoints only,
- commands avoid hidden browser state and web scraping.

See [docs/agent-usage.md](docs/agent-usage.md) for details.

Task-oriented examples for common traffic, parking, EV, transit, Tourism, and
raw API questions are in [docs/how-to.md](docs/how-to.md).

Stable JSON fields for curated outputs are documented in [docs/json-contracts.md](docs/json-contracts.md).

Data-quality caveats and diagnostics are described in [docs/data-quality.md](docs/data-quality.md).

## Open Data Hub Links

- Open Data Hub APIs: https://opendatahub.com/api/
- Data access overview: https://opendatahub.com/services/data-access/
- About Open Data Hub: https://opendatahub.com/about-us/
- Domains and datasets: https://docs.opendatahub.com/en/latest/datasets.html
- Mobility getting started: https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
- GTFS API base: https://gtfs.api.opendatahub.com
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

Agent evaluation smoke checks:

```bash
scripts/run-agent-evals.sh
```

More details are in [docs/development.md](docs/development.md).

Installation details are in [docs/install.md](docs/install.md).

Release builds are documented in [docs/release.md](docs/release.md).

Agent evaluation is documented in [docs/evaluation.md](docs/evaluation.md).

Machine-readable agent recipes are in [evals/agent/recipes.json](evals/agent/recipes.json).

The agent skill bundle is in [skills/open-data-hub-cli/SKILL.md](skills/open-data-hub-cli/SKILL.md).
