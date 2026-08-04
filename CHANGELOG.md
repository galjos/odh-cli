<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Changelog

All notable changes to `odh-cli` are documented here.

## v0.5.1 - 2026-08-04

Found by driving v0.5.0 as an agent would, with only the released binary and
the published skill.

- `traffic search` no longer matches identifier substrings. `id`, `series_id`
  and `message_id` were in the searched text, so a bare `12` matched every
  record whose message id contained 12: `--search "SS 12"` returned 32 of 64
  roadworks, mostly unrelated. Identifiers now match only as a whole term, so
  looking one up by its exact id still works.
- Road numbers match either spelling. Upstream writes both `SS 12` and `SS12`;
  `--search SS12` used to return nothing. This matters because the `--road`
  rejection under `--source content` tells the user to search instead — it was
  recommending a query that could not match.
- The `--type bike` aliases now reach cycle notices. `radweg` is a documented
  alias for that category but returned nothing, while `radroute` returned
  eight, so the natural German word gave a confident empty answer. `radroute`,
  `radweg`, `fahrrad`, `ciclabile`, `bici` and `cycle` now find the same set.

## v0.5.0 - 2026-08-04

- `odh traffic today|events|search` accept `--source content`, which answers
  from the Content API `/v1/Announcement` road bulletin instead of the Mobility
  Timeseries event feed. v0.4.1 made the CLI disclose that the Timeseries feed
  is not a live bulletin; this gives users a way to get the current data. The
  provincial bulletin is only published there now. Refs #10.
  - Supported filters: `--from`/`--to`/`--today`, `--near`/`--radius`,
    `--search`, `--type`, `--limit`, `--include-expired`.
  - `--zone-id`, `--area`, `--road` and `--type bike` are
    rejected with exit code 2 naming the flag and the reason. Announcements
    carry no zone id or severity, name the road only inside free text, and have
    no cycle-route or speed-control tag, so applying those filters would return
    fewer matches than asked for and read as "no such events".
  - Results leave `zone_id`, `road` and `severity` empty and warn that the
    field is unavailable rather than absent.
  - Validity comes from `EndTime`: empty means the announcement is still open,
    and `active` additionally requires that it has not ended yet. Upstream
    `Active` is not read — it is `true` on every PROVINCE_BZ record, including
    ones closed a year ago. `--include-expired` brings ended ones back, and
    `stale` never hides a row in this source.

## v0.4.3 - 2026-08-03

Driven by the 2026-08-03 agent eval round (11 pass, 2 partial, 0 fail),
recorded in `evals/agent/results/2026-08-03.md`.

- The GTFS archive download now honours `--timeout`. It was pinned to a flat
  two minutes, so `--timeout 10m` was silently cut back and a cold-cache
  transit query on a slow link had no way to succeed; this blocked four of
  thirteen eval attempts. The progress line also announced "up to 2m0s"
  regardless of the real budget, and running out of time produced a bare
  `context deadline exceeded`. The failure now names the budget, the cache
  path and the flag to raise, and falls back to a usable stale cache when
  one exists.
- `--json` is accepted on the commands that only ever emit JSON
  (`diagnostics *`, `mobility events`), and is a real `--format` shortcut on
  `apis` and `datasets search`, which have a meaningful `--format table`.
  Round 1 logged this with a recurrence trigger; round 2 hit it five times.
- The EV availability recipe and skill no longer hardcode `--origin ALPERIA`.
  That origin's newest measurement is from 2024 and all of its stations are
  inactive, so following the recipe reported zero chargers from stale data.
  The recipe discovers the origin instead, at a limit high enough that the
  truncation warning clears — the old `--limit 1000` returned a single origin
  and made the domain look like one provider.

## v0.4.2 - 2026-08-03

- `--limit` no longer silently caps counts on `mobility latest`,
  `diagnostics ev-charging` and `diagnostics parking-forecasts`. Filtering
  happens locally there, so the result knows the real match total and now
  reports it: `diagnostics ev-charging --limit 3` reported `current_count: 3`
  where 888 rows matched. Same defect v0.4.1 fixed for the discovery commands,
  missed because the cap lives in a shared helper. Refs #9.

## v0.4.1 - 2026-08-02

- `traffic *`, `a22 status`, and `mobility events` now report the newest row
  date they received and warn that the Mobility Timeseries event feed is not a
  live bulletin, so an empty or stale result is not evidence that roads are
  clear. They point at the live Content API alternative,
  `odh call tourism /v1/Announcement --param source=a22 --param rawsort=-LastChange`.
  The same correction was applied to the `mobility_events` and `a22_status` MCP
  tool descriptions, the agent docs, and the published skill. See issue #7.
- Fixed `GetCached` swallowing the error from a non-2xx response, which made an
  upstream 400 parse as data: `odh mobility stations --where bogus` exited 0
  with `count: 0`. Failed responses are also no longer cached.
- Fixed `odh datasets guide` printing next-step commands that do not parse.
  JSON-only commands now accept `--json` as a no-op, `--day` is `--date`, and a
  test dry-parses every command string the guide can emit.
