// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/galjos/odh-cli/internal/apis"
)

// announcementFixture mirrors the live /v1/Announcement PROVINCE_BZ shape:
// every record carries Geo, StartTime, LastChange, Shortname, both Detail
// languages and Active=true, EndTime is set only once the provider closes the
// record, and the tags come in a context plus kind pair.
const announcementFixture = `{"TotalResults":4,"TotalPages":1,"CurrentPage":1,"Items":[
	{
		"Id":"urn:announcements:provincebz:open-roadwork",
		"Active":true,
		"Source":"PROVINCE_BZ",
		"Shortname":"General Traffic Hindrance Road Work",
		"TagIds":["announcement:traffic-event","traffic-event:hindrance","traffic-event:road-work"],
		"StartTime":"2026-05-10T22:00:00Z",
		"EndTime":null,
		"LastChange":"2999-05-15T08:20:00.5968609+00:00",
		"FirstImport":"2026-05-11T06:10:00.458229+00:00",
		"Geo":{"position":{"Default":true,"Geometry":"POINT (11.25 46.42)","Latitude":46.42,"Longitude":11.25}},
		"Detail":{
			"de":{"Title":"Behinderung Arbeiten","BaseText":"Bei Girlan (km 2,300 - 2,530) Einbahnregelung wegen Arbeiten.","Language":null},
			"it":{"Title":"Ostacolo lavori","BaseText":"Presso Cornaiano (km 2,300 - 2,530) senso unico alternato per lavori.","Language":null}
		},
		"Mapping":{"ProviderProvinceBz":{"Id":"2911001","SyncTime":"2026-05-11T06:10:00.458Z"}}
	},
	{
		"Id":"urn:announcements:provincebz:closed-accident",
		"Active":true,
		"Source":"PROVINCE_BZ",
		"Shortname":"Current Situation Accident",
		"TagIds":["announcement:traffic-event","traffic-event:accident","traffic-event:current"],
		"StartTime":"2026-05-15T22:00:00Z",
		"EndTime":"2026-05-16T09:30:00.353Z",
		"LastChange":"2026-05-16T09:30:00.5968609+00:00",
		"Geo":{"position":{"Default":true,"Geometry":"POINT (11.34 46.49)","Latitude":46.49,"Longitude":11.34}},
		"Detail":{
			"de":{"Title":"Aktuelle Lage Unfall","BaseText":"Bei Bozen Süd 2 km Stau nach einem Unfall.","Language":null},
			"it":{"Title":"Situazione attuale incidente","BaseText":"Presso Bolzano sud 2 km di coda dopo un incidente.","Language":null}
		},
		"Mapping":{"ProviderProvinceBz":{"Id":"2911002","SyncTime":"2026-05-16T09:30:00.353Z"}}
	},
	{
		"Id":"urn:announcements:provincebz:running-closure",
		"Active":true,
		"Source":"PROVINCE_BZ",
		"Shortname":"General Traffic Hindrance Closure or Blockage",
		"TagIds":["announcement:traffic-event","traffic-event:closure","traffic-event:hindrance"],
		"StartTime":"2026-05-14T22:00:00Z",
		"EndTime":"2999-08-30T00:00:00Z",
		"LastChange":"2999-05-15T09:40:00.5968609+00:00",
		"Geo":{"position":{"Default":true,"Geometry":"POINT (11.903 46.61)","Latitude":46.61,"Longitude":11.903}},
		"Detail":{
			"de":{"Title":"Behinderung Sperre","BaseText":"Die Radroute Nr. 1 Bozen-Salurn ist zwischen Laag und Kurtinig GESPERRT.","Language":null},
			"it":{"Title":"Ostacolo chiusura","BaseText":"La ciclabile n. 1 Bolzano-Salorno è CHIUSA tra Laghetti e Cortina.","Language":null}
		},
		"Mapping":{"ProviderProvinceBz":{"Id":"2911003","SyncTime":"2026-05-15T09:40:00.353Z"}}
	},
	{
		"Id":"urn:announcements:provincebz:open-pass",
		"Active":true,
		"Source":"PROVINCE_BZ",
		"Shortname":"Mountain Pass Caution and Hazard",
		"TagIds":["announcement:traffic-event","traffic-event:caution","traffic-event:mountain-pass"],
		"StartTime":"2024-11-17T23:00:00Z",
		"EndTime":null,
		"LastChange":"2024-11-18T15:00:00.5968609+00:00",
		"Geo":{"position":{"Default":true,"Geometry":"POINT (10.45 46.53)","Latitude":46.53,"Longitude":10.45}},
		"Detail":{
			"de":{"Title":"Pass Vorsicht","BaseText":"Ab Trafoi (km 138,000) Fahrverbot für Fahrzeuge mit einer Länge über 10,50 m.","Language":null},
			"it":{"Title":"Passo attenzione","BaseText":"Da Trafoi (km 138,000) divieto di transito per veicoli di lunghezza superiore a 10,50 m.","Language":null}
		},
		"Mapping":{"ProviderProvinceBz":{"Id":"2911004","SyncTime":"2024-11-18T15:00:00.353Z"}}
	}
]}`

func newAnnouncementTestServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/Announcement" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("source") != "PROVINCE_BZ" || query.Get("rawsort") != "-LastChange" {
			t.Errorf("unexpected query %q", r.URL.RawQuery)
		}
		// The date range is sent server-side; the local timezone decides the
		// exact instants, so assert the ordering rather than literal values.
		begin, err := time.Parse(time.RFC3339, query.Get("begin"))
		if err != nil {
			t.Errorf("invalid begin %q: %v", query.Get("begin"), err)
		}
		end, err := time.Parse(time.RFC3339, query.Get("end"))
		if err != nil {
			t.Errorf("invalid end %q: %v", query.Get("end"), err)
		}
		if !begin.Before(end) {
			t.Errorf("begin %s is not before end %s", begin, end)
		}
		_, _ = w.Write([]byte(body))
	}))
}

type contentTrafficResponse struct {
	Source       string `json:"source"`
	SourceDetail string `json:"source_detail"`
	Endpoint     string `json:"endpoint"`
	RawCount     int    `json:"raw_count"`
	Count        int    `json:"count"`
	Events       []struct {
		ID          string    `json:"id"`
		MessageID   string    `json:"message_id"`
		Source      string    `json:"source"`
		Type        string    `json:"type"`
		Subtype     string    `json:"subtype"`
		ZoneID      string    `json:"zone_id"`
		Road        string    `json:"road"`
		Severity    string    `json:"severity"`
		Place       string    `json:"place"`
		PlaceIT     string    `json:"place_it"`
		Start       string    `json:"start"`
		End         string    `json:"end"`
		PublishedAt string    `json:"published_at"`
		Coordinates []float64 `json:"coordinates"`
		Active      bool      `json:"active"`
		Stale       bool      `json:"stale"`
	} `json:"events"`
	Warnings []string `json:"warnings"`
}

