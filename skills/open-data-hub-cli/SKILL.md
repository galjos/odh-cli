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
              "module": "github.com/galjos/odh-cli/cmd/odh@v0.1.10",
              "bins": ["odh"],
              "label": "Install odh CLI (go)",
            },
          ],
      },
  }
---

# Open Data Hub CLI

Use `odh` for public Open Data Hub API work. It is non-interactive, suitable for scripts and agents, and supports JSON output through `--json` or `--format json`. Several curated discovery/answer commands default to compact table output to avoid huge raw JSON dumps.

Open Data Hub is maintained by NOI Techpark. Most practical Tourism and Mobility tasks are about South Tyrol / the Autonomous Province of Bolzano, but do not claim every returned record is located there unless coordinates, location fields, origin metadata, or official docs support it.

The CLI retries transient `429` and `5xx` HTTP failures. Some low-risk discovery metadata such as OpenAPI specs, Tourism taxonomies, and Mobility station/type discovery can be cached locally for 24 hours. Current-data commands such as traffic, latest measurements, diagnostics, and GTFS-RT should still be treated as fresh upstream calls.

## First Checks

Run these before relying on the CLI:

```bash
odh version
odh doctor --timeout 10s
```

The traffic discovery, GTFS, transit, and filtered mobility-latest commands require `odh` `v0.1.10` or newer. If `odh version` reports an older release, install the current release into a directory that is already on PATH:

```bash
curl -fsSL https://raw.githubusercontent.com/galjos/odh-cli/main/scripts/install.sh | sh -s -- --version v0.1.10 --dir "$HOME/bin"
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

If OpenClaw's Go installer reports success but `odh version` is still old, a previous binary is probably earlier on PATH than `~/go/bin/odh`. Use the release installer above with `--dir "$HOME/bin"` or update PATH so the newer binary is resolved first.

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
odh tourism types --dataset event --limit 10 --json
```

Mobility latest measurements:

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

For EV charging, parking, or other availability questions, discover the station type and origin first, then use `mobility latest` with `--active`, `--fresh-within`, and `--sort newest`. Raw upstream latest rows can include old inactive stations before current measurements. Add `--format table` for a compact direct answer, or `--json` for structured parsing. If filtered output includes `warnings`, surface them in the answer. If the CLI prints a datatype hint, follow it instead of retrying the same empty query.

Data-quality diagnostics:

```bash
odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h --forecast-minutes 60
odh diagnostics tourism-events --date 2026-05-18 --limit 20
```

Use diagnostics before answering EV availability, parking forecast, or Tourism event-discovery questions. Parse `verdict` and `warnings`. Treat `unavailable` as "no reliable current data from the checked request", not as zero availability or proof that the entire upstream domain has no data. For `parking-forecasts`, a `current_only` verdict means current parking occupancy can be used but fresh forecast rows should not be reported. For `tourism-events`, inspect `date_status` and `location_status`; missing GPS makes radius or precise place claims weak.

Mobility type and data-type discovery:

```bash
odh mobility types --kind event
odh mobility origins --station-type TrafficSensor
odh mobility stations --station-type ParkingStation --limit 5
odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 100
odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 100 --json
```

Public transport GTFS and STA timetable data:

```bash
odh gtfs datasets
odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5
odh transit stops search auer
odh transit departures --stop "Ora, Stazione di Ora" --date 2026-05-16 --around 14:05 --mode train
odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train
odh transit stops search merano --limit 10
odh transit departures --stop-id <stop_id-from-search> --date 2026-05-16 --around 13:00 --mode train
odh transit trip --from-stop-id <origin-stop-id> --to-stop-id <destination-stop-id> --date 2026-05-16 --time 13:00 --mode train
odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday
```