- The `--limit` truncation warning was gated on `limit < 1000` while the default
  limit is 1000, so the default case never warned. `mobility origins`,
  `stations`, `datatypes`, `events` and the `traffic` commands now warn whenever
  a result fills `--limit`, including the default.
- The weekly upstream smoke workflow now checks that the live feeds (Mobility
  availability, GTFS realtime) are actually fresh, that the event-feed caveat is
  still emitted, and that the guide's commands parse, instead of only checking
  reachability and output strings.

## v0.4.0 - 2026-06-26

- Added `odh datasets guide <query>` to turn a data question into a curated
  discovery and verification path with dataset matches, commands to run next,
  and caveats that must be carried into answers.
- Exposed the same guidance through the MCP `datasets_guide` tool.
- Added discovery-first eval coverage and a live upstream smoke workflow for
  Open Data Hub command health.
- Updated README, agent docs, and the published skill guidance for the new
  dataset/source discovery flow.

## v0.3.1 - 2026-06-26

- Trimmed the documentation to essentials: rewrote the README, removed
  `ROADMAP.md` (planned work lives in GitHub issues) and the redundant
  `docs/commands.md`, `docs/how-to.md`, `docs/apis.md`, `docs/install.md`,
  `docs/data-scope.md`, and `docs/data-quality.md` (covered by `--help`,
  the README, and the remaining docs).
- Aligned the agent skill with the v0.3.0 release: version references, the
  origin-discovery rule from the 2026-06-10 eval round, and the MCP server
  mode option.

## v0.3.0 - 2026-06-10

First MCP-capable milestone release.

- Added an MCP server mode: `odh mcp serve` exposes the curated command
  surface as Model Context Protocol tools over stdio, executing each tool
  call in-process so MCP outputs match the documented CLI JSON contracts.
- Added `odh mcp tools` to list the MCP tool surface with input schemas.
- Documented MCP setup and behavior in `docs/mcp.md`.
- Recorded the first scored real-agent eval round (12 tasks, 11 pass,
  1 partial, 0 fail) in `evals/agent/results/2026-06-10.md` and documented
  the results convention in `docs/evaluation.md`.

## v0.2.5 - 2026-05-26

- Added practical built-in `--help` examples for traffic, Mobility, transit,
  diagnostics, and A22 commands.
- Added a task-oriented `docs/how-to.md` manual for common human and agent
  workflows.

## v0.2.4 - 2026-05-26

- Added a global opt-in `--timeout` flag for bounded agent and script runs.
- Added release-wide `SHA256SUMS` manifests and GitHub artifact attestations
  for release archives and Debian packages.
- Documented release verification through checksum manifests and GitHub CLI
  attestation checks.

## v0.2.3 - 2026-05-26

- Added JSON contract documentation for the key curated command outputs.
- Added golden JSON contract tests for traffic search, Mobility latest,
  diagnostics, A22 status, and Tourism type discovery.
- Added provenance fields to diagnostics, A22 status, and Tourism type JSON
  outputs.

## v0.2.2 - 2026-05-26

- Added a supported hidden `odh completion` command for bash, zsh, fish, and PowerShell completions.
- Added machine-readable agent recipes in `evals/agent/recipes.json` and validation in the agent eval smoke runner.
- Added explicit source/provenance metadata to transit JSON outputs and filtered Mobility latest output.
- Documented completions, agent recipes, and provenance fields.

## v0.2.1 - 2026-05-26

- Added a maintained roadmap and documented the public Homebrew tap.
- Added Debian package build support for Linux release artifacts.
- Updated stale v0.1 documentation references to the v0.2 project direction.
- Added agent eval tasks for realistic Merano-to-Kaltenbrunn transit routing and past-local A22 incident caveats.
- Updated the default HTTP `User-Agent` to `odh-cli/0.2`.

## v0.2.0 - 2026-05-26

First agent-friendly milestone release.

### Added

- Curated South Tyrol traffic layer for Open Data Hub `PROVINCE_BZ` events, including zone discovery, category discovery, date filtering, text search, deduplication, and stale-record warnings.
- A22 diagnostic command that separates current event rows from forecast rows and warns when data should not be treated as live incident evidence.
- Mobility discovery helpers for origins, stations, and datatypes.
- Filtered `mobility latest` queries for active, fresh, sorted availability checks.
- Data-quality diagnostics for EV charging availability, parking forecasts, and Tourism event caveats.
- GTFS dataset and GTFS-RT inspection commands.
- STA static GTFS transit commands for stop search, departures, direct trips, journey planning with transfers, and explicit delay-statistics limitations.
- Optional `transit journey --with-realtime` annotations for current GTFS-RT trip updates, service alerts, adjusted times, and transfer risk.
- Bounded GTFS archive downloads with concurrent-safe cache writes and stale-cache fallback when a refresh cannot complete.
- OpenClaw skill metadata and agent evals for repeatable agent-use checks.

### Notes

- `odh` remains an unofficial community CLI, not an official NOI Techpark/Open Data Hub product.
- Transit routing is static GTFS. Realtime data is an annotation layer, not live rerouting.
- Historical delay probability is not inferred without an archived GTFS-RT history.
- A22 live/current feeds should not be used as historical incident archives.
