// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDatasetJSONShortcutOverridesTable(t *testing.T) {
	for _, args := range [][]string{
		{"datasets", "list"},
		{"datasets", "search", "parking"},
		{"datasets", "guide", "parking"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := NewDefaultRunner().Run(context.Background(), append(args, "--format", "table", "--json"), &stdout, &stderr)
			if code != 0 || !json.Valid(stdout.Bytes()) {
				t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestBikeFeedWarningPrecedesResults(t *testing.T) {
	day := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	events, warnings := normalizeTrafficEvents([]map[string]any{{
		"evuuid":            "old-bike-notice",
		"evstart":           "2019-04-05T00:00:00Z",
		"evtransactiontime": "2025-09-15T08:00:00Z",
		"evmetadata": map[string]any{
			"subTycodeValue": "RADWEG_SPERRE",
			"placeDe":        "Radroute Gadertal",
		},
	}}, trafficQuery{Type: "bike", IncludeStale: true, Limit: 1000}, trafficArea{}, day, day)
	if len(events) != 1 || len(warnings) == 0 {
		t.Fatalf("expected the historical notice with warnings: %+v, %v", events, warnings)
	}
	if !strings.Contains(warnings[0], "2025-09-15") ||
		!strings.Contains(warnings[0], "not a live bulletin") ||
		!strings.Contains(warnings[0], "odh traffic search radroute --today --source content --json") {
		t.Fatalf("first warning must explain age and the current cycle search: %v", warnings)
	}
	for _, format := range []string{"table", "markdown"} {
		var stdout bytes.Buffer
		if err := writeTrafficOutput(&stdout, trafficResult{
			Source: "odh", Events: events, Warnings: warnings, OutputFormat: format,
		}); err != nil {
			t.Fatal(err)
		}
		text := stdout.String()
		if strings.Index(text, warnings[0]) > strings.Index(text, "Radroute Gadertal") ||
			strings.Count(text, warnings[0]) != 1 {
			t.Fatalf("%s must show the feed warning once, before the rows: %s", format, text)
		}
	}
}

func TestMobilityCoverageSeparatesPageMatchesAndResults(t *testing.T) {
	rows := []map[string]any{{"sactive": true}, {"sactive": false}, {"sactive": true}}
	result := filterMobilityLatest(rows, mobilityLatestFilter{ActiveOnly: true, Limit: 1, RequestLimit: 3})
	coverage := result.Coverage
	if coverage.FetchedCount != 3 || coverage.MatchedCount != 2 || coverage.ReturnedCount != 1 ||
		!coverage.ResultTruncated || !coverage.UpstreamMayHaveMore || coverage.UpstreamTotal != nil {
		t.Fatalf("unexpected coverage: %+v", coverage)
	}
}
