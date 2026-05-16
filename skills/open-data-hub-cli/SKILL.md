---
name: open-data-hub-cli
description: Use this skill when working with Open Data Hub, NOI Techpark data, ODH APIs, Tourism API, Mobility API, A22 traffic data, or when an agent should query Open Data Hub through the odh command-line tool instead of scraping websites.
homepage: https://github.com/galjos/odh-cli
metadata:
  {
    "openclaw":
      {
        "os": ["darwin", "linux"],
        "requires": { "bins": ["odh"] },
        "install":
          [
            {
              "id": "go",
              "kind": "go",
              "module": "github.com/galjos/odh-cli/cmd/odh@v0.1.5",
              "bins": ["odh"],
              "label": "Install odh CLI (go)",
            },
          ],
      },
  }
---

# Open Data Hub CLI

Use `odh` for public Open Data Hub API work. It is JSON-first, non-interactive, and suitable for scripts and agents.

Open Data Hub is maintained by NOI Techpark. Most practical Tourism and Mobility tasks are about South Tyrol / the Autonomous Province of Bolzano, but do not claim every returned record is located there unless coordinates, location fields, origin metadata, or official docs support it.

## First Checks

Run these before relying on the CLI:

```bash
odh version
odh doctor --timeout 10s
```

If `odh` is not installed in a normal shell, install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh
```

Or build it from source:

```bash
go build -o odh ./cmd/odh
```

In OpenClaw, this skill declares `odh` as a required binary. If OpenClaw marks the skill as needing setup, use the offered installer before trying to answer with Open Data Hub data.

When running from a source checkout, use `./odh` instead of `odh`.

## API Discovery

List known API surfaces:

```bash
odh apis
odh datasets search parking
```

Fetch OpenAPI specs as JSON:

```bash
odh openapi mobility
odh openapi tourism
```

Use `odh call` for known endpoints rather than scraping a UI:

```bash
odh call tourism /v1/ODHActivityPoi \
  --param pagenumber=1 \
  --param pagesize=1 \
  --param seed=42
```

For geographic claims, inspect returned fields. Useful examples:

- Tourism: `GpsInfo`, `LocationInfo`, `RegionInfo`, `MunicipalityInfo`, `LicenseInfo`.
- Mobility: `sorigin`, `scode`, `stype`, `scoordinate`, `smetadata`.

## Curated Commands

Tourism point of interest:

```bash
odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
odh tourism types --dataset event --limit 10
```

Mobility latest measurements:

```bash
odh mobility latest \
  --station-type EChargingStation \
  --data-type number-available \
  --limit 5
```

Mobility type and data-type discovery:

```bash
odh mobility types --kind event
odh mobility stations --station-type ParkingStation --limit 5
odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 100
```

Public transport GTFS and STA timetable data:

```bash
odh gtfs datasets
odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5
odh transit stops search auer
odh transit departures --stop "Ora, Stazione di Ora" --date 2026-05-16 --around 14:05 --mode train
odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train
odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday
```

Use these commands when the user asks about trains, buses, public-transport stops, STA timetables, GTFS, GTFS-RT, live trip updates, or whether delay probability can be computed. `odh transit trip` only finds direct static GTFS trip matches; it does not do transfer routing. `odh transit delay-stats` currently returns `supported: false` because delay probability requires historical GTFS-RT snapshots, not just the current live feed. Do not infer historical delay probability from one realtime response.

South Tyrol traffic events, roadworks, closures, and road events:

```bash
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic events --area unterland --from 2026-05-16 --to 2026-05-16 --type closure --json
odh traffic today --near 46.42,11.25 --radius 15km --format json
odh traffic today --area bozen-unterland --json
```

Prefer these commands over raw `odh mobility events --origin PROVINCE_BZ` when a user asks for roadworks, roadblocks, closures, or traffic events in areas such as Unterland, Ueberetsch, Bozen-Unterland, Salurn, Kaltern, Tramin, Eppan, Auer, Neumarkt, Kurtatsch, Margreid, or Montan. Traffic commands query Open Data Hub `PROVINCE_BZ`, default to table output, and support `--json` for structured parsing. The traffic layer filters by date, maps upstream categories to stable names, deduplicates repeated event rows, hides stale open-ended records by default, and warns about stale or date-mismatched records. If the user needs an exact official public traffic bulletin, compare with the official traffic service outside this CLI and state the source used.

A22 traffic diagnostics:

```bash
odh mobility events --origin A22 --latest --limit 20
odh a22 status --limit 10
```

## Interpretation Rules

- Parse stdout as JSON for data commands; add `--json` before parsing `odh traffic` output.
- Treat nonzero exit codes as failures.
- Treat stderr as diagnostics, not data.
- Prefer `odh` and official OpenAPI specs over scraping Open Data Hub web pages.
- Treat South Tyrol as the common regional context, not as a universal record-level guarantee.
- Verify location-sensitive answers from coordinates, origins, and metadata in the JSON.
- For roadworks and closures, prefer `odh traffic today` or `odh traffic events` before falling back to raw Mobility events.
- For public transport, prefer `odh gtfs` and `odh transit` before falling back to raw API calls.
- For historical delay probability, report the `odh transit delay-stats` caveat instead of guessing.
- Do not infer live A22 traffic from `TrafficForecast` rows alone.
- Prefer `odh a22 status` for A22 because it reports current-event availability and warns when forecast rows are not current incident data.
- Use `--where` and `--param key=value` instead of manually constructing query strings when a curated command supports them.

## Official References

- https://opendatahub.com/api/
- https://opendatahub.com/services/data-access/
- https://opendatahub.com/about-us/
- https://docs.opendatahub.com/en/latest/datasets.html
- https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
