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
| filters | all | `--from`/`--to`/`--today`, `--near`/`--radius`, `--search`, `--type`, `--limit`, `--include-expired` |
| rejected filters | none | `--zone-id`, `--area`, `--road`, `--type bike` |
| never populated | none | `zone_id`, `zone`, `zone_it`, `road`, `road_name`, `severity`, `series_id` |

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
