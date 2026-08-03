<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Development

## Structure

- `cmd/odh` - executable entrypoint.
- `internal/apis` - typed registry of known API surfaces.
- `internal/cache` - small TTL file cache for low-risk discovery responses.
- `internal/client` - context-aware HTTP client.
- `internal/commands` - Cobra command tree, validation, and command execution.
- `internal/openapi` - OpenAPI JSON/YAML normalization.
- `internal/output` - deterministic JSON output helpers.
- `docs` - user, API, agent, data-quality, and development docs.
- `evals` - agent task evals and scoring fixtures.
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

## Agent Evals

Agent evals live outside the CLI command surface. They check whether agents can use existing commands correctly before new features are added.

```bash
scripts/run-agent-evals.sh
```

Set `ODH_EVAL_BIN=odh` to test an installed binary instead of the local source tree.

The task set and scoring rubric are documented in [evaluation.md](evaluation.md). Prefer diagnostic warnings over new natural-language commands when a repeated agent failure traces back to upstream data quality.

## CLI Contract

The CLI uses Cobra internally, but the automation contract stays stable:

- data commands write results to stdout,
- diagnostics and errors go to stderr,
- usage errors return exit code `2`,
- runtime failures return exit code `1`,
- current-data commands must not be served from the generic HTTP cache,
- curated JSON fields used by agents are documented in
  [json-contracts.md](json-contracts.md) and should be protected by tests when
  changed.

The generic HTTP cache is only for low-risk discovery surfaces such as OpenAPI specs, Tourism taxonomy values, Mobility type/origin/station/datatype discovery, and similar static-ish metadata. Live feeds, latest measurements, traffic events, diagnostics, and GTFS-RT responses should remain fresh unless a command has an explicit domain-specific cache contract such as the static GTFS archive cache.

## Build Metadata

Development builds default to `0.4.3-dev` with best-effort VCS metadata. Release builds can stamp metadata through Go linker flags:

```bash
go build \
  -ldflags "-X github.com/galjos/odh-cli/internal/version.Version=0.4.3 -X github.com/galjos/odh-cli/internal/version.Commit=$(git rev-parse --short HEAD) -X github.com/galjos/odh-cli/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o odh ./cmd/odh
```

Check the result with:

```bash
./odh version
./odh doctor --network=false
```

## Project Direction

The command surface stays intentionally narrow: a small audited core plus focused discovery, diagnostic, traffic, A22, GTFS, transit, and MCP commands for the public Tourism, Mobility, and GTFS APIs. Add command surface only when repeated eval failures show the same missing bounded upstream vocabulary or mechanical data-access step. Planned work is tracked in [GitHub issues](https://github.com/galjos/odh-cli/issues).
