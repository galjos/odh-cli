// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/galjos/odh-cli/internal/apis"
	"github.com/galjos/odh-cli/internal/client"
)

func TestBuildURLWithParams(t *testing.T) {
	values := url.Values{}
	values.Add("seed", "42")
	values.Add("fields", "Detail.en.Title,GpsInfo")

	got, err := BuildURL("https://example.com/base", "/v1/ODHActivityPoi", values)
	if err != nil {
		t.Fatalf("BuildURL returned error: %v", err)
	}
	want := "https://example.com/base/v1/ODHActivityPoi?fields=Detail.en.Title%2CGpsInfo&seed=42"
	if got != want {
		t.Fatalf("BuildURL = %q, want %q", got, want)
	}
}

func TestRunAPIsOutputsJSON(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"apis"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if _, ok := decoded["apis"]; !ok {
		t.Fatalf("missing apis key: %s", stdout.String())
	}
}

func TestRunDatasetsSearchFindsParking(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"datasets", "search", "parking"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id": "mobility.parking"`) ||
		!strings.Contains(stdout.String(), "odh mobility stations --station-type ParkingStation") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunDatasetsSearchFindsTrafficEvents(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"datasets", "search", "roadworks"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id": "mobility.traffic-events"`) ||
		!strings.Contains(stdout.String(), "odh traffic today --area ueberetsch-unterland") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunDatasetsListSupportsDomainAndTable(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"datasets", "list", "--domain", "tourism", "--format", "table"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "tourism.poi") || strings.Contains(stdout.String(), "mobility.parking") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunVersionOutputsJSON(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded["version"] == "" || decoded["goos"] == "" || decoded["goarch"] == "" {
		t.Fatalf("missing version fields: %s", stdout.String())
	}
}

func TestRunDoctorWithoutNetwork(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"doctor", "--network=false"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok": true`) || !strings.Contains(stdout.String(), `"api_registry"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunCallUsesRegistryAndParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/items" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("unexpected limit %q", got)
		}
		_, _ = w.Write([]byte(`{"items":[1,2]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "test", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"call", "test", "/items", "--param", "limit=2"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"items"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunDoctorChecksNetworkTargets(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = true
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{
		{Name: "tourism", BaseURL: server.URL, Public: true},
		{Name: "mobility", BaseURL: server.URL, Public: true},
	})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	for _, path := range []string{"/swagger/v1/swagger.json", "/v2/apispec", "/v2/flat", "/v2/flat,event/A22/latest"} {
		if !seen[path] {
			t.Fatalf("doctor did not request %s; seen %#v", path, seen)
		}
	}
	if !strings.Contains(stdout.String(), `"ok": true`) || !strings.Contains(stdout.String(), `"a22_events"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunDoctorReturnsFailureForNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/apispec" {
			http.Error(w, "broken", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{
		{Name: "tourism", BaseURL: server.URL, Public: true},
		{Name: "mobility", BaseURL: server.URL, Public: true},
	})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"doctor"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run exit = %d, want 1; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok": false`) || !strings.Contains(stdout.String(), `"status_code": 502`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunOpenAPIConvertsYAMLToJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte("openapi: 3.0.1\ninfo:\n  title: Test\n"))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "test", BaseURL: server.URL, OpenAPIURL: server.URL + "/apispec", Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"openapi", "test"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"openapi": "3.0.1"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTourismPOIBuildsExpectedQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/v1/ODHActivityPoi" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if query.Get("pagesize") != "1" || query.Get("seed") != "42" || query.Get("fields") != "Detail.en.Title,GpsInfo" {
			t.Fatalf("unexpected query %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"Items":[{"Id":"one"}]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"tourism", "poi", "--limit", "1", "--seed", "42", "--fields", "Detail.en.Title,GpsInfo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"Items"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTourismTypesBuildsExpectedQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/v1/EventShortTypes" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if query.Get("pagesize") != "2" || query.Get("pagenumber") != "3" || query.Get("seed") != "42" {
			t.Fatalf("unexpected query %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"Items":[{"Id":"sport"},{"Id":"music"}]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"tourism", "types",
		"--dataset", "event",
		"--limit", "2",
		"--page", "3",
		"--seed", "42",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"dataset": "event"`) || !strings.Contains(stdout.String(), `"count": 2`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunMobilityLatestRequiresFlags(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "latest"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--station-type is required") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunMobilityLatestSupportsWhere(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,node/EChargingStation/number-available/latest" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("where"); got != "sactive.eq.true" {
			t.Fatalf("unexpected where %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"mobility", "latest",
		"--station-type", "EChargingStation",
		"--data-type", "number-available",
		"--where", "sactive.eq.true",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"data"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunMobilityTypesBuildsEventPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"id":"A22"},{"id":"PROVINCE_BZ"}]`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "types", "--kind", "event"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"kind": "event"`) || !strings.Contains(stdout.String(), `"count": 2`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunMobilityDatatypesSummarizesByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat/TrafficSensor/*" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"A22:1","sorigin":"A22","tname":"Average Flow","tdescription":"Flow","tunit":"vehicles / hour"},
			{"scode":"A22:2","sorigin":"A22","tname":"Average Flow","tdescription":"Flow","tunit":"vehicles / hour"},
			{"scode":"OTHER:1","sorigin":"OTHER","tname":"Average Flow","tdescription":"Flow","tunit":"vehicles / hour"},
			{"scode":"A22:1","sorigin":"A22","tname":"Average Density","tdescription":"Density","tunit":"vehicles / km"}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "datatypes", "--station-type", "TrafficSensor", "--origin", "A22"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name": "Average Flow"`) || !strings.Contains(stdout.String(), `"station_count": 2`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "OTHER") {
		t.Fatalf("origin filter leaked OTHER origin: %s", stdout.String())
	}
}

func TestRunMobilityStationsFiltersByOrigin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat/ParkingStation" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "3" {
			t.Fatalf("unexpected limit %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"1","sorigin":"skidata","sname":"P1"},
			{"scode":"2","sorigin":"other","sname":"P2"},
			{"scode":"3","sorigin":"skidata","sname":"P3"}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "stations", "--station-type", "ParkingStation", "--origin", "skidata", "--limit", "3"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"count": 2`) || strings.Contains(stdout.String(), `"sname": "P2"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunMobilityEventsWrapsEmptyA22Events(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/A22/latest" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "events", "--origin", "A22"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"origin": "A22"`) || !strings.Contains(stdout.String(), `"count": 0`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTrafficEventsFiltersAreaTypeRoadAndDedupes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/2026-05-16/2026-05-16" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("unexpected limit %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[
			{
				"evuuid":"sp13-1",
				"evseriesuuid":"series-sp13",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evtransactiontime":"2026-05-16 08:00:00.000+0200",
				"evmetadata":{
					"messageId":"m-sp13-1",
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageZoneDescIt":"Bolzano-Bassa Atesina",
					"messageStreetNr":"LS/SP 13",
					"messageStreetInternetDescDe":"St. Pauls-Unterrain",
					"subTycodeValue":"BAUSTELLE",
					"messageGradDescDe":"Behinderung",
					"placeDe":"zwischen St. Pauls und Unterrain",
					"placeIt":"tra San Paolo e Riva di Sotto"
				},
				"evlgeometry":{"coordinates":[11.25,46.42]}
			},
			{
				"evuuid":"sp13-duplicate",
				"evseriesuuid":"series-sp13-duplicate",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evtransactiontime":"2026-05-16 08:01:00.000+0200",
				"evmetadata":{
					"messageId":"m-sp13-2",
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageStreetNr":"LS/SP 13",
					"messageStreetInternetDescDe":"St. Pauls-Unterrain",
					"subTycodeValue":"BAUSTELLE",
					"placeDe":"zwischen St. Pauls und Unterrain"
				},
				"evlgeometry":{"coordinates":[11.25,46.42]}
			},
			{
				"evuuid":"sp16",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageStreetNr":"LS/SP 16",
					"messageStreetInternetDescDe":"Autobahn-Tramin",
					"subTycodeValue":"SPERRE",
					"placeDe":"bei Tramin"
				}
			},
			{
				"evuuid":"zone4",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"4",
					"messageZoneDescDe":"Salten-Schlern",
					"messageStreetNr":"LS/SP 13",
					"subTycodeValue":"BAUSTELLE",
					"placeDe":"anderer Bezirk"
				}
			}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"traffic", "events",
		"--source", "odh",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--area", "ueberetsch-unterland",
		"--type", "roadworks",
		"--road", "SP13",
		"--limit", "5",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Count    int `json:"count"`
		RawCount int `json:"raw_count"`
		Events   []struct {
			Type   string `json:"type"`
			Road   string `json:"road"`
			Place  string `json:"place"`
			Active bool   `json:"active"`
		} `json:"events"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.RawCount != 4 || decoded.Count != 1 || len(decoded.Events) != 1 {
		t.Fatalf("unexpected decoded output: %#v", decoded)
	}
	if got := decoded.Events[0]; got.Type != "roadworks" || got.Road != "LS/SP 13" || !got.Active || !strings.Contains(got.Place, "St. Pauls") {
		t.Fatalf("unexpected event: %#v", got)
	}
	if !containsWarning(decoded.Warnings, "deduplicated 2 raw matching rows to 1 events") {
		t.Fatalf("expected dedupe warning, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficEventsHidesStaleOpenEndedRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/2026-05-16/2026-05-16" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{
				"evuuid":"stale-open-ended",
				"evstart":"2025-01-01 00:00:00.000+0100",
				"evtransactiontime":"2025-01-01 08:00:00.000+0100",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageStreetNr":"LS/SP 13",
					"subTycodeValue":"BAUSTELLE",
					"placeDe":"old open-ended event"
				}
			},
			{
				"evuuid":"long-running",
				"evstart":"2025-10-13 00:00:00.000+0200",
				"evend":"2026-05-30 23:59:00.000+0200",
				"evtransactiontime":"2025-10-13 08:00:00.000+0200",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageStreetNr":"LS/SP 13",
					"subTycodeValue":"BAUSTELLE",
					"placeDe":"long-running event with an explicit end"
				}
			}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"traffic", "events",
		"--source", "odh",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--area", "bozen-unterland",
		"--type", "roadworks",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Count  int `json:"count"`
		Events []struct {
			ID    string `json:"id"`
			Stale bool   `json:"stale"`
		} `json:"events"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.Count != 1 || len(decoded.Events) != 1 || decoded.Events[0].ID != "long-running" || !decoded.Events[0].Stale {
		t.Fatalf("unexpected decoded output: %#v", decoded)
	}
	if !containsWarning(decoded.Warnings, "1 stale open-ended matching events were hidden") {
		t.Fatalf("expected hidden stale warning, got %#v", decoded.Warnings)
	}
}

func TestRunTrafficEventsSupportsNearFilterAndTableOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/2026-05-16/2026-05-16" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{
				"evuuid":"near",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageStreetNr":"LS/SP 13",
					"messageStreetInternetDescDe":"St. Pauls-Unterrain",
					"subTycodeValue":"BAUSTELLE",
					"placeDe":"bei St. Pauls"
				},
				"evlgeometry":{"coordinates":[11.25,46.42]}
			},
			{
				"evuuid":"far",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageStreetNr":"LS/SP 13",
					"messageStreetInternetDescDe":"St. Pauls-Unterrain",
					"subTycodeValue":"BAUSTELLE",
					"placeDe":"far away"
				},
				"evlgeometry":{"coordinates":[12.00,47.00]}
			}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"traffic", "events",
		"--source", "odh",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--near", "46.42,11.25",
		"--radius", "2km",
		"--format", "table",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "TYPE") ||
		!strings.Contains(stdout.String(), "ROAD") ||
		!strings.Contains(stdout.String(), "roadworks") ||
		!strings.Contains(stdout.String(), "LS/SP 13") ||
		strings.Contains(stdout.String(), "far away") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTrafficEventsRejectsUnknownSource(t *testing.T) {
	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: "https://example.com", Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"traffic", "events", "--source", "unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unsupported traffic source") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunTrafficEventsRejectsUnknownFormat(t *testing.T) {
	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: "https://example.com", Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"traffic", "events", "--format", "csv"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unsupported format") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunA22StatusWarnsOnEmptyEventsAndFutureForecast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/flat,event/A22/latest":
			_, _ = w.Write([]byte(`{"data":[]}`))
		case "/v2/flat/TrafficForecast/forecast/latest":
			_, _ = w.Write([]byte(`{"data":[{"sname":"Bolzano Nord","mvalue":"regular","mvalidtime":"2999-01-01 00:00:00.000+0000"}]}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"a22", "status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Open Data Hub returned no current A22 events") ||
		!strings.Contains(stdout.String(), "future valid_time") {
		t.Fatalf("expected warnings, got: %s", stdout.String())
	}
}

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}

func newTestRunner(t *testing.T, entries []apis.API) *Runner {
	t.Helper()
	if entries == nil {
		entries = []apis.API{{Name: "tourism", BaseURL: "https://example.com", Public: true}}
	}
	registry, err := apis.NewRegistry(entries)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}
	return &Runner{
		Registry: registry,
		Client:   client.NewWithHTTPClient(http.DefaultClient, "odh-cli-test"),
	}
}
