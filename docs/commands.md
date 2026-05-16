<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Commands

## `odh version`

Prints build metadata. JSON is the default so scripts can capture exact binary provenance.

```bash
odh version
odh version --format text
```

Flags:

- `--format json` - default, machine-readable JSON.
- `--format text` - compact human-readable line.

## `odh doctor`

Runs non-interactive health checks for the CLI and selected public Open Data Hub endpoints.

```bash
odh doctor
odh doctor --network=false
odh doctor --timeout 5s
```

Example output with `--network=false`:

```json
{
  "ok": true,
  "version": {
    "version": "0.1.4-dev",
    "commit": "unknown",
    "date": "unknown",
    "goos": "darwin",
    "goarch": "arm64"
  },
  "checks": [
    {
      "name": "version",
      "ok": true,
      "message": "0.1.4-dev"
    },
    {
      "name": "api_registry",
      "ok": true,
      "message": "6 APIs configured"
    }
  ]
}
```

Flags:

- `--network` - run network reachability checks; default `true`.
- `--timeout duration` - overall timeout for doctor checks; default `10s`.

The command returns exit code `1` if any required check fails.

## `odh apis`

Lists the known API registry.

```bash
odh apis
```

Default output:

```json
{
  "apis": [
    {
      "name": "mobility",
      "title": "Open Data Hub Timeseries / Mobility API",
      "base_url": "https://mobility.api.opendatahub.com"
    }
  ]
}
```

Flags:

- `--format json` - default, machine-readable JSON.
- `--format table` - simple human-readable table.

## `odh datasets list`

Lists a small curated catalog of useful Open Data Hub entry points.

```bash
odh datasets list
odh datasets list --domain mobility --format table
```

Flags:

- `--domain tourism|mobility` - optional domain filter.
- `--format json` - default, machine-readable JSON.
- `--format table` - simple human-readable table.

## `odh datasets search <query>`

Searches the curated dataset catalog by title, description, keywords, endpoint, and suggested command.

```bash
odh datasets search parking
odh datasets search weather --domain tourism
```

Flags:

- `--domain tourism|mobility` - optional domain filter.
- `--format json` - default, machine-readable JSON.
- `--format table` - simple human-readable table.

## `odh openapi <api>`

Fetches the known OpenAPI spec for an API and writes it as JSON.

```bash
odh openapi mobility
odh openapi tourism
```

If the upstream spec is YAML, `odh` converts it to JSON before writing to stdout.

## `odh call <api> <path>`

Calls a path under a known API base URL.

```bash
odh call tourism /v1/ODHActivityPoi \
  --param pagenumber=1 \
  --param pagesize=1 \
  --param seed=42 \
  --param fields=Detail.en.Title,GpsInfo
```

Flags:

- `--param key=value` - query parameter. Repeatable.

The response must be JSON. Invalid JSON is treated as a command failure.

## `odh tourism poi`

Curated wrapper for the Tourism API `ODHActivityPoi` endpoint.

```bash
odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
```

Flags:

- `--limit n` - maps to `pagesize`; default `1`.
- `--page n` - maps to `pagenumber`; default `1`.
- `--seed value` - stable randomization seed.
- `--fields value` - comma-separated fields.
- `--param key=value` - additional query parameter. Repeatable.

## `odh tourism types`

Fetches common Tourism taxonomy/type datasets and wraps the upstream `Items` list with `dataset`, `endpoint`, and `count` fields.

```bash
odh tourism types --dataset poi --limit 20
odh tourism types --dataset event --limit 20
odh tourism types --dataset event-topic --limit 20
odh tourism types --dataset accommodation --limit 20
```

Supported datasets:

- `poi`
- `event`
- `event-topic`
- `accommodation`
- `article`
- `venue`
- `tag`

Flags:

- `--dataset value` - taxonomy dataset; default `poi`.
- `--limit n` - maps to `pagesize`; default `100`.
- `--page n` - maps to `pagenumber`; default `1`.
- `--seed value` - stable randomization seed.
- `--param key=value` - additional query parameter. Repeatable.

## `odh mobility latest`

Curated wrapper for Mobility API latest measurements.

```bash
odh mobility latest \
  --station-type EChargingStation \
  --data-type number-available \
  --limit 5
```

Flags:

- `--station-type value` - required.
- `--data-type value` - required.
- `--representation value` - default `flat,node`.
- `--limit n` - default `5`.
- `--offset n` - default `0`.
- `--where expr` - Open Data Hub where filter.
- `--param key=value` - additional query parameter. Repeatable.

## `odh mobility types`

Lists Mobility API station, event, or edge types.

```bash
odh mobility types --kind station
odh mobility types --kind event
odh mobility types --kind edge
```

Flags:

