// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/galjos/odh-cli/internal/output"
	"github.com/spf13/cobra"
)

func (r *Runner) newMobilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mobility",
		Short: "Curated Mobility API commands",
		RunE:  requireSubcommand,
	}

	// types
	var typesKind string
	var typesLimit int
	var typesFormat string
	var typesJSON bool
	typesCmd := &cobra.Command{
		Use:   "types",
		Short: "Discover Mobility type values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&typesFormat, typesJSON)
			format, err := normalizeOutputFormat(typesFormat)
			if err != nil {
				return err
			}
			if typesLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			path, normalizedKind, err := mobilityTypesPath(typesKind)
			if err != nil {
				return err
			}
			api, _ := r.Registry.Find("mobility")
			values := url.Values{}
			values.Set("limit", strconv.Itoa(typesLimit))
			requestURL, err := BuildURL(api.BaseURL, path, values)
			if err != nil {
				return err
			}
			value, err := r.fetchJSONValueCached(cmd.Context(), requestURL, 24*time.Hour)
			if err != nil {
				return err
			}
			items := extractDataList(value)
			return writeMobilityTypesOutput(cmd.OutOrStdout(), mobilityTypesOutput{
				Kind:   normalizedKind,
				Count:  len(items),
				Types:  items,
				Format: format,
			})
		},
	}
	typesCmd.Flags().StringVar(&typesKind, "kind", "station", "type kind: station, event, or edge")
	typesCmd.Flags().IntVar(&typesLimit, "limit", 200, "maximum number of types to request")
	typesCmd.Flags().StringVar(&typesFormat, "format", "table", "output format: json, table, or markdown")
	typesCmd.Flags().BoolVar(&typesJSON, "json", false, "shortcut for --format json")

	// origins
	var originsStationType string
	var originsRepresentation string
	var originsLimit int
	var originsParams []string
	originsCmd := &cobra.Command{
		Use:   "origins",
		Short: "Discover Mobility station origins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(originsStationType) == "" {
				return fmt.Errorf("--station-type is required")
			}
			if originsLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			api, _ := r.Registry.Find("mobility")
			path := fmt.Sprintf("/v2/%s/%s", url.PathEscape(originsRepresentation), url.PathEscape(originsStationType))
			path = strings.ReplaceAll(path, "%2C", ",")
			values := url.Values{}
			for _, p := range originsParams {
				key, value, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}
			values.Set("limit", strconv.Itoa(originsLimit))
			requestURL, err := BuildURL(api.BaseURL, path, values)
			if err != nil {
				return err
			}
			value, err := r.fetchJSONValueCached(cmd.Context(), requestURL, 24*time.Hour)
			if err != nil {
				return err
			}
			records := extractDataList(value)
			origins := summarizeOrigins(records)
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"station_type": originsStationType,
				"endpoint":     requestURL,
				"record_count": len(records),
				"count":        len(origins),
				"origins":      origins,
			})
		},
	}
	originsCmd.Flags().StringVar(&originsStationType, "station-type", "", "station type, for example TrafficSensor")
	originsCmd.Flags().StringVar(&originsRepresentation, "representation", "flat", "API representation")
	originsCmd.Flags().IntVar(&originsLimit, "limit", 1000, "maximum station records to inspect")
	originsCmd.Flags().StringArrayVar(&originsParams, "param", nil, "additional query parameter as key=value; repeatable; values may contain commas")

	// stations
	var stationsStationType string
	var stationsRepresentation string
	var stationsOrigin string
	var stationsLimit int
	var stationsOffset int
	var stationsWhere string
	var stationsParams []string
	stationsCmd := &cobra.Command{
		Use:   "stations",
		Short: "List Mobility stations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(stationsStationType) == "" {
				return fmt.Errorf("--station-type is required")
			}
			if stationsLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			if stationsOffset < 0 {
				return fmt.Errorf("--offset must not be negative")
			}
			api, _ := r.Registry.Find("mobility")
			path := fmt.Sprintf("/v2/%s/%s", url.PathEscape(stationsRepresentation), url.PathEscape(stationsStationType))
			path = strings.ReplaceAll(path, "%2C", ",")
			values := url.Values{}
			for _, p := range stationsParams {
				key, value, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}
			values.Set("limit", strconv.Itoa(stationsLimit))
			values.Set("offset", strconv.Itoa(stationsOffset))
			if strings.TrimSpace(stationsWhere) != "" {
				values.Set("where", stationsWhere)
			}
			requestURL, err := BuildURL(api.BaseURL, path, values)
			if err != nil {
				return err
			}
			value, err := r.fetchJSONValueCached(cmd.Context(), requestURL, 24*time.Hour)
			if err != nil {
				return err
			}
			records := extractDataList(value)
			stations := filterStationsByOrigin(records, stationsOrigin)
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"station_type": stationsStationType,
				"origin":       strings.TrimSpace(stationsOrigin),
				"record_count": len(records),
				"count":        len(stations),
				"stations":     stations,
			})
		},
	}
	stationsCmd.Flags().StringVar(&stationsStationType, "station-type", "", "station type, for example ParkingStation")
	stationsCmd.Flags().StringVar(&stationsRepresentation, "representation", "flat", "API representation")
	stationsCmd.Flags().StringVar(&stationsOrigin, "origin", "", "optional sorigin filter, for example A22")
	stationsCmd.Flags().IntVar(&stationsLimit, "limit", 20, "maximum stations to request")
	stationsCmd.Flags().IntVar(&stationsOffset, "offset", 0, "pagination offset")
	stationsCmd.Flags().StringVar(&stationsWhere, "where", "", "Open Data Hub where filter")
	stationsCmd.Flags().StringArrayVar(&stationsParams, "param", nil, "additional query parameter as key=value; repeatable; values may contain commas")

	// datatypes
	var datatypesStationType string
	var datatypesRepresentation string
	var datatypesOrigin string
	var datatypesLimit int
	var datatypesParams []string
	var datatypesFormat string
	var datatypesJSON bool
	datatypesCmd := &cobra.Command{
		Use:   "datatypes",
		Short: "Summarize Mobility data types",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&datatypesFormat, datatypesJSON)
			format, err := normalizeOutputFormat(datatypesFormat)
			if err != nil {
				return err
			}
			if strings.TrimSpace(datatypesStationType) == "" {
				return fmt.Errorf("--station-type is required")
			}
			if datatypesLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			api, _ := r.Registry.Find("mobility")
			path := fmt.Sprintf("/v2/%s/%s/*", url.PathEscape(datatypesRepresentation), url.PathEscape(datatypesStationType))
			path = strings.ReplaceAll(path, "%2C", ",")
			values := url.Values{}
			for _, p := range datatypesParams {
				key, value, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}
			values.Set("limit", strconv.Itoa(datatypesLimit))
			requestURL, err := BuildURL(api.BaseURL, path, values)
			if err != nil {
				return err
			}
			value, err := r.fetchJSONValueCached(cmd.Context(), requestURL, 24*time.Hour)
			if err != nil {
				return err
			}
			records := extractDataList(value)
			summary := summarizeDatatypes(records, datatypesOrigin)
			return writeMobilityDatatypesOutput(cmd.OutOrStdout(), mobilityDatatypesOutput{
				StationType: datatypesStationType,
				Origin:      strings.TrimSpace(datatypesOrigin),
				RecordCount: len(records),
				Count:       len(summary),
				Datatypes:   summary,
				Warnings:    mobilityDatatypeDiscoveryWarnings(datatypesStationType, summary, datatypesLimit, len(records)),
				Format:      format,
			})
		},
	}
	datatypesCmd.Flags().StringVar(&datatypesStationType, "station-type", "", "station type, for example TrafficSensor")
	datatypesCmd.Flags().StringVar(&datatypesRepresentation, "representation", "flat", "API representation")
	datatypesCmd.Flags().StringVar(&datatypesOrigin, "origin", "", "optional sorigin filter, for example A22")
	datatypesCmd.Flags().IntVar(&datatypesLimit, "limit", 1000, "maximum station/datatype records to inspect")
	datatypesCmd.Flags().StringArrayVar(&datatypesParams, "param", nil, "additional query parameter as key=value; repeatable; values may contain commas")
	datatypesCmd.Flags().StringVar(&datatypesFormat, "format", "table", "output format: json, table, or markdown")
	datatypesCmd.Flags().BoolVar(&datatypesJSON, "json", false, "shortcut for --format json")

	// events
	var eventsOrigin string
	var eventsLatest bool
	var eventsRepresentation string
	var eventsLimit int
	var eventsParams []string
	eventsCmd := &cobra.Command{
		Use:   "events",
		Short: "List Mobility events",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(eventsOrigin) == "" {
				return fmt.Errorf("--origin is required")
			}
			if eventsLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			api, _ := r.Registry.Find("mobility")
			path := fmt.Sprintf("/v2/%s,event/%s", url.PathEscape(eventsRepresentation), url.PathEscape(eventsOrigin))
			path = strings.ReplaceAll(path, "%2C", ",")
			if eventsLatest {
				path += "/latest"
			}
			values := url.Values{}
			for _, p := range eventsParams {
				key, value, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}
			values.Set("limit", strconv.Itoa(eventsLimit))
			requestURL, err := BuildURL(api.BaseURL, path, values)
			if err != nil {
				return err
			}
			value, err := r.fetchJSONValue(cmd.Context(), requestURL)
			if err != nil {
				return err
			}
			events := extractDataList(value)
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"origin": eventsOrigin,
				"latest": eventsLatest,
				"count":  len(events),
				"events": events,
			})
		},
	}
	eventsCmd.Flags().StringVar(&eventsOrigin, "origin", "", "event origin, for example A22")
	eventsCmd.Flags().BoolVar(&eventsLatest, "latest", true, "request latest events")
	eventsCmd.Flags().StringVar(&eventsRepresentation, "representation", "flat", "API representation")
	eventsCmd.Flags().IntVar(&eventsLimit, "limit", 20, "maximum events to request")
	eventsCmd.Flags().StringArrayVar(&eventsParams, "param", nil, "additional query parameter as key=value; repeatable; values may contain commas")

	// latest
	var latestStationType string
	var latestDataType string
	var latestRepresentation string
	var latestLimit int
	var latestRequestLimit int
	var latestOffset int
	var latestOrigin string
	var latestActive bool
	var latestFreshWithin string
	var latestSortMode string
	var latestWhere string
	var latestParams []string
	var latestFormat string
	var latestJSON bool
	latestCmd := &cobra.Command{
		Use:   "latest",
		Short: "Query latest Mobility time-series measurements",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&latestFormat, latestJSON)
			format, err := normalizeOutputFormat(latestFormat)
			if err != nil {
				return err
			}
			if strings.TrimSpace(latestStationType) == "" {
				return fmt.Errorf("--station-type is required")
			}
			if strings.TrimSpace(latestDataType) == "" {
				return fmt.Errorf("--data-type is required")
			}
			if latestLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			if latestOffset < 0 {
				return fmt.Errorf("--offset must not be negative")
			}
			if latestRequestLimit < 0 {
				return fmt.Errorf("--request-limit must not be negative")
			}
			sortModeValue, err := normalizeMobilityLatestSort(latestSortMode)
			if err != nil {
				return err
			}
			freshDuration, err := parseFreshWithin(latestFreshWithin)
			if err != nil {
				return err
			}
			api, _ := r.Registry.Find("mobility")
			path := fmt.Sprintf("/v2/%s/%s/%s/latest", url.PathEscape(latestRepresentation), url.PathEscape(latestStationType), url.PathEscape(latestDataType))
			path = strings.ReplaceAll(path, "%2C", ",")
			values := url.Values{}
			for _, p := range latestParams {
				key, value, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}
			localProcessing := mobilityLatestNeedsLocalProcessing(latestOrigin, latestActive, freshDuration, sortModeValue)
			upstreamLimit := latestLimit
			if localProcessing {
				upstreamLimit = latestRequestLimit
				if upstreamLimit == 0 {
					upstreamLimit = max(latestLimit, 1000)
				}
				if upstreamLimit < latestLimit {
					upstreamLimit = latestLimit
				}
			}
			values.Set("limit", strconv.Itoa(upstreamLimit))
			values.Set("offset", strconv.Itoa(latestOffset))
			if strings.TrimSpace(latestWhere) != "" {
				values.Set("where", latestWhere)
			}
			requestURL, err := BuildURL(api.BaseURL, path, values)
			if err != nil {
				return err
			}
			if localProcessing || format != "json" {
				value, err := r.fetchJSONValue(cmd.Context(), requestURL)
				if err != nil {
					return err
				}
				records := extractDataList(value)
				resultRequestLimit := 0
				if localProcessing {
					resultRequestLimit = upstreamLimit
				}
				result := filterMobilityLatest(records, mobilityLatestFilter{
					StationType:   latestStationType,
					DataType:      latestDataType,
					Origin:        latestOrigin,
					ActiveOnly:    latestActive,
					FreshWithin:   latestFreshWithin,
					FreshDuration: freshDuration,
					Sort:          sortModeValue,
					Limit:         latestLimit,
					RequestLimit:  resultRequestLimit,
					Endpoint:      requestURL,
					Now:           time.Now(),
				})
				return writeMobilityLatestOutput(cmd.OutOrStdout(), result, format)
			}
			return r.fetchJSONCobra(cmd.Context(), requestURL, cmd.OutOrStdout())
		},
	}
	latestCmd.Flags().StringVar(&latestStationType, "station-type", "", "station type, for example EChargingStation")
	latestCmd.Flags().StringVar(&latestDataType, "data-type", "", "data type, for example number-available")
	latestCmd.Flags().StringVar(&latestRepresentation, "representation", "flat,node", "API representation")
	latestCmd.Flags().IntVar(&latestLimit, "limit", 5, "number of measurements to request")
	latestCmd.Flags().IntVar(&latestRequestLimit, "request-limit", 0, "raw upstream rows to request before local filtering")
	latestCmd.Flags().IntVar(&latestOffset, "offset", 0, "pagination offset")
	latestCmd.Flags().StringVar(&latestOrigin, "origin", "", "optional sorigin filter, for example ALPERIA")
	latestCmd.Flags().BoolVar(&latestActive, "active", false, "keep only active stations")
	latestCmd.Flags().StringVar(&latestFreshWithin, "fresh-within", "", "keep only rows with mvalidtime within this age, for example 24h or 7d")
	latestCmd.Flags().StringVar(&latestSortMode, "sort", "upstream", "local sort: upstream, newest, oldest, or station")
	latestCmd.Flags().StringVar(&latestWhere, "where", "", "Open Data Hub where filter")
	latestCmd.Flags().StringArrayVar(&latestParams, "param", nil, "additional query parameter as key=value; repeatable; values may contain commas")
	latestCmd.Flags().StringVar(&latestFormat, "format", "json", "output format: json, table, or markdown")
	latestCmd.Flags().BoolVar(&latestJSON, "json", false, "shortcut for --format json")

	cmd.AddCommand(typesCmd)
	cmd.AddCommand(originsCmd)
	cmd.AddCommand(stationsCmd)
	cmd.AddCommand(datatypesCmd)
	cmd.AddCommand(eventsCmd)
	cmd.AddCommand(latestCmd)
	return cmd
}