func runContentTrafficCommand(t *testing.T, server *httptest.Server, args []string) contentTrafficResponse {
	t.Helper()
	runner := newTestRunner(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded contentTrafficResponse
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	return decoded
}

func TestRunTrafficEventsContentSourceMapsAnnouncements(t *testing.T) {
	server := newAnnouncementTestServer(t, announcementFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "events",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--json",
	})
	if decoded.Source != "content" || !strings.Contains(decoded.SourceDetail, "/v1/Announcement") {
		t.Fatalf("unexpected source: %q / %q", decoded.Source, decoded.SourceDetail)
	}
	if !strings.Contains(decoded.Endpoint, "/v1/Announcement") || !strings.Contains(decoded.Endpoint, "source=PROVINCE_BZ") {
		t.Fatalf("unexpected endpoint: %s", decoded.Endpoint)
	}
	if decoded.RawCount != 4 || decoded.Count != 3 {
		t.Fatalf("unexpected counts: raw=%d count=%d", decoded.RawCount, decoded.Count)
	}

	byID := map[string]int{}
	for i, event := range decoded.Events {
		byID[event.ID] = i
	}
	if _, ok := byID["urn:announcements:provincebz:closed-accident"]; ok {
		t.Fatalf("ended announcement should be hidden without --include-expired: %#v", decoded.Events)
	}

	roadwork := decoded.Events[byID["urn:announcements:provincebz:open-roadwork"]]
	if roadwork.Type != "roadworks" || roadwork.Subtype != "hindrance,road-work" {
		t.Fatalf("unexpected roadwork classification: %#v", roadwork)
	}
	if roadwork.Source != "content" || roadwork.MessageID != "2911001" {
		t.Fatalf("unexpected roadwork identity: %#v", roadwork)
	}
	if roadwork.Start != "2026-05-10T22:00:00Z" || roadwork.End != "" {
		t.Fatalf("open announcement must keep an empty end: %#v", roadwork)
	}
	if roadwork.PublishedAt != "2999-05-15T08:20:00.5968609+00:00" {
		t.Fatalf("unexpected published_at: %#v", roadwork)
	}
	if !strings.HasPrefix(roadwork.Place, "Bei Girlan") || !strings.HasPrefix(roadwork.PlaceIT, "Presso Cornaiano") {
		t.Fatalf("unexpected place text: %#v", roadwork)
	}
	if len(roadwork.Coordinates) != 2 || roadwork.Coordinates[0] != 11.25 || roadwork.Coordinates[1] != 46.42 {
		t.Fatalf("unexpected coordinates: %#v", roadwork.Coordinates)
	}
	if !roadwork.Active || roadwork.Stale {
		t.Fatalf("open recently changed announcement must be active and fresh: %#v", roadwork)
	}
	if roadwork.ZoneID != "" || roadwork.Road != "" || roadwork.Severity != "" {
		t.Fatalf("this source cannot populate zone_id, road, or severity: %#v", roadwork)
	}

	closure := decoded.Events[byID["urn:announcements:provincebz:running-closure"]]
	if closure.Type != "closure" || closure.End != "2999-08-30T00:00:00Z" || !closure.Active {
		t.Fatalf("unexpected closure: %#v", closure)
	}

	// The mountain-pass context tag wins over the caution kind tag, and an
	// untouched open record is reported stale without being hidden.
	pass := decoded.Events[byID["urn:announcements:provincebz:open-pass"]]
	if pass.Type != "mountain-pass" || pass.Subtype != "caution,mountain-pass" || !pass.Active || !pass.Stale {
		t.Fatalf("unexpected mountain pass: %#v", pass)
	}

	if !containsWarning(decoded.Warnings, "1 matching announcements had already ended when this query ran and were hidden") {
		t.Fatalf("expected hidden-ended warning, got %#v", decoded.Warnings)
	}
	if !containsWarning(decoded.Warnings, "1 returned announcements have not changed upstream for more than 30 days") {
		t.Fatalf("expected stale-meaning warning, got %#v", decoded.Warnings)
	}
	if !containsWarning(decoded.Warnings, "this source cannot populate the event fields zone_id, zone, zone_it") {
		t.Fatalf("expected unavailable-field warning, got %#v", decoded.Warnings)
	}
	if !containsWarning(decoded.Warnings, "source is the Open Data Hub Content API /v1/Announcement feed for PROVINCE_BZ") {
		t.Fatalf("expected source warning, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficEventsContentIncludeExpiredReturnsEndedAnnouncements(t *testing.T) {
	server := newAnnouncementTestServer(t, announcementFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "events",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--include-expired",
		"--json",
	})
	if decoded.Count != 4 {
		t.Fatalf("unexpected count %d: %#v", decoded.Count, decoded.Events)
	}
	for _, event := range decoded.Events {
		if event.ID != "urn:announcements:provincebz:closed-accident" {
			continue
		}
		if event.Type != "traffic" || event.Subtype != "accident,current" || event.Active {
			t.Fatalf("an ended announcement must not be reported active: %#v", event)
		}
	}
	if !containsWarning(decoded.Warnings, "they are included because --include-expired was passed") {
		t.Fatalf("expected include-expired warning, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficSearchContentFiltersByTypeAndText(t *testing.T) {
	server := newAnnouncementTestServer(t, announcementFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "search", "radroute",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--type", "closure",
		"--json",
	})
	if decoded.Count != 1 || decoded.Events[0].ID != "urn:announcements:provincebz:running-closure" {
		t.Fatalf("unexpected search result: %#v", decoded.Events)
	}

	empty := runContentTrafficCommand(t, server, []string{
		"traffic", "search", "radroute",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--type", "roadworks",
		"--json",
	})
	if empty.Count != 0 {
		t.Fatalf("expected no matches, got %#v", empty.Events)
	}
	if !containsWarning(empty.Warnings, `no open PROVINCE_BZ announcements matched search "radroute", type roadworks`) {
		t.Fatalf("expected no-match warning, got %#v", empty.Warnings)
	}
}

func TestRunTrafficTodayContentWarnsWhenLimitTruncates(t *testing.T) {
	server := newAnnouncementTestServer(t, strings.Replace(announcementFixture, `"TotalResults":4`, `"TotalResults":40`, 1))
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "today",
		"--source", "content",
		"--limit", "4",
		"--include-expired",
		"--json",
	})
	if !containsWarning(decoded.Warnings, "the Content API reports 40 announcements in this date range but --limit=4 fetched only 4") {
		t.Fatalf("expected truncation warning, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficContentWarnsThatIncludeStaleDoesNothing(t *testing.T) {
	server := newAnnouncementTestServer(t, announcementFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "events",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--include-stale",
		"--json",
	})
	if decoded.Count != 3 {
		t.Fatalf("--include-stale must not change the result set: %#v", decoded.Events)
	}
	if !containsWarning(decoded.Warnings, "--include-stale has no effect with --source content") {
		t.Fatalf("expected include-stale warning, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficContentRejectsFiltersThisSourceCannotAnswer(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "road",
			args: []string{"--road", "SP13"},
			want: "--road is not supported with --source content",
		},
		{
			name: "type bike",
			args: []string{"--type", "bike"},
			want: "--type bike is not supported with --source content",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unsupported filter must not reach upstream: %s", r.URL)
			}))
			defer server.Close()

			runner := newTestRunner(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}})
			var stdout, stderr bytes.Buffer
			args := append([]string{"traffic", "today", "--source", "content", "--json"}, testCase.args...)
			code := runner.Run(context.Background(), args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("Run exit = %d, want 2 (stdout %s, stderr %s)", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), testCase.want) {
				t.Fatalf("unexpected stderr: %s", stderr.String())
			}
		})
	}
}

