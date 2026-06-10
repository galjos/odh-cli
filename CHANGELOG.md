<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Changelog

All notable changes to `odh-cli` are documented here.

## Unreleased

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