Use these commands when the user asks about trains, buses, public-transport stops, STA timetables, GTFS, GTFS-RT, live trip updates, or whether delay probability can be computed. The first transit command may download a large static GTFS archive and cache it for 24 hours; retry once if the upstream download is interrupted. Transit commands default to compact table output; add `--json` when you need stop-match arrays, archive metadata, or match-mode fields. If the output warns that a stop query matched many stops, run `odh transit stops search <query> --limit 5` and rerun with `--stop-id`, `--from-stop-id`, or `--to-stop-id`. Parent station IDs are valid and expand to their child platform stops. This is the preferred agent pattern for ambiguous station names like Merano and Bolzano. `odh transit trip` only finds direct static GTFS trip matches; it does not do transfer routing. `odh transit delay-stats` currently returns `supported: false` because delay probability requires historical GTFS-RT snapshots, not just the current live feed. Do not infer historical delay probability from one realtime response.

South Tyrol traffic events, roadworks, closures, and road events:

```bash
odh traffic zones
odh traffic categories
odh traffic today --area ueberetsch-unterland --type roadworks --format table
odh traffic today --zone-id 6 --type closure --json
odh traffic events --area unterland --from 2026-05-16 --to 2026-05-16 --type closure --json
odh traffic search "road closed badia" --today --zone-id 6 --json
odh traffic today --near 46.42,11.25 --radius 15km --format json
odh traffic today --area bozen-unterland --json
```

Prefer these commands over raw `odh mobility events --origin PROVINCE_BZ` when a user asks for roadworks, roadblocks, closures, or traffic events. Use `odh traffic zones` to discover upstream `PROVINCE_BZ` zone IDs and `--zone-id` for broad regional filters. Use `odh traffic categories` to discover valid `--type` values. Use `odh traffic search <text>` for towns, roads, place names, and natural-language wording. The CLI intentionally does not hardcode local village aliases; if a user uses a multilingual or local place name, broaden it in the agent layer and then query with `traffic search` and/or `--zone-id`. Traffic commands query Open Data Hub `PROVINCE_BZ`, default to table output, and support `--json` for structured parsing. The traffic layer filters by date, maps upstream categories to stable names, deduplicates repeated event rows, hides stale open-ended records by default, and warns about stale or date-mismatched records. If the user needs an exact official public traffic bulletin, compare with the official traffic service outside this CLI and state the source used.

A22 traffic diagnostics:

```bash
odh mobility events --origin A22 --latest --limit 20
odh a22 status --limit 10
```

## Interpretation Rules

- Parse stdout as JSON only after adding `--json` or `--format json` to commands that default to table output, including `traffic`, `a22 status`, `transit`, `tourism types`, and `mobility types` / `mobility datatypes`.
- Treat nonzero exit codes as failures.
- Treat exit code `2` as a bad invocation and exit code `1` as a runtime/upstream problem.
- Treat stderr as diagnostics, not data.
- Prefer `odh` and official OpenAPI specs over scraping Open Data Hub web pages.
- Treat South Tyrol as the common regional context, not as a universal record-level guarantee.
- Verify location-sensitive answers from coordinates, origins, and metadata in the JSON.
- For roadworks and closures, prefer `odh traffic today` or `odh traffic events` before falling back to raw Mobility events.
- For public transport, prefer `odh gtfs` and `odh transit` before falling back to raw API calls.
- For historical delay probability, report the `odh transit delay-stats` caveat instead of guessing.
- Use `odh diagnostics` for EV availability, parking forecasts, and Tourism event caveats before making factual current-data claims.
- Do not infer live A22 traffic from `TrafficForecast` rows alone.
- Prefer `odh a22 status` for A22 because it reports current-event availability and warns when forecast rows are not current incident data. Use its default table for direct answers, or `--json --raw` when raw upstream rows are required.
- Use `--where` and `--param key=value` instead of manually constructing query strings when a curated command supports them.

## Agent Evals

The upstream project includes agent eval tasks in `evals/agent/tasks.json` and a live smoke runner:

```bash
scripts/run-agent-evals.sh
```

Use those evals to decide whether a repeated agent failure belongs in docs, skill guidance, agent reasoning, or a narrow CLI feature. Do not ask for natural-language helper commands unless the evals show repeated friction that cannot be solved by existing discovery commands.

## Official References

- https://opendatahub.com/api/
- https://opendatahub.com/services/data-access/
- https://opendatahub.com/about-us/
- https://docs.opendatahub.com/en/latest/datasets.html
- https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
