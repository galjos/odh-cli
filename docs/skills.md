<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Skills

The repository includes an agent skill for using `odh`:

- `skills/open-data-hub-cli/SKILL.md`

The skill is intentionally small and contains only operational guidance for agents. It is suitable for ClawHub-style registries that publish a folder containing `SKILL.md`.

ClawHub publishing uses the ClawHub CLI:

```bash
clawhub skill publish skills/open-data-hub-cli --version 0.1.0 --clawscan-note "Uses network access only through the user-installed odh CLI to query public Open Data Hub endpoints."
```

Publishing to ClawHub makes the skill public under ClawHub's registry terms.