func (r *Runner) newA22Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "a22",
		Short: "A22 Mobility commands",
		RunE:  requireSubcommand,
	}

	var statusLimit int
	var statusFormat string
	var statusJSON bool
	var statusRaw bool
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check A22 status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&statusFormat, statusJSON)
			format, err := normalizeOutputFormat(statusFormat)
			if err != nil {
				return err
			}
			if statusLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			api, _ := r.Registry.Find("mobility")
			events, eventsURL, err := r.fetchListFromMobility(cmd.Context(), api.BaseURL, "/v2/flat,event/A22/latest", statusLimit)
			if err != nil {
				return err
			}
			forecasts, forecastsURL, err := r.fetchListFromMobility(cmd.Context(), api.BaseURL, "/v2/flat/TrafficForecast/forecast/latest", statusLimit)
			if err != nil {
				return err
			}

			warnings := make([]string, 0)
			if len(events) == 0 {
				warnings = append(warnings, "Open Data Hub returned no current A22 events.")
			}
			for _, forecast := range forecasts {
				validTime := parseODHTime(asString(forecast["mvalidtime"]))
				if validTime != nil && validTime.After(time.Now().Add(24*time.Hour)) {
					warnings = append(warnings, "A22 forecast contains future valid_time values; treat it as forecast data, not current incident data.")
					break
				}
			}

			return writeA22StatusOutput(cmd.OutOrStdout(), a22StatusOutput{
				Source: "Open Data Hub Mobility API",
				Events: a22FeedOutput{
					Endpoint: eventsURL,
					Count:    len(events),
					Summary:  summarizeA22Records(events),
					Items:    rawItemsWhen(statusRaw, events),
				},
				Forecast: a22FeedOutput{
					Endpoint: forecastsURL,
					Count:    len(forecasts),
					Summary:  summarizeA22Records(forecasts),
					Items:    rawItemsWhen(statusRaw, forecasts),
				},
				Warnings: warnings,
				Format:   format,
			})
		},
	}
	statusCmd.Flags().IntVar(&statusLimit, "limit", 20, "maximum records to request from each A22 feed")
	statusCmd.Flags().StringVar(&statusFormat, "format", "table", "output format: json, table, or markdown")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "shortcut for --format json")
	statusCmd.Flags().BoolVar(&statusRaw, "raw", false, "include raw upstream event and forecast rows in JSON output")

	cmd.AddCommand(statusCmd)
	return cmd
}

