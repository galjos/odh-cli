<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Data Quality

`odh` should make weak upstream data visible instead of hiding it behind a convenient answer. The diagnostics commands are meant for agents and scripts that need to decide whether a dataset is currently usable before answering a user.

## Diagnostic Commands

```bash
odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h --request-limit 10000 --limit 10
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h --forecast-minutes 60
odh diagnostics tourism-events --date 2026-05-18 --limit 20
```

Each command returns JSON with:

- `domain` - the checked data area.
- `source` - the upstream Open Data Hub API used.
- `source_detail` - the narrower endpoint or data-family checked.
- `verdict` - `usable`, `usable_with_caveats`, `usable_with_forecast`, `current_only`, or `unavailable`.
- `warnings` - caveats that must be surfaced when answering.

Treat `unavailable` as "no reliable current data from this checked request", not as zero availability, zero events, or proof that the whole domain has no data.

## Stronger Areas

The current CLI is most reliable for:

- current parking occupancy when queried with `--active` and `--fresh-within`,
- South Tyrol roadworks and traffic summaries through `odh traffic`,
- static STA timetable discovery and direct trip matching through `odh transit`,
- weather and other Tourism records when returned fields include current dates and clear location metadata.

For these areas, agents should still report timestamps, locations, and warnings from the JSON output.

## Known Caveats

EV charging availability can return old or inactive records first in raw Mobility output. Use:

```bash
odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h
```

If the verdict is `unavailable`, do not present stale 2015-2017 rows as current charger availability.

Parking forecasts may exist as data types such as `parking-forecast-60`, but the forecast rows can be stale even when current occupancy is fresh. Use:

```bash
odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h
```

If the verdict is `current_only`, answer with current occupancy only and state that fresh forecast rows were not available.

Tourism event filtering has upstream semantics that are not always intuitive. `onlyactive=true` can still return rows where `ActiveToday` is false, and many records can lack `GpsInfo`. Use:

```bash
odh diagnostics tourism-events --date 2026-05-18
```

When warnings mention missing GPS, avoid precise radius or place claims for those events unless another returned field supports the location.

## Agent Rule

For EV availability, parking forecasts, or Tourism event discovery, run the matching `odh diagnostics` command before making a factual claim. A clean answer should distinguish:

- fresh current data,
- stale data hidden by filters,
- upstream caveats,
- unsupported or unavailable claims.
