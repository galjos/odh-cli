<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Contributing

Keep this project boring and predictable.

## Principles

- JSON-first output.
- Diagnostics on stderr.
- Nonzero exit codes for failures.
- No interactive prompts.
- Public unauthenticated examples unless a feature explicitly documents auth.
- Small packages and clear tests.

## Before Opening a PR

Run:

```bash
go fmt ./...
go test ./...
go vet ./...
go build ./cmd/odh
```

If your change touches live Open Data Hub behavior, also run:

```bash
ODH_LIVE_TESTS=1 go test ./internal/commands -run Live
```

## License Hygiene

Source files should carry SPDX headers. Code is MPL-2.0. Documentation, examples, configuration, and metadata are CC0-1.0 unless noted otherwise.
