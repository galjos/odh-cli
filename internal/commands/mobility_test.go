// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestFilterMobilityLatest(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	records := []map[string]any{
		{
			"scode":      "S1",
			"sname":      "Station 1",
			"sorigin":    "ORIGIN1",
			"sactive":    true,
			"mvalidtime": "2026-05-19T11:30:00", // Fresh
		},
		{
			"scode":      "S2",
			"sname":      "Station 2",
			"sorigin":    "ORIGIN1",
			"sactive":    false, // Inactive
			"mvalidtime": "2026-05-19T11:30:00",
		},
		{
			"scode":      "S3",
			"sname":      "Station 3",
			"sorigin":    "ORIGIN2", // Different origin
			"sactive":    true,
			"mvalidtime": "2026-05-19T11:30:00",
		},
		{
			"scode":      "S4",
			"sname":      "Station 4",
			"sorigin":    "ORIGIN1",
			"sactive":    true,
			"mvalidtime": "2026-05-18T11:30:00", // Stale (24h+ old)
		},
	}

	tests := []struct {
		name     string
		filter   mobilityLatestFilter
		expected int
	}{
		{
			name: "No filtering",
			filter: mobilityLatestFilter{
				Now:   now,
				Limit: 10,
			},
			expected: 4,
		},
		{
			name: "Filter by origin",
			filter: mobilityLatestFilter{
				Origin: "ORIGIN1",
				Now:    now,
				Limit:  10,
			},
			expected: 3,
		},
		{
			name: "Filter by active",
			filter: mobilityLatestFilter{
				ActiveOnly: true,
				Now:        now,
				Limit:      10,
			},
			expected: 3,
		},
		{
			name: "Filter by freshness (1h)",
			filter: mobilityLatestFilter{
				FreshDuration: 1 * time.Hour,
				Now:           now,
				Limit:         10,
			},
			expected: 3,
		},
		{
			name: "Combined filter",
			filter: mobilityLatestFilter{
				Origin:        "ORIGIN1",
				ActiveOnly:    true,
				FreshDuration: 1 * time.Hour,
				Now:           now,
				Limit:         10,
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterMobilityLatest(records, tt.filter)
			if len(result.Measurements) != tt.expected {
				t.Errorf("expected %d measurements, got %d", tt.expected, len(result.Measurements))
			}
		})
	}
}

func TestSummarizeDatatypes(t *testing.T) {
	records := []map[string]any{
		{"tname": "temp", "tdescription": "Temperature", "tunit": "C", "scode": "S1", "sorigin": "O1"},
		{"tname": "temp", "tdescription": "Temperature", "tunit": "C", "scode": "S2", "sorigin": "O1"},
		{"tname": "hum", "tdescription": "Humidity", "tunit": "%", "scode": "S1", "sorigin": "O1"},
		{"tname": "temp", "tdescription": "Temperature", "tunit": "C", "scode": "S3", "sorigin": "O2"},
	}

	t.Run("No origin filter", func(t *testing.T) {
		summary := summarizeDatatypes(records, "")
		if len(summary) != 2 {
			t.Errorf("expected 2 datatypes, got %d", len(summary))
		}
		for _, s := range summary {
			if s.Name == "temp" && s.StationCount != 3 {
				t.Errorf("expected 3 stations for temp, got %d", s.StationCount)
			}
		}
	})

	t.Run("Filter by origin", func(t *testing.T) {
		summary := summarizeDatatypes(records, "O1")
		if len(summary) != 2 {
			t.Errorf("expected 2 datatypes, got %d", len(summary))
		}
		for _, s := range summary {
			if s.Name == "temp" && s.StationCount != 2 {
				t.Errorf("expected 2 stations for temp under O1, got %d", s.StationCount)
			}
		}
	})
}

type mockTransport struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

func TestRunnerWithMock(t *testing.T) {
	mockData := map[string]any{
		"data": []any{
			map[string]any{"tname": "mock-type", "scode": "M1", "sorigin": "MOCK"},
		},
	}
	encoded, _ := json.Marshal(mockData)

	transport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(string(encoded))),
				Header:     make(http.Header),
			}, nil
		},
	}

	runner := NewDefaultRunner()
	runner.Client = runner.Client.WithTransport(transport)

	var stdout, stderr strings.Builder
	args := []string{"mobility", "datatypes", "--station-type", "Any"}
	code := runner.Run(context.Background(), args, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
		t.Fatalf("failed to parse stdout as JSON: %v", err)
	}

	count, ok := result["count"].(float64)
	if !ok || count != 1 {
		t.Errorf("expected count 1 in JSON output, got %v", result["count"])
	}
}
