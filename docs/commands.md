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
    "version": "0.1.9-dev",
    "commit": "unknown",
    "date": "unknown",
    "goos": "darwin",
    "goarch": "arm64"
  },
  "checks": [
    {
      "name": "version",
      "ok": true,
      "message": "0.1.9-dev"
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

OpenAPI responses are cached locally for 24 hours because they are low-risk discovery metadata.

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

Tourism type responses can be served from the local 24-hour metadata cache.

## `odh diagnostics`

Data-quality checks for areas where raw upstream data can be stale, semantically surprising, or incomplete.

```bash
odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h --forecast-minutes 60
odh diagnostics tourism-events --date 2026-05-18 --limit 20
```

The diagnostics commands return JSON with `domain`, `source`, `verdict`, and `warnings`. Treat `unavailable` as "no reliable current data from the checked request", not as zero availability or proof that the whole upstream domain has no records.

### `odh diagnostics ev-charging`

Checks `EChargingStation/number-available/latest` through the filtered Mobility latest path.

Flags:

- `--origin value` - optional `sorigin` filter, for example `ALPERIA`.
- `--fresh-within duration` - freshness window; default `24h`.
- `--limit n` - maximum filtered rows to include; default `10`.
- `--request-limit n` - upstream rows to inspect before filtering; default `10000`.

### `odh diagnostics parking-forecasts`

Compares current parking occupancy (`free`) with a parking forecast data type such as `parking-forecast-60`.

Flags:

- `--origin value` - optional `sorigin` filter, for example `Municipality Merano`.
- `--forecast-minutes n` - forecast horizon; default `60`.
- `--fresh-within duration` - freshness window for current and forecast rows; default `2h`.
- `--limit n` - maximum filtered rows to include per feed; default `10`.
- `--request-limit n` - upstream rows to inspect before filtering; default `10000`.

### `odh diagnostics tourism-events`

Checks Tourism `EventShort` rows for local date status, upstream `ActiveToday` caveats, and missing GPS fields.

Flags:

- `--date YYYY-MM-DD` - date used for local active checks; default today.
- `--only-active` - request upstream `onlyactive=true`; default `true`.
- `--limit n` - number of upstream events to inspect; default `20`.
- `--page n` - page number; default `1`.
- `--param key=value` - additional query parameter. Repeatable.

## `odh mobility latest`

Curated wrapper for Mobility API latest measurements.

```bash
odh mobility latest \
  --station-type EChargingStation \
  --data-type number-available \
  --origin ALPERIA \
  --active \
  --fresh-within 24h \
  --sort newest \
  --request-limit 1000 \
  --limit 5
```

Flags:

- `--station-type value` - required.
- `--data-type value` - required.
- `--representation value` - default `flat,node`.
- `--limit n` - final number of measurements to output; default `5`.
- `--request-limit n` - upstream rows to inspect before local filtering. Used with local filters; default is at least `1000`.
- `--offset n` - default `0`.
- `--origin value` - local `sorigin` filter, case-insensitive, for example `ALPERIA`.
- `--active` - keep only rows whose station is active.
- `--fresh-within duration` - keep only rows whose `mvalidtime` is within this age, for example `24h` or `7d`.
- `--sort upstream|newest|oldest|station` - local sort mode; default `upstream`.
- `--where expr` - Open Data Hub where filter.
- `--param key=value` - additional query parameter. Repeatable.

Without local flags, output is the raw upstream JSON response. With `--origin`, `--active`, `--fresh-within`, or a local `--sort`, the command returns a wrapper containing `station_type`, `data_type`, `raw_count`, `count`, `measurements`, and optional `warnings`. This keeps the raw command lightweight while allowing agents to avoid stale or inactive availability rows.

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

This discovery response can be served from the local 24-hour metadata cache.

## `odh mobility origins`

Summarizes available `sorigin` values for a Mobility station type.

```bash
odh mobility origins --station-type TrafficSensor
odh mobility origins --station-type ParkingStation --limit 1000
```

Flags:

- `--station-type value` - required.
- `--representation value` - default `flat`.
- `--limit n` - maximum station records to inspect; default `1000`.
- `--param key=value` - additional query parameter. Repeatable.

The output wraps discovered origin names with station counts and a few station-code samples. Use an origin from this command with `odh mobility stations --origin ...`, `odh mobility datatypes --origin ...`, or origin-specific Mobility event commands when the upstream feed supports them.

This discovery response can be served from the local 24-hour metadata cache.

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

This station-metadata response can be served from the local 24-hour metadata cache. Latest measurements are not served from this cache.

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

This discovery response can be served from the local 24-hour metadata cache.

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

## `odh gtfs datasets`

Lists Open Data Hub GTFS datasets and their metadata.

```bash
odh gtfs datasets
odh gtfs datasets --format table
```

Flags:

- `--format json|table` - output format; default `json`.

The JSON output includes `endpoint`, `count`, and `datasets`. Dataset metadata can expose available realtime feeds such as `trip-updates`, `vehicle-positions`, and `service-alerts`.

## `odh gtfs realtime`

Fetches a GTFS-RT JSON feed from Open Data Hub.

```bash
odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5
odh gtfs realtime --dataset sta-time-tables --feed vehicle-positions --route-id 101
odh gtfs realtime --dataset sta-time-tables --feed service-alerts --raw
```

Flags:

- `--dataset value` - GTFS dataset id; default `sta-time-tables`.
- `--feed trip-updates|vehicle-positions|service-alerts` - realtime feed; default `trip-updates`.
- `--limit n` - maximum entities to include; default `20`, use `0` for all.
- `--trip-id value` - optional `trip_id` filter for trip updates.
- `--route-id value` - optional `route_id` filter for trip updates or vehicle positions.
- `--raw` - write the upstream JSON feed without wrapping or filtering when no filters are active and `--limit 0` is used.

## `odh transit stops search <query>`

Searches static GTFS stops from a cached Open Data Hub GTFS archive.

```bash
odh transit stops search auer
odh transit stops search "Ora, Stazione di Ora" --limit 5
odh transit stops search merano --limit 10
```

Flags:

- `--dataset value` - GTFS dataset id; default `sta-time-tables`.
- `--limit n` - maximum stops to return; default `20`.
- `--cache-dir dir` - directory for cached GTFS archives.
- `--refresh` - force a new archive download.

Stop search supports common German/Italian place aliases such as `auer` / `ora`, `brenner` / `brennero`, `bozen` / `bolzano`, and `meran` / `merano`.

## `odh transit departures`

Finds static GTFS departures for matched stops near a time.

```bash
odh transit departures --stop "Ora, Stazione di Ora" --date 2026-05-16 --around 14:05
odh transit departures --stop auer --date 2026-05-16 --around 14:05 --window 30m --mode train
odh transit departures --stop-id it:22021:210:52:31041 --date 2026-05-16 --around 14:05 --mode train
```

Flags:

- `--dataset value` - GTFS dataset id; default `sta-time-tables`.
- `--stop value` - stop search query; required unless `--stop-id` is used.
- `--stop-id value` - exact GTFS `stop_id` or `parent_station` from `odh transit stops search`; required unless `--stop` is used.
- `--date YYYY-MM-DD` - service date; default today.
- `--around HH:MM` - optional departure time to search around.
- `--window duration` - time window around `--around`; default `15m`.
- `--mode all|train|bus|cable-car` - route-type filter; default `all`.
- `--limit n` - maximum departures to return; default `20`.
- `--cache-dir dir` - directory for cached GTFS archives.
- `--refresh` - force a new archive download.

The command reads static GTFS. It is useful for timetable evidence, not for live delay probability. Use `--stop-id` after `odh transit stops search` when a broad stop name matches many platforms, bus bays, or nearby stops. If the selected ID is a GTFS parent station, the CLI expands it to its child platform stops. Output includes `stop_match_mode` so agents can tell whether matching used a fuzzy query, exact stop id, or parent station.

## `odh transit trip`

Looks for direct static GTFS trip matches between two stop queries.

```bash
odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train
odh transit trip --from-stop-id it:22021:210:52:31041 --to-stop-id it:22003:500:52:20001 --date 2026-05-16 --time 14:05 --mode train
```

Flags:

- `--dataset value` - GTFS dataset id; default `sta-time-tables`.
- `--from value` - origin stop query; required unless `--from-stop-id` is used.
- `--from-stop-id value` - exact origin GTFS `stop_id` or `parent_station` from `odh transit stops search`.
- `--to value` - destination stop query; required unless `--to-stop-id` is used.
- `--to-stop-id value` - exact destination GTFS `stop_id` or `parent_station` from `odh transit stops search`.
- `--date YYYY-MM-DD` - service date; default today.
- `--time HH:MM` - origin departure time; required.
- `--window duration` - time window around `--time`; default `15m`.
- `--mode all|train|bus|cable-car` - route-type filter; default `all`.
- `--limit n` - maximum direct trip matches to return; default `20`.
- `--cache-dir dir` - directory for cached GTFS archives.
- `--refresh` - force a new archive download.

This command does not perform transfer routing. If no direct trip matches, the JSON output includes a warning. Output includes `from_match_mode` and `to_match_mode`; use stop-id or parent-station mode for agent workflows where names like Merano or Bolzano return many matches.

## `odh transit delay-stats`

Reports whether historical delay probability can be computed.

```bash
odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday
```

Flags:

- `--from value` - origin stop query.
- `--to value` - destination stop query.
- `--time HH:MM` - origin departure time.
- `--weekday value` - weekday filter, for example `saturday`.
- `--since value` - requested archive range, for example `90d`.

The current implementation returns `supported: false` because Open Data Hub GTFS exposes current static GTFS and live GTFS-RT, not an archived GTFS-RT history. The CLI intentionally does not guess delay probability from one live feed snapshot.

## `odh traffic zones`

List the upstream Open Data Hub `PROVINCE_BZ` traffic zone IDs used by traffic events.

```bash
odh traffic zones
odh traffic zones --json
```

Use these IDs with `--zone-id` when an agent needs a broad upstream region without relying on local aliases.

## `odh traffic categories`

List the stable traffic category names accepted by `--type` and the upstream subtype values they map from.

```bash
odh traffic categories
odh traffic categories --json
```

Use these names with `odh traffic today`, `odh traffic events`, or `odh traffic search`.

## `odh traffic today`

Opinionated wrapper for current South Tyrol traffic events from Open Data Hub Mobility `PROVINCE_BZ`.

```bash
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic today --zone-id 6 --type closure --json
odh traffic today --area unterland --type closure --format markdown
odh traffic today --near 46.42,11.25 --radius 15km
odh traffic today --area bozen-unterland --json
```

The command filters to today, collapses duplicate event rows, hides expired and future items by default, and emits warnings when upstream timestamps look stale. Table output is the default for direct answers; use `--json` or `--format json` when an agent needs structured output. `odh traffic` intentionally stays on Open Data Hub data; if a human-facing traffic bulletin is required, compare the result with the official traffic service outside this CLI and mention the source used.

Flags:

- `--source odh` - traffic source; `odh` is the only supported source.
- `--zone-id value` - Open Data Hub `PROVINCE_BZ` `messageZoneId` filter; run `odh traffic zones` to list known IDs.
- `--area value` - compatibility convenience for existing broad area aliases such as `ueberetsch-unterland`, `bozen-unterland`, `unterland`, or `ueberetsch`; prefer `--zone-id` when an agent wants the upstream ODH traffic zone directly.
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

## `odh traffic search`

Free-text search over the normalized ODH traffic event fields: zone, zone ID, road, road name, German/Italian place text, subtype, severity, and event type.

```bash
odh traffic search "road closed badia" --today --zone-id 6 --json
odh traffic search zwischenwasser --from 2026-05-16 --to 2026-05-16 --include-stale
odh traffic search "SP244 closure" --today --format table
```

The search command is intentionally generic. It does not hardcode village aliases. Agents should use `odh traffic zones` and their own geographic reasoning when they need to broaden a user phrase such as a town, valley, or multilingual place name. The CLI only normalizes common traffic terms such as `closed`, `closure`, `roadblock`, `sperre`, `gesperrt`, `roadworks`, and `baustelle`.

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
