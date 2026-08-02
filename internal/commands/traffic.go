// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/galjos/odh-cli/internal/output"
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
deduplicated and stale open-ended rows are hidden by default.`,
		Example: `  odh traffic zones
  odh traffic categories
  odh traffic today --area ueberetsch-unterland --type roadworks
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
			normalizedFormat, err := normalizeTrafficFormat(zonesFormat)
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
			normalizedFormat, err := normalizeTrafficFormat(catsFormat)
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
default table is meant for humans; use --json for agents and scripts.`,
		Example: `  odh traffic today --area ueberetsch-unterland --type roadworks
  odh traffic today --near 46.42,11.25 --radius 15km --json
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
	cmd.Flags().StringVar(&query.Source, "source", "odh", "traffic source: odh")
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
	format, err := normalizeTrafficFormat(query.Format)
	if err != nil {
		return trafficQuery{}, err
	}
	query.Format = format
	return query, nil
}

func (r *Runner) runTrafficQueryCobra(ctx context.Context, query trafficQuery, stdout, stderr io.Writer) error {
	if source := strings.ToLower(strings.TrimSpace(query.Source)); source != "" && source != "odh" {
		return fmt.Errorf("unsupported traffic source %q; supported source: odh", query.Source)
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
	Events       []trafficEvent `json:"events"`
	Warnings     []string       `json:"warnings,omitempty"`
	OutputFormat string         `json:"-"`
	IncludeRaw   bool           `json:"-"`
}

func normalizeTrafficEvents(raw []map[string]any, query trafficQuery, area trafficArea, fromDay, toDay time.Time) ([]trafficEvent, []string) {
	events := make([]trafficEvent, 0, len(raw))
	now := time.Now()
	staleCount := 0
	hiddenStaleOpenEndedCount := 0
	expiredCount := 0
	futureCount := 0
	for _, record := range raw {
		event := normalizeTrafficEvent(record, query.Raw, now)
		if !trafficZoneMatches(event, query.ZoneID) {
			continue
		}
		if !trafficAreaMatches(event, area) {
			continue
		}
		if !trafficTypeMatches(event, query.Type) {
			continue
		}
		if !trafficRoadMatches(event, query.Road) {
			continue
		}
		if !trafficNearMatches(event, query.Near, query.Radius) {
			continue
		}
		if !trafficSearchMatches(event, query.Search) {
			continue
		}
		active := eventActiveInRange(event, fromDay, toDay)
		event.Active = active
		if !active && !query.IncludeExpired {
			if eventEndBefore(event, fromDay) {
				expiredCount++
			} else {
				futureCount++
			}
			continue
		}
		if event.Stale && !eventHasEnd(event) && !query.IncludeStale {
			hiddenStaleOpenEndedCount++
			continue
		}
		if event.Stale {
			staleCount++
		}
		events = append(events, event)
	}
	deduped := dedupeTrafficEvents(events)
	sort.SliceStable(deduped, func(i, j int) bool {
		if deduped[i].Start != deduped[j].Start {
			return deduped[i].Start < deduped[j].Start
		}
		if deduped[i].Road != deduped[j].Road {
			return deduped[i].Road < deduped[j].Road
		}
		return deduped[i].Place < deduped[j].Place
	})

	warnings := make([]string, 0)
	if len(events) != len(deduped) {
		warnings = append(warnings, fmt.Sprintf("deduplicated %d raw matching rows to %d events", len(events), len(deduped)))
	}
	if staleCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d matching events have transaction or publish timestamps older than 30 days", staleCount))
	}
	if hiddenStaleOpenEndedCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d stale open-ended matching events were hidden; pass --include-stale to inspect them", hiddenStaleOpenEndedCount))
	}
	if expiredCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d expired matching events were hidden", expiredCount))
	}
	if futureCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d future matching events were hidden", futureCount))
	}
	if len(deduped) == 0 {
		if warning := trafficNoMatchesWarning(query, area, hiddenStaleOpenEndedCount); warning != "" {
			warnings = append(warnings, warning)
		}
	}
	if warning := mobilityTruncationWarning("returned", "raw event rows", query.Limit, len(raw), "traffic completeness"); warning != "" {
		warnings = append(warnings, warning)
	}
	warnings = append(warnings, timeseriesEventFeedWarning(newestTrafficEventTimestamp(deduped), "PROVINCE_BZ"))
	warnings = append(warnings, "source is Open Data Hub PROVINCE_BZ; compare with the official traffic service before presenting this as a complete live road bulletin")
	return deduped, warnings
}

