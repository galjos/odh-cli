<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Agent Evals

These evals test whether an agent can use `odh` cleanly without the CLI taking over the agent's reasoning layer.

The goal is not to make `odh` answer broad natural-language questions directly. The goal is to verify that an agent can:

- discover the right API surface,
- choose stable CLI filters instead of guessing hidden IDs,
- parse JSON output,
- notice stale-data and unsupported-capability warnings,
- run diagnostics before making claims from data areas with known freshness or semantics caveats,
- state Open Data Hub limitations clearly.

## Files

- `tasks.json` - real user-style prompts, expected command paths, scoring criteria, and common failure modes.
- `../../scripts/run-agent-evals.sh` - live command-affordance checks for the CLI surfaces used by the eval tasks.

## Run The Smoke Evals

From the repository root:

```bash
scripts/run-agent-evals.sh
```

The runner calls public unauthenticated Open Data Hub endpoints. It requires `go`, `jq`, and network access.

To test an installed binary instead of `go run`:

```bash
ODH_EVAL_BIN=odh scripts/run-agent-evals.sh
```

## Manual Agent Eval Protocol

Use each `prompt` in `tasks.json` as a fresh agent task. The agent may use the `odh` CLI and public official sources when the task explicitly requires comparison, but it should not scrape unrelated websites by default.

Score each task as:

- `pass` - uses the expected command path, handles caveats, and gives a source-aware answer.
- `partial` - reaches useful data but misses a warning, uses a less direct command, or overstates certainty.
- `fail` - guesses, scrapes when `odh` should be used first, ignores stale/unsupported warnings, or invents data.

For every failure, decide whether the fix belongs in:

- documentation or skill guidance,
- an eval task clarification,
- a narrow CLI discovery feature,
- or the agent's own reasoning layer.

Keep the CLI clean: add command surface only after repeated eval failures show the same missing bounded upstream vocabulary or mechanical data-access step.
