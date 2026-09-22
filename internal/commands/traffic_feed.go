// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"strings"
	"time"
)

// Shared Timeseries feed helpers. They live with the traffic code but are used
// beyond it: mobility events and A22 status report the same feed caveats.

// newestRecordTimestamp returns the most recent event timestamp across raw
// Mobility records, for callers that have not normalized them into events yet.
func newestRecordTimestamp(records []map[string]any) *time.Time {
	var newest *time.Time
	for _, record := range records {
		for _, key := range []string{"evtransactiontime", "evpublishtime"} {
			parsed := parseODHTime(asString(record[key]))
			if parsed == nil {
				continue
			}
			if newest == nil || parsed.After(*newest) {
				newest = parsed
			}
		}
	}
	return newest
}

// newestTrafficEventTimestamp returns the most recent transaction or publish
// timestamp across the given events, or nil when none of them carry one.
func newestTrafficEventTimestamp(events []trafficEvent) *time.Time {
	var newest *time.Time
	for _, event := range events {
		for _, value := range []string{event.TransactionTime, event.PublishedAt} {
			parsed := parseODHTime(value)
			if parsed == nil {
				continue
			}
			if newest == nil || parsed.After(*newest) {
				newest = parsed
			}
		}
	}
	return newest
}

// timeseriesEventFeedWarning reports the newest row this response carried rather
// than asserting an upstream feed status the CLI never checks, so the caveat
// stays true whatever upstream does next.
func timeseriesEventFeedWarning(newest *time.Time, announcementSource string) string {
	replacement := fmt.Sprintf("odh call tourism /v1/Announcement --param source=%s --param rawsort=-LastChange", announcementSource)
	if newest == nil {
		return fmt.Sprintf("this Mobility Timeseries event feed returned no dated rows; an empty result is not evidence that roads are clear, so cross-check current notices with: %s", replacement)
	}
	return fmt.Sprintf("the newest row in this Mobility Timeseries event response is dated %s; this feed is not a live bulletin, and neither a stale nor an empty result is evidence that roads are clear, so cross-check current notices with: %s", newest.UTC().Format("2006-01-02"), replacement)
}

// announcementSourceForOrigin maps a Mobility event origin to the Content API
// Announcement source that carries current notices for the same roads.
func announcementSourceForOrigin(origin string) string {
	if strings.EqualFold(strings.TrimSpace(origin), "a22") {
		return "a22"
	}
	return announcementTrafficSource
}