func normalizeTrafficEvent(record map[string]any, includeRaw bool, now time.Time) trafficEvent {
	metadata, _ := record["evmetadata"].(map[string]any)
	event := trafficEvent{
		ID:              firstNonEmpty(asString(record["evuuid"]), asString(record["evname"])),
		SeriesID:        asString(record["evseriesuuid"]),
		MessageID:       asString(metadata["messageId"]),
		Source:          "odh",
		Subtype:         strings.TrimSpace(asString(metadata["subTycodeValue"])),
		Severity:        firstNonEmpty(asString(metadata["messageGradDescDe"]), asString(metadata["messageGradDescIt"])),
		ZoneID:          asString(metadata["messageZoneId"]),
		Zone:            strings.TrimSpace(asString(metadata["messageZoneDescDe"])),
		ZoneIT:          strings.TrimSpace(asString(metadata["messageZoneDescIt"])),
		Road:            normalizeRoad(asString(metadata["messageStreetNr"])),
		RoadName:        strings.TrimSpace(asString(metadata["messageStreetInternetDescDe"])),
		Place:           cleanTrafficText(asString(metadata["placeDe"])),
		PlaceIT:         cleanTrafficText(asString(metadata["placeIt"])),
		Start:           asString(record["evstart"]),
		End:             asString(record["evend"]),
		PublishedAt:     firstNonEmpty(asString(metadata["publishDateTime"]), asString(metadata["publisherDateTime"])),
		TransactionTime: asString(record["evtransactiontime"]),
		Coordinates:     extractCoordinates(record["evlgeometry"]),
	}
	event.Type = classifyTrafficType(event)
	if includeRaw {
		event.Raw = record
	}
	event.Stale = trafficEventStale(event, now)
	return event
}

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

func normalizeTrafficFormat(value string) (string, error) {
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

func writeTrafficTable(stdout io.Writer, result trafficResult) error {
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
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	return nil
}

func writeTrafficMarkdown(stdout io.Writer, result trafficResult) error {
	fmt.Fprintln(stdout, "| type | road | place | time | active | stale |")
	fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- | --- |")
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
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "\n> warning: %s\n", warning)
	}
	return nil
}

func trafficZoneMatches(event trafficEvent, zoneIDs string) bool {
	values := trafficZoneIDValues(zoneIDs)
	if len(values) == 0 {
		return true
	}
	return containsString(values, event.ZoneID)
}

func trafficAreaMatches(event trafficEvent, area trafficArea) bool {
	if area.Name == "" {
		return true
	}
	if len(area.ZoneIDs) > 0 && !containsString(area.ZoneIDs, event.ZoneID) {
		return false
	}
	if len(area.Keywords) == 0 {
		return true
	}
	haystack := normalizeTrafficSearchText(strings.Join([]string{event.Zone, event.ZoneIT, event.Road, event.RoadName, event.Place, event.PlaceIT}, " "))
	for _, keyword := range area.Keywords {
		keyword = normalizeTrafficSearchText(keyword)
		if keyword != "" && strings.Contains(haystack, keyword) {
			return true
		}
	}
	return false
}

func trafficTypeMatches(event trafficEvent, filter string) bool {
	filter = normalizeTrafficTypeName(filter)
	if filter == "" || filter == "all" {
		return true
	}
	if event.Type == filter {
		return true
	}
	if filter == "closure" && strings.Contains(strings.ToLower(event.Place), "sperre") {
		return true
	}
	if filter == "bike" && strings.Contains(strings.ToLower(event.RoadName), "rad") {
		return true
	}
	if filter == "mountain-pass" && textContainsPass(event.RoadName+" "+event.Place) {
		return true
	}
	return false
}

