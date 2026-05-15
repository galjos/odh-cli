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
odh apis
odh openapi mobility
odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
odh mobility latest --station-type EChargingStation --data-type number-available --limit 5
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

## Why No MCP Yet

The project is structured so an MCP server can reuse the registry and HTTP client later. v0.1 ships the CLI first because it is simpler to review, easier to test, and immediately useful in scripts.