func mobilityTypesPath(kind string) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "station", "stations":
		return "/v2/flat", "station", nil
	case "event", "events":
		return "/v2/flat,event", "event", nil
	case "edge", "edges":
		return "/v2/flat,edge", "edge", nil
	default:
		return "", "", fmt.Errorf("unsupported mobility type kind %q", kind)
	}
}

type datatypeSummary struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Unit         string   `json:"unit,omitempty"`
	StationCount int      `json:"station_count"`
	Origins      []string `json:"origins,omitempty"`
}

type originSummary struct {
	Name           string   `json:"name"`
	StationCount   int      `json:"station_count"`
	StationSamples []string `json:"station_samples,omitempty"`
}

type mobilityLatestFilter struct {
	StationType   string
	DataType      string
	Origin        string
	ActiveOnly    bool
	FreshWithin   string
	FreshDuration time.Duration
	Sort          string
	Limit         int
	RequestLimit  int
	Endpoint      string
	Now           time.Time
}

type mobilityLatestResult struct {
	Source       string           `json:"source"`
	SourceDetail string           `json:"source_detail"`
	StationType  string           `json:"station_type"`
	DataType     string           `json:"data_type"`
	Origin       string           `json:"origin,omitempty"`
	ActiveOnly   bool             `json:"active_only,omitempty"`
	FreshWithin  string           `json:"fresh_within,omitempty"`
	Sort         string           `json:"sort"`
	Endpoint     string           `json:"endpoint"`
	RawCount     int              `json:"raw_count"`
	Count        int              `json:"count"`
	Measurements []map[string]any `json:"measurements"`
	Warnings     []string         `json:"warnings,omitempty"`
}

