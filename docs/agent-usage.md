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
- Treat exit code `2` as a bad invocation and exit code `1` as a runtime failure.
- Do not prompt interactively.
- Emit JSON by default, except `odh traffic` which defaults to table output and supports `--json`.
- Keep examples public and unauthenticated.

This means agents can call `odh`, parse stdout as JSON, and treat stderr plus exit code as failure context.

Some discovery responses are cached locally for 24 hours to keep repeated agent loops fast. This cache is limited to static-ish metadata such as OpenAPI specs, Tourism taxonomy values, and Mobility type/origin/station/datatype discovery. Do not assume current-data commands are cached: latest measurements, traffic events, diagnostics, and GTFS-RT should be treated as fresh upstream calls unless a command documents an explicit cache.

The HTTP client retries transient `429` and `5xx` failures with bounded exponential backoff. If a command still exits `1`, treat it as a real runtime failure and do not silently invent data.

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
odh mobility origins --station-type TrafficSensor
odh mobility stations --station-type ParkingStation --limit 5
odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 100
odh mobility events --origin A22 --latest --limit 20
odh gtfs datasets
odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5
odh transit stops search auer
odh transit departures --stop "Ora, Stazione di Ora" --date 2026-05-16 --around 14:05
odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train
odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic events --area bozen-unterland --from 2026-05-16 --to 2026-05-16 --json
odh traffic today --near 46.42,11.25 --radius 15km --json
odh mobility latest --station-type EChargingStation --data-type number-available --origin ALPERIA --active --fresh-within 24h --sort newest --request-limit 1000 --limit 5
odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h
odh diagnostics tourism-events --date 2026-05-18
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
odh mobility origins --station-type ParkingStation
odh mobility stations --station-type ParkingStation --limit 5
```

## Mobility Availability Questions

For current EV charging, parking, or similar availability questions, avoid parsing raw latest rows in upstream order. Open Data Hub can return old inactive stations first for some station/data-type combinations.

Use discovery first, then filtered latest measurements:

```bash
odh mobility origins --station-type EChargingStation --limit 1000
odh mobility datatypes --station-type EChargingStation --origin ALPERIA --limit 1000
odh mobility latest --station-type EChargingStation --data-type number-available --origin ALPERIA --active --fresh-within 24h --sort newest --request-limit 1000 --limit 10
```

The filtered `mobility latest` output wraps measurements with `raw_count`, `count`, and `warnings`. Report warnings when filters hide stale or inactive rows, and increase `--request-limit` when a question needs broader coverage than the inspected upstream rows.

## Data Quality Diagnostics

Before answering EV availability, parking forecast, or Tourism event-discovery questions, run the matching diagnostic command:

```bash
odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h --forecast-minutes 60
odh diagnostics tourism-events --date 2026-05-18 --limit 20
```

Diagnostics return a `verdict` and `warnings`. Surface warnings in the final answer. Treat `unavailable` as "the checked request does not contain reliable current data", not as zero chargers, zero parking spaces, or proof that no Tourism events exist.

Use these defaults:

- EV availability: `--active --fresh-within 24h` through `odh diagnostics ev-charging`.
- Parking forecasts: current occupancy can be usable even when forecasts are stale; if the verdict is `current_only`, answer only with current occupancy.
- Tourism events: inspect `date_status`, `ActiveToday`, and `location_status`; missing GPS makes radius or place claims weak.

## Public Transport And Delay Questions

For public-transport timetable or live-feed questions, prefer the GTFS and transit commands before raw API calls:

```bash
odh datasets search train
odh gtfs datasets
odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5
odh transit stops search auer
odh transit departures --stop "Ora, Stazione di Ora" --date 2026-05-16 --around 14:05 --mode train
odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train
odh transit stops search merano --limit 10
odh transit departures --stop-id <stop_id-from-search> --date 2026-05-16 --around 13:00 --mode train
odh transit trip --from-stop-id <origin-stop-id> --to-stop-id <destination-stop-id> --date 2026-05-16 --time 13:00 --mode train
```

The transit layer reads the static STA GTFS archive and caches it locally for 24 hours. A cold cache download can be large, so transit commands allow a longer archive-download timeout than normal API calls. Stop search supports common German/Italian aliases such as `auer` / `ora`, `brenner` / `brennero`, and `bozen` / `bolzano`.

If `transit departures` or `transit trip` returns a warning that a stop query matched many stops, narrow the stop wording with `odh transit stops search <query> --limit 5` and rerun with the returned `stop_id`: `--stop-id` for departures, or `--from-stop-id` / `--to-stop-id` for trip matching. Parent station IDs are valid and expand to their child platform stops. This is the preferred agent pattern for ambiguous station names such as Merano or Bolzano.

`odh transit trip` only finds direct static GTFS trip matches. It does not perform full connection planning or transfer routing. If the user asks for a multi-leg journey, report that limitation and use the returned stop matches plus GTFS-RT feed as supporting data, not as a full journey planner.

For historical delay probability or "usual delay minutes", use:

```bash
odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday
```

This currently returns `supported: false`. That is intentional: the public live GTFS-RT feed can describe current trip updates, but probability needs archived snapshots over time. Do not infer historical probabilities from a single live feed response.

## Handling Failures

Agents should treat exit code `2` as a usage bug in the invocation and exit code `1` as a runtime problem such as HTTP failure, invalid JSON, or unavailable upstream service.

`odh doctor` is the preferred first command when an agent needs to distinguish a bad invocation from a local install problem or an upstream Open Data Hub reachability problem.

## Traffic Data Caveat

Open Data Hub Mobility feeds can expose different traffic concepts as station measurements, events, and forecasts. For South Tyrol roadworks, closures, road events, and traffic restrictions, prefer the opinionated traffic layer:

```bash
odh traffic zones
odh traffic categories
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic today --zone-id 6 --type closure --json
odh traffic events --area unterland --from 2026-05-16 --to 2026-05-16 --type closure --json
odh traffic search "road closed badia" --today --zone-id 6 --json
odh traffic today --near 46.42,11.25 --radius 15km
```

Traffic commands query Open Data Hub `PROVINCE_BZ` events and default to table output. Use `--json` or `--format json` for downstream parsing and `--format table` or `--format markdown` for direct human answers. Use `odh traffic zones` to discover upstream zone IDs, then pass `--zone-id` when broad regional filtering is more reliable than a local place-name alias. The traffic layer deduplicates rows, maps German/Italian event categories to stable English category names, filters expired/future rows by date, hides stale open-ended records by default, and warns when timestamps look stale.

Use `odh traffic search <text>` for towns, roads, place names, and natural-language traffic wording. The CLI intentionally does not hardcode local village aliases; agents should broaden multilingual or local place names themselves, then call `traffic search` and/or `--zone-id`. The search layer only applies generic traffic-term normalization such as `closed`/`closure`/`roadblock` to `sperre`/`gesperrt`, and `roadworks` to `baustelle`.

If a user needs an exact public traffic bulletin, compare the Open Data Hub result with the official traffic service outside this CLI and state both the source and timestamp used. Do not imply that `odh traffic` silently switched to another upstream feed.

Agents should not infer live A22 traffic solely from `TrafficForecast` rows. Prefer `odh a22 status` when checking A22 because it reports current-event availability and warns when forecast timestamps indicate non-current data.

If the user asks for current traffic conditions, report the timestamp and feed type used. If Open Data Hub has no current A22 event rows, say that directly instead of converting forecast rows into live incidents.

## Why No MCP Yet

The project is structured so an MCP server can reuse the registry and HTTP client later. v0.1 ships the CLI first because it is simpler to review, easier to test, and immediately useful in scripts.

## Agent Evals

Use [evaluation.md](evaluation.md) and [../evals/agent/tasks.json](../evals/agent/tasks.json) to test whether agents can answer realistic Open Data Hub questions with the existing CLI surface.

The evals are intentionally outside the CLI. If an agent fails a prompt, first improve the skill guidance or the agent's command path. Add CLI surface only when repeated failures show the same missing bounded upstream vocabulary or mechanical discovery step.