// zoneInferenceFixture places four open announcements by coordinate only: one
// inside Bozen-Unterland, one inside Pustertal, one 70 km east of every
// reference point, and one with no Geo block at all.
const zoneInferenceFixture = `{"TotalResults":4,"TotalPages":1,"CurrentPage":1,"Items":[
	{
		"Id":"urn:announcements:provincebz:bozen",
		"Active":true,
		"Source":"PROVINCE_BZ",
		"Shortname":"General Traffic Hindrance Road Work",
		"TagIds":["announcement:traffic-event","traffic-event:hindrance","traffic-event:road-work"],
		"StartTime":"2026-05-10T22:00:00Z",
		"EndTime":null,
		"LastChange":"2999-05-15T08:20:00.5968609+00:00",
		"Geo":{"position":{"Default":true,"Geometry":"POINT (11.34 46.49)","Latitude":46.49,"Longitude":11.34}},
		"Detail":{"de":{"Title":"Behinderung Arbeiten","BaseText":"Bei Bozen Süd Arbeiten."},"it":{"Title":"Ostacolo lavori","BaseText":"Presso Bolzano sud lavori."}},
		"Mapping":{"ProviderProvinceBz":{"Id":"3001"}}
	},
	{
		"Id":"urn:announcements:provincebz:pustertal",
		"Active":true,
		"Source":"PROVINCE_BZ",
		"Shortname":"General Traffic Hindrance Closure or Blockage",
		"TagIds":["announcement:traffic-event","traffic-event:closure","traffic-event:hindrance"],
		"StartTime":"2026-05-10T22:00:00Z",
		"EndTime":null,
		"LastChange":"2999-05-15T08:20:00.5968609+00:00",
		"Geo":{"position":{"Default":true,"Geometry":"POINT (12.224 46.615)","Latitude":46.615,"Longitude":12.224}},
		"Detail":{"de":{"Title":"Behinderung Sperre","BaseText":"Im Pustertal gesperrt."},"it":{"Title":"Ostacolo chiusura","BaseText":"In Val Pusteria chiusa."}},
		"Mapping":{"ProviderProvinceBz":{"Id":"3002"}}
	},
	{
		"Id":"urn:announcements:provincebz:far-away",
		"Active":true,
		"Source":"PROVINCE_BZ",
		"Shortname":"General Traffic Hindrance Road Work",
		"TagIds":["announcement:traffic-event","traffic-event:hindrance","traffic-event:road-work"],
		"StartTime":"2026-05-10T22:00:00Z",
		"EndTime":null,
		"LastChange":"2999-05-15T08:20:00.5968609+00:00",
		"Geo":{"position":{"Default":true,"Geometry":"POINT (12.6 46.0)","Latitude":46.0,"Longitude":12.6}},
		"Detail":{"de":{"Title":"Behinderung Arbeiten","BaseText":"Weit weg von jedem Referenzpunkt."},"it":{"Title":"Ostacolo lavori","BaseText":"Lontano da ogni punto di riferimento."}},
		"Mapping":{"ProviderProvinceBz":{"Id":"3003"}}
	},
	{
		"Id":"urn:announcements:provincebz:no-geo",
		"Active":true,
		"Source":"PROVINCE_BZ",
		"Shortname":"General Traffic Hindrance Road Work",
		"TagIds":["announcement:traffic-event","traffic-event:hindrance","traffic-event:road-work"],
		"StartTime":"2026-05-10T22:00:00Z",
		"EndTime":null,
		"LastChange":"2999-05-15T08:20:00.5968609+00:00",
		"Detail":{"de":{"Title":"Behinderung Arbeiten","BaseText":"Ohne Koordinaten."},"it":{"Title":"Ostacolo lavori","BaseText":"Senza coordinate."}},
		"Mapping":{"ProviderProvinceBz":{"Id":"3004"}}
	}
]}`

