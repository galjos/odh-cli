<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Skills

The repository includes an agent skill for using `odh`:

- `skills/open-data-hub-cli/SKILL.md`

The skill is intentionally small and contains only operational guidance for agents. It is suitable for ClawHub-style registries that publish a folder containing `SKILL.md`.

The skill should stay focused on runtime behavior:

- prefer `odh` commands over browser scraping,
- run `odh doctor` before relying on upstream data,
- parse stdout as JSON only after adding `--json` or `--format json` to commands that default to compact table output,
- treat stderr as diagnostics,
- verify geographic claims from returned fields,
- use `odh gtfs` and `odh transit` for public-transport timetable and live-feed questions,
- use `odh traffic today` or `odh traffic events` for South Tyrol roadworks, closures, and road events from Open Data Hub `PROVINCE_BZ`,
- use `odh diagnostics` before answering EV availability, parking forecast, or Tourism event-discovery questions,
- use `odh a22 status` for A22 traffic checks.

The skill frontmatter also declares the `odh` runtime binary through `metadata.openclaw.requires.bins` and exposes a Go installer for OpenClaw setup flows. This is the ClawHub/OpenClaw-native way to tell an agent host that the skill depends on an external CLI.

ClawHub publishing uses the ClawHub CLI:

```bash
clawhub skill publish "$(pwd)/skills/open-data-hub-cli" --version 0.3.1 --clawscan-note "Uses network access only through the odh CLI to query public Open Data Hub endpoints. Declares odh as a required binary and provides an OpenClaw Go installer hint."
```

Publishing to ClawHub makes the skill public under ClawHub's registry terms.
