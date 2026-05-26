// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

func TestRunNoArgsReturnsUsageError(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("expected usage on stderr, got %s", stderr.String())
	}
}

func TestRunParentCommandRequiresSubcommand(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "usage: odh mobility <subcommand>") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunHelpDoesNotExposeCompletionCommand(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "completion") {
		t.Fatalf("completion command leaked into help: %s", stdout.String())
	}
}

func TestRunCompletionGeneratesShellScript(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"completion", "zsh"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "#compdef odh") {
		t.Fatalf("unexpected completion output: %s", stdout.String())
	}
}

func TestRunCompletionRejectsUnsupportedShell(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"completion", "tcsh"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run exit = %d, stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid argument") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
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

func TestRunDatasetsSearchFindsGTFS(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"datasets", "search", "train delay"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id": "gtfs.transit"`) ||
		!strings.Contains(stdout.String(), "odh transit departures") {
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

func TestRunGlobalTimeoutCancelsHTTPCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(time.Second):
			_, _ = w.Write([]byte(`{"too":"late"}`))
		}
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "test", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"--timeout", "20ms", "call", "test", "/slow"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run exit = %d, want 1, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %s", stderr.String())
	}
}

func TestRunGlobalTimeoutRejectsNegativeDuration(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"--timeout", "-1s", "apis"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--timeout must not be negative") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunGlobalTimeoutCoexistsWithDoctorTimeout(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"--timeout", "1s", "doctor", "--network=false"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok": true`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunCallPreservesCommaValuesInParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ODHActivityPoi" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("fields"); got != "Detail.en.Title,GpsInfo" {
			t.Fatalf("unexpected fields %q from raw query %q", got, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"Items":[{"Id":"one"}]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"call", "tourism", "/v1/ODHActivityPoi", "--param", "fields=Detail.en.Title,GpsInfo", "--param", "pagesize=1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"Items"`) {
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

func TestRunGTFSDatasetsListsRealtimeFeeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"sta-time-tables":{
				"description":"Public transportation data provided by STA",
				"origin":"sta.bz.it",
				"license":"CC0",
				"endpoint":"https://example.test/raw.zip",
				"metadata":{"modes":["bus","train"]},
				"realtime":{"trip_updates":"https://example.test/trip-updates"}
			}
		}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"gtfs", "datasets"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id": "sta-time-tables"`) ||
		!strings.Contains(stdout.String(), `"trip_updates"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunGTFSRealtimeFiltersTripUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/realtime/sta-time-tables/trip-updates" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"header":{"timestamp":1778924144},
			"entity":[
				{"id":"one","trip_update":{"trip":{"trip_id":"match","route_id":"REG"},"stop_time_update":[{"arrival":{"delay":60}}]}},
				{"id":"two","trip_update":{"trip":{"trip_id":"other","route_id":"REG"},"stop_time_update":[{"arrival":{"delay":0}}]}}
			]
		}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"gtfs", "realtime", "--dataset", "sta-time-tables", "--feed", "trip-updates", "--trip-id", "match"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		EntityCount int `json:"entity_count"`
		Count       int `json:"count"`
		Entities    []struct {
			ID string `json:"id"`
		} `json:"entities"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.EntityCount != 1 || decoded.Count != 1 || decoded.Entities[0].ID != "one" {
		t.Fatalf("unexpected decoded output: %#v", decoded)
	}
}

func TestRunTransitStopsSearchUsesAuerAlias(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"transit", "stops", "search", "auer", "--cache-dir", t.TempDir(), "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name": "Ora, Stazione di Ora"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "loading GTFS archive") {
		t.Fatalf("expected cold-cache GTFS progress on stderr, got %s", stderr.String())
	}
}

func TestRunTransitStopsSearchFallsBackToStaleGTFSCacheOnRefreshTimeout(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "sta-time-tables.zip")
	if err := os.WriteFile(cachePath, gtfsZip, 0o644); err != nil {
		t.Fatalf("write stale cache: %v", err)
	}
	staleTime := time.Now().Add(-2 * defaultGTFSCacheTTL)
	if err := os.Chtimes(cachePath, staleTime, staleTime); err != nil {
		t.Fatalf("mark stale cache: %v", err)
	}

	originalTimeout := gtfsDownloadTimeout
	gtfsDownloadTimeout = 25 * time.Millisecond
	t.Cleanup(func() { gtfsDownloadTimeout = originalTimeout })

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		<-r.Context().Done()
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"transit", "stops", "search", "auer", "--cache-dir", cacheDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("expected one GTFS archive download attempt, got %d", got)
	}

	var decoded transitStopsSearchOutput
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if !decoded.Archive.Cached || !decoded.Archive.Stale {
		t.Fatalf("expected stale cached archive metadata, got %#v", decoded.Archive)
	}
	if !strings.Contains(decoded.Archive.Warning, "stale cached GTFS archive") {
		t.Fatalf("expected archive warning, got %#v", decoded.Archive)
	}
	if !strings.Contains(stderr.String(), "warning: using stale cached GTFS archive") {
		t.Fatalf("expected stale-cache warning on stderr, got %s", stderr.String())
	}
	if decoded.Count == 0 || !strings.Contains(stdout.String(), `"name": "Ora, Stazione di Ora"`) {
		t.Fatalf("expected stops from stale cache, got %#v", decoded.Stops)
	}

	stdout.Reset()
	stderr.Reset()
	code = runner.Run(context.Background(), []string{"transit", "stops", "search", "auer", "--cache-dir", cacheDir, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second Run exit = %d, stderr = %s", code, stderr.String())
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("expected recent refresh failure marker to suppress another download attempt, got %d attempts", got)
	}
	if !strings.Contains(stderr.String(), "refresh failed recently") {
		t.Fatalf("expected recent refresh failure warning on stderr, got %s", stderr.String())
	}
}

