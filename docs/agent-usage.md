<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Agent Usage

`odh` is designed for automation and AI coding agents.

## Contract

- Use stdout only for command results.
- Use stderr for diagnostics.
- Return nonzero exit codes on errors.
- Do not prompt interactively.
- Emit JSON by default, except `odh traffic` which defaults to table output and supports `--json`.
- Keep examples public and unauthenticated.

This means agents can call `odh`, parse stdout as JSON, and treat stderr plus exit code as failure context.

## Safe Starter Commands

```bash
odh version
odh doctor --timeout 5s
odh apis
odh datasets search parking
odh openapi mobility
odh tourism types --dataset event --limit 10
odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
odh mobility types --kind event
odh mobility stations --station-type ParkingStation --limit 5
odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 100
odh mobility events --origin A22 --latest --limit 20
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic events --area bozen-unterland --from 2026-05-16 --to 2026-05-16 --json
odh traffic today --near 46.42,11.25 --radius 15km --json
odh mobility latest --station-type EChargingStation --data-type number-available --limit 5
odh a22 status --limit 10
```

## Data Scope Rule

Open Data Hub is maintained by NOI Techpark and many of the current public Tourism and Mobility datasets are centered on South Tyrol / the Autonomous Province of Bolzano. Do not claim that every record is in South Tyrol unless the returned data supports that.

When geography matters, agents should inspect fields such as:

- Tourism: `GpsInfo`, `LocationInfo`, `RegionInfo`, `MunicipalityInfo`, `LicenseInfo`.
- Mobility: `sorigin`, `scode`, `stype`, `scoordinate`, `smetadata`.

Use [data-scope.md](data-scope.md) as the project-level source note.

## Generic Calls

When an endpoint is known from OpenAPI docs, use `odh call` instead of scraping a UI:

```bash
odh call tourism /v1/ODHActivityPoi \
  --param pagenumber=1 \
  --param pagesize=1 \
  --param seed=42
```

When the endpoint is not known yet, start with catalog and type discovery:

```bash
odh datasets search <topic>
odh tourism types --dataset event
odh mobility types --kind station
odh mobility stations --station-type ParkingStation --limit 5
```

## Handling Failures

Agents should treat exit code `2` as a usage bug in the invocation and exit code `1` as a runtime problem such as HTTP failure, invalid JSON, or unavailable upstream service.

`odh doctor` is the preferred first command when an agent needs to distinguish a bad invocation from a local install problem or an upstream Open Data Hub reachability problem.

## Traffic Data Caveat

Open Data Hub Mobility feeds can expose different traffic concepts as station measurements, events, and forecasts. For South Tyrol roadworks, closures, road events, and traffic restrictions, prefer the opinionated traffic layer:

```bash
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic events --area unterland --from 2026-05-16 --to 2026-05-16 --type closure --json
odh traffic today --near 46.42,11.25 --radius 15km
```

Traffic commands query Open Data Hub `PROVINCE_BZ` events and default to table output. Use `--json` or `--format json` for downstream parsing and `--format table` or `--format markdown` for direct human answers. The traffic layer deduplicates rows, maps German/Italian event categories to stable English category names, filters expired/future rows by date, hides stale open-ended records by default, and warns when timestamps look stale.

If a user needs an exact public traffic bulletin, compare the Open Data Hub result with the official traffic service outside this CLI and state both the source and timestamp used. Do not imply that `odh traffic` silently switched to another upstream feed.

Agents should not infer live A22 traffic solely from `TrafficForecast` rows. Prefer `odh a22 status` when checking A22 because it reports current-event availability and warns when forecast timestamps indicate non-current data.

If the user asks for current traffic conditions, report the timestamp and feed type used. If Open Data Hub has no current A22 event rows, say that directly instead of converting forecast rows into live incidents.

## Why No MCP Yet

The project is structured so an MCP server can reuse the registry and HTTP client later. v0.1 ships the CLI first because it is simpler to review, easier to test, and immediately useful in scripts.
