// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

// Package mcpserver exposes the curated odh command surface as Model
// Context Protocol tools. Every tool executes the corresponding CLI
// command in-process and returns its documented JSON output unchanged,
// so the JSON contracts in docs/json-contracts.md apply to MCP results
// as well.
package mcpserver

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

type paramType string

const (
	paramString    paramType = "string"
	paramInteger   paramType = "integer"
	paramBoolean   paramType = "boolean"
	paramStringMap paramType = "object"
)

type param struct {
	name     string
	flag     string // CLI flag without leading dashes; empty means positional argument
	typ      paramType
	desc     string
	required bool
}

type toolSpec struct {
	name           string
	desc           string
	base           []string
	forceJSON      bool // append --json for commands that default to table output
	params         []param
	timeoutSeconds int // 0 means defaultTimeoutSeconds
}

const (
	defaultTimeoutSeconds = 60
	// Transit commands may download a GTFS archive on a cold cache,
	// which the CLI bounds at two minutes before falling back.
	transitTimeoutSeconds = 180
)

func (t toolSpec) timeout() time.Duration {
	seconds := t.timeoutSeconds
	if seconds <= 0 {
		seconds = defaultTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (t toolSpec) inputSchema() map[string]any {
	properties := map[string]any{}
	required := []string{}
	for _, p := range t.params {
		property := map[string]any{
			"type":        string(p.typ),
			"description": p.desc,
		}
		if p.typ == paramStringMap {
			property["additionalProperties"] = map[string]any{"type": "string"}
		}
		properties[p.name] = property
		if p.required {
			required = append(required, p.name)
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func (t toolSpec) findParam(name string) (param, bool) {
	for _, p := range t.params {
		if p.name == name {
			return p, true
		}
	}
	return param{}, false
}

// buildArgs converts validated tool arguments into a CLI argument vector.
func buildArgs(spec toolSpec, raw json.RawMessage) ([]string, error) {
	input := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, fmt.Errorf("arguments must be a JSON object: %w", err)
		}
	}
	for name := range input {
		if _, ok := spec.findParam(name); !ok {
			return nil, fmt.Errorf("unknown argument %q for tool %q", name, spec.name)
		}
	}
	for _, p := range spec.params {
		if _, ok := input[p.name]; p.required && !ok {
			return nil, fmt.Errorf("argument %q is required for tool %q", p.name, spec.name)
		}
	}

	args := append([]string{}, spec.base...)

	// Positional arguments keep their declared order directly after the
	// command path.
	for _, p := range spec.params {
		if p.flag != "" {
			continue
		}
		value, ok := input[p.name]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("argument %q must be a string", p.name)
		}
		args = append(args, text)
	}

	if spec.forceJSON {
		args = append(args, "--json")
	}

	for _, p := range spec.params {
		if p.flag == "" {
			continue
		}
		value, ok := input[p.name]
		if !ok {
			continue
		}
		switch p.typ {
		case paramString:
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("argument %q must be a string", p.name)
			}
			if text != "" {
				args = append(args, "--"+p.flag, text)
			}
		case paramInteger:
			number, ok := value.(float64)
			if !ok || number != math.Trunc(number) {
				return nil, fmt.Errorf("argument %q must be an integer", p.name)
			}
			args = append(args, "--"+p.flag, strconv.Itoa(int(number)))
		case paramBoolean:
			truthy, ok := value.(bool)
			if !ok {
				return nil, fmt.Errorf("argument %q must be a boolean", p.name)
			}
			if truthy {
				args = append(args, "--"+p.flag)
			}
		case paramStringMap:
			entries, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("argument %q must be an object of string values", p.name)
			}
			keys := make([]string, 0, len(entries))
			for key := range entries {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				text, ok := entries[key].(string)
				if !ok {
					return nil, fmt.Errorf("argument %q value for key %q must be a string", p.name, key)
				}
				args = append(args, "--"+p.flag, key+"="+text)
			}
		default:
			return nil, fmt.Errorf("argument %q has unsupported type %q", p.name, p.typ)
		}
	}
	return args, nil
}

