// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type trafficQuery struct {
	Source         string
	ZoneID         string
	Area           string
	Type           string
	Road           string
	Near           string
	Radius         string
	From           string
	To             string
	Search         string
	Today          bool
	Format         string
	Limit          int
	Raw            bool
	IncludeExpired bool
	IncludeStale   bool
}

type trafficEvent struct {
	ID              string         `json:"id"`
	SeriesID        string         `json:"series_id,omitempty"`
	MessageID       string         `json:"message_id,omitempty"`
	Source          string         `json:"source"`
	Type            string         `json:"type"`
	Subtype         string         `json:"subtype,omitempty"`
	Severity        string         `json:"severity,omitempty"`
	ZoneID          string         `json:"zone_id,omitempty"`
	Zone            string         `json:"zone,omitempty"`
	ZoneIT          string         `json:"zone_it,omitempty"`
	Road            string         `json:"road,omitempty"`
	RoadName        string         `json:"road_name,omitempty"`
	Place           string         `json:"place,omitempty"`
	PlaceIT         string         `json:"place_it,omitempty"`
	Start           string         `json:"start,omitempty"`
	End             string         `json:"end,omitempty"`
	PublishedAt     string         `json:"published_at,omitempty"`
	TransactionTime string         `json:"transaction_time,omitempty"`
	Coordinates     []float64      `json:"coordinates,omitempty"`
	Active          bool           `json:"active"`
	Stale           bool           `json:"stale"`
	Raw             map[string]any `json:"raw,omitempty"`
}

type trafficArea struct {
	Name     string
	ZoneIDs  []string
	Keywords []string
}

type trafficZone struct {
	ZoneID string `json:"zone_id"`
	Name   string `json:"name"`
}

type trafficCategory struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Aliases          []string `json:"aliases,omitempty"`
	UpstreamSubtypes []string `json:"upstream_subtypes,omitempty"`
}

type trafficZonesResult struct {
	Source       string        `json:"source"`
	SourceDetail string        `json:"source_detail"`
	Zones        []trafficZone `json:"zones"`
	OutputFormat string        `json:"-"`
}

type trafficCategoriesResult struct {
	Source       string            `json:"source"`
	SourceDetail string            `json:"source_detail"`
	Categories   []trafficCategory `json:"categories"`
	OutputFormat string            `json:"-"`
}

