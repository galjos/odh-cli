// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolSpecsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range toolSpecs {
		if spec.name == "" || spec.desc == "" || len(spec.base) == 0 {
			t.Fatalf("tool %q needs name, description, and base command", spec.name)
		}
		if seen[spec.name] {
			t.Fatalf("duplicate tool name %q", spec.name)
		}
		seen[spec.name] = true
		paramNames := map[string]bool{}
		for _, p := range spec.params {
			if p.name == "" || p.desc == "" {
				t.Fatalf("tool %q has a parameter without name or description", spec.name)
			}
			if paramNames[p.name] {
				t.Fatalf("tool %q has duplicate parameter %q", spec.name, p.name)
			}
			paramNames[p.name] = true
			if p.flag == "" && p.typ != paramString {
				t.Fatalf("tool %q positional parameter %q must be a string", spec.name, p.name)
			}
			if p.flag == "" && !p.required {
				t.Fatalf("tool %q positional parameter %q must be required", spec.name, p.name)
			}
		}
		if _, err := json.Marshal(spec.inputSchema()); err != nil {
			t.Fatalf("tool %q schema does not marshal: %v", spec.name, err)
		}
	}
}

func TestBuildArgs(t *testing.T) {
	specByName := map[string]toolSpec{}
	for _, spec := range toolSpecs {
		specByName[spec.name] = spec
	}

	tests := []struct {
		tool      string
		arguments string
		want      []string
		wantErr   string
	}{
		{
			tool:      "version",
			arguments: "",
			want:      []string{"version"},
		},
		{
			tool:      "datasets_guide",
			arguments: `{"query":"ev charging availability","domain":"mobility","limit":1}`,
			want: []string{
				"datasets", "guide", "ev charging availability",
				"--domain", "mobility", "--limit", "1",
			},
		},
		{
			tool:      "traffic_search",
			arguments: `{"query":"road closed badia","today":true,"zone_id":"6","include_stale":false,"limit":500}`,
			want: []string{
				"traffic", "search", "road closed badia", "--json",
				"--today", "--zone-id", "6", "--limit", "500",
			},
		},
		{
			tool:      "mobility_latest",
			arguments: `{"station_type":"EChargingStation","data_type":"number-available","active":true,"fresh_within":"24h","sort":"newest","request_limit":1000}`,
			want: []string{
				"mobility", "latest",
				"--station-type", "EChargingStation",
				"--data-type", "number-available",
				"--active", "--fresh-within", "24h",
				"--sort", "newest", "--request-limit", "1000",
			},
		},
		{
			tool:      "call_api",
			arguments: `{"api":"tourism","path":"/v1/Event","params":{"pagesize":"1","fields":"Id,Detail.en.Title"}}`,
			want: []string{
				"call", "tourism", "/v1/Event",
				"--param", "fields=Id,Detail.en.Title",
				"--param", "pagesize=1",
			},
		},
		{
			tool:      "transit_journey",
			arguments: `{"from_stop_id":"Parentit:22021:301","to_stop_id":"it:22021:730:0:1150","date":"2026-06-10","time":"16:40","with_realtime":true}`,
			want: []string{
				"transit", "journey", "--json",
				"--from-stop-id", "Parentit:22021:301",
				"--to-stop-id", "it:22021:730:0:1150",
				"--date", "2026-06-10", "--time", "16:40",
				"--with-realtime",
			},
		},
		{
			tool:      "mobility_latest",
			arguments: `{"data_type":"number-available"}`,
			wantErr:   `argument "station_type" is required`,
		},
		{
			tool:      "traffic_search",
			arguments: `{"query":"badia","limit":"many"}`,
			wantErr:   `argument "limit" must be an integer`,
		},
		{
			tool:      "traffic_search",
			arguments: `{"query":"badia","bogus":true}`,
			wantErr:   `unknown argument "bogus"`,
		},
	}

	for _, test := range tests {
		spec, ok := specByName[test.tool]
		if !ok {
			t.Fatalf("unknown tool %q in test", test.tool)
		}
		got, err := buildArgs(spec, json.RawMessage(test.arguments))
		if test.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("tool %q: want error containing %q, got %v", test.tool, test.wantErr, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("tool %q: unexpected error: %v", test.tool, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("tool %q args:\n got %q\nwant %q", test.tool, got, test.want)
		}
	}
}

func connectTestSession(t *testing.T, exec ExecFunc) *mcp.ClientSession {
	t.Helper()
	server := New("test", exec)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestServerListsAllTools(t *testing.T) {
	session := connectTestSession(t, func(ctx context.Context, args []string, stdout, stderr io.Writer) int {
		return 0
	})
	listed := map[string]bool{}
	for tool, err := range session.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatalf("list tools: %v", err)
		}
		listed[tool.Name] = true
	}
	if len(listed) != len(toolSpecs) {
		t.Fatalf("listed %d tools, want %d", len(listed), len(toolSpecs))
	}
	for _, name := range []string{"traffic_search", "mobility_latest", "transit_journey", "doctor"} {
		if !listed[name] {
			t.Fatalf("tool %q missing from listing", name)
		}
	}
}

