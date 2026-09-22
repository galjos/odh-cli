// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func normalizeTrafficEvents(raw []map[string]any, query trafficQuery, area trafficArea, fromDay, toDay time.Time) ([]trafficEvent, []string) {
	events := make([]trafficEvent, 0, len(raw))
	now := time.Now()
	staleCount := 0
	hiddenStaleOpenEndedCount := 0
	expiredCount := 0
	futureCount := 0
	for _, record := range raw {
		event := normalizeTrafficEvent(record, query.Raw, now)
		if !trafficZoneMatches(event, query.ZoneID) {
			continue
		}
		if !trafficAreaMatches(event, area) {
			continue
		}
		if !trafficTypeMatches(event, query.Type) {
			continue
		}
		if !trafficRoadMatches(event, query.Road) {
			continue
		}
		if !trafficNearMatches(event, query.Near, query.Radius) {
			continue
		}
		if !trafficSearchMatches(event, query.Search) {
			continue
		}
		active := eventActiveInRange(event, fromDay, toDay)
		event.Active = active
		if !active && !query.IncludeExpired {
			if eventEndBefore(event, fromDay) {
				expiredCount++
			} else {
				futureCount++
			}
			continue
		}
		if event.Stale && !eventHasEnd(event) && !query.IncludeStale {
			hiddenStaleOpenEndedCount++
			continue
		}
		if event.Stale {
			staleCount++
		}
		events = append(events, event)
	}
	deduped := dedupeTrafficEvents(events)
	sort.SliceStable(deduped, func(i, j int) bool {
		if deduped[i].Start != deduped[j].Start {
			return deduped[i].Start < deduped[j].Start
		}
		if deduped[i].Road != deduped[j].Road {
			return deduped[i].Road < deduped[j].Road
		}
		return deduped[i].Place < deduped[j].Place
	})

	feedWarning := timeseriesEventFeedWarning(newestTrafficEventTimestamp(deduped), "PROVINCE_BZ")
	if normalizeTrafficTypeName(query.Type) == "bike" {
		feedWarning += "; active reflects the stored date range, not verified current status. For current cycle-route notices, run: odh traffic search radroute --today --source content --json"
	}
	warnings := []string{feedWarning}
	if len(events) != len(deduped) {
		warnings = append(warnings, fmt.Sprintf("deduplicated %d raw matching rows to %d events", len(events), len(deduped)))
	}
	if staleCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d matching events have transaction or publish timestamps older than 30 days", staleCount))
	}
	if hiddenStaleOpenEndedCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d stale open-ended matching events were hidden; pass --include-stale to inspect them", hiddenStaleOpenEndedCount))
	}
	if expiredCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d expired matching events were hidden", expiredCount))
	}
	if futureCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d future matching events were hidden", futureCount))
	}
	if len(deduped) == 0 {
		if warning := trafficNoMatchesWarning(query, area, hiddenStaleOpenEndedCount); warning != "" {
			warnings = append(warnings, warning)
		}
	}
	if warning := mobilityTruncationWarning("returned", "raw event rows", query.Limit, len(raw), "traffic completeness"); warning != "" {
		warnings = append(warnings, warning)
	}
	warnings = append(warnings, "source is Open Data Hub PROVINCE_BZ; compare with the official traffic service before presenting this as a complete live road bulletin")
	return deduped, warnings
}

func normalizeTrafficEvent(record map[string]any, includeRaw bool, now time.Time) trafficEvent {
	metadata, _ := record["evmetadata"].(map[string]any)
	event := trafficEvent{
		ID:              firstNonEmpty(asString(record["evuuid"]), asString(record["evname"])),
		SeriesID:        asString(record["evseriesuuid"]),
		MessageID:       asString(metadata["messageId"]),
		Source:          "odh",
		Subtype:         strings.TrimSpace(asString(metadata["subTycodeValue"])),
		Severity:        firstNonEmpty(asString(metadata["messageGradDescDe"]), asString(metadata["messageGradDescIt"])),
		ZoneID:          asString(metadata["messageZoneId"]),
		Zone:            strings.TrimSpace(asString(metadata["messageZoneDescDe"])),
		ZoneIT:          strings.TrimSpace(asString(metadata["messageZoneDescIt"])),
		Road:            normalizeRoad(asString(metadata["messageStreetNr"])),
		RoadName:        strings.TrimSpace(asString(metadata["messageStreetInternetDescDe"])),
		Place:           cleanTrafficText(asString(metadata["placeDe"])),
		PlaceIT:         cleanTrafficText(asString(metadata["placeIt"])),
		Start:           asString(record["evstart"]),
		End:             asString(record["evend"]),
		PublishedAt:     firstNonEmpty(asString(metadata["publishDateTime"]), asString(metadata["publisherDateTime"])),
		TransactionTime: asString(record["evtransactiontime"]),
		Coordinates:     extractCoordinates(record["evlgeometry"]),
	}
	event.Type = classifyTrafficType(event)
	if includeRaw {
		event.Raw = record
	}
	event.Stale = trafficEventStale(event, now)
	return event
}

func eventActiveInRange(event trafficEvent, fromDay, toDay time.Time) bool {
	start := parseODHTime(event.Start)
	end := parseODHTime(event.End)
	rangeStart := startOfDay(fromDay)
	rangeEnd := endOfDay(toDay)
	if start != nil && start.After(rangeEnd) {
		return false
	}
	if end != nil && end.Before(rangeStart) {
		return false
	}
	return true
}

func eventEndBefore(event trafficEvent, fromDay time.Time) bool {
	end := parseODHTime(event.End)
	return end != nil && end.Before(startOfDay(fromDay))
}

func eventHasEnd(event trafficEvent) bool {
	return parseODHTime(event.End) != nil
}

func trafficEventStale(event trafficEvent, now time.Time) bool {
	for _, value := range []string{event.TransactionTime, event.PublishedAt} {
		parsed := parseODHTime(value)
		if parsed != nil {
			return parsed.Before(now.AddDate(0, 0, -30))
		}
	}
	return false
}

func dedupeTrafficEvents(events []trafficEvent) []trafficEvent {
	seen := map[string]struct{}{}
	result := make([]trafficEvent, 0, len(events))
	for _, event := range events {
		key := strings.Join([]string{
			event.ZoneID,
			event.Road,
			event.RoadName,
			normalizeDedupText(event.Place),
			event.Start,
			event.End,
		}, "|")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, event)
	}
	return result
}

func classifyTrafficType(event trafficEvent) string {
	subtype := strings.ToUpper(strings.TrimSpace(event.Subtype))
	combined := strings.ToLower(event.RoadName + " " + event.Place)
	switch subtype {
	case "BAUSTELLE":
		return "roadworks"
	case "SPERRE":
		if textContainsPass(combined) {
			return "mountain-pass"
		}
		return "closure"
	case "RADWEG_SPERRE":
		return "bike"
	case "VERANSTALTUNG":
		return "event"
	case "RADARKONTROLLE":
		return "radar"
	case "STAU", "UNFALL", "SCHNEEFALL", "AMPELREGELUNG", "VORSICHT", "FREI BEFAHRBAR":
		if textContainsPass(combined) {
			return "mountain-pass"
		}
		return "traffic"
	default:
		if strings.Contains(combined, "radroute") || strings.Contains(combined, "radweg") {
			return "bike"
		}
		if textContainsPass(combined) {
			return "mountain-pass"
		}
		return "traffic"
	}
}

func normalizeDedupText(value string) string {
	value = strings.ToLower(cleanTrafficText(value))
	return strings.Join(strings.Fields(value), " ")
}
