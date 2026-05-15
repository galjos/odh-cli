<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# API Registry

The registry is intentionally explicit. It gives scripts and agents stable names for public Open Data Hub API surfaces.

| Name | Alias | Base URL | OpenAPI URL |
| --- | --- | --- | --- |
| `tourism` | `content` | `https://tourism.opendatahub.com` | `https://tourism.opendatahub.com/swagger/v1/swagger.json` |
| `mobility` | | `https://mobility.api.opendatahub.com` | `https://mobility.api.opendatahub.com/v2/apispec` |
| `gtfs` | | `https://gtfs.api.opendatahub.com` | |
| `gbfs` | | `https://gbfs.api.opendatahub.com` | |
| `transmodel` | | `https://transmodel.api.opendatahub.com` | |
| `alpinebits` | | `https://alpinebits.opendatahub.com` | |

Only Tourism and Mobility have curated commands in v0.1 because those are the public endpoints verified by the current smoke tests.

Useful references:

- https://opendatahub.com/api/
- https://opendatahub.com/services/data-access/
- https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
- https://tourism.opendatahub.com/swagger/index.html
- https://mobility.api.opendatahub.com/v2/apispec
