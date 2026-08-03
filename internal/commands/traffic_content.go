// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	trafficSourceODH     = "odh"
	trafficSourceContent = "content"
)

// announcementTrafficSource is the Content API Announcement source carrying the
// provincial road bulletin that the traffic commands cover.
const announcementTrafficSource = "PROVINCE_BZ"

func normalizeTrafficSource(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", trafficSourceODH:
		return trafficSourceODH, nil
	case trafficSourceContent:
		return trafficSourceContent, nil
	default:
		return "", usageErrorf("unsupported traffic source %q; supported sources: odh, content", value)
	}
}

// rejectUnsupportedContentTrafficFlags fails filters the Announcement feed
// cannot answer. Applying them partially would return fewer matches than the
// user asked for, which reads as "no such events".
func rejectUnsupportedContentTrafficFlags(query trafficQuery) error {
	if strings.TrimSpace(query.ZoneID) != "" {
		return usageErrorf("--zone-id is not supported with --source content; Announcement records carry no zone id, only free-text place descriptions. Use --source odh, or narrow with --near or --search")
	}
	if area := normalizeAreaAlias(query.Area); area != "" && area != "all" {
		return usageErrorf("--area is not supported with --source content; the area aliases resolve to zone ids, which Announcement records do not carry. Use --source odh, or narrow with --near or --search")
	}
	if road := strings.TrimSpace(query.Road); road != "" {
		return usageErrorf("--road is not supported with --source content; Announcement records name the road only inside free-text descriptions. Use --source odh, or --search %q", road)
	}
	switch normalizeTrafficTypeName(query.Type) {
	case "bike":
		return usageErrorf("--type bike is not supported with --source content; Announcement records carry no cycle tag, and cycle notices are filed under the same tags as any other closure or roadwork, so a tag filter would drop real matches. Use --source odh, or --search radroute")
	}
	return nil
}

func (r *Runner) runContentTrafficQueryCobra(ctx context.Context, query trafficQuery, fromDay, toDay time.Time, stdout io.Writer) error {
	api, _ := r.Registry.Find("tourism")
	values := url.Values{}
	values.Set("source", announcementTrafficSource)
	values.Set("pagenumber", "1")
	values.Set("pagesize", strconv.Itoa(query.Limit))
	values.Set("begin", startOfDay(fromDay).UTC().Format(time.RFC3339))
	values.Set("end", endOfDay(toDay).UTC().Format(time.RFC3339))
	values.Set("rawsort", "-LastChange")
	requestURL, err := BuildURL(api.BaseURL, "/v1/Announcement", values)
	if err != nil {
		return err
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		return err
	}
	records := mapsFromList(extractItemsList(value))
	events, warnings := normalizeContentTrafficEvents(records, announcementTotalResults(value), query, fromDay, toDay)
	return writeTrafficOutput(stdout, trafficResult{
		Source:       trafficSourceContent,
		SourceDetail: "Open Data Hub Tourism Content API /v1/Announcement " + announcementTrafficSource + " road bulletin",
		Endpoint:     requestURL,
		From:         fromDay.Format("2006-01-02"),
		To:           toDay.Format("2006-01-02"),
		Type:         normalizeTrafficTypeName(query.Type),
		Search:       strings.TrimSpace(query.Search),
		RawCount:     len(records),
		Count:        len(events),
		Events:       events,
		Warnings:     warnings,
		OutputFormat: query.Format,
		IncludeRaw:   query.Raw,
	})
}

