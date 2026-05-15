<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Development

## Structure

- `cmd/odh` - executable entrypoint.
- `internal/apis` - typed registry of known API surfaces.
- `internal/client` - context-aware HTTP client.
- `internal/commands` - CLI parsing and command execution.
- `internal/openapi` - OpenAPI JSON/YAML normalization.
- `internal/output` - deterministic JSON output helpers.
- `docs` - user, API, agent, and development docs.
- `examples` - small shell examples.

## Local Checks

```bash
go fmt ./...
go test ./...
go vet ./...
go build ./cmd/odh
```

If `golangci-lint` is installed:

```bash
golangci-lint run
```

## Live Smoke Tests

Unit tests do not require Open Data Hub network availability. Live smoke tests are opt-in:

```bash
ODH_LIVE_TESTS=1 go test ./internal/commands -run Live
```

The live tests call public unauthenticated Tourism and Mobility endpoints.

## Release Notes

v0.1 intentionally avoids generated clients and packaging. It includes a small audited core plus focused discovery and diagnostic commands for the public Tourism and Mobility APIs.

The next useful milestones are:

- more curated commands for common endpoints,
- broader dataset search once a suitable public metadata endpoint is selected,
- MCP server mode reusing the same internal packages,
- binary release workflow and package-manager distribution.
