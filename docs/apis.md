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

The Mobility commands include a narrow A22 diagnostic path:

- `odh mobility events --origin A22 --latest` checks current A22 events.
- `odh mobility datatypes --station-type TrafficSensor --origin A22` discovers A22 traffic-sensor data types.
- `odh a22 status` combines current events with the traffic forecast feed and emits warnings when Open Data Hub data should not be treated as current incident data.

Useful references:

- https://opendatahub.com/api/
- https://opendatahub.com/services/data-access/
- https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
- https://tourism.opendatahub.com/swagger/index.html
- https://mobility.api.opendatahub.com/v2/apispec
