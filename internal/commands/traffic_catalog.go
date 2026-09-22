// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"strings"
	"time"
)

func normalizeTrafficTypeFilter(value string) (string, error) {
	normalized := normalizeTrafficTypeName(value)
	switch normalized {
	case "", "all", "roadworks", "closure", "event", "traffic", "mountain-pass", "bike", "radar":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported traffic type %q", value)
	}
}

func normalizeTrafficTypeName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all", "any":
		return "all"
	case "roadwork", "roadworks", "works", "baustelle", "baustellen":
		return "roadworks"
	case "closure", "closures", "closed", "sperre", "sperren":
		return "closure"
	case "events", "event", "veranstaltung", "veranstaltungen":
		return "event"
	case "traffic", "incident", "incidents", "stau", "unfall", "warning":
		return "traffic"
	case "mountain-pass", "mountain-pass-closure", "pass", "passes":
		return "mountain-pass"
	case "bike", "cycle", "cycling", "radweg":
		return "bike"
	case "radar", "speed", "speed-control":
		return "radar"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func knownTrafficZones() []trafficZone {
	return []trafficZone{
		{ZoneID: "1", Name: "Vinschgau"},
		{ZoneID: "2", Name: "Burggrafenamt"},
		{ZoneID: "3", Name: "Bozen-Unterland"},
		{ZoneID: "4", Name: "Salten-Schlern"},
		{ZoneID: "5", Name: "Eisacktal-Wipptal"},
		{ZoneID: "6", Name: "Pustertal"},
		{ZoneID: "7", Name: "Ausserhalb Südtirol"},
	}
}

func knownTrafficCategories() []trafficCategory {
	return []trafficCategory{
		{
			Name:             "roadworks",
			Description:      "Roadworks and construction notices.",
			Aliases:          []string{"roadwork", "works", "baustelle", "baustellen"},
			UpstreamSubtypes: []string{"BAUSTELLE"},
		},
		{
			Name:             "closure",
			Description:      "Road closures and blocked-road notices.",
			Aliases:          []string{"closed", "closures", "sperre", "sperren", "gesperrt"},
			UpstreamSubtypes: []string{"SPERRE"},
		},
		{
			Name:             "event",
			Description:      "Road impacts caused by public events.",
			Aliases:          []string{"events", "veranstaltung", "veranstaltungen"},
			UpstreamSubtypes: []string{"VERANSTALTUNG"},
		},
		{
			Name:             "traffic",
			Description:      "Traffic incidents, congestion, warnings, and general restrictions.",
			Aliases:          []string{"incident", "incidents", "stau", "unfall", "warning"},
			UpstreamSubtypes: []string{"AMPELREGELUNG", "FREI BEFAHRBAR", "SCHNEEFALL", "STAU", "UNFALL", "VORSICHT"},
		},
		{
			Name:             "mountain-pass",
			Description:      "Pass and mountain-road closures or restrictions.",
			Aliases:          []string{"mountain-pass-closure", "pass", "passes"},
			UpstreamSubtypes: []string{"SPERRE", "WINTERSPERRE"},
		},
		{
			Name:             "bike",
			Description:      "Cycle-route and bike-path closures.",
			Aliases:          []string{"cycle", "cycling", "radweg"},
			UpstreamSubtypes: []string{"RADWEG_SPERRE"},
		},
		{
			Name:             "radar",
			Description:      "Speed-control notices.",
			Aliases:          []string{"speed", "speed-control"},
			UpstreamSubtypes: []string{"RADARKONTROLLE"},
		},
	}
}

func validateTrafficZoneIDs(value string) error {
	known := map[string]struct{}{}
	for _, zone := range knownTrafficZones() {
		known[zone.ZoneID] = struct{}{}
	}
	for _, zoneID := range trafficZoneIDValues(value) {
		if _, ok := known[zoneID]; !ok {
			return fmt.Errorf("unknown traffic zone-id %q; run odh traffic zones", zoneID)
		}
	}
	return nil
}

