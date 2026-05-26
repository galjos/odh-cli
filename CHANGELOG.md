<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Changelog

All notable changes to `odh-cli` are documented here.

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