func (r *Runner) newTrafficCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "traffic",
		Short: "Opinionated Open Data Hub traffic commands",
		Long: `Opinionated helpers for Open Data Hub PROVINCE_BZ traffic events.

Use these commands for roadworks, closures, road events, bike notices, and
traffic notices before falling back to raw mobility event calls. Results are
deduplicated and stale open-ended rows are hidden by default.

--source odh reads the Mobility Timeseries event feed and supports --zone-id,
--area and --road. --source content reads the Content API Announcement road
bulletin, which is the feed the province still updates, but which carries no
zone, road, or severity fields; there --zone-id and --area match by geographic
inference from historical zone coordinates, and --road is rejected.`,
		Example: `  odh traffic zones
  odh traffic categories
  odh traffic today --area ueberetsch-unterland --type roadworks
  odh traffic today --source content --json
  odh traffic search badia --today --json`,
		RunE: requireSubcommand,
	}

	var zonesFormat string
	var zonesJSON bool
	zonesCmd := &cobra.Command{
		Use:   "zones",
		Short: "List traffic zones",
		Example: `  odh traffic zones
  odh traffic zones --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zonesJSON {
				zonesFormat = "json"
			}
			normalizedFormat, err := normalizeOutputFormat(zonesFormat)
			if err != nil {
				return err
			}
			return writeTrafficZonesOutput(cmd.OutOrStdout(), trafficZonesResult{
				Source:       "odh",
				SourceDetail: "Open Data Hub Mobility API PROVINCE_BZ traffic zones",
				Zones:        knownTrafficZones(),
				OutputFormat: normalizedFormat,
			})
		},
	}
	zonesCmd.Flags().StringVar(&zonesFormat, "format", "table", "output format: json, table, or markdown")
	zonesCmd.Flags().BoolVar(&zonesJSON, "json", false, "shortcut for --format json")

	var catsFormat string
	var catsJSON bool
	categoriesCmd := &cobra.Command{
		Use:   "categories",
		Short: "List traffic event categories",
		Example: `  odh traffic categories
  odh traffic categories --format markdown`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if catsJSON {
				catsFormat = "json"
			}
			normalizedFormat, err := normalizeOutputFormat(catsFormat)
			if err != nil {
				return err
			}
			return writeTrafficCategoriesOutput(cmd.OutOrStdout(), trafficCategoriesResult{
				Source:       "odh",
				SourceDetail: "Open Data Hub Mobility API PROVINCE_BZ traffic event categories",
				Categories:   knownTrafficCategories(),
				OutputFormat: normalizedFormat,
			})
		},
	}
	categoriesCmd.Flags().StringVar(&catsFormat, "format", "table", "output format: json, table, or markdown")
	categoriesCmd.Flags().BoolVar(&catsJSON, "json", false, "shortcut for --format json")

	var todayQuery trafficQuery
	var todayJSON bool
	todayCmd := &cobra.Command{
		Use:   "today",
		Short: "Query today's traffic events",
		Long: `Query traffic events active today from Open Data Hub PROVINCE_BZ.

Use --area, --zone-id, --road, --type, or --near to narrow the answer. The
default table is meant for humans; use --json for agents and scripts.

--source content queries the Content API Announcement bulletin instead, where
--road is rejected because the feed cannot answer it, and --zone-id and --area
match by inferring a zone from the announcement's coordinates.`,
		Example: `  odh traffic today --area ueberetsch-unterland --type roadworks
  odh traffic today --near 46.42,11.25 --radius 15km --json
  odh traffic today --source content --type closure --json
  odh --timeout 20s traffic today --area bozen-unterland --format markdown`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if todayJSON {
				todayQuery.Format = "json"
			}
			today := time.Now().Format("2006-01-02")
			todayQuery.From = today
			todayQuery.To = today
			q, err := finalizeTrafficFlags(todayQuery)
			if err != nil {
				return err
			}
			return r.runTrafficQueryCobra(cmd.Context(), q, cmd.OutOrStdout(), cmd.OutOrStderr())
		},
	}
	addTrafficCobraFlags(todayCmd, &todayQuery)
	todayCmd.Flags().BoolVar(&todayJSON, "json", false, "shortcut for --format json")

	var eventsQuery trafficQuery
	var eventsJSON bool
	eventsCmd := &cobra.Command{
		Use:   "events",
		Short: "Query traffic events by date",
		Long: `Query Open Data Hub PROVINCE_BZ traffic events for an explicit date range.

If neither --from nor --to is set, the command defaults to today.`,
		Example: `  odh traffic events --from 2026-05-16 --to 2026-05-16 --area bozen-unterland --json
  odh traffic events --road SP13 --type closure --format table
  odh traffic events --source content --from 2026-08-01 --to 2026-08-03 --json
  odh traffic events --zone-id 4 --include-stale --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if eventsJSON {
				eventsQuery.Format = "json"
			}
			if strings.TrimSpace(eventsQuery.From) == "" && strings.TrimSpace(eventsQuery.To) == "" {
				today := time.Now().Format("2006-01-02")
				eventsQuery.From = today
				eventsQuery.To = today
			}
			if strings.TrimSpace(eventsQuery.From) == "" {
				eventsQuery.From = eventsQuery.To
			}
			if strings.TrimSpace(eventsQuery.To) == "" {
				eventsQuery.To = eventsQuery.From
			}
			q, err := finalizeTrafficFlags(eventsQuery)
			if err != nil {
				return err
			}
			return r.runTrafficQueryCobra(cmd.Context(), q, cmd.OutOrStdout(), cmd.OutOrStderr())
		},
	}
	addTrafficCobraFlags(eventsCmd, &eventsQuery)
	eventsCmd.Flags().BoolVar(&eventsJSON, "json", false, "shortcut for --format json")

	var searchQuery trafficQuery
	var searchJSON bool
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search traffic events by text",
		Long: `Search traffic events by road, place, message text, or upstream metadata.

By default, search checks today's events. Add --from/--to for a different date
range or --include-stale when you explicitly want hidden open-ended rows.`,
		Example: `  odh traffic search badia --today --json
  odh traffic search "St. Pauls" --from 2026-05-16 --to 2026-05-16 --include-stale
  odh traffic search radroute --today --source content --json
  odh traffic search neustift --zone-id 6 --type closure --format table`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if searchJSON {
				searchQuery.Format = "json"
			}
			searchQuery.Search = strings.Join(args, " ")
			if searchQuery.Today || (strings.TrimSpace(searchQuery.From) == "" && strings.TrimSpace(searchQuery.To) == "") {
				today := time.Now().Format("2006-01-02")
				searchQuery.From = today
				searchQuery.To = today
			}
			if strings.TrimSpace(searchQuery.From) == "" {
				searchQuery.From = searchQuery.To
			}
			if strings.TrimSpace(searchQuery.To) == "" {
				searchQuery.To = searchQuery.From
			}
			q, err := finalizeTrafficFlags(searchQuery)
			if err != nil {
				return err
			}
			return r.runTrafficQueryCobra(cmd.Context(), q, cmd.OutOrStdout(), cmd.OutOrStderr())
		},
	}
	addTrafficCobraFlags(searchCmd, &searchQuery)
	searchCmd.Flags().BoolVar(&searchQuery.Today, "today", false, "search today's traffic events")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "shortcut for --format json")

	cmd.AddCommand(zonesCmd)
	cmd.AddCommand(categoriesCmd)
	cmd.AddCommand(todayCmd)
	cmd.AddCommand(eventsCmd)
	cmd.AddCommand(searchCmd)
	return cmd
}

func addTrafficCobraFlags(cmd *cobra.Command, query *trafficQuery) {
	cmd.Flags().StringVar(&query.Source, "source", "odh", "traffic source: odh (Mobility Timeseries events) or content (Content API Announcement bulletin)")
	cmd.Flags().StringVar(&query.ZoneID, "zone-id", "", "ODH PROVINCE_BZ messageZoneId filter, for example 6")
	cmd.Flags().StringVar(&query.Area, "area", "", "area alias, for example ueberetsch-unterland")
	cmd.Flags().StringVar(&query.Type, "type", "all", "type filter: all, roadworks, closure, event, traffic, mountain-pass, bike, or radar")
	cmd.Flags().StringVar(&query.Road, "road", "", "road filter, for example SP13 or SS42")
	cmd.Flags().StringVar(&query.Near, "near", "", "coordinate filter as lat,lon")
	cmd.Flags().StringVar(&query.Radius, "radius", "15km", "radius for --near, for example 15km")
	cmd.Flags().StringVar(&query.From, "from", "", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&query.To, "to", "", "end date YYYY-MM-DD")
	cmd.Flags().StringVar(&query.Format, "format", "table", "output format: json, table, or markdown")
	cmd.Flags().IntVar(&query.Limit, "limit", 1000, "maximum raw events to request")
	cmd.Flags().BoolVar(&query.Raw, "raw", false, "include raw upstream event objects in JSON output")
	cmd.Flags().BoolVar(&query.IncludeExpired, "include-expired", false, "include expired events after local date filtering")
	cmd.Flags().BoolVar(&query.IncludeStale, "include-stale", false, "include stale open-ended events that are hidden by default")
}

func finalizeTrafficFlags(query trafficQuery) (trafficQuery, error) {
	if query.Limit < 1 {
		return trafficQuery{}, fmt.Errorf("--limit must be greater than zero")
	}
	format, err := normalizeOutputFormat(query.Format)
	if err != nil {
		return trafficQuery{}, err
	}
	query.Format = format
	return query, nil
}

func (r *Runner) runTrafficQueryCobra(ctx context.Context, query trafficQuery, stdout, stderr io.Writer) error {
	source, err := normalizeTrafficSource(query.Source)
	if err != nil {
		return err
	}
	if source == trafficSourceContent {
		if err := rejectUnsupportedContentTrafficFlags(query); err != nil {
			return err
		}
	}
	if strings.TrimSpace(query.Near) != "" {
		if _, _, _, err := parseNearRadius(query.Near, query.Radius); err != nil {
			return err
		}
	}
	fromDay, toDay, err := parseTrafficDateRange(query.From, query.To)
	if err != nil {
		return err
	}
	area, err := resolveTrafficArea(query.Area)
	if err != nil {
		return err
	}
	if strings.TrimSpace(query.ZoneID) != "" {
		if err := validateTrafficZoneIDs(query.ZoneID); err != nil {
			return err
		}
	}
	if _, err := normalizeTrafficTypeFilter(query.Type); err != nil {
		return err
	}
	if source == trafficSourceContent {
		return r.runContentTrafficQueryCobra(ctx, query, area, fromDay, toDay, stdout)
	}
	return r.runODHTrafficQueryCobra(ctx, query, area, fromDay, toDay, stdout, stderr)
}

func (r *Runner) runODHTrafficQueryCobra(ctx context.Context, query trafficQuery, area trafficArea, fromDay, toDay time.Time, stdout, stderr io.Writer) error {
	api, _ := r.Registry.Find("mobility")
	path := fmt.Sprintf("/v2/flat,event/PROVINCE_BZ/%s/%s", fromDay.Format("2006-01-02"), toDay.Format("2006-01-02"))
	values := url.Values{}
	values.Set("limit", strconv.Itoa(query.Limit))
	requestURL, err := BuildURL(api.BaseURL, path, values)
	if err != nil {
		return err
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		return err
	}
	rawEvents := extractDataList(value)
	events, warnings := normalizeTrafficEvents(rawEvents, query, area, fromDay, toDay)
	return writeTrafficOutput(stdout, trafficResult{
		Source:       "odh",
		SourceDetail: "Open Data Hub Mobility API PROVINCE_BZ traffic events",
		Endpoint:     requestURL,
		From:         fromDay.Format("2006-01-02"),
		To:           toDay.Format("2006-01-02"),
		ZoneID:       strings.TrimSpace(query.ZoneID),
		Area:         area.Name,
		Type:         normalizeTrafficTypeName(query.Type),
		Search:       strings.TrimSpace(query.Search),
		RawCount:     len(rawEvents),
		Count:        len(events),
		Coverage:     pageCoverage(len(rawEvents), len(events), len(events), query.Limit),
		Events:       events,
		Warnings:     warnings,
		OutputFormat: query.Format,
		IncludeRaw:   query.Raw,
	})
}

type trafficResult struct {
	Source       string         `json:"source"`
	SourceDetail string         `json:"source_detail"`
	Endpoint     string         `json:"endpoint"`
	From         string         `json:"from"`
	To           string         `json:"to"`
	ZoneID       string         `json:"zone_id,omitempty"`
	Area         string         `json:"area,omitempty"`
	Type         string         `json:"type,omitempty"`
	Search       string         `json:"search,omitempty"`
	RawCount     int            `json:"raw_count"`
	Count        int            `json:"count"`
	Coverage     resultCoverage `json:"coverage"`
	Events       []trafficEvent `json:"events"`
	Warnings     []string       `json:"warnings,omitempty"`
	OutputFormat string         `json:"-"`
	IncludeRaw   bool           `json:"-"`
}
