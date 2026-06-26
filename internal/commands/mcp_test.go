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
)

func TestMCPRequiresSubcommand(t *testing.T) {
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mcp"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage: odh mcp <subcommand>") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestMCPToolsListsToolSurface(t *testing.T) {
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mcp", "tools"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}
	var payload struct {
		Count int `json:"count"`
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"input_schema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if payload.Count == 0 || payload.Count != len(payload.Tools) {
		t.Fatalf("count = %d with %d tools", payload.Count, len(payload.Tools))
	}
	names := map[string]bool{}
	for _, tool := range payload.Tools {
		names[tool.Name] = true
		if tool.Description == "" {
			t.Fatalf("tool %q has no description", tool.Name)
		}
		if tool.InputSchema["type"] != "object" {
			t.Fatalf("tool %q schema type is not object", tool.Name)
		}
	}
	for _, name := range []string{"doctor", "datasets_guide", "traffic_search", "mobility_latest", "transit_journey", "diagnostics_ev_charging"} {
		if !names[name] {
			t.Fatalf("expected tool %q in listing", name)
		}
	}
	if names["mcp"] || names["completion"] {
		t.Fatal("mcp and completion must not be exposed as tools")
	}
}
