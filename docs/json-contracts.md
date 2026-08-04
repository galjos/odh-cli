<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# JSON Contracts

This document lists the JSON fields agents and scripts can rely on for the
curated command surfaces. It is not a full schema for every raw upstream object.
Raw upstream objects inside fields such as `measurements`, `items`, or `raw`
can change with Open Data Hub.

Stable contract fields are additive: patch releases may add fields, but should
not rename or remove the fields listed here without a changelog note.

## Common Fields

Curated JSON outputs should expose provenance when the answer depends on source
semantics:

- `source` - upstream API or feed family.
- `source_detail` - narrower endpoint or data-family description.
- `endpoint` - concrete upstream URL when useful for audit/debugging.
- `warnings` - caveats that agents must surface in answers.

## `odh traffic today|events|search --json`

Stable top-level fields:

- `source`
- `source_detail`
- `endpoint`
- `from`
- `to`
- `zone_id`
- `area`
- `type`
- `search`
- `raw_count`
- `count`
- `events`
- `warnings`

Stable event fields:

- `id`
- `series_id`
- `message_id`
- `source`
- `type`
- `subtype`
- `severity`
- `zone_id`
- `zone`
- `zone_it`
- `road`
- `road_name`
- `place`
- `place_it`
- `start`
- `end`
- `published_at`
- `transaction_time`
- `coordinates`
- `active`
- `stale`

`raw` is present only with `--raw` and mirrors upstream data.

### `--source odh` and `--source content`

Both sources emit the same envelope and the same event fields. They differ in
which of those fields can be populated and which filters are accepted.

| | `--source odh` (default) | `--source content` |
| --- | --- | --- |
| upstream | Mobility Timeseries `/v2/flat,event/PROVINCE_BZ` | Content API `/v1/Announcement?source=PROVINCE_BZ` |
| `source` | `odh` | `content` |
| filters | all | `--from`/`--to`/`--today`, `--near`/`--radius`, `--search`, `--type`, `--limit`, `--include-expired`, `--zone-id`/`--area` (inferred, see below) |
| rejected filters | none | `--road`, `--type bike` |
| never populated (event fields) | none | `zone_id`, `zone`, `zone_it`, `road`, `road_name`, `severity`, `series_id` |

With `--source content`:

- `subtype` holds the upstream `traffic-event:*` tags, comma-joined and sorted,
  for example `hindrance,road-work`. `type` is derived from them.
- `end` is empty while the announcement is open. The provider sets an end time
  only when the event ends, so an empty `end` means ongoing, not unknown.
- `active` is true only when the announcement overlaps the requested date range
  **and** has not ended yet. Upstream `Active` is not read: it is `true` on
  every PROVINCE_BZ record, including ones closed a year ago.
- `published_at` and `transaction_time` both carry the record's `LastChange`.
  This feed has no provider publication timestamp, so `published_at` here means
  "last changed upstream", not "published by the province". `stale` derives from it.
- `--include-expired` is scoped to now, not to the requested range: an
  announcement that has already ended is hidden even when it was live during the
  range you asked for. Under `--source odh` the same flag means "ended before the
  range started". Pass `--include-expired` for historical range queries.
- Timestamps are passed through in the upstream encoding, and the two sources
  differ: `--source odh` emits `2006-01-02 15:04:05.000-0700`, `--source content`
  emits RFC 3339. Parse permissively.
- `stale` reports that `LastChange` is more than 30 days old. It never hides a
  row here, because an open announcement stays valid until the provider closes
  it. `--include-stale` is accepted and warns that it has no effect.
- A rejected filter is a usage error (exit code 2) naming the flag and the
  reason. The filter is never silently dropped.

#### `--zone-id` and `--area` under `--source content`

Announcement records carry coordinates but no zone. These two filters therefore
match by **geographic inference**, not by reading a field:

- The CLI ships a reference table of ~1100 rounded coordinates, each tagged with
  the zone the Mobility Timeseries event feed recorded there. It is derived from
  the historical feed by `scripts/generate-traffic-zone-points.go` and committed,
  not fetched at runtime. Grid cells that more than one zone claimed are dropped
  rather than assigned.
- An announcement matches when the nearest reference point's zone is among the
  requested zones and lies within 2.0 km. Leave-one-out over the table, the
  nearest other point carries the same zone 98.2% of the time within that bound
  and 82.9% between 2 and 3 km.
- Beyond the bound, or with no coordinates, the announcement is **unassignable**:
  it is excluded and counted in a warning, so a thin result is distinguishable
  from an empty road network.
- The inferred zone is used only to filter. It is never written to `zone_id`,
  `zone` or `zone_it`, which stay empty exactly as they do without the filter.
- A warning naming the inference is always emitted when either filter is used.
  Surface it: the answer is "near coordinates historically tagged with this
  zone", not "the province filed this under this zone".
- Top-level `zone_id` and `area` echo the filter you passed. They are not read
  from any record and say nothing about the events below them.
- `--zone-id` and `--area` narrow independently, as they do under `--source odh`:
  an announcement must satisfy both, so disjoint values return nothing.

Three limits are worth knowing:

- Area aliases that also filter on place names under `--source odh` (`kaltern`,
  `eppan`, `unterland`, and the other municipality aliases) narrow **only by
  zone** here, so the result covers the whole zone. A warning says so.
