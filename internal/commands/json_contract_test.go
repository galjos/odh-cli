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
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/galjos/odh-cli/internal/apis"
)

func TestJSONContractTrafficSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/2026-05-16/2026-05-16" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{
			"evuuid":"badia-closure",
			"evseriesuuid":"series-badia",
			"evstart":"2026-05-16 08:00:00.000+0200",
			"evend":"2026-05-16 18:00:00.000+0200",
			"evtransactiontime":"2999-05-16 07:45:00.000+0200",
			"evmetadata":{
				"messageId":"message-badia",
				"messageZoneId":"6",
				"messageZoneDescDe":"Pustertal",
				"messageZoneDescIt":"Val Pusteria",
				"messageStreetNr":"LS/SP 244",
				"messageStreetInternetDescDe":"Gadertal",
				"subTycodeValue":"SPERRE",
				"messageGradDescDe":"Sperre",
				"placeDe":"Badia",
				"placeIt":"Badia"
			},
			"evlgeometry":{"coordinates":[11.903,46.61]}
		}]}`))
	}))
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}}, []string{
		"traffic", "search", "road closed badia",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--zone-id", "6",
		"--json",
	})
	assertGoldenJSON(t, "traffic-search.json", stdout, server.URL)
}

func TestJSONContractTrafficContentSearch(t *testing.T) {
	server := newAnnouncementTestServer(t, announcementFixture)
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}}, []string{
		"traffic", "search", "radroute",
		"--source", "content",
		"--from", "2026-05-16",
		"--to", "2026-05-16",
		"--json",
	})
	assertGoldenJSON(t, "traffic-content-search.json", normalizeAnnouncementRange(stdout), server.URL)
}

// normalizeAnnouncementRange pins the begin/end query parameters, which the
// local timezone shifts, so the golden endpoint stays comparable.
var announcementRangeParams = regexp.MustCompile(`(begin|end)=[^&"]*`)

func normalizeAnnouncementRange(actual []byte) []byte {
	return announcementRangeParams.ReplaceAll(actual, []byte("$1=RANGE"))
}

func TestJSONContractMobilityOrigins(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat/TrafficSensor" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"A22:1","sorigin":"A22","sname":"A22 one"},
			{"scode":"A22:2","sorigin":"A22","sname":"A22 two"}
		]}`))
	}))
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}}, []string{
		"mobility", "origins",
		"--station-type", "TrafficSensor",
		"--limit", "2",
		"--json",
	})
	assertGoldenJSON(t, "mobility-origins.json", stdout, server.URL)
}

func TestJSONContractMobilityStations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat/ParkingStation" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"scode":"park-1","sorigin":"skidata","sname":"Park One"},
			{"scode":"park-2","sorigin":"other","sname":"Park Two"}
		]}`))
	}))
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}}, []string{
		"mobility", "stations",
		"--station-type", "ParkingStation",
		"--origin", "skidata",
		"--limit", "2",
		"--json",
	})
	assertGoldenJSON(t, "mobility-stations.json", stdout, server.URL)
}

func TestJSONContractMobilityEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,event/PROVINCE_BZ/latest" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{
			"evuuid":"badia-closure",
			"evcategory":"SPERRE",
			"evtransactiontime":"2024-03-28 09:00:00.000+0100"
		}]}`))
	}))
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}}, []string{
		"mobility", "events",
		"--origin", "PROVINCE_BZ",
		"--latest",
		"--limit", "1",
	})
	assertGoldenJSON(t, "mobility-events.json", stdout, server.URL)
}

func TestJSONContractMobilityLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,node/ParkingStation/free/latest" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{
			"scode":"park-1",
			"sname":"Park One",
			"sorigin":"Municipality Merano",
			"sactive":true,
			"mvalue":42,
			"mvalidtime":"2999-05-16 08:00:00.000+0000"
		}]}`))
	}))
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}}, []string{
		"mobility", "latest",
		"--station-type", "ParkingStation",
		"--data-type", "free",
		"--origin", "Municipality Merano",
		"--active",
		"--fresh-within", "999999h",
		"--sort", "newest",
		"--request-limit", "5",
		"--limit", "1",
		"--json",
	})
	assertGoldenJSON(t, "mobility-latest.json", stdout, server.URL)
}

func TestJSONContractDiagnosticsEVCharging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/flat,node/EChargingStation/number-available/latest" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{
			"scode":"charger-1",
			"sorigin":"ALPERIA",
			"sactive":true,
			"mvalue":3,
			"mvalidtime":"2999-05-16 08:00:00.000+0000"
		}]}`))
	}))
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}}, []string{
		"diagnostics", "ev-charging",
		"--origin", "ALPERIA",
		"--fresh-within", "999999h",
		"--request-limit", "5",
		"--limit", "1",
	})
	assertGoldenJSON(t, "diagnostics-ev-charging.json", stdout, server.URL)
}

func TestJSONContractA22Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/flat,event/A22/latest":
			_, _ = w.Write([]byte(`{"data":[]}`))
		case "/v2/flat/TrafficForecast/forecast/latest":
			_, _ = w.Write([]byte(`{"data":[{
				"scode":"forecast-1",
				"sname":"Bolzano Nord",
				"mvalue":"regular",
				"mvalidtime":"2999-01-01 00:00:00.000+0000"
			}]}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "mobility", BaseURL: server.URL, Public: true}}, []string{
		"a22", "status",
		"--json",
		"--raw",
		"--limit", "1",
	})
	assertGoldenJSON(t, "a22-status.json", stdout, server.URL)
}

func TestJSONContractTourismTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/EventShortTypes" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"Items":[{
			"Id":"type-1",
			"Key":"music",
			"Type":"event",
			"TypeDesc":{"en":"Music","de":"Musik","it":"Musica"}
		}]}`))
	}))
	defer server.Close()

	stdout := runContractCommand(t, []apis.API{{Name: "tourism", BaseURL: server.URL, Public: true}}, []string{
		"tourism", "types",
		"--dataset", "event",
		"--limit", "1",
		"--json",
	})
	assertGoldenJSON(t, "tourism-types.json", stdout, server.URL)
}

func runContractCommand(t *testing.T, entries []apis.API, args []string) []byte {
	t.Helper()
	runner := newTestRunner(t, entries)
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	return stdout.Bytes()
}

func assertGoldenJSON(t *testing.T, name string, actual []byte, serverURL string) {
	t.Helper()
	expectedPath := filepath.Join("testdata", "golden", name)
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", expectedPath, err)
	}
	actual = bytes.ReplaceAll(actual, []byte(serverURL), []byte("https://example.test"))

	var expectedValue any
	if err := json.Unmarshal(expected, &expectedValue); err != nil {
		t.Fatalf("invalid golden JSON %s: %v", expectedPath, err)
	}
	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("invalid actual JSON: %v\n%s", err, string(actual))
	}
	if !reflect.DeepEqual(expectedValue, actualValue) {
		t.Fatalf("JSON contract mismatch for %s\nwant:\n%s\ngot:\n%s", name, strings.TrimSpace(string(expected)), strings.TrimSpace(string(actual)))
	}
}
