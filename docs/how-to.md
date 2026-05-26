<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# How To Use odh

This page is a task manual for humans and agents. For exact flags, run
`odh <command> --help`.

Use `--json` or `--format json` whenever another program needs to parse stdout.
Use the global timeout before the subcommand when the caller has a latency
budget:

```bash
odh --timeout 20s traffic today --area ueberetsch-unterland --json
```

## Find Roadworks Or Closures Today

Start with discovery if you do not know the area or type filter:

```bash
odh traffic zones
odh traffic categories
```

Then query the curated traffic layer:

```bash
odh traffic today --area ueberetsch-unterland --type roadworks
odh traffic today --near 46.42,11.25 --radius 15km --json
odh traffic search badia --today --json
```

For an exact date:

```bash
odh traffic events --from 2026-05-16 --to 2026-05-16 --area bozen-unterland --json
```

Read warnings. The traffic layer uses Open Data Hub `PROVINCE_BZ`, hides stale
open-ended rows by default, and is not a full replacement for the official live
traffic bulletin.

## Check A22 Data

Use:

```bash
odh a22 status --limit 10 --json
```

The command keeps A22 event rows separate from `TrafficForecast` rows. Do not
present forecast rows as live incidents. For historical local incidents, Open
Data Hub live/current feeds may not be enough.

## Check Parking Availability

Discover the origin if needed:

```bash
odh mobility origins --station-type ParkingStation
odh mobility datatypes --station-type ParkingStation --origin "Municipality Merano"
```

Then query current fresh rows:

```bash
odh mobility latest --station-type ParkingStation --data-type free --origin "Municipality Merano" --active --fresh-within 2h --sort newest --request-limit 10000 --format table
```

For forecast reliability, run:

```bash
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h
```

If the diagnostic verdict is `current_only`, report current free spaces but do
not report stale forecast values as live predictions.

## Check EV Charging Availability

Raw EV latest rows can be stale. Start with diagnostics:

```bash
odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h
```

If the diagnostic finds fresh active rows, use the recommended command from its
JSON output or run:

```bash
odh mobility latest --station-type EChargingStation --data-type number-available --origin ALPERIA --active --fresh-within 24h --sort newest --request-limit 10000 --json
```

If no fresh active rows are found, say that current availability is unavailable
from the inspected Open Data Hub feed instead of showing old values.

## Plan A Public Transport Journey

First resolve ambiguous stops:

```bash
odh transit stops search merano --limit 10
odh transit stops search "Ora, Stazione" --limit 10 --json
```

Then use stop IDs for the actual route:

```bash
odh transit journey --from-stop-id Parentit:22021:301 --to-stop-id it:22021:730:0:1150 --date 2026-05-21 --time 16:40 --max-transfers 3 --with-realtime --json
```

Use `--with-realtime` for GTFS-RT annotations. The route search still uses the
static STA GTFS timetable and does not live-reroute around delays.

For a direct connection only:

```bash
odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train --json
```

For departures at one stop:

```bash
odh transit departures --stop-id Parentit:22021:301 --date 2026-05-21 --around 16:40 --mode train --json
```

## Answer Historical Delay Probability Questions

Use the explicit limitation command:

```bash
odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday --json
```

The CLI currently has no archived GTFS-RT history, so it must not invent delay
probabilities or usual delay minutes from live feeds.

## Inspect Tourism Event Caveats

For Tourism events, check date and location semantics before making precise
"active today" or "near me" claims:

```bash
odh diagnostics tourism-events --date 2026-05-18 --limit 20
odh tourism types --dataset event --limit 20 --json
```

Some event rows can have missing GPS fields or surprising `onlyactive`
semantics. Surface diagnostic warnings when answering.

## Fall Back To Raw APIs

Use discovery before raw calls:

```bash
odh apis
odh datasets search traffic
odh openapi mobility
odh openapi tourism
```

Then call known endpoints directly:

```bash
odh call mobility /v2/flat,node/ParkingStation/free/latest --param limit=5
odh call tourism /v1/ODHActivityPoi --param pagesize=1 --param fields=Detail.en.Title,GpsInfo
```

Prefer curated commands when they exist because they add deduplication,
freshness filters, provenance fields, and caveat warnings.
