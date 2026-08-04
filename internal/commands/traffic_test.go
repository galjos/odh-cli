// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import "testing"

// The v0.5.0 field test found `--search "SS12"` returning nothing while
// `--search "SS 12"` returned mostly-irrelevant rows, because the haystack
// included opaque identifiers: a bare "12" matched every record whose
// message id contained 12.
func TestTrafficSearchIgnoresIdentifierSubstrings(t *testing.T) {
	event := trafficEvent{
		ID:        "urn:announcements:provincebz:2489d10e-53b1-51aa-a8c4-ca85d662cdac",
		MessageID: "3016412",
		Place:     "Bei Girlan Einbahnregelung wegen Arbeiten",
	}
	if trafficSearchMatches(event, "301") {
		t.Error("a substring of the message id must not match")
	}
	if trafficSearchMatches(event, "2489d10e") {
		t.Error("a substring of the id must not match")
	}
	if !trafficSearchMatches(event, "3016412") {
		t.Error("the whole message id must still match")
	}
	if !trafficSearchMatches(event, "girlan") {
		t.Error("ordinary text must still match")
	}
}

// Road numbers appear upstream both as "SS 12" and "SS12"; either spelling
// must find either, so the --road rejection can suggest a query that works.
func TestTrafficSearchMatchesRoadNumberEitherSpelling(t *testing.T) {
	spaced := trafficEvent{Place: "Kreuzung mit der SS 12 Brennerstaatsstrasse km 470"}
	tight := trafficEvent{Place: "Baustelle auf der SS12 bei Branzoll"}
	for _, query := range []string{"SS12", "SS 12", "ss12"} {
		if !trafficSearchMatches(spaced, query) {
			t.Errorf("%q did not match spaced road text", query)
		}
		if !trafficSearchMatches(tight, query) {
			t.Errorf("%q did not match unspaced road text", query)
		}
	}
	// The join must not let unrelated adjacent words collapse into a match.
	if trafficSearchMatches(trafficEvent{Place: "Klausen 12 Uhr"}, "SS12") {
		t.Error("SS12 must not match unrelated text that merely contains 12")
	}
}

// The bike rejection tells the user to search instead, so the category's own
// aliases must all reach the same notices. "radweg" returning nothing while
// "radroute" returned eight is what made the suggestion untrustworthy.
func TestTrafficSearchCycleAliasesAgree(t *testing.T) {
	event := trafficEvent{Place: "Die Radroute zwischen Klughammer und Gmund wurde GESPERRT"}
	for _, query := range []string{"radroute", "radweg", "fahrrad", "ciclabil", "bici", "cycle"} {
		if !trafficSearchMatches(event, query) {
			t.Errorf("%q did not match a cycle-route notice", query)
		}
	}
}
