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
- parse stdout as JSON,
- treat stderr as diagnostics,
- verify geographic claims from returned fields,
- use `odh a22 status` for A22 traffic checks.

ClawHub publishing uses the ClawHub CLI:

```bash
clawhub skill publish skills/open-data-hub-cli --version 0.1.1 --clawscan-note "Uses network access only through the user-installed odh CLI to query public Open Data Hub endpoints. Guidance is limited to public read-only Open Data Hub commands and official data-scope caveats."
```

Publishing to ClawHub makes the skill public under ClawHub's registry terms.
