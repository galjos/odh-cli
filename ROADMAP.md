<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Roadmap

`odh` should stay a clean, source-grounded CLI for Open Data Hub. Prefer better
discovery, diagnostics, docs, and evals over broad natural-language shortcuts.

## Near Term

- Keep the agent eval set close to real OpenClaw/Mira questions and add a task
  whenever an agent repeatedly struggles with the same command path.
- Keep `evals/agent/recipes.json` aligned with the highest-value command paths
  so agents have machine-readable examples without adding natural-language CLI
  shortcuts.
- Keep package-manager installation current through the
  [`galjos/homebrew-odh`](https://github.com/galjos/homebrew-odh) tap.
- Improve metadata discovery for datasets, station types, origins, datatypes,
  traffic zones, and documented caveats without duplicating the upstream APIs.
- Keep the OpenClaw/ClawHub skill aligned with the latest released CLI and
  require `odh doctor --timeout 10s` before answering data questions.

## Tracking Issues

- [#1](https://github.com/galjos/odh-cli/issues/1) Publish Homebrew tap for `odh` - done.
- [#2](https://github.com/galjos/odh-cli/issues/2) Run and expand real-agent evals after v0.2 - first scored round in [evals/agent/results/2026-06-10.md](evals/agent/results/2026-06-10.md).
- [#3](https://github.com/galjos/odh-cli/issues/3) Improve dataset and source metadata discovery.
- [#4](https://github.com/galjos/odh-cli/issues/4) Evaluate MCP server mode after CLI stabilizes.
- [#5](https://github.com/galjos/odh-cli/issues/5) Publish a signed APT repository.

## Later

- Explore MCP server mode once the CLI command surface and evals have stayed
  stable across several real agent-use sessions.
- Consider generated clients only if they reduce maintenance risk for verified
  upstream APIs.

## Explicit Non-Goals For Now

- Historical A22 incident reconstruction from live feeds.
- Historical train-delay probabilities without a real GTFS-RT archive.
- Full live public-transport rerouting.
- Hardcoded place aliases that duplicate what an agent can discover from
  upstream zones, stops, coordinates, or plain text search.
