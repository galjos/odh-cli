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
