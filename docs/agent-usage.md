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
- Do not prompt interactively.
- Emit JSON by default.
- Keep examples public and unauthenticated.

This means agents can call `odh`, parse stdout as JSON, and treat stderr plus exit code as failure context.

## Safe Starter Commands

```bash
odh version
odh doctor --timeout 5s
odh apis
odh openapi mobility
odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
odh mobility types --kind event
odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 100
odh mobility events --origin A22 --latest --limit 20
odh mobility latest --station-type EChargingStation --data-type number-available --limit 5
odh a22 status --limit 10
```

## Generic Calls

When an endpoint is known from OpenAPI docs, use `odh call` instead of scraping a UI:

```bash
odh call tourism /v1/ODHActivityPoi \
  --param pagenumber=1 \
  --param pagesize=1 \
  --param seed=42
```

## Handling Failures

Agents should treat exit code `2` as a usage bug in the invocation and exit code `1` as a runtime problem such as HTTP failure, invalid JSON, or unavailable upstream service.

`odh doctor` is the preferred first command when an agent needs to distinguish a bad invocation from a local install problem or an upstream Open Data Hub reachability problem.

## Traffic Data Caveat

Open Data Hub Mobility feeds can expose different traffic concepts as station measurements, events, and forecasts. Agents should not infer live A22 traffic solely from `TrafficForecast` rows. Prefer `odh a22 status` when checking A22 because it reports current-event availability and warns when forecast timestamps indicate non-current data.

## Why No MCP Yet

The project is structured so an MCP server can reuse the registry and HTTP client later. v0.1 ships the CLI first because it is simpler to review, easier to test, and immediately useful in scripts.