func TestServerCallToolSuccessWithDiagnostics(t *testing.T) {
	var gotArgs []string
	session := connectTestSession(t, func(ctx context.Context, args []string, stdout, stderr io.Writer) int {
		gotArgs = args
		fmt.Fprintln(stdout, `{"zones": []}`)
		fmt.Fprintln(stderr, "warning: example diagnostic")
		return 0
	})
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "traffic_zones",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	want := []string{"traffic", "zones", "--json"}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("exec args: got %q want %q", gotArgs, want)
	}
	if len(result.Content) != 2 {
		t.Fatalf("want 2 content blocks, got %d", len(result.Content))
	}
	first, ok := result.Content[0].(*mcp.TextContent)
	if !ok || first.Text != `{"zones": []}` {
		t.Fatalf("unexpected first content: %#v", result.Content[0])
	}
	second, ok := result.Content[1].(*mcp.TextContent)
	if !ok || !strings.Contains(second.Text, "warning: example diagnostic") {
		t.Fatalf("unexpected second content: %#v", result.Content[1])
	}
}

func TestServerCallToolFailure(t *testing.T) {
	session := connectTestSession(t, func(ctx context.Context, args []string, stdout, stderr io.Writer) int {
		fmt.Fprintln(stderr, "unknown traffic zone-id \"99\"")
		return 2
	})
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "traffic_search",
		Arguments: map[string]any{"query": "badia", "zone_id": "99"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Fatal("want IsError for nonzero exit code")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "unknown traffic zone-id") || !strings.Contains(text.Text, "exit code 2") {
		t.Fatalf("unexpected error content: %#v", result.Content[0])
	}
}

func TestServerCallToolRejectsInvalidArguments(t *testing.T) {
	called := false
	session := connectTestSession(t, func(ctx context.Context, args []string, stdout, stderr io.Writer) int {
		called = true
		return 0
	})
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "transit_delay_stats",
		Arguments: map[string]any{"from": "auer"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Fatal("want IsError for missing required argument")
	}
	if called {
		t.Fatal("exec must not run when arguments are invalid")
	}
}

func TestToolsListingMatchesSpecs(t *testing.T) {
	tools := Tools()
	if len(tools) != len(toolSpecs) {
		t.Fatalf("Tools() returned %d entries, want %d", len(tools), len(toolSpecs))
	}
	for i, info := range tools {
		if info.Name != toolSpecs[i].name {
			t.Fatalf("Tools()[%d] = %q, want %q", i, info.Name, toolSpecs[i].name)
		}
		if info.InputSchema["type"] != "object" {
			t.Fatalf("tool %q schema must have type object", info.Name)
		}
	}
}