func TestRunTrafficContentZoneIDFiltersByInferredZone(t *testing.T) {
	server := newAnnouncementTestServer(t, zoneInferenceFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "events",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--zone-id", "3",
		"--json",
	})
	if decoded.RawCount != 4 || decoded.Count != 1 {
		t.Fatalf("unexpected counts: raw=%d count=%d events=%#v", decoded.RawCount, decoded.Count, decoded.Events)
	}
	if decoded.Events[0].ID != "urn:announcements:provincebz:bozen" {
		t.Fatalf("unexpected match: %#v", decoded.Events)
	}
	// The zone decided the match; writing it onto the event would turn the
	// inference into a field the record does not carry.
	if decoded.Events[0].ZoneID != "" {
		t.Fatalf("an inferred zone must not be written into the event: %#v", decoded.Events[0])
	}
	if !containsWarning(decoded.Warnings, "--zone-id with --source content is geographic inference, not a field") {
		t.Fatalf("expected inference warning, got %#v", decoded.Warnings)
	}
	if !containsWarning(decoded.Warnings, "zone_id, zone and zone_it stay empty there") {
		t.Fatalf("inference warning must say no zone is emitted, got %#v", decoded.Warnings)
	}
	if !containsWarning(decoded.Warnings, "2 announcements passed every other filter but could not be placed in any zone") {
		t.Fatalf("expected unassignable count, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficContentAreaFiltersByInferredZone(t *testing.T) {
	server := newAnnouncementTestServer(t, zoneInferenceFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "today",
		"--source", "content",
		"--area", "pustertal",
		"--json",
	})
	if decoded.Count != 1 || decoded.Events[0].ID != "urn:announcements:provincebz:pustertal" {
		t.Fatalf("unexpected match: %#v", decoded.Events)
	}
	if !containsWarning(decoded.Warnings, "--area with --source content is geographic inference, not a field") {
		t.Fatalf("expected inference warning, got %#v", decoded.Warnings)
	}
	// pustertal is a zone-only alias, so it carries no place keywords to lose.
	for _, warning := range decoded.Warnings {
		if strings.Contains(warning, "narrows to zone") {
			t.Fatalf("a zone-only alias must not warn about dropped keywords: %q", warning)
		}
	}
}

