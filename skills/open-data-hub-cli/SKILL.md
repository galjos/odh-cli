---
name: open-data-hub-cli
description: Use this skill when working with Open Data Hub, NOI Techpark data, ODH APIs, Tourism API, Mobility API, A22 traffic data, or when an agent should query Open Data Hub through the odh command-line tool instead of scraping websites.
homepage: https://github.com/galjos/odh-cli
metadata:
  {
    "openclaw":
      {
        "os": ["darwin", "linux"],
        "requires": { "bins": ["odh"] },
        "install":
          [
            {
              "id": "go",
              "kind": "go",
              "module": "github.com/galjos/odh-cli/cmd/odh@v0.1.0",
              "bins": ["odh"],
              "label": "Install odh CLI (go)",
            },
          ],
      },
  }
---

# Open Data Hub CLI

Use `odh` for public Open Data Hub API work. It is JSON-first, non-interactive, and suitable for scripts and agents.

Open Data Hub is maintained by NOI Techpark. Most practical Tourism and Mobility tasks are about South Tyrol / the Autonomous Province of Bolzano, but do not claim every returned record is located there unless coordinates, location fields, origin metadata, or official docs support it.

## First Checks

Run these before relying on the CLI:

```bash
odh version
odh doctor --timeout 10s
```

If `odh` is not installed, use the latest release from `galjos/odh-cli` or build it from source:

```bash
go build -o odh ./cmd/odh
```

In OpenClaw, this skill declares `odh` as a required binary. If OpenClaw marks the skill as needing setup, use the offered installer before trying to answer with Open Data Hub data.

When running from a source checkout, use `./odh` instead of `odh`.

## API Discovery

List known API surfaces:

```bash
odh apis
```

Fetch OpenAPI specs as JSON:

```bash
odh openapi mobility
odh openapi tourism
```

Use `odh call` for known endpoints rather than scraping a UI:

```bash
odh call tourism /v1/ODHActivityPoi \
  --param pagenumber=1 \
  --param pagesize=1 \
  --param seed=42
```

For geographic claims, inspect returned fields. Useful examples:

- Tourism: `GpsInfo`, `LocationInfo`, `RegionInfo`, `MunicipalityInfo`, `LicenseInfo`.
- Mobility: `sorigin`, `scode`, `stype`, `scoordinate`, `smetadata`.

## Curated Commands

Tourism point of interest:

```bash
odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
```

Mobility latest measurements:

```bash
odh mobility latest \
  --station-type EChargingStation \
  --data-type number-available \
  --limit 5
```

Mobility type and data-type discovery:

```bash
odh mobility types --kind event
odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 100
```

A22 traffic diagnostics:

```bash
odh mobility events --origin A22 --latest --limit 20
odh a22 status --limit 10
```

## Interpretation Rules

- Parse stdout as JSON.
- Treat nonzero exit codes as failures.
- Treat stderr as diagnostics, not data.
- Prefer `odh` and official OpenAPI specs over scraping Open Data Hub web pages.
- Treat South Tyrol as the common regional context, not as a universal record-level guarantee.
- Verify location-sensitive answers from coordinates, origins, and metadata in the JSON.
- Do not infer live A22 traffic from `TrafficForecast` rows alone.
- Prefer `odh a22 status` for A22 because it reports current-event availability and warns when forecast rows are not current incident data.
- Use `--where` and `--param key=value` instead of manually constructing query strings when a curated command supports them.

## Official References

- https://opendatahub.com/api/
- https://opendatahub.com/services/data-access/
- https://opendatahub.com/about-us/
- https://docs.opendatahub.com/en/latest/datasets.html
- https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html