func normalizeContentTrafficEvents(raw []map[string]any, totalResults int, query trafficQuery, fromDay, toDay time.Time) ([]trafficEvent, []string) {
	events := make([]trafficEvent, 0, len(raw))
	now := time.Now()
	endedCount := 0
	futureCount := 0
	for _, record := range raw {
		event := normalizeContentTrafficEvent(record, query.Raw, now)
		if !contentTrafficTypeMatches(event, query.Type) {
			continue
		}
		if !trafficNearMatches(event, query.Near, query.Radius) {
			continue
		}
		if !trafficSearchMatches(event, query.Search) {
			continue
		}
		ended := contentTrafficEnded(event, now)
		event.Active = eventActiveInRange(event, fromDay, toDay) && !ended
		if ended {
			endedCount++
		}
		if !event.Active && !query.IncludeExpired {
			if !ended {
				futureCount++
			}
			continue
		}
		events = append(events, event)
	}
	deduped := dedupeTrafficEvents(events)
	sort.SliceStable(deduped, func(i, j int) bool {
		if deduped[i].Start != deduped[j].Start {
			return deduped[i].Start < deduped[j].Start
		}
		return deduped[i].Place < deduped[j].Place
	})

	staleCount := 0
	for _, event := range deduped {
		if event.Stale {
			staleCount++
		}
	}

	warnings := make([]string, 0)
	if len(events) != len(deduped) {
		warnings = append(warnings, fmt.Sprintf("deduplicated %d raw matching rows to %d events", len(events), len(deduped)))
	}
	if endedCount > 0 {
		if query.IncludeExpired {
			warnings = append(warnings, fmt.Sprintf("%d matching announcements had already ended when this query ran; they are included because --include-expired was passed", endedCount))
		} else {
			warnings = append(warnings, fmt.Sprintf("%d matching announcements had already ended when this query ran and were hidden; pass --include-expired to inspect them", endedCount))
		}
	}
	if futureCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d matching announcements start after the selected date range and were hidden", futureCount))
	}
	if staleCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d returned announcements have not changed upstream for more than 30 days; in this feed an announcement stays open until the provider sets an end time, so an old timestamp is not evidence that it is over", staleCount))
	}
	if query.IncludeStale {
		warnings = append(warnings, "--include-stale has no effect with --source content; no announcements are hidden by staleness in this source")
	}
	if len(deduped) == 0 {
		if warning := contentTrafficNoMatchesWarning(query, endedCount, totalResults, len(raw)); warning != "" {
			warnings = append(warnings, warning)
		}
	}
	if warning := contentTrafficTruncationWarning(totalResults, len(raw), query.Limit); warning != "" {
		warnings = append(warnings, warning)
	}
	warnings = append(warnings, "this source cannot populate zone_id, zone, zone_it, road, road_name, severity, or series_id; an empty value there means the Content API does not carry the field, not that the event has no zone, road, or severity")
	warnings = append(warnings, "source is the Open Data Hub Content API /v1/Announcement feed for "+announcementTrafficSource+"; compare with the official traffic service before presenting this as a complete live road bulletin")
	return deduped, warnings
}

func normalizeContentTrafficEvent(record map[string]any, includeRaw bool, now time.Time) trafficEvent {
	detail, _ := record["Detail"].(map[string]any)
	german, _ := detail["de"].(map[string]any)
	italian, _ := detail["it"].(map[string]any)
	tags := contentTrafficTags(record["TagIds"])
	event := trafficEvent{
		ID:              asString(record["Id"]),
		MessageID:       contentProviderMessageID(record["Mapping"]),
		Source:          trafficSourceContent,
		Type:            contentTrafficType(tags),
		Subtype:         strings.Join(tags, ","),
		Place:           cleanTrafficText(firstNonEmpty(asString(german["BaseText"]), asString(german["Title"]), asString(record["Shortname"]))),
		PlaceIT:         cleanTrafficText(firstNonEmpty(asString(italian["BaseText"]), asString(italian["Title"]))),
		Start:           asString(record["StartTime"]),
		End:             asString(record["EndTime"]),
		PublishedAt:     asString(record["LastChange"]),
		TransactionTime: asString(record["LastChange"]),
		Coordinates:     contentTrafficCoordinates(record["Geo"]),
	}
	if includeRaw {
		event.Raw = record
	}
	event.Stale = trafficEventStale(event, now)
	return event
}