- `--kind station|event|edge` - default `station`.
- `--limit n` - maximum records to request; default `200`.

The output wraps the upstream list with `kind`, `count`, and `types` fields so agents can check whether discovery returned data before using it.

## `odh mobility stations`

Lists Mobility stations for a station type.

```bash
odh mobility stations --station-type ParkingStation --limit 5
odh mobility stations --station-type EChargingStation --limit 5
odh mobility stations --station-type TrafficSensor --origin A22 --limit 10
```

Flags:

- `--station-type value` - required.
- `--origin value` - optional `sorigin` filter, for example `A22`.
- `--representation value` - default `flat`.
- `--limit n` - maximum stations to request; default `20`.
- `--offset n` - pagination offset; default `0`.
- `--where expr` - Open Data Hub where filter.
- `--param key=value` - additional query parameter. Repeatable.

The output wraps matching records with `station_type`, `origin`, `record_count`, `count`, and `stations`.

## `odh mobility datatypes`

Summarizes available data types for a Mobility station type.

```bash
odh mobility datatypes \
  --station-type TrafficSensor \
  --origin A22 \
  --limit 100
```

Flags:

- `--station-type value` - required.
- `--origin value` - optional `sorigin` filter, for example `A22`.
- `--representation value` - default `flat`.
- `--limit n` - maximum station/data-type records to inspect; default `1000`.
- `--param key=value` - additional query parameter. Repeatable.

The output groups records by data type name and includes station counts, units, descriptions, and origins.

## `odh mobility events`

Fetches Mobility events for an origin.

```bash
odh mobility events --origin A22 --latest --limit 20
```

Flags:

- `--origin value` - required event origin.
- `--latest` - request latest events; default `true`.
- `--representation value` - default `flat`.
- `--limit n` - maximum events to request; default `20`.
- `--param key=value` - additional query parameter. Repeatable.

## `odh traffic today`

Opinionated wrapper for current South Tyrol traffic events from Open Data Hub Mobility `PROVINCE_BZ`.

```bash
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic today --area unterland --type closure --format markdown
odh traffic today --near 46.42,11.25 --radius 15km
odh traffic today --area bozen-unterland --json
```

The command filters to today, collapses duplicate event rows, hides expired and future items by default, and emits warnings when upstream timestamps look stale. Table output is the default for direct answers; use `--json` or `--format json` when an agent needs structured output. `odh traffic` intentionally stays on Open Data Hub data; if a human-facing traffic bulletin is required, compare the result with the official traffic service outside this CLI and mention the source used.

Flags:

- `--source odh` - traffic source; `odh` is the only supported source.
- `--area value` - area alias such as `ueberetsch-unterland`, `bozen-unterland`, `unterland`, `ueberetsch`, `salurn`, `kaltern`, `tramin`, `eppan`, `auer`, `neumarkt`, `kurtatsch`, `margreid`, or `montan`.
- `--type all|roadworks|closure|event|traffic|mountain-pass|bike|radar` - category filter; default `all`.
- `--road value` - road filter, for example `SP13`, `LS/SP 13`, or `SS42`.
- `--near lat,lon` - coordinate filter.
- `--radius value` - radius for `--near`; default `15km`.
- `--format json|table|markdown` - output format; default `table`.
- `--json` - shortcut for `--format json`.
- `--limit n` - maximum raw rows requested from Open Data Hub before local filtering; default `1000`.
- `--raw` - include raw upstream event objects in JSON output.
- `--include-expired` - include events outside the selected date range.
- `--include-stale` - include stale open-ended events hidden by default.

## `odh traffic events`

Date-range variant of the traffic wrapper.

```bash
odh traffic events --area bozen-unterland --from 2026-05-16 --to 2026-05-16
odh traffic events --area ueberetsch --from 2026-05-16 --to 2026-05-17 --type roadworks --road SP13
```

If both `--from` and `--to` are omitted, the command uses today's date. If only one side is supplied, it uses the same date for both sides.

Additional flags:

- `--from YYYY-MM-DD` - start date.
- `--to YYYY-MM-DD` - end date.

## `odh a22 status`

Diagnostic wrapper for A22 traffic-related Mobility feeds.

```bash
odh a22 status --limit 10
```

It checks current A22 events and the `TrafficForecast/forecast/latest` feed. The command returns JSON with `events`, `forecast`, and `warnings`. Warnings are part of the contract: for example, the command explicitly reports when Open Data Hub returns no current A22 events or when forecast timestamps are future-dated.

Flags:

- `--limit n` - maximum records to request from each feed; default `20`.

## Exit Codes

- `0` - success.
- `1` - runtime failure, such as HTTP failure or invalid upstream JSON.
- `2` - command usage error.