func trafficRoadMatches(event trafficEvent, road string) bool {
	road = normalizeRoad(road)
	if road == "" {
		return true
	}
	filter := compactRoadToken(road)
	eventRoad := compactRoadToken(event.Road)
	return strings.EqualFold(event.Road, road) ||
		(filter != "" && strings.Contains(eventRoad, filter)) ||
		strings.Contains(strings.ToLower(event.RoadName), strings.ToLower(road))
}

func trafficNearMatches(event trafficEvent, near, radius string) bool {
	if strings.TrimSpace(near) == "" {
		return true
	}
	lat, lon, radiusKM, err := parseNearRadius(near, radius)
	if err != nil || len(event.Coordinates) < 2 {
		return false
	}
	eventLon, eventLat := event.Coordinates[0], event.Coordinates[1]
	return haversineKM(lat, lon, eventLat, eventLon) <= radiusKM
}

func trafficSearchMatches(event trafficEvent, search string) bool {
	if strings.TrimSpace(search) == "" {
		return true
	}
	haystack := normalizeTrafficSearchText(strings.Join([]string{
		event.ID,
		event.SeriesID,
		event.MessageID,
		event.Source,
		event.Type,
		event.Subtype,
		event.Severity,
		event.ZoneID,
		event.Zone,
		event.ZoneIT,
		event.Road,
		event.RoadName,
		event.Place,
		event.PlaceIT,
	}, " "))
	groups := trafficSearchTermGroups(search)
	if len(groups) == 0 {
		return true
	}
	for _, group := range groups {
		matched := false
		for _, term := range group {
			term = normalizeTrafficSearchText(term)
			if term != "" && strings.Contains(haystack, term) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func trafficSearchTermGroups(search string) [][]string {
	terms := strings.Fields(normalizeTrafficSearchText(search))
	groups := make([][]string, 0, len(terms))
	for _, term := range terms {
		if trafficSearchStopword(term) {
			continue
		}
		groups = append(groups, trafficSearchAlternatives(term))
	}
	return groups
}

func trafficSearchStopword(term string) bool {
	switch term {
	case "a", "an", "am", "and", "are", "auf", "bei", "by", "der", "die", "das", "del", "della", "den", "des", "di", "for", "heute", "in", "is", "la", "le", "near", "on", "road", "roads", "route", "street", "streets", "strasse", "strassen", "the", "today", "um", "und", "via", "why":
		return true
	default:
		return false
	}
}

func trafficSearchAlternatives(term string) []string {
	switch term {
	case "blocked", "closed", "closure", "closures", "roadblock", "roadblocks":
		return []string{"closure", "closed", "blocked", "sperre", "sperren", "gesperrt"}
	case "baustelle", "baustellen", "construction", "roadwork", "roadworks", "works":
		return []string{"roadworks", "baustelle", "baustellen", "arbeiten"}
	case "event", "events", "veranstaltung", "veranstaltungen":
		return []string{"event", "events", "veranstaltung", "veranstaltungen"}
	case "sperre", "sperren", "gesperrt":
		return []string{"sperre", "sperren", "gesperrt", "closure", "closed", "blocked"}
	default:
		return []string{term}
	}
}

func trafficNoMatchesWarning(query trafficQuery, area trafficArea, hiddenStaleOpenEndedCount int) string {
	parts := make([]string, 0, 5)
	if search := strings.TrimSpace(query.Search); search != "" {
		parts = append(parts, fmt.Sprintf("search %q", search))
	}
	if zoneID := strings.TrimSpace(query.ZoneID); zoneID != "" {
		parts = append(parts, "zone-id "+zoneID)
	}
	if area.Name != "" {
		parts = append(parts, "area "+area.Name)
	}
	if eventType := normalizeTrafficTypeName(query.Type); eventType != "" && eventType != "all" {
		parts = append(parts, "type "+eventType)
	}
	if road := strings.TrimSpace(query.Road); road != "" {
		parts = append(parts, "road "+road)
	}
	if near := strings.TrimSpace(query.Near); near != "" {
		parts = append(parts, "near "+near)
	}
	if len(parts) == 0 {
		return ""
	}
	warning := "no current ODH PROVINCE_BZ traffic events matched " + strings.Join(parts, ", ") + " in the selected date range"
	if hiddenStaleOpenEndedCount > 0 {
		warning += "; stale open-ended matches may exist, rerun with --include-stale to inspect them"
	}
	return warning
}

func eventActiveInRange(event trafficEvent, fromDay, toDay time.Time) bool {
	start := parseODHTime(event.Start)
	end := parseODHTime(event.End)
	rangeStart := startOfDay(fromDay)
	rangeEnd := endOfDay(toDay)
	if start != nil && start.After(rangeEnd) {
		return false
	}
	if end != nil && end.Before(rangeStart) {
		return false
	}
	return true
}

func eventEndBefore(event trafficEvent, fromDay time.Time) bool {
	end := parseODHTime(event.End)
	return end != nil && end.Before(startOfDay(fromDay))
}

func eventHasEnd(event trafficEvent) bool {
	return parseODHTime(event.End) != nil
}

// newestRecordTimestamp returns the most recent event timestamp across raw
// Mobility records, for callers that have not normalized them into events yet.
func newestRecordTimestamp(records []map[string]any) *time.Time {
	var newest *time.Time
	for _, record := range records {
		for _, key := range []string{"evtransactiontime", "evpublishtime"} {
			parsed := parseODHTime(asString(record[key]))
			if parsed == nil {
				continue
			}
			if newest == nil || parsed.After(*newest) {
				newest = parsed
			}
		}
	}
	return newest
}

// newestTrafficEventTimestamp returns the most recent transaction or publish
// timestamp across the given events, or nil when none of them carry one.
func newestTrafficEventTimestamp(events []trafficEvent) *time.Time {
	var newest *time.Time
	for _, event := range events {
		for _, value := range []string{event.TransactionTime, event.PublishedAt} {
			parsed := parseODHTime(value)
			if parsed == nil {
				continue
			}
			if newest == nil || parsed.After(*newest) {
				newest = parsed
			}
		}
	}
	return newest
}

// timeseriesEventFeedWarning reports the newest row this response carried rather
// than asserting an upstream feed status the CLI never checks, so the caveat
// stays true whatever upstream does next.
func timeseriesEventFeedWarning(newest *time.Time, announcementSource string) string {
	replacement := fmt.Sprintf("odh call tourism /v1/Announcement --param source=%s --param rawsort=-LastChange", announcementSource)
	if newest == nil {
		return fmt.Sprintf("this Mobility Timeseries event feed returned no dated rows; an empty result is not evidence that roads are clear, so cross-check current notices with: %s", replacement)
	}
	return fmt.Sprintf("the newest row in this Mobility Timeseries event response is dated %s; this feed is not a live bulletin, and neither a stale nor an empty result is evidence that roads are clear, so cross-check current notices with: %s", newest.UTC().Format("2006-01-02"), replacement)
}

// announcementSourceForOrigin maps a Mobility event origin to the Content API
// Announcement source that carries current notices for the same roads.
func announcementSourceForOrigin(origin string) string {
	if strings.EqualFold(strings.TrimSpace(origin), "a22") {
		return "a22"
	}
	return "PROVINCE_BZ"
}

func trafficEventStale(event trafficEvent, now time.Time) bool {
	for _, value := range []string{event.TransactionTime, event.PublishedAt} {
		parsed := parseODHTime(value)
		if parsed != nil {
			return parsed.Before(now.AddDate(0, 0, -30))
		}
	}
	return false
}

func dedupeTrafficEvents(events []trafficEvent) []trafficEvent {
	seen := map[string]struct{}{}
	result := make([]trafficEvent, 0, len(events))
	for _, event := range events {
		key := strings.Join([]string{
			event.ZoneID,
			event.Road,
			event.RoadName,
			normalizeDedupText(event.Place),
			event.Start,
			event.End,
		}, "|")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, event)
	}
	return result
}

func classifyTrafficType(event trafficEvent) string {
	subtype := strings.ToUpper(strings.TrimSpace(event.Subtype))
	combined := strings.ToLower(event.RoadName + " " + event.Place)
	switch subtype {
	case "BAUSTELLE":
		return "roadworks"
	case "SPERRE":
		if textContainsPass(combined) {
			return "mountain-pass"
		}
		return "closure"
	case "RADWEG_SPERRE":
		return "bike"
	case "VERANSTALTUNG":
		return "event"
	case "RADARKONTROLLE":
		return "radar"
	case "STAU", "UNFALL", "SCHNEEFALL", "AMPELREGELUNG", "VORSICHT", "FREI BEFAHRBAR":
		if textContainsPass(combined) {
			return "mountain-pass"
		}
		return "traffic"
	default:
		if strings.Contains(combined, "radroute") || strings.Contains(combined, "radweg") {
			return "bike"
		}
		if textContainsPass(combined) {
			return "mountain-pass"
		}
		return "traffic"
	}
}

func normalizeTrafficTypeFilter(value string) (string, error) {
	normalized := normalizeTrafficTypeName(value)
	switch normalized {
	case "", "all", "roadworks", "closure", "event", "traffic", "mountain-pass", "bike", "radar":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported traffic type %q", value)
	}
}

func normalizeTrafficTypeName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all", "any":
		return "all"
	case "roadwork", "roadworks", "works", "baustelle", "baustellen":
		return "roadworks"
	case "closure", "closures", "closed", "sperre", "sperren":
		return "closure"
	case "events", "event", "veranstaltung", "veranstaltungen":
		return "event"
	case "traffic", "incident", "incidents", "stau", "unfall", "warning":
		return "traffic"
	case "mountain-pass", "mountain-pass-closure", "pass", "passes":
		return "mountain-pass"
	case "bike", "cycle", "cycling", "radweg":
		return "bike"
	case "radar", "speed", "speed-control":
		return "radar"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func knownTrafficZones() []trafficZone {
	return []trafficZone{
		{ZoneID: "1", Name: "Vinschgau"},
		{ZoneID: "2", Name: "Burggrafenamt"},
		{ZoneID: "3", Name: "Bozen-Unterland"},
		{ZoneID: "4", Name: "Salten-Schlern"},
		{ZoneID: "5", Name: "Eisacktal-Wipptal"},
		{ZoneID: "6", Name: "Pustertal"},
		{ZoneID: "7", Name: "Ausserhalb Südtirol"},
	}
}

func knownTrafficCategories() []trafficCategory {
	return []trafficCategory{
		{
			Name:             "roadworks",
			Description:      "Roadworks and construction notices.",
			Aliases:          []string{"roadwork", "works", "baustelle", "baustellen"},
			UpstreamSubtypes: []string{"BAUSTELLE"},
		},
		{
			Name:             "closure",
			Description:      "Road closures and blocked-road notices.",
			Aliases:          []string{"closed", "closures", "sperre", "sperren", "gesperrt"},
			UpstreamSubtypes: []string{"SPERRE"},
		},
		{
			Name:             "event",
			Description:      "Road impacts caused by public events.",
			Aliases:          []string{"events", "veranstaltung", "veranstaltungen"},
			UpstreamSubtypes: []string{"VERANSTALTUNG"},
		},
		{
			Name:             "traffic",
			Description:      "Traffic incidents, congestion, warnings, and general restrictions.",
			Aliases:          []string{"incident", "incidents", "stau", "unfall", "warning"},
			UpstreamSubtypes: []string{"AMPELREGELUNG", "FREI BEFAHRBAR", "SCHNEEFALL", "STAU", "UNFALL", "VORSICHT"},
		},
		{
			Name:             "mountain-pass",
			Description:      "Pass and mountain-road closures or restrictions.",
			Aliases:          []string{"mountain-pass-closure", "pass", "passes"},
			UpstreamSubtypes: []string{"SPERRE", "WINTERSPERRE"},
		},
		{
			Name:             "bike",
			Description:      "Cycle-route and bike-path closures.",
			Aliases:          []string{"cycle", "cycling", "radweg"},
			UpstreamSubtypes: []string{"RADWEG_SPERRE"},
		},
		{
			Name:             "radar",
			Description:      "Speed-control notices.",
			Aliases:          []string{"speed", "speed-control"},
			UpstreamSubtypes: []string{"RADARKONTROLLE"},
		},
	}
}

func validateTrafficZoneIDs(value string) error {
	known := map[string]struct{}{}
	for _, zone := range knownTrafficZones() {
		known[zone.ZoneID] = struct{}{}
	}
	for _, zoneID := range trafficZoneIDValues(value) {
		if _, ok := known[zoneID]; !ok {
			return fmt.Errorf("unknown traffic zone-id %q; run odh traffic zones", zoneID)
		}
	}
	return nil
}

func trafficZoneIDValues(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func resolveTrafficArea(value string) (trafficArea, error) {
	normalized := normalizeAreaAlias(value)
	if normalized == "" || normalized == "all" {
		return trafficArea{}, nil
	}
	areas := map[string]trafficArea{
		"bozen-unterland": {Name: "bozen-unterland", ZoneIDs: []string{"3"}},
		"ueberetsch-unterland": {
			Name:    "ueberetsch-unterland",
			ZoneIDs: []string{"3"},
			Keywords: []string{
				"Salurn", "Neumarkt", "Auer", "Montan", "Tramin", "Kurtatsch", "Margreid", "Laag", "Buchholz", "Gfrill",
				"Eppan", "Kaltern", "St. Pauls", "Unterrain", "Girlan", "Missian", "Montiggl", "Laimburg", "Mendel", "Sigmundskron", "Frangart",
			},
		},
		"unterland":            {Name: "unterland", ZoneIDs: []string{"3"}, Keywords: []string{"Salurn", "Neumarkt", "Auer", "Montan", "Tramin", "Kurtatsch", "Margreid", "Laag", "Buchholz", "Gfrill"}},
		"ueberetsch":           {Name: "ueberetsch", ZoneIDs: []string{"3"}, Keywords: []string{"Eppan", "Kaltern", "St. Pauls", "Unterrain", "Girlan", "Missian", "Montiggl", "Laimburg", "Mendel", "Sigmundskron", "Frangart"}},
		"bozen":                {Name: "bozen", ZoneIDs: []string{"3"}, Keywords: []string{"Bozen", "Bolzano", "Stadtgemeinde Bozen"}},
		"salurn":               {Name: "salurn", ZoneIDs: []string{"3"}, Keywords: []string{"Salurn"}},
		"kaltern":              {Name: "kaltern", ZoneIDs: []string{"3"}, Keywords: []string{"Kaltern"}},
		"tramin":               {Name: "tramin", ZoneIDs: []string{"3"}, Keywords: []string{"Tramin"}},
		"eppan":                {Name: "eppan", ZoneIDs: []string{"3"}, Keywords: []string{"Eppan", "St. Pauls", "Girlan", "Missian", "Unterrain", "Montiggl", "Frangart"}},
		"auer":                 {Name: "auer", ZoneIDs: []string{"3"}, Keywords: []string{"Auer"}},
		"neumarkt":             {Name: "neumarkt", ZoneIDs: []string{"3"}, Keywords: []string{"Neumarkt"}},
		"kurtatsch":            {Name: "kurtatsch", ZoneIDs: []string{"3"}, Keywords: []string{"Kurtatsch"}},
		"margreid":             {Name: "margreid", ZoneIDs: []string{"3"}, Keywords: []string{"Margreid"}},
		"montan":               {Name: "montan", ZoneIDs: []string{"3"}, Keywords: []string{"Montan"}},
		"burggrafenamt":        {Name: "burggrafenamt", ZoneIDs: []string{"2"}},
		"eisacktal-wipptal":    {Name: "eisacktal-wipptal", ZoneIDs: []string{"5"}},
		"pustertal":            {Name: "pustertal", ZoneIDs: []string{"6"}},
		"vinschgau":            {Name: "vinschgau", ZoneIDs: []string{"1"}},
		"salten-schlern":       {Name: "salten-schlern", ZoneIDs: []string{"4"}},
		"ausserhalb-suedtirol": {Name: "ausserhalb-suedtirol", ZoneIDs: []string{"7"}},
	}
	area, ok := areas[normalized]
	if !ok {
		return trafficArea{}, fmt.Errorf("unknown traffic area %q", value)
	}
	return area, nil
}

func parseTrafficDateRange(from, to string) (time.Time, time.Time, error) {
	fromDay, err := parseTrafficDate(from)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	toDay, err := parseTrafficDate(to)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if toDay.Before(fromDay) {
		return time.Time{}, time.Time{}, fmt.Errorf("--to must not be before --from")
	}
	return fromDay, toDay, nil
}

func parseTrafficDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q; use YYYY-MM-DD", value)
	}
	return parsed, nil
}

func parseNearRadius(near, radius string) (float64, float64, float64, error) {
	latText, lonText, ok := strings.Cut(strings.TrimSpace(near), ",")
	if !ok {
		return 0, 0, 0, fmt.Errorf("--near must use lat,lon")
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(latText), 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid latitude in --near: %w", err)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(lonText), 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid longitude in --near: %w", err)
	}
	radiusKM, err := parseRadiusKM(radius)
	if err != nil {
		return 0, 0, 0, err
	}
	return lat, lon, radiusKM, nil
}

