<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# API Registry

The registry is intentionally explicit. It gives scripts and agents stable names for public Open Data Hub API surfaces.

Open Data Hub is maintained by NOI Techpark and exposes datasets in domains such as Tourism, Mobility, and Other. The current official docs describe it as a platform for European ecosystem data, while the practical public Tourism and Mobility datasets used by this CLI are strongly centered on South Tyrol / the Autonomous Province of Bolzano. See [data-scope.md](data-scope.md) for the source-backed interpretation rule.

| Name | Alias | Base URL | OpenAPI URL |
| --- | --- | --- | --- |
| `tourism` | `content` | `https://tourism.opendatahub.com` | `https://tourism.opendatahub.com/swagger/v1/swagger.json` |
| `mobility` | | `https://mobility.api.opendatahub.com` | `https://mobility.api.opendatahub.com/v2/apispec` |
| `gtfs` | | `https://gtfs.api.opendatahub.com` | |
| `gbfs` | | `https://gbfs.api.opendatahub.com` | |
| `transmodel` | | `https://transmodel.api.opendatahub.com` | |
| `alpinebits` | | `https://alpinebits.opendatahub.com` | |

Only Tourism and Mobility have curated commands in v0.1 because those are the public endpoints verified by the current smoke tests.

## API Notes

Tourism / Content API:

- Official API description: datasets from several domains, primarily tourism.
- Common data categories include accommodations, events, POIs, weather, snow data, regions, municipalities, tourism associations, and related tourism objects.
- The Tourism Data Browser explicitly presents this as South Tyrolean open data, but agents should still inspect returned location fields before making precise location claims.

Mobility / Time Series API:

- Official API description: historical and real-time time-series data, primarily in the mobility domain.
- API v2 uses station concepts such as station type, data type, origin, metadata, and measurements.
- Use `odh mobility types` and `odh mobility datatypes` for discovery before assuming a station type or origin exists.
- Use `odh mobility stations --station-type <type>` after station-type discovery to inspect concrete station metadata, coordinates, origins, and provider fields.

Standards APIs:

- `gtfs`, `gbfs`, `transmodel`, and `alpinebits` are registered so agents can discover their base URLs consistently.
- v0.1 does not yet provide curated wrappers for these APIs.

The Mobility commands include a narrow A22 diagnostic path:

- `odh mobility events --origin A22 --latest` checks current A22 events.
- `odh mobility stations --station-type TrafficSensor --origin A22` lists A22 traffic-sensor station records.
- `odh mobility datatypes --station-type TrafficSensor --origin A22` discovers A22 traffic-sensor data types.
- `odh a22 status` combines current events with the traffic forecast feed and emits warnings when Open Data Hub data should not be treated as current incident data.

## Useful References

- https://opendatahub.com/api/
- https://opendatahub.com/services/data-access/
- https://opendatahub.com/about-us/
- https://docs.opendatahub.com/en/latest/datasets.html
- https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
- https://tourism.databrowser.opendatahub.com/
- https://tourism.opendatahub.com/swagger/index.html
- https://mobility.api.opendatahub.com/v2/apispec