var toolSpecs = []toolSpec{
	{
		name: "version",
		desc: "Report odh CLI build and version metadata as JSON.",
		base: []string{"version"},
	},
	{
		name: "doctor",
		desc: "Check local CLI health and upstream Open Data Hub API reachability. Run this before answering data questions when freshness or availability matters.",
		base: []string{"doctor"},
	},
	{
		name: "apis_list",
		desc: "List the known public Open Data Hub API surfaces with base and OpenAPI URLs.",
		base: []string{"apis"},
	},
	{
		name: "datasets_search",
		desc: "Search the curated Open Data Hub dataset catalog for common entry points such as parking, charging, or events.",
		base: []string{"datasets", "search"},
		params: []param{
			{name: "query", typ: paramString, desc: "free-text catalog search, for example: parking", required: true},
			{name: "domain", flag: "domain", typ: paramString, desc: "optional domain filter, for example tourism or mobility"},
		},
	},
	{
		name: "datasets_guide",
		desc: "Suggest a discovery-first command path for an Open Data Hub data question. Returns matching datasets, discovery commands, verification commands, and caveats so agents avoid guessing upstream vocabulary.",
		base: []string{"datasets", "guide"},
		params: []param{
			{name: "query", typ: paramString, desc: "free-text data question, for example: parking forecast or A22 traffic", required: true},
			{name: "domain", flag: "domain", typ: paramString, desc: "optional domain filter, for example tourism or mobility"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum number of matching datasets to guide; 0 means no limit"},
		},
	},
	{
		name: "call_api",
		desc: "Call any registered Open Data Hub API path with query parameters and return raw upstream JSON. Escape hatch for data the curated tools do not cover; prefer curated tools when one fits.",
		base: []string{"call"},
		params: []param{
			{name: "api", typ: paramString, desc: "registered API name, for example tourism or mobility", required: true},
			{name: "path", typ: paramString, desc: "API path, for example /v1/Event", required: true},
			{name: "params", flag: "param", typ: paramStringMap, desc: "query parameters as key/value strings; values may contain commas"},
		},
	},
	{
		name: "tourism_poi",
		desc: "Query Tourism points of interest. Inspect returned coordinates and origins before making location claims.",
		base: []string{"tourism", "poi"},
		params: []param{
			{name: "limit", flag: "limit", typ: paramInteger, desc: "number of POIs to request (default 1)"},
			{name: "page", flag: "page", typ: paramInteger, desc: "page number"},
			{name: "seed", flag: "seed", typ: paramString, desc: "stable randomization seed"},
			{name: "fields", flag: "fields", typ: paramString, desc: "comma-separated upstream fields, for example Detail.en.Title,GpsInfo"},
		},
	},
	{
		name:      "tourism_types",
		desc:      "Discover Tourism taxonomy values (POI, event, accommodation, venue, article, and tag types) before filtering Tourism queries.",
		base:      []string{"tourism", "types"},
		forceJSON: true,
		params: []param{
			{name: "dataset", flag: "dataset", typ: paramString, desc: "taxonomy dataset: poi, event, event-topic, accommodation, article, venue, or tag (default poi)"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "number of type records to request (default 100)"},
			{name: "page", flag: "page", typ: paramInteger, desc: "page number"},
		},
	},
	{
		name:      "mobility_types",
		desc:      "Discover Mobility station, event, or edge types. Run this before assuming a station type name.",
		base:      []string{"mobility", "types"},
		forceJSON: true,
		params: []param{
			{name: "kind", flag: "kind", typ: paramString, desc: "type kind: station, event, or edge (default station)"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum number of types to request (default 200)"},
		},
	},
	{
		name: "mobility_origins",
		desc: "Discover data-provider origins for a Mobility station type. Run this before filtering any query by origin, even for seemingly obvious origins such as A22.",
		base: []string{"mobility", "origins"},
		params: []param{
			{name: "station_type", flag: "station-type", typ: paramString, desc: "station type, for example TrafficSensor", required: true},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum station records to inspect (default 1000); raise it if origins look incomplete"},
		},
	},
	{
		name: "mobility_stations",
		desc: "List Mobility station metadata (names, codes, coordinates, capacity) for a station type. Station metadata is not a measurement; use mobility_latest for current values. The origin filter is applied locally after fetching, so use a generous limit.",
		base: []string{"mobility", "stations"},
		params: []param{
			{name: "station_type", flag: "station-type", typ: paramString, desc: "station type, for example ParkingStation", required: true},
			{name: "origin", flag: "origin", typ: paramString, desc: "optional sorigin filter, for example A22"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum stations to request (default 20); raise it when filtering by origin"},
			{name: "where", flag: "where", typ: paramString, desc: "optional Open Data Hub where filter"},
		},
	},
	{
		name:      "mobility_datatypes",
		desc:      "Summarize available measurement data types for a Mobility station type and origin. A catalogued data type is not proof that open rows exist; verify with mobility_latest. Use a high limit to avoid truncating the summary.",
		base:      []string{"mobility", "datatypes"},
		forceJSON: true,
		params: []param{
			{name: "station_type", flag: "station-type", typ: paramString, desc: "station type, for example TrafficSensor", required: true},
			{name: "origin", flag: "origin", typ: paramString, desc: "optional sorigin filter, for example A22"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum station/datatype records to inspect (default 1000); use 10000 for complete summaries"},
		},
	},
	{
		name: "mobility_latest",
		desc: "Query latest Mobility time-series measurements with freshness, active-station, and sorting filters. For current-availability questions always set active=true, a fresh_within window, and sort=newest, and surface the returned warnings.",
		base: []string{"mobility", "latest"},
		params: []param{
			{name: "station_type", flag: "station-type", typ: paramString, desc: "station type, for example EChargingStation", required: true},
			{name: "data_type", flag: "data-type", typ: paramString, desc: "data type, for example number-available", required: true},
			{name: "origin", flag: "origin", typ: paramString, desc: "optional sorigin filter, for example ALPERIA"},
			{name: "active", flag: "active", typ: paramBoolean, desc: "keep only active stations"},
			{name: "fresh_within", flag: "fresh-within", typ: paramString, desc: "keep only rows with mvalidtime within this age, for example 24h or 7d"},
			{name: "sort", flag: "sort", typ: paramString, desc: "local sort: upstream, newest, oldest, or station"},
			{name: "request_limit", flag: "request-limit", typ: paramInteger, desc: "raw upstream rows to request before local filtering; raise it when warnings mention hidden later rows"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "number of measurements to return (default 5)"},
			{name: "where", flag: "where", typ: paramString, desc: "optional Open Data Hub where filter"},
		},
	},
	{
		name: "mobility_events",
		desc: "Query raw Mobility events for an origin, for example A22. This Timeseries event feed is not a live bulletin and not a historical incident archive: an empty or unchanged result proves nothing about current conditions, and the response's own warnings report the newest row date it received. For current notices use call_api with api=tourism, path=/v1/Announcement, param source=a22 (or source=PROVINCE_BZ for provincial roads) and rawsort=-LastChange.",
		base: []string{"mobility", "events"},
		params: []param{
			{name: "origin", flag: "origin", typ: paramString, desc: "event origin, for example A22"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum events to request (default 20)"},
		},
	},
	{
		name: "diagnostics_ev_charging",
		desc: "Data-quality verdict for EV charging availability. Run this before reporting charger availability; if the verdict is unavailable, do not present stale rows as current.",
		base: []string{"diagnostics", "ev-charging"},
		params: []param{
			{name: "origin", flag: "origin", typ: paramString, desc: "optional sorigin filter, for example ALPERIA"},
			{name: "fresh_within", flag: "fresh-within", typ: paramString, desc: "freshness window for current availability rows (default 24h)"},
			{name: "request_limit", flag: "request-limit", typ: paramInteger, desc: "raw upstream rows to inspect (default 10000)"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum filtered rows to include (default 10)"},
		},
	},
	{
		name: "diagnostics_parking_forecasts",
		desc: "Data-quality verdict for parking occupancy and forecasts. If the verdict is current_only, report current occupancy but omit stale forecast rows.",
		base: []string{"diagnostics", "parking-forecasts"},
		params: []param{
			{name: "origin", flag: "origin", typ: paramString, desc: "optional sorigin filter, for example Municipality Merano"},
			{name: "fresh_within", flag: "fresh-within", typ: paramString, desc: "freshness window for current and forecast rows (default 2h)"},
			{name: "forecast_minutes", flag: "forecast-minutes", typ: paramInteger, desc: "parking forecast horizon in minutes (default 60)"},
			{name: "request_limit", flag: "request-limit", typ: paramInteger, desc: "raw upstream rows to inspect (default 10000)"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum filtered rows to include per feed (default 10)"},
		},
	},
	{
		name: "diagnostics_tourism_events",
		desc: "Data-quality verdict for Tourism events: checks ActiveToday consistency, local date status, and GPS availability. Run this before making near-me-today event claims.",
		base: []string{"diagnostics", "tourism-events"},
		params: []param{
			{name: "date", flag: "date", typ: paramString, desc: "date used for local active-today checks, YYYY-MM-DD (default today)"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "number of upstream events to inspect (default 20)"},
		},
	},
	{
		name:      "traffic_zones",
		desc:      "List the Open Data Hub PROVINCE_BZ traffic zones with their zone IDs. Use this to scope traffic queries instead of guessing place aliases.",
		base:      []string{"traffic", "zones"},
		forceJSON: true,
	},
	{
		name:      "traffic_categories",
		desc:      "List stable traffic event type filters (roadworks, closure, event, traffic, and others) with their upstream subtypes and aliases.",
		base:      []string{"traffic", "categories"},
		forceJSON: true,
	},
	{
		name:      "traffic_search",
		desc:      "Search South Tyrol PROVINCE_BZ road events by free text with date, zone, and type filters. Stale open-ended rows are hidden by default and must not be presented as confirmed current closures; the source is a public bulletin feed, not a complete live road bulletin. Pass source=content to search the Content API Announcement bulletin instead, which is the feed the province still updates.",
		base:      []string{"traffic", "search"},
		forceJSON: true,
		params: []param{
			{name: "source", flag: "source", typ: paramString, desc: "traffic source: odh (Mobility Timeseries events, supports zone_id/area/road) or content (Content API Announcement bulletin, the feed the province still updates; road and type bike are rejected there, zone_id and area match by geographic inference from historical zone coordinates rather than a zone field, and zone_id, road and severity are absent from the results)"},
			{name: "query", typ: paramString, desc: "free-text search, for example: road closed badia", required: true},
			{name: "today", flag: "today", typ: paramBoolean, desc: "restrict to today's events"},
			{name: "from", flag: "from", typ: paramString, desc: "start date YYYY-MM-DD"},
			{name: "to", flag: "to", typ: paramString, desc: "end date YYYY-MM-DD"},
			{name: "zone_id", flag: "zone-id", typ: paramString, desc: "PROVINCE_BZ messageZoneId filter, for example 6; discover with traffic_zones"},
			{name: "area", flag: "area", typ: paramString, desc: "area alias, for example ueberetsch-unterland"},
			{name: "type", flag: "type", typ: paramString, desc: "type filter: all, roadworks, closure, event, traffic, mountain-pass, bike, or radar"},
			{name: "road", flag: "road", typ: paramString, desc: "road filter, for example SP13 or SS42"},
			{name: "near", flag: "near", typ: paramString, desc: "coordinate filter as lat,lon"},
			{name: "radius", flag: "radius", typ: paramString, desc: "radius for near, for example 15km"},
			{name: "include_stale", flag: "include-stale", typ: paramBoolean, desc: "include stale open-ended events for inspection as historical context only"},
			{name: "include_expired", flag: "include-expired", typ: paramBoolean, desc: "include expired events after local date filtering"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum raw events to request (default 1000)"},
		},
	},
	{
		name:      "traffic_today",
		desc:      "Summarize today's South Tyrol road events (roadworks, closures, notices) for an area or zone, deduplicated, with stale rows hidden by default. Pass source=content for the Content API Announcement bulletin, which is the feed the province still updates; it has no zone, road, or severity fields.",
		base:      []string{"traffic", "today"},
		forceJSON: true,
		params: []param{
			{name: "source", flag: "source", typ: paramString, desc: "traffic source: odh (Mobility Timeseries events, supports zone_id/area/road) or content (Content API Announcement bulletin, the feed the province still updates; road and type bike are rejected there, zone_id and area match by geographic inference from historical zone coordinates rather than a zone field, and zone_id, road and severity are absent from the results)"},
			{name: "area", flag: "area", typ: paramString, desc: "area alias, for example ueberetsch-unterland"},
			{name: "zone_id", flag: "zone-id", typ: paramString, desc: "PROVINCE_BZ messageZoneId filter; discover with traffic_zones"},
			{name: "type", flag: "type", typ: paramString, desc: "type filter: all, roadworks, closure, event, traffic, mountain-pass, bike, or radar"},
			{name: "road", flag: "road", typ: paramString, desc: "road filter, for example SP13 or SS42"},
			{name: "near", flag: "near", typ: paramString, desc: "coordinate filter as lat,lon"},
			{name: "radius", flag: "radius", typ: paramString, desc: "radius for near, for example 15km"},
			{name: "include_stale", flag: "include-stale", typ: paramBoolean, desc: "include stale open-ended events for inspection as historical context only"},
			{name: "include_expired", flag: "include-expired", typ: paramBoolean, desc: "include already-ended announcements or expired events after local date filtering"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum raw events to request (default 1000)"},
		},
	},
	{
		name:      "traffic_events",
		desc:      "List South Tyrol road events for an explicit date range, area, or zone, deduplicated, with stale rows hidden by default. Pass source=content for the Content API Announcement bulletin, which is the feed the province still updates; it has no zone, road, or severity fields.",
		base:      []string{"traffic", "events"},
		forceJSON: true,
		params: []param{
			{name: "source", flag: "source", typ: paramString, desc: "traffic source: odh (Mobility Timeseries events, supports zone_id/area/road) or content (Content API Announcement bulletin, the feed the province still updates; road and type bike are rejected there, zone_id and area match by geographic inference from historical zone coordinates rather than a zone field, and zone_id, road and severity are absent from the results)"},
			{name: "from", flag: "from", typ: paramString, desc: "start date YYYY-MM-DD"},
			{name: "to", flag: "to", typ: paramString, desc: "end date YYYY-MM-DD"},
			{name: "area", flag: "area", typ: paramString, desc: "area alias, for example ueberetsch-unterland"},
			{name: "zone_id", flag: "zone-id", typ: paramString, desc: "PROVINCE_BZ messageZoneId filter; discover with traffic_zones"},
			{name: "type", flag: "type", typ: paramString, desc: "type filter: all, roadworks, closure, event, traffic, mountain-pass, bike, or radar"},
			{name: "road", flag: "road", typ: paramString, desc: "road filter, for example SP13 or SS42"},
			{name: "near", flag: "near", typ: paramString, desc: "coordinate filter as lat,lon"},
			{name: "radius", flag: "radius", typ: paramString, desc: "radius for near, for example 15km"},
			{name: "include_stale", flag: "include-stale", typ: paramBoolean, desc: "include stale open-ended events for inspection as historical context only"},
			{name: "include_expired", flag: "include-expired", typ: paramBoolean, desc: "include expired events after local date filtering"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum raw events to request (default 1000)"},
		},
	},
	{
		name:      "a22_status",
		desc:      "Inspect A22 Brenner motorway event and forecast feeds. Forecast rows are not current incident evidence. The event feed is a Timeseries feed, not a live bulletin: an empty or stale event result is not evidence that no incidents exist or that roads are clear, and the response's warnings report the newest row date it received. Report it as \"this feed returned no data\" and check current notices with call_api, api=tourism, path=/v1/Announcement, param source=a22 and rawsort=-LastChange.",
		base:      []string{"a22", "status"},
		forceJSON: true,
		params: []param{
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum records to request from each A22 feed (default 20)"},
		},
	},
	{
		name: "gtfs_datasets",
		desc: "List Open Data Hub GTFS datasets with static and realtime feed URLs.",
		base: []string{"gtfs", "datasets"},
	},
	{
		name: "gtfs_realtime",
		desc: "Inspect current GTFS-RT trip updates, vehicle positions, or service alerts. This is a live snapshot, not a delay history; missing entities do not prove trips are on time.",
		base: []string{"gtfs", "realtime"},
		params: []param{
			{name: "dataset", flag: "dataset", typ: paramString, desc: "GTFS dataset id (default sta-time-tables)"},
			{name: "feed", flag: "feed", typ: paramString, desc: "feed: trip-updates, vehicle-positions, or service-alerts"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum entities to include (default 20)"},
			{name: "route_id", flag: "route-id", typ: paramString, desc: "optional route_id filter"},
			{name: "trip_id", flag: "trip-id", typ: paramString, desc: "optional GTFS trip_id filter for trip-updates"},
		},
	},
	{
		name:           "transit_stops_search",
		desc:           "Search public-transport stops in the static STA GTFS timetable and return stop IDs. Run this first and switch to exact stop IDs when a place name matches many platforms or bus bays. Stop names use local (often Italian) spellings.",
		base:           []string{"transit", "stops", "search"},
		forceJSON:      true,
		timeoutSeconds: transitTimeoutSeconds,
		params: []param{
			{name: "query", typ: paramString, desc: "stop name search text, for example: merano", required: true},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum stops to return (default 20)"},
		},
	},
	{
		name:           "transit_departures",
		desc:           "List static GTFS departures for a stop around a time. Use stop_id from transit_stops_search for unambiguous results; parent station IDs expand to their platforms.",
		base:           []string{"transit", "departures"},
		forceJSON:      true,
		timeoutSeconds: transitTimeoutSeconds,
		params: []param{
			{name: "stop", flag: "stop", typ: paramString, desc: "stop search query; prefer stop_id when names are ambiguous"},
			{name: "stop_id", flag: "stop-id", typ: paramString, desc: "exact GTFS stop_id or parent_station"},
			{name: "date", flag: "date", typ: paramString, desc: "service date YYYY-MM-DD (default today)"},
			{name: "around", flag: "around", typ: paramString, desc: "departure time HH:MM to search around"},
			{name: "window", flag: "window", typ: paramString, desc: "time window around the time, for example 15m"},
			{name: "mode", flag: "mode", typ: paramString, desc: "mode filter: all, train, bus, or cable-car"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum departures to return (default 20)"},
		},
	},
	{
		name:           "transit_trip",
		desc:           "Find direct static GTFS trips between two stops. Returns only direct matches; use transit_journey for connections with transfers.",
		base:           []string{"transit", "trip"},
		forceJSON:      true,
		timeoutSeconds: transitTimeoutSeconds,
		params: []param{
			{name: "from", flag: "from", typ: paramString, desc: "origin stop query"},
			{name: "to", flag: "to", typ: paramString, desc: "destination stop query"},
			{name: "from_stop_id", flag: "from-stop-id", typ: paramString, desc: "exact origin GTFS stop_id or parent_station"},
			{name: "to_stop_id", flag: "to-stop-id", typ: paramString, desc: "exact destination GTFS stop_id or parent_station"},
			{name: "date", flag: "date", typ: paramString, desc: "service date YYYY-MM-DD (default today)"},
			{name: "time", flag: "time", typ: paramString, desc: "origin departure time HH:MM"},
			{name: "window", flag: "window", typ: paramString, desc: "time window around the time, for example 15m"},
			{name: "mode", flag: "mode", typ: paramString, desc: "mode filter: all, train, bus, or cable-car"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum direct trip matches to return (default 20)"},
		},
	},
	{
		name:           "transit_journey",
		desc:           "Plan static GTFS transfer journeys between two stops. Routing is static timetable data; with_realtime only annotates the static journeys with current GTFS-RT delays, alerts, and transfer risk, it does not reroute live. Missing realtime matches do not prove trips are on time.",
		base:           []string{"transit", "journey"},
		forceJSON:      true,
		timeoutSeconds: transitTimeoutSeconds,
		params: []param{
			{name: "from", flag: "from", typ: paramString, desc: "origin stop query"},
			{name: "to", flag: "to", typ: paramString, desc: "destination stop query"},
			{name: "from_stop_id", flag: "from-stop-id", typ: paramString, desc: "exact origin GTFS stop_id or parent_station"},
			{name: "to_stop_id", flag: "to-stop-id", typ: paramString, desc: "exact destination GTFS stop_id or parent_station"},
			{name: "date", flag: "date", typ: paramString, desc: "service date YYYY-MM-DD (default today)"},
			{name: "time", flag: "time", typ: paramString, desc: "earliest origin departure time HH:MM"},
			{name: "max_transfers", flag: "max-transfers", typ: paramInteger, desc: "maximum transfers (default 3)"},
			{name: "max_duration", flag: "max-duration", typ: paramString, desc: "maximum journey duration to search (default 6h)"},
			{name: "min_transfer", flag: "min-transfer", typ: paramString, desc: "minimum transfer time, for example 3m"},
			{name: "mode", flag: "mode", typ: paramString, desc: "mode filter: all, train, bus, or cable-car"},
			{name: "with_realtime", flag: "with-realtime", typ: paramBoolean, desc: "annotate static journeys with current GTFS-RT trip updates, alerts, and transfer risk"},
			{name: "limit", flag: "limit", typ: paramInteger, desc: "maximum journeys to return (default 3)"},
		},
	},
	{
		name:           "transit_delay_stats",
		desc:           "Report whether historical delay statistics are available for a connection. Historical delay probability is unsupported without an archived GTFS-RT history; this tool states that explicitly instead of guessing.",
		base:           []string{"transit", "delay-stats"},
		forceJSON:      true,
		timeoutSeconds: transitTimeoutSeconds,
		params: []param{
			{name: "from", flag: "from", typ: paramString, desc: "origin stop query", required: true},
			{name: "to", flag: "to", typ: paramString, desc: "destination stop query", required: true},
			{name: "time", flag: "time", typ: paramString, desc: "origin departure time HH:MM"},
			{name: "weekday", flag: "weekday", typ: paramString, desc: "weekday filter, for example saturday"},
		},
	},
}
