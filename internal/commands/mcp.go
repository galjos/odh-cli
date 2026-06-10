// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/galjos/odh-cli/internal/client"
	"github.com/galjos/odh-cli/internal/mcpserver"
	"github.com/galjos/odh-cli/internal/output"
	"github.com/galjos/odh-cli/internal/version"
	"github.com/spf13/cobra"
)

func (r *Runner) newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run odh as a Model Context Protocol server",
		RunE:  requireSubcommand,
	}
	cmd.AddCommand(r.newMCPServeCmd())
	cmd.AddCommand(r.newMCPToolsCmd())
	return cmd
}

func (r *Runner) newMCPServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Serve curated odh commands as MCP tools on stdio",
		Long: `Serve the curated odh command surface as Model Context Protocol tools
on stdin/stdout. Each tool call executes the matching CLI command
in-process, so tool outputs are the documented odh JSON contracts.
The server runs until the MCP client disconnects.`,
		Example: `  odh mcp serve
  claude mcp add odh -- odh mcp serve`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "odh MCP server listening on stdio")
			return mcpserver.ServeStdio(cmd.Context(), version.Current().Version, r.mcpExec)
		},
	}
}

func (r *Runner) newMCPToolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "List the MCP tool surface as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tools := mcpserver.Tools()
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"count": len(tools),
				"tools": tools,
			})
		},
	}
}

// mcpExec runs one tool invocation on a fresh sub-runner so concurrent
// MCP tool calls never share mutable runner state.
func (r *Runner) mcpExec(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	sub := &Runner{
		Registry: r.Registry,
		Client:   client.New(30 * time.Second),
	}
	return sub.Run(ctx, args, stdout, stderr)
}
