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
    "version": "0.1.1-dev",
    "commit": "unknown",
    "date": "unknown",
    "goos": "darwin",
    "goarch": "arm64"
  },
  "checks": [
    {
      "name": "version",
      "ok": true,
      "message": "0.1.1-dev"
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
