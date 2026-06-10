<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# MCP Server Mode

`odh mcp serve` runs `odh` as a Model Context Protocol server on
stdin/stdout. It exposes the curated command surface as MCP tools so MCP
clients such as Claude Code, Claude Desktop, or other agent runtimes can
query Open Data Hub without shelling out to the CLI.

Every tool call executes the matching CLI command in-process. Tool
output is therefore byte-identical to the documented CLI behavior:

- the first content block is the command's stdout, which follows the
  stable fields in [json-contracts.md](json-contracts.md),
- a second content block carries stderr diagnostics (for example
  stale-GTFS-cache warnings) when the command emitted any,
- a nonzero exit code becomes an MCP tool error (`isError: true`) with
  the CLI's stderr message, so usage errors and runtime failures stay
  visible to the agent.

Commands that default to compact table output are forced to `--json`
over MCP.

## Setup

Claude Code:

```bash
claude mcp add odh -- odh mcp serve
```

Generic `mcpServers` JSON configuration:

```json
{
  "mcpServers": {
    "odh": {
      "command": "odh",
      "args": ["mcp", "serve"]
    }
  }
}
```

The server needs no authentication and calls only public unauthenticated
Open Data Hub endpoints, exactly like the CLI.

## Tool Surface

The tool surface mirrors the curated commands: discovery
(`apis_list`, `datasets_search`, `mobility_types`, `mobility_origins`,
`mobility_datatypes`, `tourism_types`, `traffic_zones`,
`traffic_categories`, `gtfs_datasets`, `transit_stops_search`), data
queries (`mobility_latest`, `mobility_stations`, `mobility_events`,
`tourism_poi`, `traffic_search`, `traffic_today`, `traffic_events`,
`a22_status`, `gtfs_realtime`, `transit_departures`, `transit_trip`,
`transit_journey`, `transit_delay_stats`), data-quality verdicts
(`diagnostics_ev_charging`, `diagnostics_parking_forecasts`,
`diagnostics_tourism_events`), health and provenance (`doctor`,
`version`), and the raw escape hatch (`call_api`).

List the full tool surface with input schemas without speaking MCP:

```bash
odh mcp tools
```

Tool descriptions and the server instructions carry the same caveats as
the agent skill: discover before filtering, surface warnings, run
diagnostics before answering from areas with known freshness problems,
and treat unsupported capabilities as explicit answers rather than gaps
to guess around.

## Behavior Notes

- Tool calls are bounded: 60 seconds per call, 180 seconds for
  `transit_*` tools because a cold GTFS archive download may take up to
  two minutes before the CLI falls back to a cached copy.
- Concurrent tool calls are safe; each call runs on a fresh internal
  runner. The HTTP discovery cache in `~/.cache/odh-cli` is shared and
  concurrency-safe.
- `odh mcp serve` ignores stdin framing other than newline-delimited
  JSON-RPC; it is meant to be launched by an MCP client, not typed into.
- Do not combine the global `--timeout` flag with `mcp serve`; it would
  bound the lifetime of the whole server, not of individual tool calls.
- The shell completion command and the MCP commands themselves are not
  exposed as tools.

## Scope

MCP mode adds no new data access or reasoning: it is the same curated,
read-only, public-endpoint surface as the CLI, with the same warnings
and JSON contracts. Anything out of scope for the CLI (historical
GTFS-RT archives, live rerouting, A22 incident history) is equally out
of scope over MCP and is reported explicitly by the relevant tools.
