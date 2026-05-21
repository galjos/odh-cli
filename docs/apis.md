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
| `gtfs` | | `https://gtfs.api.opendatahub.com` | `https://gtfs.api.opendatahub.com/v1/apispec` |
| `gbfs` | | `https://gbfs.api.opendatahub.com` | |
| `transmodel` | | `https://transmodel.api.opendatahub.com` | `https://transmodel.api.opendatahub.com/apispec` |
| `alpinebits` | | `https://alpinebits.opendatahub.com` | |

Tourism, Mobility, and GTFS have curated commands in v0.1 because those are the public endpoints verified by the current smoke tests.

## API Notes

Tourism / Content API:

- Official API description: datasets from several domains, primarily tourism.
- Common data categories include accommodations, events, POIs, weather, snow data, regions, municipalities, tourism associations, and related tourism objects.
- The Tourism Data Browser explicitly presents this as South Tyrolean open data, but agents should still inspect returned location fields before making precise location claims.

Mobility / Time Series API:

- Official API description: historical and real-time time-series data, primarily in the mobility domain.
- API v2 uses station concepts such as station type, data type, origin, metadata, and measurements.
- Use `odh mobility types`, `odh mobility origins`, and `odh mobility datatypes` for discovery before assuming a station type, origin, or data type exists.
- Use `odh mobility stations --station-type <type>` after station-type discovery to inspect concrete station metadata, coordinates, origins, and provider fields.
- Use `odh traffic today` or `odh traffic events` for South Tyrol roadworks, closures, road events, and traffic restrictions from Open Data Hub `PROVINCE_BZ`.
- Use `odh traffic zones` to list upstream `PROVINCE_BZ` traffic zone IDs, and `--zone-id` for broad region filters.
- Use `odh traffic categories` to list the stable `--type` names and upstream subtype hints.
- Use `odh traffic search <text>` for roads, towns, and plain-language traffic questions while staying on Open Data Hub data.

Standards APIs:

- `gtfs` has curated wrappers for dataset listing, GTFS-RT realtime feeds, stop search, static departures, direct static timetable matches, and static transfer journeys.
- `gbfs`, `transmodel`, and `alpinebits` are registered so agents can discover their base URLs consistently.
- v0.1 does not yet provide curated wrappers for `gbfs`, `transmodel`, or `alpinebits`.
- `odh transit delay-stats` intentionally reports unsupported because the live GTFS API does not provide a historical GTFS-RT archive for probability calculations.

The Mobility commands include a narrow A22 diagnostic path:

- `odh traffic today --area ueberetsch-unterland --type roadworks` summarizes current Open Data Hub `PROVINCE_BZ` roadwork events.
- `odh traffic search "road closed badia" --today --zone-id 6 --json` searches event text and uses the upstream Pustertal zone ID.
- `odh traffic events --area bozen-unterland --from YYYY-MM-DD --to YYYY-MM-DD` queries a specific traffic-event date range.
- `odh traffic today --near 46.42,11.25 --radius 15km --json` filters Open Data Hub `PROVINCE_BZ` events by coordinates.
- `odh gtfs datasets` lists Open Data Hub GTFS datasets and their realtime-feed metadata.
- `odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5` inspects current GTFS-RT trip updates.
- `odh transit stops search auer` searches static STA timetable stops with common German/Italian place aliases.
- `odh transit departures --stop "Ora, Stazione di Ora" --date YYYY-MM-DD --around HH:MM` inspects static GTFS departures near a time.
- `odh transit trip --from auer --to brenner --date YYYY-MM-DD --time HH:MM --mode train` looks for direct static GTFS trip matches.
- `odh transit journey --from auer --to brenner --date YYYY-MM-DD --time HH:MM --max-transfers 2` plans static GTFS transfer itineraries.
- `odh transit journey --from auer --to brenner --date YYYY-MM-DD --time HH:MM --max-transfers 2 --with-realtime --json` annotates returned static journeys with matching current GTFS-RT delays, alerts, and transfer risk.
- `odh mobility events --origin A22 --latest` checks current A22 events.
- `odh mobility origins --station-type TrafficSensor` lists `sorigin` values before choosing one for sensor or datatype commands.
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
