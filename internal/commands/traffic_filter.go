// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"slices"
	"strings"
)

func trafficZoneMatches(event trafficEvent, zoneIDs string) bool {
	values := trafficZoneIDValues(zoneIDs)
	if len(values) == 0 {
		return true
	}
	return containsString(values, event.ZoneID)
}

func trafficAreaMatches(event trafficEvent, area trafficArea) bool {
	if area.Name == "" {
		return true
	}
	if len(area.ZoneIDs) > 0 && !containsString(area.ZoneIDs, event.ZoneID) {
		return false
	}
	if len(area.Keywords) == 0 {
		return true
	}
	haystack := normalizeTrafficSearchText(strings.Join([]string{event.Zone, event.ZoneIT, event.Road, event.RoadName, event.Place, event.PlaceIT}, " "))
	for _, keyword := range area.Keywords {
		keyword = normalizeTrafficSearchText(keyword)
		if keyword != "" && strings.Contains(haystack, keyword) {
			return true
		}
	}
	return false
}

func trafficTypeMatches(event trafficEvent, filter string) bool {
	filter = normalizeTrafficTypeName(filter)
	if filter == "" || filter == "all" {
		return true
	}
	if event.Type == filter {
		return true
	}
	if filter == "closure" && strings.Contains(strings.ToLower(event.Place), "sperre") {
		return true
	}
	if filter == "bike" && strings.Contains(strings.ToLower(event.RoadName), "rad") {
		return true
	}
	if filter == "mountain-pass" && textContainsPass(event.RoadName+" "+event.Place) {
		return true
	}
	return false
}

func trafficRoadMatches(event trafficEvent, road string) bool {
	road = normalizeRoad(road)
	if road == "" {
		return true
	}
	filter := compactRoadToken(road)
	eventRoad := compactRoadToken(event.Road)
	return strings.EqualFold(event.Road, road) ||
		(filter != "" && strings.Contains(eventRoad, filter)) ||
		strings.Contains(strings.ToLower(event.RoadName), strings.ToLower(road))
}

func trafficNearMatches(event trafficEvent, near, radius string) bool {
	if strings.TrimSpace(near) == "" {
		return true
	}
	lat, lon, radiusKM, err := parseNearRadius(near, radius)
	if err != nil || len(event.Coordinates) < 2 {
		return false
	}
	eventLon, eventLat := event.Coordinates[0], event.Coordinates[1]
	return haversineKM(lat, lon, eventLat, eventLon) <= radiusKM
}

