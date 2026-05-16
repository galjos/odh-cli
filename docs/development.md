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
- `scripts` - release and maintenance scripts.

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

## Build Metadata

Development builds default to `0.1.5-dev` with best-effort VCS metadata. Release builds can stamp metadata through Go linker flags:

```bash
go build \
  -ldflags "-X github.com/galjos/odh-cli/internal/version.Version=0.1.5 -X github.com/galjos/odh-cli/internal/version.Commit=$(git rev-parse --short HEAD) -X github.com/galjos/odh-cli/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o odh ./cmd/odh
```

Check the result with:

```bash
./odh version
./odh doctor --network=false
```

## Release Notes

v0.1 intentionally avoids generated clients and package-manager publishing. It includes a small audited core plus focused discovery and diagnostic commands for the public Tourism, Mobility, and GTFS APIs.

The next useful milestones are:

- more curated commands for common endpoints,
- richer dataset metadata once a suitable public upstream catalog endpoint is selected,
- MCP server mode reusing the same internal packages,
- package-manager distribution.
