// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/galjos/odh-cli/internal/output"
)

func writeTrafficOutput(stdout io.Writer, result trafficResult) error {
	switch result.OutputFormat {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		return writeTrafficTable(stdout, result)
	case "markdown", "md":
		return writeTrafficMarkdown(stdout, result)
	default:
		return fmt.Errorf("unsupported format %q", result.OutputFormat)
	}
}

func writeTrafficZonesOutput(stdout io.Writer, result trafficZonesResult) error {
	switch result.OutputFormat {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ZONE_ID\tNAME")
		for _, zone := range result.Zones {
			fmt.Fprintf(tw, "%s\t%s\n", zone.ZoneID, zone.Name)
		}
		return tw.Flush()
	case "markdown", "md":
		fmt.Fprintln(stdout, "| zone_id | name |")
		fmt.Fprintln(stdout, "| --- | --- |")
		for _, zone := range result.Zones {
			fmt.Fprintf(stdout, "| %s | %s |\n", escapeMarkdown(zone.ZoneID), escapeMarkdown(zone.Name))
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.OutputFormat)
	}
}

func writeTrafficCategoriesOutput(stdout io.Writer, result trafficCategoriesResult) error {
	switch result.OutputFormat {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "NAME\tDESCRIPTION\tUPSTREAM_SUBTYPES")
		for _, category := range result.Categories {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", category.Name, category.Description, strings.Join(category.UpstreamSubtypes, ","))
		}
		return tw.Flush()
	case "markdown", "md":
		fmt.Fprintln(stdout, "| name | description | upstream_subtypes |")
		fmt.Fprintln(stdout, "| --- | --- | --- |")
		for _, category := range result.Categories {
			fmt.Fprintf(stdout, "| %s | %s | %s |\n",
				escapeMarkdown(category.Name),
				escapeMarkdown(category.Description),
				escapeMarkdown(strings.Join(category.UpstreamSubtypes, ", ")),
			)
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.OutputFormat)
	}
}

func writeTrafficTable(stdout io.Writer, result trafficResult) error {
	warnings := result.Warnings
	if result.Source == trafficSourceODH && len(warnings) > 0 {
		fmt.Fprintf(stdout, "warning: %s\n\n", warnings[0])
		warnings = warnings[1:]
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TYPE\tROAD\tPLACE\tTIME\tACTIVE\tSTALE")
	for _, event := range result.Events {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%t\t%t\n",
			event.Type,
			firstNonEmpty(event.Road, event.RoadName),
			compactText(firstNonEmpty(event.Place, event.PlaceIT), 90),
			compactRange(event.Start, event.End),
			event.Active,
			event.Stale,
		)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	for _, warning := range warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	return nil
}

func writeTrafficMarkdown(stdout io.Writer, result trafficResult) error {
	warnings := result.Warnings
	if result.Source == trafficSourceODH && len(warnings) > 0 {
		fmt.Fprintf(stdout, "> warning: %s\n\n", warnings[0])
		warnings = warnings[1:]
	}
	fmt.Fprintln(stdout, "| type | road | place | time | active | stale |")
	fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- |")
	for _, event := range result.Events {
		fmt.Fprintf(stdout, "| %s | %s | %s | %s | %t | %t |\n",
			escapeMarkdown(event.Type),
			escapeMarkdown(firstNonEmpty(event.Road, event.RoadName)),
			escapeMarkdown(compactText(firstNonEmpty(event.Place, event.PlaceIT), 90)),
			escapeMarkdown(compactRange(event.Start, event.End)),
			event.Active,
			event.Stale,
		)
	}
	for _, warning := range warnings {
		fmt.Fprintf(stdout, "\n> warning: %s\n", warning)
	}
	return nil
}

func compactRange(start, end string) string {
	start = compactDate(start)
	end = compactDate(end)
	if start == "" {
		return end
	}
	if end == "" || end == start {
		return start
	}
	return start + " - " + end
}

func compactDate(value string) string {
	parsed := parseODHTime(value)
	if parsed == nil {
		return strings.TrimSpace(value)
	}
	return parsed.Format("2006-01-02")
}