func trafficZoneIDValues(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func resolveTrafficArea(value string) (trafficArea, error) {
	normalized := normalizeAreaAlias(value)
	if normalized == "" || normalized == "all" {
		return trafficArea{}, nil
	}
	areas := map[string]trafficArea{
		"bozen-unterland": {Name: "bozen-unterland", ZoneIDs: []string{"3"}},
		"ueberetsch-unterland": {
			Name:    "ueberetsch-unterland",
			ZoneIDs: []string{"3"},
			Keywords: []string{
				"Salurn", "Neumarkt", "Auer", "Montan", "Tramin", "Kurtatsch", "Margreid", "Laag", "Buchholz", "Gfrill",
				"Eppan", "Kaltern", "St. Pauls", "Unterrain", "Girlan", "Missian", "Montiggl", "Laimburg", "Mendel", "Sigmundskron", "Frangart",
			},
		},
		"unterland":            {Name: "unterland", ZoneIDs: []string{"3"}, Keywords: []string{"Salurn", "Neumarkt", "Auer", "Montan", "Tramin", "Kurtatsch", "Margreid", "Laag", "Buchholz", "Gfrill"}},
		"ueberetsch":           {Name: "ueberetsch", ZoneIDs: []string{"3"}, Keywords: []string{"Eppan", "Kaltern", "St. Pauls", "Unterrain", "Girlan", "Missian", "Montiggl", "Laimburg", "Mendel", "Sigmundskron", "Frangart"}},
		"bozen":                {Name: "bozen", ZoneIDs: []string{"3"}, Keywords: []string{"Bozen", "Bolzano", "Stadtgemeinde Bozen"}},
		"salurn":               {Name: "salurn", ZoneIDs: []string{"3"}, Keywords: []string{"Salurn"}},
		"kaltern":              {Name: "kaltern", ZoneIDs: []string{"3"}, Keywords: []string{"Kaltern"}},
		"tramin":               {Name: "tramin", ZoneIDs: []string{"3"}, Keywords: []string{"Tramin"}},
		"eppan":                {Name: "eppan", ZoneIDs: []string{"3"}, Keywords: []string{"Eppan", "St. Pauls", "Girlan", "Missian", "Unterrain", "Montiggl", "Frangart"}},
		"auer":                 {Name: "auer", ZoneIDs: []string{"3"}, Keywords: []string{"Auer"}},
		"neumarkt":             {Name: "neumarkt", ZoneIDs: []string{"3"}, Keywords: []string{"Neumarkt"}},
		"kurtatsch":            {Name: "kurtatsch", ZoneIDs: []string{"3"}, Keywords: []string{"Kurtatsch"}},
		"margreid":             {Name: "margreid", ZoneIDs: []string{"3"}, Keywords: []string{"Margreid"}},
		"montan":               {Name: "montan", ZoneIDs: []string{"3"}, Keywords: []string{"Montan"}},
		"burggrafenamt":        {Name: "burggrafenamt", ZoneIDs: []string{"2"}},
		"eisacktal-wipptal":    {Name: "eisacktal-wipptal", ZoneIDs: []string{"5"}},
		"pustertal":            {Name: "pustertal", ZoneIDs: []string{"6"}},
		"vinschgau":            {Name: "vinschgau", ZoneIDs: []string{"1"}},
		"salten-schlern":       {Name: "salten-schlern", ZoneIDs: []string{"4"}},
		"ausserhalb-suedtirol": {Name: "ausserhalb-suedtirol", ZoneIDs: []string{"7"}},
	}
	area, ok := areas[normalized]
	if !ok {
		return trafficArea{}, fmt.Errorf("unknown traffic area %q", value)
	}
	return area, nil
}

func parseTrafficDateRange(from, to string) (time.Time, time.Time, error) {
	fromDay, err := parseTrafficDate(from)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	toDay, err := parseTrafficDate(to)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if toDay.Before(fromDay) {
		return time.Time{}, time.Time{}, fmt.Errorf("--to must not be before --from")
	}
	return fromDay, toDay, nil
}

func parseTrafficDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q; use YYYY-MM-DD", value)
	}
	return parsed, nil
}
