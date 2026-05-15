<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Commands

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
- `--param key=value` - additional query parameter. Repeatable.

## Exit Codes

- `0` - success.
- `1` - runtime failure, such as HTTP failure or invalid upstream JSON.
- `2` - command usage error.