- A long linear notice is published as a single point, and the two feeds may
  anchor the same notice at different ends of it. Where the stretch crosses a
  zone boundary the inferred zone can differ from the structured one even at a
  few metres' distance. Measured against announcements that also appear in the
  structured feed with an unambiguous zone, the inference agreed on 21 of 22 when measured against the 2026-08-04 bulletin.
- Zone 7 (`Ausserhalb Südtirol`) is defined by not being a place: its points run
  the Brenner axis from Modena to Austria and interleave with zones 1-6 at the
  border, so inference is weakest there.

## `odh transit journey --json`

Stable top-level fields:

- `source`
- `source_detail`
- `timetable_type`
- `dataset`
- `from_query`
- `from_stop_id`
- `from_match_mode`
- `to_query`
- `to_stop_id`
- `to_match_mode`
- `date`
- `time`
- `mode`
- `max_transfers`
- `min_transfer`
- `max_duration`
- `archive`
- `from_stops`
- `to_stops`
- `count`
- `journeys`
- `with_realtime`
- `realtime`
- `warnings`

Stable journey fields:

- `departure_time`
- `arrival_time`
- `duration`
- `transfer_count`
- `legs`
- `realtime_transfers`

Stable leg fields:

- `trip_id`
- `route_id`
- `route_short_name`
- `route_long_name`
- `route_type`
- `headsign`
- `direction_id`
- `from`
- `to`
- `realtime`

When `--with-realtime` is present, `realtime` is an annotation layer over static
GTFS routing. It is not live rerouting.

## `odh mobility origins --json`

This command always emits JSON; `--json` is accepted for consistency.

Stable top-level fields:

- `station_type`
- `endpoint`
- `record_count`
- `count`
- `origins`
- `warnings`

Stable origin fields:

- `name`
- `station_count`
- `station_samples`

`record_count` is the number of upstream station records inspected and `count`
is the number of distinct origins summarized from them. `station_samples` is
capped at five station codes and is not the full station list.

`warnings` is present only when the upstream result filled `--limit`, including
the default limit; while it is present the origin list may be incomplete.

## `odh mobility stations --json`

This command always emits JSON; `--json` is accepted for consistency.

Stable top-level fields:

- `station_type`
- `origin`
- `record_count`
- `count`
- `stations`
- `warnings`

`stations` contains Open Data Hub Mobility station rows and mirrors upstream
data. `record_count` is the number of rows upstream returned and `count` is the
number left after the local `--origin` filter; they are equal when `--origin` is
empty.

`warnings` is present only when the upstream result filled `--limit`, including
the default limit. Note that `--limit` caps the upstream request, so the
`--origin` filter only sees the rows inside that page.

## `odh mobility events`

This command has no format flag and always emits JSON.

Stable top-level fields:

- `origin`
- `latest`
- `count`
- `events`
- `warnings`

`events` contains Open Data Hub Mobility event rows and mirrors upstream data.

`warnings` is never empty: it always carries the Mobility Timeseries event feed
caveat, which reports the newest row date received and names the Content API
command to cross-check current notices with. A second entry is added when the
result filled `--limit`, including the default limit.

For South Tyrol roadworks and closures, prefer `odh traffic` over this command:
it deduplicates rows and returns the richer contract documented above.

## `odh mobility latest --json`

When no local filtering or human output is requested, this command can pass
through the raw upstream JSON. When local filtering is active, or when table /
markdown output would require local processing, JSON uses the wrapper contract.

Stable wrapper fields:

- `source`
- `source_detail`
- `station_type`
- `data_type`
- `origin`
- `active_only`
- `fresh_within`
- `sort`
- `endpoint`
- `raw_count`
- `count`
- `measurements`
- `warnings`

`measurements` contains Open Data Hub Mobility measurement rows.

`count` describes the returned rows, which `--limit` may have capped below the
number that matched the filters. When it does, `warnings` says how many matched.
Read that warning before treating `count` as a total.

## `odh diagnostics *`

Stable common fields:

- `domain`
- `source`
- `source_detail`
- `verdict`
- `warnings`

`diagnostics ev-charging` also exposes:

- `endpoint`
- `station_type`
- `data_type`
- `origin`
- `active_only`
- `fresh_within`
- `raw_count`
- `current_count`
- `measurements`
- `recommended_command`

`diagnostics parking-forecasts` also exposes:

- `origin`
- `fresh_within`
- `forecast_minutes`
- `current`
- `forecast`

`diagnostics tourism-events` also exposes:

- `date`
- `only_active`
- `endpoint`
- `count`
- `active_count`
- `events`

`current_count` and the `current`/`forecast` counts describe the rows returned
after `--limit`, which can be fewer than matched. `warnings` reports the match
total whenever the cap binds; unlike `raw_count`, the counts alone cannot reveal
it.

Treat `verdict: unavailable` as "the checked request does not contain reliable
current data", not as proof that the real-world domain has zero records.

## `odh a22 status --json`

Stable fields:

- `source`
- `source_detail`
- `events`
- `forecast`
- `warnings`

Stable `events` and `forecast` feed fields:

- `endpoint`
- `count`
- `summary`
- `items`

`items` is present only with `--raw` and mirrors upstream rows.

## `odh tourism types --json`

Stable fields:

- `source`
- `source_detail`
- `dataset`
- `endpoint`
- `count`
- `items`

`items` contains upstream taxonomy rows and can vary by Tourism API dataset.