func parseRadiusKM(value string) (float64, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimSuffix(value, "km")
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid radius %q", value)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("radius must be greater than zero")
	}
	return parsed, nil
}

func extractCoordinates(value any) []float64 {
	geometry, _ := value.(map[string]any)
	coordinates, _ := geometry["coordinates"].([]any)
	if len(coordinates) < 2 {
		return nil
	}
	lon, lonOK := numberValue(coordinates[0])
	lat, latOK := numberValue(coordinates[1])
	if !lonOK || !latOK {
		return nil
	}
	return []float64{lon, lat}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKM = 6371.0
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func startOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func endOfDay(value time.Time) time.Time {
	return startOfDay(value).Add(24*time.Hour - time.Nanosecond)
}

func normalizeRoad(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	value = strings.ReplaceAll(value, "LS / SP", "LS/SP")
	value = strings.ReplaceAll(value, "LS/ SP", "LS/SP")
	value = strings.ReplaceAll(value, "LS /SP", "LS/SP")
	return value
}

func compactRoadToken(value string) string {
	value = normalizeRoad(value)
	replacer := strings.NewReplacer(" ", "", "/", "", "-", "")
	return replacer.Replace(value)
}

func normalizeAreaAlias(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacements := map[string]string{
		"ü": "ue",
		"ö": "oe",
		"ä": "ae",
		"ß": "ss",
		"_": "-",
		" ": "-",
	}
	for old, newValue := range replacements {
		value = strings.ReplaceAll(value, old, newValue)
	}
	value = strings.Trim(value, "-")
	return value
}

func normalizeTrafficSearchText(value string) string {
	value = strings.ToLower(cleanTrafficText(value))
	replacer := strings.NewReplacer(
		"ä", "ae",
		"ö", "oe",
		"ü", "ue",
		"ß", "ss",
		"à", "a",
		"á", "a",
		"è", "e",
		"é", "e",
		"ì", "i",
		"í", "i",
		"ò", "o",
		"ó", "o",
		"ù", "u",
		"ú", "u",
		"/", " ",
		"-", " ",
		"_", " ",
		".", " ",
		",", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"?", " ",
		"!", " ",
	)
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func cleanTrafficText(value string) string {
	value = strings.ReplaceAll(value, "\\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}

func compactText(value string, max int) string {
	value = cleanTrafficText(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
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

func normalizeDedupText(value string) string {
	value = strings.ToLower(cleanTrafficText(value))
	return strings.Join(strings.Fields(value), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func textContainsPass(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "pass") || strings.Contains(value, "joch")
}

func escapeMarkdown(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
