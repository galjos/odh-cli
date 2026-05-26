<!--
SPDX-FileCopyrightText: 2026 Josef Gallmetzer

SPDX-License-Identifier: CC0-1.0
-->

# Data Scope

This project treats Open Data Hub as a public data platform with a strong South Tyrol focus, not as a guarantee that every record is located inside South Tyrol.

## What the Official Docs Say

NOI Techpark's data access page describes Open Data Hub as a stable, machine-readable channel for real-time and documented data, especially for mobility and tourism use cases:

- https://noi.bz.it/en/services/data-access
- https://opendatahub.com/services/data-access/

The Open Data Hub API page lists the main public API surfaces used by this CLI:

- Content API for datasets primarily focused on tourism.
- Time Series API for historical and real-time time-series data, primarily in the mobility domain.
- Additional standards APIs such as Transmodel, GTFS, GBFS, and AlpineBits.

Reference:

- https://opendatahub.com/api/

The current Open Data Hub documentation lists the main accessible domains as Mobility, Tourism, and Other. It also describes the broader goal as making datasets about the European ecosystem available to third parties.

Reference:

- https://docs.opendatahub.com/en/latest/datasets.html

The Open Data Hub about page describes the platform as an international Free Open Source Software project maintained by NOI Techpark. It says the project began with European mobility research data, added tourism data in 2016, and now aims to expand data collection and exchange at an international level.

Reference:

- https://opendatahub.com/about-us/

The Tourism Data Browser is more explicitly regional. It presents itself as a way to explore South Tyrolean open data and includes tourism objects such as accommodations, events, POIs, weather, snow data, regions, municipalities, and tourism associations.

Reference:

- https://tourism.databrowser.opendatahub.com/

## Practical Rule for Agents

For most current Tourism and Mobility tasks, a South Tyrol / Autonomous Province of Bolzano interpretation is a good starting assumption. Do not turn that into a universal claim.

When location matters:

- prefer coordinates and explicit location fields over dataset names,
- inspect Mobility fields such as `sorigin`, `scode`, `stype`, `scoordinate`, and `smetadata`,
- inspect Tourism fields such as `GpsInfo`, `LocationInfo`, `RegionInfo`, `MunicipalityInfo`, and `LicenseInfo`,
- mention uncertainty when a record has no coordinates or clear administrative location,
- use upstream OpenAPI specs and the official docs before making coverage claims.

## Current CLI Coverage

`odh` v0.2 focuses on public, read-only access to:

- Tourism content through `tourism.opendatahub.com`.
- Mobility time-series data through `mobility.api.opendatahub.com`.
- A small curated dataset catalog for common Tourism and Mobility entry points.
- Tourism taxonomy/type discovery for common public endpoints.
- Mobility station metadata discovery by station type.
- GTFS and STA timetable discovery through Open Data Hub GTFS endpoints.
- South Tyrol traffic event summaries over Open Data Hub `PROVINCE_BZ`.
- Data-quality diagnostics for EV charging, parking forecasts, and Tourism event caveats.
- API discovery and OpenAPI retrieval for the registered public surfaces.
- A narrow A22 diagnostic command that separates current events from forecast rows.

The CLI intentionally does not hide the upstream JSON. Scripts and agents should parse the returned metadata instead of relying on prose assumptions.