func TestRunTrafficContentAreaWithKeywordsWarnsItCoversTheWholeZone(t *testing.T) {
	server := newAnnouncementTestServer(t, zoneInferenceFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "today",
		"--source", "content",
		"--area", "kaltern",
		"--json",
	})
	// Kaltern's own keywords cannot be applied here, so the Bozen announcement
	// matches on zone alone and the caller is told the result is zone-wide.
	if decoded.Count != 1 || decoded.Events[0].ID != "urn:announcements:provincebz:bozen" {
		t.Fatalf("unexpected match: %#v", decoded.Events)
	}
	if !containsWarning(decoded.Warnings, "area kaltern narrows to zone 3 (Bozen-Unterland) only with --source content") {
		t.Fatalf("expected keyword warning, got %#v", decoded.Warnings)
	}
}

// An announcement on the far side of the guard is excluded and counted, so a
// thin result is never read as "nothing there".
func TestRunTrafficContentCountsUnassignableAnnouncementsAtTheGuardBoundary(t *testing.T) {
	anchor := isolatedTrafficZonePoint(t)
	const kmPerDegreeLat = 111.194926644
	inside := anchor.Lat + (trafficZoneGuardKM-0.01)/kmPerDegreeLat
	outside := anchor.Lat + (trafficZoneGuardKM+0.01)/kmPerDegreeLat
	body := fmt.Sprintf(`{"TotalResults":2,"TotalPages":1,"CurrentPage":1,"Items":[
		{
			"Id":"urn:inside","Active":true,"Source":"PROVINCE_BZ",
			"TagIds":["announcement:traffic-event","traffic-event:hindrance","traffic-event:road-work"],
			"StartTime":"2026-05-10T22:00:00Z","EndTime":null,"LastChange":"2999-05-15T08:20:00Z",
			"Geo":{"position":{"Latitude":%.9f,"Longitude":%.9f}},
			"Detail":{"de":{"BaseText":"Innerhalb der Distanzgrenze."}}
		},
		{
			"Id":"urn:outside","Active":true,"Source":"PROVINCE_BZ",
			"TagIds":["announcement:traffic-event","traffic-event:hindrance","traffic-event:road-work"],
			"StartTime":"2026-05-10T22:00:00Z","EndTime":null,"LastChange":"2999-05-15T08:20:00Z",
			"Geo":{"position":{"Latitude":%.9f,"Longitude":%.9f}},
			"Detail":{"de":{"BaseText":"Ausserhalb der Distanzgrenze."}}
		}
	]}`, inside, anchor.Lon, outside, anchor.Lon)

	server := newAnnouncementTestServer(t, body)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "events",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--zone-id", anchor.ZoneID,
		"--json",
	})
	if decoded.Count != 1 || decoded.Events[0].ID != "urn:inside" {
		t.Fatalf("only the announcement inside the guard may match: %#v", decoded.Events)
	}
	if !containsWarning(decoded.Warnings, "1 announcement passed every other filter but could not be placed in any zone") {
		t.Fatalf("expected one unassignable announcement, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficContentZoneFilterWithNoMatchesNamesTheFilter(t *testing.T) {
	server := newAnnouncementTestServer(t, zoneInferenceFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "today",
		"--source", "content",
		"--area", "vinschgau",
		"--json",
	})
	if decoded.Count != 0 {
		t.Fatalf("expected no matches: %#v", decoded.Events)
	}
	if !containsWarning(decoded.Warnings, "no open PROVINCE_BZ announcements matched area vinschgau") {
		t.Fatalf("an empty result must name the area it was narrowed by, got %#v", decoded.Warnings)
	}
	if !containsWarning(decoded.Warnings, "2 announcements passed every other filter but could not be placed in any zone") {
		t.Fatalf("an empty result must still report unassignable announcements, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficContentWithoutZoneFilterDoesNotInferOrWarn(t *testing.T) {
	server := newAnnouncementTestServer(t, zoneInferenceFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "today",
		"--source", "content",
		"--json",
	})
	if decoded.Count != 4 {
		t.Fatalf("an unfiltered query must keep unassignable announcements: %#v", decoded.Events)
	}
	for _, warning := range decoded.Warnings {
		if strings.Contains(warning, "geographic inference") || strings.Contains(warning, "unassignable") {
			t.Fatalf("no zone filter was applied, so nothing was inferred: %q", warning)
		}
	}
}

func TestRunTrafficContentRejectsUnknownAreaAndZone(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "unknown area", args: []string{"--area", "nowhere"}, want: `unknown traffic area "nowhere"`},
		{name: "unknown zone", args: []string{"--zone-id", "99"}, want: `unknown traffic zone-id "99"`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("an invalid filter must not reach upstream: %s", r.URL)
			}))
			defer server.Close()

			runner := newTestRunner(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}})
			var stdout, stderr bytes.Buffer
			args := append([]string{"traffic", "today", "--source", "content", "--json"}, testCase.args...)
			if code := runner.Run(context.Background(), args, &stdout, &stderr); code == 0 {
				t.Fatalf("Run exit = 0, want non-zero (stdout %s)", stdout.String())
			}
			if !strings.Contains(stderr.String(), testCase.want) {
				t.Fatalf("unexpected stderr: %s", stderr.String())
			}
		})
	}
}

