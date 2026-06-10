// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ExecFunc runs one CLI argument vector and returns its process exit
// code. Implementations must be safe for concurrent calls.
type ExecFunc func(ctx context.Context, args []string, stdout, stderr io.Writer) int

const serverInstructions = `odh exposes public Open Data Hub (NOI Techpark) data for South Tyrol
and beyond: traffic events, A22 feeds, mobility time series, public
transport timetables, GTFS-RT, tourism data, and data-quality
diagnostics.

Ground rules for answering from these tools:
- Discover before filtering: use mobility_origins, mobility_types,
  traffic_zones, and transit_stops_search instead of guessing IDs,
  origins, or place aliases.
- Surface the "warnings" arrays and stderr diagnostics in answers;
  never present stale or hidden rows as current conditions.
- Run the diagnostics_* tools before making claims from data areas
  with known freshness caveats (EV charging, parking forecasts,
  Tourism events).
- Unsupported capabilities (historical delay probabilities, past A22
  incidents) are reported explicitly; do not work around them by
  guessing.

Tool outputs are the documented odh JSON contracts; see
docs/json-contracts.md in the odh-cli repository.`

// New builds an MCP server exposing the curated odh tool surface.
// Tool calls are executed through exec.
func New(version string, exec ExecFunc) *mcp.Server {
	impl := &mcp.Implementation{
		Name:    "odh",
		Title:   "Open Data Hub CLI (unofficial)",
		Version: version,
	}
	server := mcp.NewServer(impl, &mcp.ServerOptions{Instructions: serverInstructions})
	for _, spec := range toolSpecs {
		tool := &mcp.Tool{
			Name:        spec.name,
			Description: spec.desc,
			InputSchema: spec.inputSchema(),
		}
		server.AddTool(tool, newToolHandler(spec, exec))
	}
	return server
}

// ServeStdio runs the MCP server on stdin/stdout until the client
// disconnects or ctx is canceled.
func ServeStdio(ctx context.Context, version string, exec ExecFunc) error {
	return New(version, exec).Run(ctx, &mcp.StdioTransport{})
}

func newToolHandler(spec toolSpec, exec ExecFunc) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := buildArgs(spec, req.Params.Arguments)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		ctx, cancel := context.WithTimeout(ctx, spec.timeout())
		defer cancel()

		var stdout, stderr bytes.Buffer
		exitCode := exec(ctx, args, &stdout, &stderr)
		diagnostics := strings.TrimSpace(stderr.String())
		if exitCode != 0 {
			message := diagnostics
			if message == "" {
				message = fmt.Sprintf("odh %s failed", strings.Join(args, " "))
			}
			return errorResult(fmt.Sprintf("%s (exit code %d)", message, exitCode)), nil
		}

		content := []mcp.Content{&mcp.TextContent{Text: strings.TrimSpace(stdout.String())}}
		if diagnostics != "" {
			content = append(content, &mcp.TextContent{Text: "stderr diagnostics:\n" + diagnostics})
		}
		return &mcp.CallToolResult{Content: content}, nil
	}
}

func errorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}
}

// ToolInfo describes one exposed MCP tool for listings outside an MCP
// session, such as odh mcp tools.
type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Tools returns the exposed tool surface in registration order.
func Tools() []ToolInfo {
	infos := make([]ToolInfo, 0, len(toolSpecs))
	for _, spec := range toolSpecs {
		infos = append(infos, ToolInfo{
			Name:        spec.name,
			Description: spec.desc,
			InputSchema: spec.inputSchema(),
		})
	}
	return infos
}
