// SPDX-FileCopyrightText: 2026 Josef M. Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func normalizeOutputFormat(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "json":
		return "json", nil
	case "table":
		return "table", nil
	case "markdown", "md":
		return "markdown", nil
	default:
		return "", fmt.Errorf("unsupported format %q", value)
	}
}

func applyJSONShortcut(format *string, enabled bool) {
	if enabled {
		*format = "json"
	}
}

// acceptJSONFlag registers --json on commands that only ever emit JSON so that
// suggested command strings carrying the flag still parse. The value is ignored.
func acceptJSONFlag(cmd *cobra.Command) {
	var ignored bool
	cmd.Flags().BoolVar(&ignored, "json", false, "accepted for consistency; this command always emits JSON")
}

func writePlainWarnings(stdout io.Writer, warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
}

func writeMarkdownWarnings(stdout io.Writer, warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(stdout, "\n> warning: %s\n", warning)
	}
}