func trafficSearchMatches(event trafficEvent, search string) bool {
	if strings.TrimSpace(search) == "" {
		return true
	}
	// Identifiers are matched whole, never by substring: they are opaque, and a
	// bare "12" used to match every record whose message id contained 12.
	identifiers := []string{
		normalizeTrafficSearchText(event.ID),
		normalizeTrafficSearchText(event.SeriesID),
		normalizeTrafficSearchText(event.MessageID),
	}
	haystack := normalizeTrafficSearchText(strings.Join([]string{
		event.Source,
		event.Type,
		event.Subtype,
		event.Severity,
		event.ZoneID,
		event.Zone,
		event.ZoneIT,
		event.Road,
		event.RoadName,
		event.Place,
		event.PlaceIT,
	}, " "))
	// Road numbers are written both ways upstream ("SS12" and "SS 12"), so a
	// space-stripped copy lets either spelling find either.
	squashed := strings.ReplaceAll(haystack, " ", "")
	groups := trafficSearchTermGroups(search)
	if len(groups) == 0 {
		return true
	}
	for _, group := range groups {
		matched := false
		for _, term := range group {
			term = normalizeTrafficSearchText(term)
			if term == "" {
				continue
			}
			// Alphabetic terms match at a word boundary only (prefix of a word is
			// fine: cycle aliases like "radweg" must still find "Radrouten"). An
			// infix match would turn "auer" into every "Stützmauern".
			if trafficTextHasTerm(haystack, term) || slices.Contains(identifiers, term) {
				matched = true
				break
			}
			if isRoadToken(term) && strings.Contains(squashed, term) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// trafficTextHasTerm reports whether term sits at the start of any word in
// haystack. Prefixes are allowed; infix matches inside a longer word are not.
func trafficTextHasTerm(haystack, term string) bool {
	for _, word := range strings.Fields(haystack) {
		if strings.HasPrefix(word, term) {
			return true
		}
	}
	return false
}

func trafficSearchTermGroups(search string) [][]string {
	terms := joinRoadTokens(strings.Fields(normalizeTrafficSearchText(search)))
	groups := make([][]string, 0, len(terms))
	for _, term := range terms {
		if trafficSearchStopword(term) {
			continue
		}
		groups = append(groups, trafficSearchAlternatives(term))
	}
	return groups
}

// isRoadToken reports a road number written as one word, for example ss12 or sp13.
// Only these are matched against the space-stripped haystack, so ordinary words
// cannot join across a space and match something nobody asked for.
func isRoadToken(term string) bool {
	letters := 0
	for letters < len(term) && term[letters] >= 'a' && term[letters] <= 'z' {
		letters++
	}
	if letters == 0 || letters > 3 || letters == len(term) {
		return false
	}
	for i := letters; i < len(term); i++ {
		if term[i] < '0' || term[i] > '9' {
			return false
		}
	}
	return true
}

// joinRoadTokens collapses "ss 12" into "ss12" so both spellings of a road
// number search the same way. Upstream text uses both.
func joinRoadTokens(terms []string) []string {
	joined := make([]string, 0, len(terms))
	for i := 0; i < len(terms); i++ {
		if i+1 < len(terms) && isRoadToken(terms[i]+terms[i+1]) {
			joined = append(joined, terms[i]+terms[i+1])
			i++
			continue
		}
		joined = append(joined, terms[i])
	}
	return joined
}

func trafficSearchStopword(term string) bool {
	switch term {
	case "a", "an", "am", "and", "are", "auf", "bei", "by", "der", "die", "das", "del", "della", "den", "des", "di", "for", "heute", "in", "is", "la", "le", "near", "on", "road", "roads", "route", "street", "streets", "strasse", "strassen", "the", "today", "um", "und", "via", "why":
		return true
	default:
		return false
	}
}

func trafficSearchAlternatives(term string) []string {
	switch term {
	case "blocked", "closed", "closure", "closures", "roadblock", "roadblocks":
		return []string{"closure", "closed", "blocked", "sperre", "sperren", "gesperrt"}
	case "baustelle", "baustellen", "construction", "roadwork", "roadworks", "works":
		return []string{"roadworks", "baustelle", "baustellen", "arbeiten"}
	case "event", "events", "veranstaltung", "veranstaltungen":
		return []string{"event", "events", "veranstaltung", "veranstaltungen"}
	case "sperre", "sperren", "gesperrt":
		return []string{"sperre", "sperren", "gesperrt", "closure", "closed", "blocked"}
	// The bike category's own aliases must find cycle notices: upstream writes
	// Radroute and ciclabile, so a user typing the documented "radweg" would
	// otherwise get a confident empty answer.
	case "bike", "bici", "bicicletta", "ciclabil", "ciclabile", "cycle", "cycling", "fahrrad", "rad", "radroute", "radrouten", "radweg", "radwege":
		return []string{"radroute", "radweg", "fahrrad", "ciclabil", "bicicl", "cycle"}
	default:
		return []string{term}
	}
}

func trafficNoMatchesWarning(query trafficQuery, area trafficArea, hiddenStaleOpenEndedCount int) string {
	parts := trafficFilterParts(query, area)
	if len(parts) == 0 {
		return ""
	}
	warning := "no current ODH PROVINCE_BZ traffic events matched " + strings.Join(parts, ", ") + " in the selected date range"
	if hiddenStaleOpenEndedCount > 0 {
		warning += "; stale open-ended matches may exist, rerun with --include-stale to inspect them"
	}
	return warning
}

// trafficFilterParts describes the narrowing filters a query applied, so an
// empty result can name what it was narrowed by.
func trafficFilterParts(query trafficQuery, area trafficArea) []string {
	parts := make([]string, 0, 6)
	if search := strings.TrimSpace(query.Search); search != "" {
		parts = append(parts, fmt.Sprintf("search %q", search))
	}
	if zoneID := strings.TrimSpace(query.ZoneID); zoneID != "" {
		parts = append(parts, "zone-id "+zoneID)
	}
	if area.Name != "" {
		parts = append(parts, "area "+area.Name)
	}
	if eventType := normalizeTrafficTypeName(query.Type); eventType != "" && eventType != "all" {
		parts = append(parts, "type "+eventType)
	}
	if road := strings.TrimSpace(query.Road); road != "" {
		parts = append(parts, "road "+road)
	}
	if near := strings.TrimSpace(query.Near); near != "" {
		parts = append(parts, "near "+near)
	}
	return parts
}