// contentTrafficTags returns the traffic-event tag suffixes in a stable order.
// Every record also carries announcement:traffic-event, which says nothing
// beyond "this is a road notice", so it is dropped.
func contentTrafficTags(value any) []string {
	list, _ := value.([]any)
	tags := make([]string, 0, len(list))
	for _, item := range list {
		suffix, ok := strings.CutPrefix(strings.TrimSpace(asString(item)), "traffic-event:")
		if !ok || suffix == "" {
			continue
		}
		tags = append(tags, suffix)
	}
	sort.Strings(tags)
	return tags
}

// contentTrafficType maps Announcement tags onto the traffic category names.
// A record carries a context tag (hindrance, current, mountain-pass, special)
// plus a kind tag; mountain-pass wins because a pass notice is what a
// --type mountain-pass caller is asking for. Anything unrecognized stays
// "traffic" rather than being dropped.
func contentTrafficType(tags []string) string {
	category := "traffic"
	for _, tag := range tags {
		switch tag {
		case "mountain-pass":
			return "mountain-pass"
		case "road-work":
			category = "roadworks"
		case "closure":
			category = "closure"
		case "event":
			category = "event"
		case "speed-camera":
			category = "radar"
		case "maintenance":
			category = "roadworks"
		}
	}
	return category
}

// contentTrafficTypeMatches filters on the tag-derived category only. The
// keyword fallbacks the odh source uses read road and zone fields that this
// source leaves empty.
func contentTrafficTypeMatches(event trafficEvent, filter string) bool {
	filter = normalizeTrafficTypeName(filter)
	return filter == "" || filter == "all" || event.Type == filter
}

// contentTrafficEnded reports whether the provider has closed the announcement.
// EndTime is only set when an event ends; upstream Active is true on every
// PROVINCE_BZ record, including ones closed a year ago, so it is not read.
func contentTrafficEnded(event trafficEvent, now time.Time) bool {
	end := parseODHTime(event.End)
	return end != nil && !end.After(now)
}

func contentTrafficCoordinates(value any) []float64 {
	geo, _ := value.(map[string]any)
	position, _ := geo["position"].(map[string]any)
	lon, lonOK := numberValue(position["Longitude"])
	lat, latOK := numberValue(position["Latitude"])
	if !lonOK || !latOK {
		return nil
	}
	return []float64{lon, lat}
}

// contentProviderMessageID returns the upstream provider record id from the
// Mapping block, which is keyed by provider name.
func contentProviderMessageID(value any) string {
	mapping, _ := value.(map[string]any)
	providers := make([]string, 0, len(mapping))
	for provider := range mapping {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	for _, provider := range providers {
		entry, _ := mapping[provider].(map[string]any)
		if id := strings.TrimSpace(asString(entry["Id"])); id != "" {
			return id
		}
	}
	return ""
}

func announcementTotalResults(value any) int {
	object, _ := value.(map[string]any)
	total, ok := numberValue(object["TotalResults"])
	if !ok {
		return 0
	}
	return int(total)
}

func contentTrafficTruncationWarning(totalResults, fetched, limit int) string {
	if totalResults <= fetched {
		return ""
	}
	return fmt.Sprintf("the Content API reports %d announcements in this date range but --limit=%d fetched only %d; rerun with a higher --limit when traffic completeness matters", totalResults, limit, fetched)
}

func contentTrafficNoMatchesWarning(query trafficQuery, endedCount, totalResults, fetched int) string {
	parts := trafficFilterParts(query, trafficArea{})
	if len(parts) == 0 {
		return ""
	}
	// --search and --near run locally over the fetched page, so on a truncated
	// fetch an empty result says nothing about the records that were not fetched.
	if totalResults > fetched {
		return fmt.Sprintf("none of the %d fetched announcements matched %s; the Content API reports %d in this date range, so raise --limit before concluding there are none",
			fetched, strings.Join(parts, ", "), totalResults)
	}
	scope := "no open "
	if query.IncludeExpired {
		scope = "no "
	}
	warning := scope + announcementTrafficSource + " announcements matched " + strings.Join(parts, ", ") + " in the selected date range"
	if endedCount > 0 && !query.IncludeExpired {
		warning += "; already-ended announcements matched, rerun with --include-expired to inspect them"
	}
	return warning
}