func TestRunTrafficContentAcceptsAreaAll(t *testing.T) {
	server := newAnnouncementTestServer(t, announcementFixture)
	defer server.Close()

	decoded := runContentTrafficCommand(t, server, []string{
		"traffic", "events",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--area", "all",
		"--json",
	})
	if decoded.Count != 3 {
		t.Fatalf("unexpected count %d", decoded.Count)
	}
}

func TestContentTrafficTypeMapsObservedTagPairs(t *testing.T) {
	cases := map[string]string{
		"hindrance,road-work":          "roadworks",
		"current,road-work":            "roadworks",
		"closure,hindrance":            "closure",
		"closure,current":              "closure",
		"current,event":                "event",
		"event,hindrance":              "event",
		"caution,mountain-pass":        "mountain-pass",
		"mountain-pass,road-condition": "mountain-pass",
		"event,mountain-pass":          "mountain-pass",
		"congestion,current":           "traffic",
		"accident,current":             "traffic",
		"caution,hindrance":            "traffic",
		"hindrance,road-condition":     "traffic",
		"public-transport,special":     "traffic",
		"hindrance,restriction":        "traffic",
		"":                             "traffic",
		"speed-camera":                 "radar",
		"restriction,speed-camera":     "radar",
		"maintenance":                  "roadworks",
	}
	for tags, want := range cases {
		var list []string
		if tags != "" {
			list = strings.Split(tags, ",")
		}
		if got := contentTrafficType(list); got != want {
			t.Fatalf("contentTrafficType(%q) = %q, want %q", tags, got, want)
		}
	}
}

func TestContentTrafficTagsDropTheGenericAnnouncementTag(t *testing.T) {
	tags := contentTrafficTags([]any{"announcement:traffic-event", "traffic-event:road-work", "traffic-event:hindrance", "", "other:thing"})
	if strings.Join(tags, ",") != "hindrance,road-work" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}