type mobilityTypesOutput struct {
	Kind   string           `json:"kind"`
	Count  int              `json:"count"`
	Types  []map[string]any `json:"types"`
	Format string           `json:"-"`
}

type mobilityDatatypesOutput struct {
	StationType string            `json:"station_type"`
	Origin      string            `json:"origin,omitempty"`
	RecordCount int               `json:"record_count"`
	Count       int               `json:"count"`
	Datatypes   []datatypeSummary `json:"datatypes"`
	Warnings    []string          `json:"warnings,omitempty"`
	Format      string            `json:"-"`
}

type a22RecordSummary struct {
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
	ValidTime string `json:"valid_time,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type a22FeedOutput struct {
	Endpoint string             `json:"endpoint"`
	Count    int                `json:"count"`
	Summary  []a22RecordSummary `json:"summary,omitempty"`
	Items    []map[string]any   `json:"items,omitempty"`
}

type a22StatusOutput struct {
	Source   string        `json:"source"`
	Events   a22FeedOutput `json:"events"`
	Forecast a22FeedOutput `json:"forecast"`
	Warnings []string      `json:"warnings,omitempty"`
	Format   string        `json:"-"`
}

func writeA22StatusOutput(stdout io.Writer, result a22StatusOutput) error {
	switch result.Format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "FEED\tNAME\tVALUE\tVALID_TIME")
		if len(result.Events.Summary) == 0 {
			fmt.Fprintln(tw, "events\t-\tno current events\t-")
		}
		for _, item := range result.Events.Summary {
			fmt.Fprintf(tw, "events\t%s\t%s\t%s\n", item.Name, item.Value, item.ValidTime)
		}
		for _, item := range result.Forecast.Summary {
			fmt.Fprintf(tw, "forecast\t%s\t%s\t%s\n", item.Name, item.Value, item.ValidTime)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		writePlainWarnings(stdout, result.Warnings)
		return nil
	case "markdown":
		fmt.Fprintln(stdout, "| feed | name | value | valid_time |")
		fmt.Fprintln(stdout, "| --- | --- | --- | --- |")
		if len(result.Events.Summary) == 0 {
			fmt.Fprintln(stdout, "| events | - | no current events | - |")
		}
		for _, item := range result.Events.Summary {
			fmt.Fprintf(stdout, "| events | %s | %s | %s |\n",
				escapeMarkdown(item.Name),
				escapeMarkdown(item.Value),
				escapeMarkdown(item.ValidTime),
			)
		}
		for _, item := range result.Forecast.Summary {
			fmt.Fprintf(stdout, "| forecast | %s | %s | %s |\n",
				escapeMarkdown(item.Name),
				escapeMarkdown(item.Value),
				escapeMarkdown(item.ValidTime),
			)
		}
		writeMarkdownWarnings(stdout, result.Warnings)
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.Format)
	}
}

func summarizeA22Records(records []map[string]any) []a22RecordSummary {
	summaries := make([]a22RecordSummary, 0, len(records))
	for _, record := range records {
		summaries = append(summaries, a22RecordSummary{
			Name:      firstNonEmpty(asString(record["sname"]), asString(record["scode"])),
			Value:     anyToString(record["mvalue"]),
			ValidTime: asString(record["mvalidtime"]),
			Timestamp: asString(record["_timestamp"]),
		})
	}
	return summaries
}

func rawItemsWhen(enabled bool, records []map[string]any) []map[string]any {
	if !enabled {
		return nil
	}
	return records
}

func writeMobilityTypesOutput(stdout io.Writer, result mobilityTypesOutput) error {
	switch result.Format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tDESCRIPTION")
		for _, item := range result.Types {
			fmt.Fprintf(tw, "%s\t%s\n", asString(item["id"]), compactText(asString(item["description"]), 80))
		}
		return tw.Flush()
	case "markdown":
		fmt.Fprintln(stdout, "| id | description |")
		fmt.Fprintln(stdout, "| --- | --- |")
		for _, item := range result.Types {
			fmt.Fprintf(stdout, "| %s | %s |\n",
				escapeMarkdown(asString(item["id"])),
				escapeMarkdown(compactText(asString(item["description"]), 80)),
			)
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.Format)
	}
}

func writeMobilityDatatypesOutput(stdout io.Writer, result mobilityDatatypesOutput) error {
	switch result.Format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "NAME\tDESCRIPTION\tUNIT\tSTATIONS\tORIGINS")
		for _, item := range result.Datatypes {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n",
				item.Name,
				compactText(item.Description, 72),
				item.Unit,
				item.StationCount,
				compactText(strings.Join(item.Origins, ","), 72),
			)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		writePlainWarnings(stdout, result.Warnings)
		return nil
	case "markdown":
		fmt.Fprintln(stdout, "| name | description | unit | stations | origins |")
		fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- |")
		for _, item := range result.Datatypes {
			fmt.Fprintf(stdout, "| %s | %s | %s | %d | %s |\n",
				escapeMarkdown(item.Name),
				escapeMarkdown(compactText(item.Description, 72)),
				escapeMarkdown(item.Unit),
				item.StationCount,
				escapeMarkdown(compactText(strings.Join(item.Origins, ", "), 72)),
			)
		}
		writeMarkdownWarnings(stdout, result.Warnings)
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.Format)
	}
}

func writeMobilityLatestOutput(stdout io.Writer, result mobilityLatestResult, format string) error {
	switch format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "STATION\tVALUE\tVALID_TIME\tORIGIN\tACTIVE")
		for _, record := range result.Measurements {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%t\n",
				firstNonEmpty(asString(record["sname"]), asString(record["scode"])),
				compactText(anyToString(record["mvalue"]), 40),
				firstNonEmpty(asString(record["mvalidtime"]), asString(record["_timestamp"])),
				asString(record["sorigin"]),
				asBool(record["sactive"]),
			)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		writePlainWarnings(stdout, result.Warnings)
		return nil
	case "markdown":
		fmt.Fprintln(stdout, "| station | value | valid_time | origin | active |")
		fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- |")
		for _, record := range result.Measurements {
			fmt.Fprintf(stdout, "| %s | %s | %s | %s | %t |\n",
				escapeMarkdown(firstNonEmpty(asString(record["sname"]), asString(record["scode"]))),
				escapeMarkdown(compactText(anyToString(record["mvalue"]), 40)),
				escapeMarkdown(firstNonEmpty(asString(record["mvalidtime"]), asString(record["_timestamp"]))),
				escapeMarkdown(asString(record["sorigin"])),
				asBool(record["sactive"]),
			)
		}
		writeMarkdownWarnings(stdout, result.Warnings)
		return nil
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func mobilityDatatypeDiscoveryWarnings(stationType string, summaries []datatypeSummary, limit, recordCount int) []string {
	warnings := []string{}
	if len(summaries) == 0 {
		warnings = append(warnings, fmt.Sprintf("no datatype rows matched; inspect raw station records with odh mobility stations --station-type %s --limit 5 --json", strings.TrimSpace(stationType)))
	}
	if limit > 0 && limit < 1000 && recordCount >= limit {
		warnings = append(warnings, fmt.Sprintf("inspected %d records because --limit=%d; rerun with --limit 1000 when datatype completeness matters", recordCount, limit))
	}
	return warnings
}

func mobilityLatestDatatypeHints(stationType, dataType string) []string {
	stationType = strings.ToLower(strings.TrimSpace(stationType))
	dataType = strings.ToLower(strings.TrimSpace(dataType))
	switch {
	case stationType == "parkingstation" && (dataType == "number-free" || dataType == "number-available" || dataType == "free-slots" || dataType == "available"):
		return []string{`ParkingStation current availability is usually data type "free"; discover valid names with odh mobility datatypes --station-type ParkingStation --format table`}
	case stationType == "echargingstation" && (dataType == "free" || dataType == "available"):
		return []string{`EChargingStation availability is usually data type "number-available"; discover valid names with odh mobility datatypes --station-type EChargingStation --format table`}
	default:
		return nil
	}
}

func anyToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func summarizeDatatypes(records []map[string]any, originFilter string) []datatypeSummary {
	type aggregate struct {
		datatypeSummary
		stations map[string]struct{}
		origins  map[string]struct{}
	}
	filter := strings.TrimSpace(originFilter)
	byName := map[string]*aggregate{}
	for _, record := range records {
		if filter != "" && asString(record["sorigin"]) != filter {
			continue
		}
		name := asString(record["tname"])
		if name == "" {
			continue
		}
		current, ok := byName[name]
		if !ok {
			current = &aggregate{
				datatypeSummary: datatypeSummary{
					Name:        name,
					Description: asString(record["tdescription"]),
					Unit:        asString(record["tunit"]),
				},
				stations: map[string]struct{}{},
				origins:  map[string]struct{}{},
			}
			byName[name] = current
		}
		if code := asString(record["scode"]); code != "" {
			current.stations[code] = struct{}{}
		}
		if origin := asString(record["sorigin"]); origin != "" {
			current.origins[origin] = struct{}{}
		}
	}

	summaries := make([]datatypeSummary, 0, len(byName))
	for _, current := range byName {
		current.StationCount = len(current.stations)
		current.Origins = sortedKeys(current.origins)
		summaries = append(summaries, current.datatypeSummary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})
	return summaries
}

func mobilityLatestNeedsLocalProcessing(origin string, active bool, freshDuration time.Duration, sortMode string) bool {
	return strings.TrimSpace(origin) != "" || active || freshDuration > 0 || sortMode != "upstream"
}

func filterMobilityLatest(records []map[string]any, filter mobilityLatestFilter) mobilityLatestResult {
	now := filter.Now
	if now.IsZero() {
		now = time.Now()
	}
	origin := strings.TrimSpace(filter.Origin)
	matched := make([]map[string]any, 0, len(records))
	filteredOrigin := 0
	filteredInactive := 0
	filteredStale := 0
	for _, record := range records {
		if origin != "" && !strings.EqualFold(asString(record["sorigin"]), origin) {
			filteredOrigin++
			continue
		}
		if filter.ActiveOnly && !asBool(record["sactive"]) {
			filteredInactive++
			continue
		}
		if filter.FreshDuration > 0 {
			validTime := parseODHTime(asString(record["mvalidtime"]))
			if validTime == nil || validTime.Before(now.Add(-filter.FreshDuration)) {
				filteredStale++
				continue
			}
		}
		matched = append(matched, record)
	}
	sortMobilityLatest(matched, filter.Sort)
	if filter.Limit > 0 && len(matched) > filter.Limit {
		matched = matched[:filter.Limit]
	}
	warnings := make([]string, 0)
	if filteredOrigin > 0 {
		warnings = append(warnings, fmt.Sprintf("%d rows were hidden by --origin", filteredOrigin))
	}
	if filteredInactive > 0 {
		warnings = append(warnings, fmt.Sprintf("%d inactive rows were hidden by --active", filteredInactive))
	}
	if filteredStale > 0 {
		warnings = append(warnings, fmt.Sprintf("%d stale rows were hidden by --fresh-within", filteredStale))
	}
	if filter.RequestLimit > 0 && len(records) >= filter.RequestLimit {
		warnings = append(warnings, fmt.Sprintf("inspected %d upstream rows; increase --request-limit if filters may hide later rows", filter.RequestLimit))
	}
	if len(matched) == 0 && len(warnings) > 0 {
		warnings = append(warnings, "local filters matched no rows from the inspected upstream response")
	}
	if len(matched) == 0 {
		warnings = append(warnings, mobilityLatestDatatypeHints(filter.StationType, filter.DataType)...)
	}
	return mobilityLatestResult{
		Source:       "Open Data Hub Mobility API",
		SourceDetail: "latest Mobility time-series measurements",
		StationType:  filter.StationType,
		DataType:     filter.DataType,
		Origin:       origin,
		ActiveOnly:   filter.ActiveOnly,
		FreshWithin:  strings.TrimSpace(filter.FreshWithin),
		Sort:         filter.Sort,
		Endpoint:     filter.Endpoint,
		RawCount:     len(records),
		Count:        len(matched),
		Measurements: matched,
		Warnings:     warnings,
	}
}

func sortMobilityLatest(records []map[string]any, sortMode string) {
	switch sortMode {
	case "newest":
		sort.SliceStable(records, func(i, j int) bool {
			return mobilityValidTime(records[i]).After(mobilityValidTime(records[j]))
		})
	case "oldest":
		sort.SliceStable(records, func(i, j int) bool {
			return mobilityValidTime(records[i]).Before(mobilityValidTime(records[j]))
		})
	case "station":
		sort.SliceStable(records, func(i, j int) bool {
			if asString(records[i]["sname"]) != asString(records[j]["sname"]) {
				return asString(records[i]["sname"]) < asString(records[j]["sname"])
			}
			return asString(records[i]["scode"]) < asString(records[j]["scode"])
		})
	}
}

func mobilityValidTime(record map[string]any) time.Time {
	parsed := parseODHTime(asString(record["mvalidtime"]))
	if parsed == nil {
		return time.Time{}
	}
	return *parsed
}

func normalizeMobilityLatestSort(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "upstream", "none":
		return "upstream", nil
	case "new", "newest", "newest-first", "desc", "valid-time-desc":
		return "newest", nil
	case "old", "oldest", "oldest-first", "asc", "valid-time-asc":
		return "oldest", nil
	case "station", "station-name":
		return "station", nil
	default:
		return "", fmt.Errorf("unsupported mobility latest sort %q", value)
	}
}

func parseFreshWithin(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, nil
	}
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "d")), 64)
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid --fresh-within %q; use a positive duration like 24h or 7d", value)
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid --fresh-within %q; use a positive duration like 24h or 7d", value)
	}
	return duration, nil
}

func summarizeOrigins(records []map[string]any) []originSummary {
	type aggregate struct {
		originSummary
		stations map[string]struct{}
	}
	byName := map[string]*aggregate{}
	for _, record := range records {
		name := strings.TrimSpace(asString(record["sorigin"]))
		if name == "" {
			continue
		}
		current, ok := byName[name]
		if !ok {
			current = &aggregate{
				originSummary: originSummary{Name: name},
				stations:      map[string]struct{}{},
			}
			byName[name] = current
		}
		if code := asString(record["scode"]); code != "" {
			current.stations[code] = struct{}{}
		}
	}
	origins := make([]originSummary, 0, len(byName))
	for _, current := range byName {
		samples := sortedKeys(current.stations)
		current.StationCount = len(samples)
		if len(samples) > 5 {
			samples = samples[:5]
		}
		current.StationSamples = samples
		origins = append(origins, current.originSummary)
	}
	sort.Slice(origins, func(i, j int) bool {
		return origins[i].Name < origins[j].Name
	})
	return origins
}

func extractDataList(value any) []map[string]any {
	switch typed := value.(type) {
	case []any:
		return mapsFromList(typed)
	case map[string]any:
		if data, ok := typed["data"].([]any); ok {
			return mapsFromList(data)
		}
	}
	return nil
}

func extractItemsList(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case map[string]any:
		if data, ok := typed["Items"].([]any); ok {
			return data
		}
		if data, ok := typed["data"].([]any); ok {
			return data
		}
	}
	return nil
}

func filterStationsByOrigin(records []map[string]any, originFilter string) []map[string]any {
	filter := strings.TrimSpace(originFilter)
	if filter == "" {
		return records
	}
	filtered := make([]map[string]any, 0, len(records))
	for _, record := range records {
		if asString(record["sorigin"]) == filter {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func mapsFromList(items []any) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if object, ok := item.(map[string]any); ok {
			result = append(result, object)
		}
	}
	return result
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func asBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y":
			return true
		default:
			return false
		}
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func parseODHTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02 15:04:05.000-0700",
		"2006-01-02 15:04:05-0700",
		"2006-01-02T15:04:05",
		"2006-01-02",
		time.RFC3339,
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}