func TestRunTransitDeparturesParsesStaticGTFS(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "departures",
		"--stop", "auer",
		"--date", "2026-05-16",
		"--around", "14:05",
		"--window", "5m",
		"--mode", "train",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"route_short_name": "REG"`) ||
		!strings.Contains(stdout.String(), `"departure_time": "14:05:00"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTransitDeparturesSupportsStopID(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "departures",
		"--stop-id", "ora-station",
		"--date", "2026-05-16",
		"--around", "14:05",
		"--window", "5m",
		"--mode", "train",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"stop_match_mode": "stop-id"`) ||
		!strings.Contains(stdout.String(), `"stop_id": "ora-station"`) ||
		!strings.Contains(stdout.String(), `"departure_time": "14:05:00"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "matched 50 stops") {
		t.Fatalf("stop-id mode emitted ambiguity warning: %s", stdout.String())
	}
}

func TestRunTransitDeparturesSupportsParentStationID(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "departures",
		"--stop-id", "ora-parent",
		"--date", "2026-05-16",
		"--around", "14:05",
		"--window", "5m",
		"--mode", "train",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"stop_match_mode": "parent-station"`) ||
		!strings.Contains(stdout.String(), `"stop_id": "ora-parent"`) ||
		!strings.Contains(stdout.String(), `"departure_time": "14:05:00"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTransitDeparturesRejectsStopAndStopID(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "departures",
		"--stop", "auer",
		"--stop-id", "ora-station",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "use either --stop or --stop-id") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunTransitTripFindsDirectGTFSMatch(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "trip",
		"--from", "auer",
		"--to", "brenner",
		"--date", "2026-05-16",
		"--time", "14:05",
		"--window", "5m",
		"--mode", "train",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"count": 1`) ||
		!strings.Contains(stdout.String(), `"stop_name": "Brennero"`) ||
		!strings.Contains(stdout.String(), "historical delay probability is not available") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTransitTripDefaultsToCompactTable(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "trip",
		"--from", "auer",
		"--to", "brenner",
		"--date", "2026-05-16",
		"--time", "14:05",
		"--window", "5m",
		"--mode", "train",
		"--cache-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ROUTE") ||
		!strings.Contains(stdout.String(), "trip-reg-1") ||
		strings.Contains(stdout.String(), `"from_stops"`) {
		t.Fatalf("expected compact table output, got: %s", stdout.String())
	}
}

func TestRunTransitTripSupportsStopIDs(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "trip",
		"--from-stop-id", "ora-station",
		"--to-stop-id", "brenner",
		"--date", "2026-05-16",
		"--time", "14:05",
		"--window", "5m",
		"--mode", "train",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"from_match_mode": "stop-id"`) ||
		!strings.Contains(stdout.String(), `"to_match_mode": "stop-id"`) ||
		!strings.Contains(stdout.String(), `"from_stop_id": "ora-station"`) ||
		!strings.Contains(stdout.String(), `"to_stop_id": "brenner"`) ||
		!strings.Contains(stdout.String(), `"count": 1`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTransitTripSupportsParentStationIDs(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "trip",
		"--from-stop-id", "ora-parent",
		"--to-stop-id", "brenner-parent",
		"--date", "2026-05-16",
		"--time", "14:05",
		"--window", "5m",
		"--mode", "train",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"from_match_mode": "parent-station"`) ||
		!strings.Contains(stdout.String(), `"to_match_mode": "parent-station"`) ||
		!strings.Contains(stdout.String(), `"from_stop_id": "ora-parent"`) ||
		!strings.Contains(stdout.String(), `"to_stop_id": "brenner-parent"`) ||
		!strings.Contains(stdout.String(), `"count": 1`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTransitJourneyFindsTransferItinerary(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "journey",
		"--from", "auer",
		"--to", "truden",
		"--date", "2026-05-16",
		"--time", "14:00",
		"--max-transfers", "2",
		"--min-transfer", "3m",
		"--max-duration", "2h",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Count    int `json:"count"`
		Journeys []struct {
			ArrivalTime   string `json:"arrival_time"`
			TransferCount int    `json:"transfer_count"`
			Legs          []struct {
				RouteShortName string `json:"route_short_name"`
				From           struct {
					StopName      string `json:"stop_name"`
					DepartureTime string `json:"departure_time"`
				} `json:"from"`
				To struct {
					StopName    string `json:"stop_name"`
					ArrivalTime string `json:"arrival_time"`
				} `json:"to"`
			} `json:"legs"`
		} `json:"journeys"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.Count != 1 || len(decoded.Journeys) != 1 {
		t.Fatalf("expected one journey, got: %s", stdout.String())
	}
	journey := decoded.Journeys[0]
	if journey.TransferCount != 1 || journey.ArrivalTime != "15:10:00" || len(journey.Legs) != 2 {
		t.Fatalf("unexpected journey summary: %#v", journey)
	}
	if journey.Legs[0].RouteShortName != "REG" || journey.Legs[0].From.StopName != "Ora, Stazione di Ora" || journey.Legs[0].To.StopName != "Bolzano" {
		t.Fatalf("unexpected first leg: %#v", journey.Legs[0])
	}
	if journey.Legs[1].RouteShortName != "120" || journey.Legs[1].From.StopName != "Bolzano" || journey.Legs[1].To.StopName != "Truden" {
		t.Fatalf("unexpected second leg: %#v", journey.Legs[1])
	}
	if !containsWarning(decoded.Warnings, "static GTFS timetable") {
		t.Fatalf("missing static GTFS warning: %#v", decoded.Warnings)
	}
}

func TestRunTransitJourneyAnnotatesRealtime(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	timestamp := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/dataset/sta-time-tables/raw":
			_, _ = w.Write(gtfsZip)
		case "/v1/realtime/sta-time-tables/trip-updates":
			fmt.Fprintf(w, `{
				"header":{"timestamp":%d},
				"entity":[
					{"id":"rt-reg","trip_update":{"trip":{"trip_id":"trip-reg-2","route_id":"route-reg"},"stop_time_update":[{"stop_id":"bozen","stop_sequence":2,"arrival":{"delay":300},"departure":{"delay":300}}]}},
					{"id":"rt-bus","trip_update":{"trip":{"trip_id":"trip-bus-2","route_id":"route-bus"},"stop_time_update":[{"stop_id":"bozen","stop_sequence":1,"departure":{"delay":120}},{"stop_id":"truden","stop_sequence":2,"arrival":{"delay":120}}]}}
				]
			}`, timestamp)
		case "/v1/realtime/sta-time-tables/service-alerts":
			fmt.Fprintf(w, `{
				"header":{"timestamp":%d},
				"entity":[
					{"id":"alert-bus","alert":{"cause":"CONSTRUCTION","effect":"SIGNIFICANT_DELAYS","informed_entity":[{"route_id":"route-bus"}],"header_text":{"translation":[{"language":"en","text":"Road works near Truden"}]}}}
				]
			}`, timestamp)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "journey",
		"--from", "auer",
		"--to", "truden",
		"--date", "2026-05-16",
		"--time", "14:00",
		"--max-transfers", "2",
		"--min-transfer", "3m",
		"--max-duration", "2h",
		"--with-realtime",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		WithRealtime bool `json:"with_realtime"`
		Realtime     struct {
			TripUpdateEntityCount   int   `json:"trip_update_entity_count"`
			ServiceAlertEntityCount int   `json:"service_alert_entity_count"`
			MatchedTripUpdateCount  int   `json:"matched_trip_update_count"`
			MatchedAlertCount       int   `json:"matched_alert_count"`
			FeedTimestampUnix       int64 `json:"feed_timestamp_unix"`
		} `json:"realtime"`
		Journeys []struct {
			RealtimeTransfers []struct {
				Status                 string `json:"status"`
				ScheduledBufferSeconds int    `json:"scheduled_buffer_seconds"`
				AdjustedBufferSeconds  *int   `json:"adjusted_buffer_seconds"`
			} `json:"realtime_transfers"`
			Legs []struct {
				Realtime struct {
					Status                string `json:"status"`
					DelaySeconds          *int   `json:"delay_seconds"`
					AdjustedDepartureTime string `json:"adjusted_departure_time"`
					AdjustedArrivalTime   string `json:"adjusted_arrival_time"`
					Alerts                []struct {
						ID     string `json:"id"`
						Header string `json:"header"`
					} `json:"alerts"`
				} `json:"realtime"`
			} `json:"legs"`
		} `json:"journeys"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if !decoded.WithRealtime || decoded.Realtime.TripUpdateEntityCount != 2 || decoded.Realtime.ServiceAlertEntityCount != 1 ||
		decoded.Realtime.MatchedTripUpdateCount != 2 || decoded.Realtime.MatchedAlertCount != 1 || decoded.Realtime.FeedTimestampUnix != timestamp {
		t.Fatalf("unexpected realtime summary: %#v\n%s", decoded.Realtime, stdout.String())
	}
	if len(decoded.Journeys) != 1 || len(decoded.Journeys[0].Legs) != 2 {
		t.Fatalf("expected two-leg journey, got: %s", stdout.String())
	}
	firstLegRT := decoded.Journeys[0].Legs[0].Realtime
	if firstLegRT.Status != "updated" || firstLegRT.DelaySeconds == nil || *firstLegRT.DelaySeconds != 300 ||
		firstLegRT.AdjustedArrivalTime != "14:30:00" {
		t.Fatalf("unexpected first leg realtime: %#v", firstLegRT)
	}
	secondLegRT := decoded.Journeys[0].Legs[1].Realtime
	if secondLegRT.Status != "updated" || secondLegRT.DelaySeconds == nil || *secondLegRT.DelaySeconds != 120 ||
		secondLegRT.AdjustedDepartureTime != "14:37:00" || len(secondLegRT.Alerts) != 1 || secondLegRT.Alerts[0].ID != "alert-bus" {
		t.Fatalf("unexpected second leg realtime: %#v", secondLegRT)
	}
	if got := decoded.Journeys[0].RealtimeTransfers[0]; got.Status != "ok" || got.ScheduledBufferSeconds != 600 ||
		got.AdjustedBufferSeconds == nil || *got.AdjustedBufferSeconds != 420 {
		t.Fatalf("unexpected transfer realtime: %#v", got)
	}
	if !containsWarning(decoded.Warnings, "GTFS-RT annotations") {
		t.Fatalf("missing GTFS-RT caveat warning: %#v", decoded.Warnings)
	}
}

func TestRunTransitJourneyWarnsWhenRealtimeDoesNotMatch(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	timestamp := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/dataset/sta-time-tables/raw":
			_, _ = w.Write(gtfsZip)
		case "/v1/realtime/sta-time-tables/trip-updates":
			fmt.Fprintf(w, `{"header":{"timestamp":%d},"entity":[{"id":"other","trip_update":{"trip":{"trip_id":"not-the-returned-trip"},"stop_time_update":[{"arrival":{"delay":60}}]}}]}`, timestamp)
		case "/v1/realtime/sta-time-tables/service-alerts":
			fmt.Fprintf(w, `{"header":{"timestamp":%d},"entity":[]}`, timestamp)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "journey",
		"--from", "auer",
		"--to", "truden",
		"--date", "2026-05-16",
		"--time", "14:00",
		"--max-transfers", "2",
		"--max-duration", "2h",
		"--with-realtime",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Realtime struct {
			TripUpdateEntityCount  int `json:"trip_update_entity_count"`
			MatchedTripUpdateCount int `json:"matched_trip_update_count"`
		} `json:"realtime"`
		Journeys []struct {
			Legs []struct {
				Realtime struct {
					Status string `json:"status"`
				} `json:"realtime"`
			} `json:"legs"`
		} `json:"journeys"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.Realtime.TripUpdateEntityCount != 1 || decoded.Realtime.MatchedTripUpdateCount != 0 {
		t.Fatalf("unexpected realtime counts: %#v", decoded.Realtime)
	}
	if decoded.Journeys[0].Legs[0].Realtime.Status != "no-update" {
		t.Fatalf("expected no-update leg status, got: %s", stdout.String())
	}
	if !containsWarning(decoded.Warnings, "none matched the returned journey trip IDs") {
		t.Fatalf("missing unmatched realtime warning: %#v", decoded.Warnings)
	}
}

func TestRunTransitJourneyPrefersExactDestinationOverStreetName(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "journey",
		"--from", "auer",
		"--to", "brenner",
		"--date", "2026-05-16",
		"--time", "14:00",
		"--max-transfers", "0",
		"--max-duration", "2h",
		"--cache-dir", t.TempDir(),
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Count    int `json:"count"`
		Journeys []struct {
			Legs []struct {
				To struct {
					StopName string `json:"stop_name"`
				} `json:"to"`
			} `json:"legs"`
		} `json:"journeys"`
		ToStops []gtfsStop `json:"to_stops"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.Count != 1 || len(decoded.Journeys) != 1 || len(decoded.Journeys[0].Legs) != 1 {
		t.Fatalf("expected one direct journey, got: %s", stdout.String())
	}
	if got := decoded.Journeys[0].Legs[0].To.StopName; got != "Brennero" {
		t.Fatalf("expected exact Brennero destination, got %q in %s", got, stdout.String())
	}
	for _, stop := range decoded.ToStops {
		if stop.Name == "Bolzano, Via Brennero" {
			t.Fatalf("street-name match leaked into exact destination selector: %s", stdout.String())
		}
	}
}

func TestRunTransitJourneyDefaultsToCompactTable(t *testing.T) {
	gtfsZip := buildTestGTFSZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dataset/sta-time-tables/raw" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write(gtfsZip)
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "gtfs", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"transit", "journey",
		"--from", "auer",
		"--to", "truden",
		"--date", "2026-05-16",
		"--time", "14:00",
		"--max-transfers", "2",
		"--max-duration", "2h",
		"--cache-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "JOURNEY") ||
		!strings.Contains(stdout.String(), "summary") ||
		!strings.Contains(stdout.String(), "Truden") ||
		strings.Contains(stdout.String(), `"journeys"`) {
		t.Fatalf("expected compact journey table, got: %s", stdout.String())
	}
}

func TestRunTransitDelayStatsReportsUnsupported(t *testing.T) {
	runner := newTestRunner(t, nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"transit", "delay-stats", "--from", "auer", "--to", "brenner", "--time", "14:05", "--weekday", "saturday", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"supported": false`) ||
		!strings.Contains(stdout.String(), "no historical GTFS-RT archive") {
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
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"dataset": "event"`) || !strings.Contains(stdout.String(), `"count": 2`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunTourismTypesDefaultsToCompactTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/EventShortTypes" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"Items":[{"Id":"music","Key":"Music","Type":"Type","TypeDesc":{"en":"Music","de":"Musik","it":"Musica"}}]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"tourism", "types", "--dataset", "event", "--limit", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ID") ||
		!strings.Contains(stdout.String(), "music") ||
		strings.Contains(stdout.String(), `"items"`) {
		t.Fatalf("expected compact table output, got: %s", stdout.String())
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

func TestRunMobilityLatestFiltersAndSorts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,node/EChargingStation/number-available/latest" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("unexpected limit %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"alperia-old","sorigin":"ALPERIA","sname":"Old station","sactive":true,"mvalidtime":"2025-01-01 08:00:00.000+0000"},
			{"scode":"alperia-inactive","sorigin":"ALPERIA","sname":"Inactive station","sactive":false,"mvalidtime":"2026-05-18 10:00:00.000+0000"},
			{"scode":"other-fresh","sorigin":"OTHER","sname":"Other station","sactive":true,"mvalidtime":"2026-05-18 11:00:00.000+0000"},
			{"scode":"alperia-fresh-a","sorigin":"ALPERIA","sname":"Fresh A","sactive":true,"mvalidtime":"2026-05-18 09:00:00.000+0000"},
			{"scode":"alperia-fresh-b","sorigin":"ALPERIA","sname":"Fresh B","sactive":true,"mvalidtime":"2026-05-18 12:00:00.000+0000"}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"mobility", "latest",
		"--station-type", "EChargingStation",
		"--data-type", "number-available",
		"--origin", "alperia",
		"--active",
		"--sort", "newest",
		"--request-limit", "5",
		"--limit", "2",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}

	var got mobilityLatestResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if got.StationType != "EChargingStation" || got.DataType != "number-available" || got.Origin != "alperia" {
		t.Fatalf("unexpected summary: %+v", got)
	}
	if got.RawCount != 5 || got.Count != 2 || len(got.Measurements) != 2 {
		t.Fatalf("unexpected counts: raw=%d count=%d len=%d", got.RawCount, got.Count, len(got.Measurements))
	}
	if asString(got.Measurements[0]["scode"]) != "alperia-fresh-b" || asString(got.Measurements[1]["scode"]) != "alperia-fresh-a" {
		t.Fatalf("unexpected sorted measurements: %+v", got.Measurements)
	}
	if !strings.Contains(strings.Join(got.Warnings, "\n"), "inactive rows were hidden") {
		t.Fatalf("missing active warning: %+v", got.Warnings)
	}
	if !strings.Contains(strings.Join(got.Warnings, "\n"), "rows were hidden by --origin") {
		t.Fatalf("missing origin warning: %+v", got.Warnings)
	}
}

func TestFilterMobilityLatestFiltersFreshWithin(t *testing.T) {
	records := []map[string]any{
		{
			"scode":      "fresh",
			"sorigin":    "ALPERIA",
			"sactive":    true,
			"mvalidtime": "2026-05-18 09:00:00.000+0000",
		},
		{
			"scode":      "stale",
			"sorigin":    "ALPERIA",
			"sactive":    true,
			"mvalidtime": "2026-05-16 08:00:00.000+0000",
		},
		{
			"scode":      "invalid-time",
			"sorigin":    "ALPERIA",
			"sactive":    true,
			"mvalidtime": "not-a-time",
		},
	}

	got := filterMobilityLatest(records, mobilityLatestFilter{
		StationType:   "EChargingStation",
		DataType:      "number-available",
		FreshWithin:   "24h",
		FreshDuration: 24 * time.Hour,
		Sort:          "newest",
		Limit:         10,
		RequestLimit:  100,
		Endpoint:      "https://example.test",
		Now:           time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC),
	})

	if got.RawCount != 3 || got.Count != 1 || len(got.Measurements) != 1 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if asString(got.Measurements[0]["scode"]) != "fresh" {
		t.Fatalf("unexpected measurement: %+v", got.Measurements[0])
	}
	if !strings.Contains(strings.Join(got.Warnings, "\n"), "2 stale rows were hidden by --fresh-within") {
		t.Fatalf("missing stale warning: %+v", got.Warnings)
	}
}

func TestRunMobilityLatestHintsCommonParkingDatatypeMistake(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,node/ParkingStation/number-free/latest" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"mobility", "latest",
		"--station-type", "ParkingStation",
		"--data-type", "number-free",
		"--origin", "Municipality Merano",
		"--format", "table",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `data type "free"`) {
		t.Fatalf("expected datatype hint, got: %s", stdout.String())
	}
}

func TestRunDiagnosticsEVChargingWarnsWhenNoFreshRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,node/EChargingStation/number-available/latest" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"old","sorigin":"ALPERIA","sactive":true,"mvalidtime":"2017-01-01 08:00:00.000+0000"},
			{"scode":"inactive","sorigin":"ALPERIA","sactive":false,"mvalidtime":"2999-05-18 09:00:00.000+0000"}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"diagnostics", "ev-charging",
		"--origin", "ALPERIA",
		"--fresh-within", "24h",
		"--request-limit", "2",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"verdict": "unavailable"`) ||
		!strings.Contains(stdout.String(), "no fresh active EV availability rows found") ||
		!strings.Contains(stdout.String(), `"current_count": 0`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunDiagnosticsParkingForecastsKeepsStaleForecastUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/flat,node/ParkingStation/free/latest":
			_, _ = w.Write([]byte(`{"data":[
				{"scode":"park","sorigin":"Municipality Merano","sactive":true,"mvalidtime":"2999-05-18 09:30:00.000+0000","mvalue":42}
			]}`))
		case "/v2/flat,node/ParkingStation/parking-forecast-60/latest":
			_, _ = w.Write([]byte(`{"data":[
				{"scode":"park","sorigin":"Municipality Merano","sactive":true,"mvalidtime":"2017-05-17 09:30:00.000+0000","mvalue":40}
			]}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"diagnostics", "parking-forecasts",
		"--origin", "Municipality Merano",
		"--fresh-within", "48h",
		"--forecast-minutes", "60",
		"--request-limit", "2",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"verdict": "current_only"`) ||
		!strings.Contains(stdout.String(), "no fresh parking forecast rows found") ||
		!strings.Contains(stdout.String(), `"count": 1`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunDiagnosticsTourismEventsReportsOnlyActiveCaveats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/EventShort" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("onlyactive") != "true" || query.Get("pagesize") != "1" {
			t.Fatalf("unexpected query %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"Items":[
			{"Id":"event-one","Active":true,"ActiveToday":false,"StartDate":"2018-01-22T09:00:00","EndDate":"2018-01-25T11:00:00","GpsInfo":null,"EventTitle":{"en":"Workshop"}}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"diagnostics", "tourism-events",
		"--date", "2026-05-18",
		"--limit", "1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"verdict": "unavailable"`) ||
		!strings.Contains(stdout.String(), `"date_status": "expired"`) ||
		!strings.Contains(stdout.String(), "ActiveToday=false") ||
		!strings.Contains(stdout.String(), "missing GpsInfo") {
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
	code := runner.Run(context.Background(), []string{"mobility", "types", "--kind", "event", "--json"}, &stdout, &stderr)
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
	code := runner.Run(context.Background(), []string{"mobility", "datatypes", "--station-type", "TrafficSensor", "--origin", "A22", "--json"}, &stdout, &stderr)
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

func TestRunMobilityDatatypesDefaultsToCompactTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat/TrafficSensor/*" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"A22:1","sorigin":"A22","tname":"Average Flow","tdescription":"Flow","tunit":"vehicles / hour"}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "datatypes", "--station-type", "TrafficSensor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "NAME") ||
		!strings.Contains(stdout.String(), "Average Flow") ||
		strings.Contains(stdout.String(), `"datatypes"`) {
		t.Fatalf("expected compact table output, got: %s", stdout.String())
	}
}

func TestRunMobilityDatatypesWarnsWhenSmallLimitMayHideValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat/ParkingStation/*" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Fatalf("unexpected limit %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"park-1","sorigin":"Municipality Merano","tname":"free","tdescription":"Free parking spaces","tunit":"spaces"}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "datatypes", "--station-type", "ParkingStation", "--limit", "1", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "rerun with --limit 1000") {
		t.Fatalf("expected limit warning, got: %s", stdout.String())
	}
}

func TestRunMobilityOriginsSummarizesStationOrigins(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat/TrafficSensor" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("unexpected limit %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"A22:1","sorigin":"A22","sname":"A22 one"},
			{"scode":"A22:2","sorigin":"A22","sname":"A22 two"},
			{"scode":"SIAG:1","sorigin":"SIAG","sname":"SIAG one"},
			{"scode":"missing","sname":"missing origin"}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "origins", "--station-type", "TrafficSensor", "--limit", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		StationType string `json:"station_type"`
		RecordCount int    `json:"record_count"`
		Count       int    `json:"count"`
		Origins     []struct {
			Name           string   `json:"name"`
			StationCount   int      `json:"station_count"`
			StationSamples []string `json:"station_samples"`
		} `json:"origins"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.StationType != "TrafficSensor" || decoded.RecordCount != 4 || decoded.Count != 2 || len(decoded.Origins) != 2 {
		t.Fatalf("unexpected decoded output: %#v", decoded)
	}
	if decoded.Origins[0].Name != "A22" || decoded.Origins[0].StationCount != 2 || len(decoded.Origins[0].StationSamples) != 2 {
		t.Fatalf("unexpected A22 origin summary: %#v", decoded.Origins[0])
	}
	if decoded.Origins[1].Name != "SIAG" || decoded.Origins[1].StationCount != 1 {
		t.Fatalf("unexpected SIAG origin summary: %#v", decoded.Origins[1])
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

func TestRunTrafficEventsUeberetschUnterlandExcludesBozenCity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/2026-05-16/2026-05-16" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{
				"evuuid":"bozen-city",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageStreetInternetDescDe":"Stadtgemeinde Bozen",
					"subTycodeValue":"SPERRE",
					"placeDe":"Die Straße Untervirgl ist wegen Bauarbeiten gesperrt."
				}
			},
			{
				"evuuid":"ueberetsch",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"messageStreetNr":"LS/SP 13",
					"messageStreetInternetDescDe":"St. Pauls-Unterrain",
					"subTycodeValue":"BAUSTELLE",
					"placeDe":"zwischen St. Pauls und Unterrain"
				}
			}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"traffic", "events",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--area", "ueberetsch-unterland",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Count  int `json:"count"`
		Events []struct {
			ID string `json:"id"`
		} `json:"events"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.Count != 1 || len(decoded.Events) != 1 || decoded.Events[0].ID != "ueberetsch" {
		t.Fatalf("unexpected decoded output: %#v", decoded)
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

func TestRunTrafficZonesOutputsKnownZoneIDs(t *testing.T) {
	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: "https://example.com", Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"traffic", "zones", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Zones []struct {
			ZoneID string `json:"zone_id"`
			Name   string `json:"name"`
		} `json:"zones"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(decoded.Zones) != 7 {
		t.Fatalf("unexpected zones: %#v", decoded.Zones)
	}
	if decoded.Zones[5].ZoneID != "6" || decoded.Zones[5].Name != "Pustertal" {
		t.Fatalf("expected Pustertal zone 6, got %#v", decoded.Zones[5])
	}
}

func TestRunTrafficCategoriesOutputsKnownFilters(t *testing.T) {
	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: "https://example.com", Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"traffic", "categories", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Categories []struct {
			Name             string   `json:"name"`
			Aliases          []string `json:"aliases"`
			UpstreamSubtypes []string `json:"upstream_subtypes"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(decoded.Categories) != 7 {
		t.Fatalf("unexpected categories: %#v", decoded.Categories)
	}
	if decoded.Categories[0].Name != "roadworks" || !containsString(decoded.Categories[0].UpstreamSubtypes, "BAUSTELLE") {
		t.Fatalf("expected roadworks category first, got %#v", decoded.Categories[0])
	}
	if decoded.Categories[1].Name != "closure" || !containsString(decoded.Categories[1].Aliases, "gesperrt") {
		t.Fatalf("expected closure category, got %#v", decoded.Categories[1])
	}
}

func TestRunTrafficEventsSupportsZoneIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/2026-05-16/2026-05-16" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{
				"evuuid":"zone-6",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"6",
					"messageZoneDescDe":"Pustertal",
					"messageStreetNr":"LS/SP 244",
					"subTycodeValue":"SPERRE",
					"placeDe":"Sperre bei Zwischenwasser"
				}
			},
			{
				"evuuid":"zone-3",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"subTycodeValue":"SPERRE",
					"placeDe":"Sperre bei Bozen"
				}
			}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"traffic", "events",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--zone-id", "6",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		ZoneID string `json:"zone_id"`
		Count  int    `json:"count"`
		Events []struct {
			ID     string `json:"id"`
			ZoneID string `json:"zone_id"`
		} `json:"events"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.ZoneID != "6" || decoded.Count != 1 || len(decoded.Events) != 1 || decoded.Events[0].ID != "zone-6" || decoded.Events[0].ZoneID != "6" {
		t.Fatalf("unexpected decoded output: %#v", decoded)
	}
}

func TestRunTrafficSearchMatchesTextAndGenericClosureTerms(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/2026-05-16/2026-05-16" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{
				"evuuid":"closed-road",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"6",
					"messageZoneDescDe":"Pustertal",
					"messageStreetNr":"LS/SP 244",
					"messageStreetInternetDescDe":"Gadertal",
					"subTycodeValue":"SPERRE",
					"placeDe":"Zwischenwasser: Straße aus Sicherheitsgründen gesperrt."
				}
			},
			{
				"evuuid":"other-road",
				"evstart":"2026-05-16 00:00:00.000+0200",
				"evend":"2026-05-16 23:59:00.000+0200",
				"evmetadata":{
					"messageZoneId":"3",
					"messageZoneDescDe":"Bozen-Unterland",
					"subTycodeValue":"BAUSTELLE",
					"placeDe":"Bauarbeiten in Bozen"
				}
			}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"traffic", "search", "road", "closed", "zwischenwasser",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Search string `json:"search"`
		Count  int    `json:"count"`
		Events []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"events"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.Search != "road closed zwischenwasser" || decoded.Count != 1 || len(decoded.Events) != 1 || decoded.Events[0].ID != "closed-road" || decoded.Events[0].Type != "closure" {
		t.Fatalf("unexpected decoded output: %#v", decoded)
	}
}

func TestRunTrafficSearchWarnsWhenOnlyStaleOpenEndedRowsMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/2026-05-16/2026-05-16" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{
				"evuuid":"stale-badia",
				"evstart":"2025-01-01 00:00:00.000+0100",
				"evtransactiontime":"2025-01-01 08:00:00.000+0100",
				"evmetadata":{
					"messageZoneId":"6",
					"messageZoneDescDe":"Pustertal",
					"subTycodeValue":"SPERRE",
					"placeDe":"Sperre bei Badia"
				}
			}
		]}`))
	}))
	defer server.Close()

	runner := newTestRunner(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}})
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{
		"traffic", "search", "badia",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Count    int      `json:"count"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if decoded.Count != 0 {
		t.Fatalf("expected no current results, got %#v", decoded)
	}
	if !containsWarning(decoded.Warnings, "stale open-ended matching events were hidden") ||
		!containsWarning(decoded.Warnings, `no current ODH PROVINCE_BZ traffic events matched search "badia"`) {
		t.Fatalf("expected stale/no-match warnings, got %#v", decoded.Warnings)
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

func TestRunA22StatusJSONRawIncludesUpstreamRows(t *testing.T) {
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
	code := runner.Run(context.Background(), []string{"a22", "status", "--json", "--raw"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Forecast struct {
			Items []map[string]any `json:"items"`
		} `json:"forecast"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(decoded.Forecast.Items) != 1 || asString(decoded.Forecast.Items[0]["sname"]) != "Bolzano Nord" {
		t.Fatalf("expected raw forecast item, got: %#v", decoded.Forecast.Items)
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

func buildTestGTFSZip(t *testing.T) []byte {
	t.Helper()
	files := map[string]string{
		"stops.txt": strings.Join([]string{
			"stop_id,stop_name,stop_lat,stop_lon,zone_id,location_type,parent_station",
			"ora-parent,\"Ora\",46.36075232,11.29752794,,1,",
			"ora-station,\"Ora, Stazione di Ora\",46.36075232,11.29752794,,,ora-parent",
			"brenner-parent,\"Brennero\",47.00268073,11.50557389,,1,",
			"brenner,\"Brennero\",47.00268073,11.50557389,,,brenner-parent",
			"bozen,\"Bolzano\",46.498,11.354,,,",
			"bolzano-via-brennero,\"Bolzano, Via Brennero\",46.501,11.354,,,",
			"truden,\"Truden\",46.322,11.349,,,",
			"",
		}, "\n"),
		"routes.txt": strings.Join([]string{
			"route_id,agency_id,route_short_name,route_long_name,route_type",
			"route-reg,sta,REG,,2",
			"route-bus,sta,120,,3",
			"",
		}, "\n"),
		"trips.txt": strings.Join([]string{
			"route_id,service_id,trip_id,trip_headsign,direction_id",
			"route-reg,weekday-sat,trip-reg-1,Brennero,1",
			"route-reg,weekday-sat,trip-reg-2,Bolzano,1",
			"route-bus,weekday-sat,trip-bus-1,Ora,0",
			"route-bus,weekday-sat,trip-bus-2,Truden,0",
			"route-bus,weekday-sat,trip-bus-3,Via Brennero,0",
			"",
		}, "\n"),
		"calendar.txt": strings.Join([]string{
			"service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date",
			"weekday-sat,0,0,0,0,0,1,0,20260101,20261231",
			"",
		}, "\n"),
		"calendar_dates.txt": "service_id,date,exception_type\n",
		"stop_times.txt": strings.Join([]string{
			"trip_id,arrival_time,departure_time,stop_id,stop_sequence",
			"trip-reg-1,14:04:00,14:05:00,ora-station,1",
			"trip-reg-1,15:39:00,15:40:00,brenner,2",
			"trip-reg-2,14:04:00,14:05:00,ora-station,1",
			"trip-reg-2,14:25:00,14:26:00,bozen,2",
			"trip-bus-1,14:05:00,14:05:00,ora-station,1",
			"trip-bus-2,14:35:00,14:35:00,bozen,1",
			"trip-bus-2,15:10:00,15:10:00,truden,2",
			"trip-bus-3,14:06:00,14:06:00,ora-station,1",
			"trip-bus-3,14:40:00,14:40:00,bolzano-via-brennero,2",
			"",
		}, "\n"),
	}
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	for name, content := range files {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buffer.Bytes()
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
