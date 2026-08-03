<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Evaluation

`odh` should stay a clean data-access CLI. Evaluation work therefore lives outside the command surface.

The evaluation layer answers one question: can an agent use the existing CLI correctly for realistic Open Data Hub tasks?

## Agent Eval Tasks

The task set is in [../evals/agent/tasks.json](../evals/agent/tasks.json). It contains user-style prompts, expected command paths, pass criteria, and common failure modes.

The recipe set is in [../evals/agent/recipes.json](../evals/agent/recipes.json). It contains machine-readable command recipes, parse targets, and caveats that agents can use as stable starting paths before composing a final answer.

Current task themes:

- South Tyrol traffic text search and stale-event handling.
- Unterland / Ueberetsch roadworks and closure summaries.
- A22 traffic data discovery and forecast caveats.
- A22 current-feed limitations for past local incidents.
- Public-transport delay-probability limitations.
- Public-transport stop-ID disambiguation for station names with many matches.
- Public-transport journey planning from realistic local-location prompts.
- Mobility parking discovery.
- Mobility latest freshness and inactive-station filtering for availability questions.
- Data-quality diagnostics for EV availability, parking forecasts, and Tourism event caveats.

## Smoke Runner

Run the command-affordance checks from the repository root:

```bash
scripts/run-agent-evals.sh
```

By default, this runs the local source tree with `go run ./cmd/odh`. For direct source-checkout commands, use `go run ./cmd/odh version`, not `go run ./cmd/odh -- version`. To test an installed binary:

```bash
ODH_EVAL_BIN=odh scripts/run-agent-evals.sh
```

The smoke runner is intentionally live: it calls public unauthenticated Open Data Hub endpoints. It should not be treated like a hermetic unit test or required in normal CI.

The smoke runner also validates the recipe file shape so recipes remain a maintained artifact rather than stale documentation.

## Manual Scoring

For each prompt in `tasks.json`, run a fresh agent attempt and score it:

- `pass` - expected command path, source-aware answer, caveats handled.
- `partial` - useful data, but a missed warning, weak source handling, or unnecessary raw call.
- `fail` - guessed answer, stale data presented as current, unsupported capability invented, or wrong command family.

Record scored rounds in `evals/agent/results/` as one dated file per round,
including setup, per-task scores, failure analysis, and fix-category
decisions. The first round is
[../evals/agent/results/2026-06-10.md](../evals/agent/results/2026-06-10.md).

When a `partial` or `fail` turns on how much the answer should have been
trusted, name the dimension it failed on:

- `recency` - age of the newest dated row against the feed's own cadence.
- `coverage` - whether the response saw everything: `--limit` fill, `--request-limit`
  inspection cap, rows hidden by filters.
- `feed_semantics` - whether the feed is the kind of thing the question needs:
  Timeseries event vs live bulletin, forecast vs current, static GTFS vs live.
- `scope_match` - whether returned rows belong to the geography the question implied.
- `upstream_health` - HTTP status, retries, cache fallback, `doctor` verdict.

These are reviewer vocabulary for scoring a round. They are deliberately **not**
CLI output fields and must not become any: see issue #6, which rejected shipping
them as a `reliability` score. A scalar cannot carry the next command that the
warnings already name, and it would re-compress exactly what v0.4.1 decompressed.

Record the failed command path and the reason. Then decide the fix category:

- Docs or skill guidance.
- Eval wording.
- Agent reasoning.
- Narrow CLI feature.

Only the last category should change the CLI, and only when repeated eval failures point to the same missing bounded upstream vocabulary or mechanical discovery step.

## Clean CLI Rule

Do not add natural-language commands just because one eval prompt is difficult.

Prefer:

- discovery commands for bounded upstream vocabularies,
- stable JSON contracts,
- explicit unsupported responses,
- clear warnings.
- diagnostic commands when the upstream data is present but frequently stale or semantically surprising.

Avoid:

- hardcoded local place aliases,
- agent reasoning inside the CLI,
- endpoint-specific wrappers for one-off questions,
- silent fallback to non-ODH sources.
