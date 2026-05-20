// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/galjos/odh-cli/internal/output"
	"github.com/spf13/cobra"
)

const (
	defaultTransitWindow = 15 * time.Minute
	defaultGTFSCacheTTL  = 24 * time.Hour
	gtfsDownloadTimeout  = 2 * time.Minute
	maxGTFSArchiveBytes  = 200 * 1024 * 1024
)

type gtfsStop struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Lat           float64 `json:"lat,omitempty"`
	Lon           float64 `json:"lon,omitempty"`
	ParentStation string  `json:"parent_station,omitempty"`
}

type gtfsRoute struct {
	ID        string `json:"id"`
	ShortName string `json:"short_name,omitempty"`
	LongName  string `json:"long_name,omitempty"`
	Type      string `json:"type,omitempty"`
}

type gtfsTrip struct {
	ID          string `json:"id"`
	RouteID     string `json:"route_id"`
	ServiceID   string `json:"service_id"`
	Headsign    string `json:"headsign,omitempty"`
	DirectionID string `json:"direction_id,omitempty"`
}

type gtfsStopTime struct {
	TripID        string `json:"trip_id"`
	ArrivalTime   string `json:"arrival_time"`
	DepartureTime string `json:"departure_time"`
	StopID        string `json:"stop_id"`
	StopSequence  int    `json:"stop_sequence"`
}

type transitDeparture struct {
	TripID         string `json:"trip_id"`
	RouteID        string `json:"route_id"`
	RouteShortName string `json:"route_short_name,omitempty"`
	RouteLongName  string `json:"route_long_name,omitempty"`
	RouteType      string `json:"route_type,omitempty"`
	Headsign       string `json:"headsign,omitempty"`
	DirectionID    string `json:"direction_id,omitempty"`
	StopID         string `json:"stop_id"`
	StopName       string `json:"stop_name"`
	ArrivalTime    string `json:"arrival_time"`
	DepartureTime  string `json:"departure_time"`
	StopSequence   int    `json:"stop_sequence"`
}

type transitTripMatch struct {
	TripID         string            `json:"trip_id"`
	RouteID        string            `json:"route_id"`
	RouteShortName string            `json:"route_short_name,omitempty"`
	RouteLongName  string            `json:"route_long_name,omitempty"`
	RouteType      string            `json:"route_type,omitempty"`
	Headsign       string            `json:"headsign,omitempty"`
	DirectionID    string            `json:"direction_id,omitempty"`
	From           transitStopOnTrip `json:"from"`
	To             transitStopOnTrip `json:"to"`
	TransferCount  int               `json:"transfer_count"`
}

type transitStopOnTrip struct {
	StopID        string `json:"stop_id"`
	StopName      string `json:"stop_name"`
	ArrivalTime   string `json:"arrival_time,omitempty"`
	DepartureTime string `json:"departure_time,omitempty"`
	StopSequence  int    `json:"stop_sequence"`
}

type gtfsArchiveInfo struct {
	Dataset  string `json:"dataset"`
	Endpoint string `json:"endpoint"`
	Path     string `json:"path,omitempty"`
	Cached   bool   `json:"cached"`
}

type transitStopSelector struct {
	Query string
	ID    string
}

func (r *Runner) newTransitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transit",
		Short: "GTFS-based transit routing and search",
		RunE:  requireSubcommand,
	}

	// stops
	stopsCmd := &cobra.Command{
		Use:   "stops",
		Short: "GTFS stop commands",
		RunE:  requireSubcommand,
	}
	var searchDataset string
	var searchLimit int
	var searchCacheDir string
	var searchRefresh bool
	var searchFormat string
	var searchJSON bool
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search GTFS stops by name",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&searchFormat, searchJSON)
			format, err := normalizeOutputFormat(searchFormat)
			if err != nil {
				return err
			}
			if searchLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			query := strings.Join(args, " ")
			archive, err := r.fetchGTFSArchive(cmd.Context(), searchDataset, searchCacheDir, searchRefresh)
			if err != nil {
				return err
			}
			stops, err := readGTFSStops(archive.Path)
			if err != nil {
				return err
			}
			matches := searchGTFSStops(stops, query, searchLimit)
			return writeTransitStopsSearchOutput(cmd.OutOrStdout(), transitStopsSearchOutput{
				Dataset: searchDataset,
				Query:   query,
				Archive: archive,
				Count:   len(matches),
				Stops:   matches,
				Format:  format,
			})
		},
	}
	searchCmd.Flags().StringVar(&searchDataset, "dataset", defaultGTFSDataset, "GTFS dataset id")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "maximum stops to return")
	searchCmd.Flags().StringVar(&searchCacheDir, "cache-dir", "", "directory for cached GTFS archives")
	searchCmd.Flags().BoolVar(&searchRefresh, "refresh", false, "refresh cached GTFS archive")
	searchCmd.Flags().StringVar(&searchFormat, "format", "table", "output format: json, table, or markdown")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "shortcut for --format json")
	stopsCmd.AddCommand(searchCmd)

	// departures
	var depDataset string
	var depStopQuery string
	var depStopID string
	var depDate string
	var depAround string
	var depWindow string
	var depMode string
	var depLimit int
	var depCacheDir string
	var depRefresh bool
	var depFormat string
	var depJSON bool
	departuresCmd := &cobra.Command{
		Use:   "departures",
		Short: "List departures from a stop",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&depFormat, depJSON)
			format, err := normalizeOutputFormat(depFormat)
			if err != nil {
				return err
			}
			selector := transitStopSelector{Query: depStopQuery, ID: depStopID}
			if err := selector.validate("stop", "stop-id"); err != nil {
				return err
			}
			if depLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			query, err := parseTransitTimeQuery(depDate, depAround, depWindow)
			if err != nil {
				return err
			}
			routeTypes, err := transitModeRouteTypes(depMode)
			if err != nil {
				return err
			}
			archive, err := r.fetchGTFSArchive(cmd.Context(), depDataset, depCacheDir, depRefresh)
			if err != nil {
				return err
			}
			result, err := findTransitDepartures(archive.Path, selector, query, routeTypes, depLimit)
			if err != nil {
				return err
			}
			warnings := appendTransitStopMatchWarning(nil, "stop", "stop-id", selector, result.StopMatchMode, len(result.Stops))
			return writeTransitDeparturesOutput(cmd.OutOrStdout(), transitDeparturesOutput{
				Dataset:       depDataset,
				StopQuery:     depStopQuery,
				StopID:        depStopID,
				StopMatchMode: result.StopMatchMode,
				Date:          query.Date.Format("2006-01-02"),
				Around:        query.AroundText,
				Window:        query.Window.String(),
				Mode:          normalizeTransitModeName(depMode),
				Archive:       archive,
				MatchedStops:  result.Stops,
				Count:         len(result.Departures),
				Departures:    result.Departures,
				Warnings:      warnings,
				Format:        format,
			})
		},
	}
	departuresCmd.Flags().StringVar(&depDataset, "dataset", defaultGTFSDataset, "GTFS dataset id")
	departuresCmd.Flags().StringVar(&depStopQuery, "stop", "", "stop search query")
	departuresCmd.Flags().StringVar(&depStopID, "stop-id", "", "exact GTFS stop_id or parent_station")
	departuresCmd.Flags().StringVar(&depDate, "date", time.Now().Format("2006-01-02"), "service date YYYY-MM-DD")
	departuresCmd.Flags().StringVar(&depAround, "around", "", "departure time HH:MM to search around")
	departuresCmd.Flags().StringVar(&depWindow, "window", defaultTransitWindow.String(), "time window around --around, for example 15m")
	departuresCmd.Flags().StringVar(&depMode, "mode", "all", "mode filter: all, train, bus, or cable-car")
	departuresCmd.Flags().IntVar(&depLimit, "limit", 20, "maximum departures to return")
	departuresCmd.Flags().StringVar(&depCacheDir, "cache-dir", "", "directory for cached GTFS archives")
	departuresCmd.Flags().BoolVar(&depRefresh, "refresh", false, "refresh cached GTFS archive")
	departuresCmd.Flags().StringVar(&depFormat, "format", "table", "output format: json, table, or markdown")
	departuresCmd.Flags().BoolVar(&depJSON, "json", false, "shortcut for --format json")

	// trip
	var tripDataset string
	var tripFromQuery string
	var tripFromStopID string
	var tripToQuery string
	var tripToStopID string
	var tripDate string
	var tripTime string
	var tripWindow string
	var tripMode string
	var tripLimit int
	var tripCacheDir string
	var tripRefresh bool
	var tripFormat string
	var tripJSON bool
	tripCmd := &cobra.Command{
		Use:   "trip",
		Short: "Search direct trips between stops",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&tripFormat, tripJSON)
			format, err := normalizeOutputFormat(tripFormat)
			if err != nil {
				return err
			}
			fromSelector := transitStopSelector{Query: tripFromQuery, ID: tripFromStopID}
			toSelector := transitStopSelector{Query: tripToQuery, ID: tripToStopID}
			if err := fromSelector.validate("from", "from-stop-id"); err != nil {
				return err
			}
			if err := toSelector.validate("to", "to-stop-id"); err != nil {
				return err
			}
			if strings.TrimSpace(tripTime) == "" {
				return fmt.Errorf("--time is required")
			}
			if tripLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			query, err := parseTransitTimeQuery(tripDate, tripTime, tripWindow)
			if err != nil {
				return err
			}
			routeTypes, err := transitModeRouteTypes(tripMode)
			if err != nil {
				return err
			}
			archive, err := r.fetchGTFSArchive(cmd.Context(), tripDataset, tripCacheDir, tripRefresh)
			if err != nil {
				return err
			}
			result, err := findTransitTripMatches(archive.Path, fromSelector, toSelector, query, routeTypes, tripLimit)
			if err != nil {
				return err
			}
			warnings := make([]string, 0)
			if len(result.Matches) == 0 {
				warnings = append(warnings, "no direct GTFS trip matched; this command does not perform transfer routing")
			}
			warnings = appendTransitStopMatchWarning(warnings, "from", "from-stop-id", fromSelector, result.FromMatchMode, len(result.FromStops))
			warnings = appendTransitStopMatchWarning(warnings, "to", "to-stop-id", toSelector, result.ToMatchMode, len(result.ToStops))
			warnings = append(warnings, "historical delay probability is not available from the live GTFS API without an archived GTFS-RT snapshot dataset")
			return writeTransitTripOutput(cmd.OutOrStdout(), transitTripOutput{
				Dataset:       tripDataset,
				FromQuery:     tripFromQuery,
				FromStopID:    tripFromStopID,
				FromMatchMode: result.FromMatchMode,
				ToQuery:       tripToQuery,
				ToStopID:      tripToStopID,
				ToMatchMode:   result.ToMatchMode,
				Date:          query.Date.Format("2006-01-02"),
				Time:          query.AroundText,
				Window:        query.Window.String(),
				Mode:          normalizeTransitModeName(tripMode),
				Archive:       archive,
				FromStops:     result.FromStops,
				ToStops:       result.ToStops,
				Count:         len(result.Matches),
				Matches:       result.Matches,
				Warnings:      warnings,
				Format:        format,
			})
		},
	}
	tripCmd.Flags().StringVar(&tripDataset, "dataset", defaultGTFSDataset, "GTFS dataset id")
	tripCmd.Flags().StringVar(&tripFromQuery, "from", "", "origin stop query")
	tripCmd.Flags().StringVar(&tripFromStopID, "from-stop-id", "", "exact origin GTFS stop_id or parent_station")
	tripCmd.Flags().StringVar(&tripToQuery, "to", "", "destination stop query")
	tripCmd.Flags().StringVar(&tripToStopID, "to-stop-id", "", "exact destination GTFS stop_id or parent_station")
	tripCmd.Flags().StringVar(&tripDate, "date", time.Now().Format("2006-01-02"), "service date YYYY-MM-DD")
	tripCmd.Flags().StringVar(&tripTime, "time", "", "origin departure time HH:MM")
	tripCmd.Flags().StringVar(&tripWindow, "window", defaultTransitWindow.String(), "time window around --time, for example 15m")
	tripCmd.Flags().StringVar(&tripMode, "mode", "all", "mode filter: all, train, bus, or cable-car")
	tripCmd.Flags().IntVar(&tripLimit, "limit", 20, "maximum direct trip matches to return")
	tripCmd.Flags().StringVar(&tripCacheDir, "cache-dir", "", "directory for cached GTFS archives")
	tripCmd.Flags().BoolVar(&tripRefresh, "refresh", false, "refresh cached GTFS archive")
	tripCmd.Flags().StringVar(&tripFormat, "format", "table", "output format: json, table, or markdown")
	tripCmd.Flags().BoolVar(&tripJSON, "json", false, "shortcut for --format json")

	// delay-stats
	var dsFrom string
	var depTo string
	var dsTime string
	var dsWeekday string
	var dsSince string
	var dsFormat string
	var dsJSON bool
	delayStatsCmd := &cobra.Command{
		Use:     "delay-stats",
		Aliases: []string{"delay-probability"},
		Short:   "Historical delay statistics",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&dsFormat, dsJSON)
			format, err := normalizeOutputFormat(dsFormat)
			if err != nil {
				return err
			}
			return writeTransitDelayStatsOutput(cmd.OutOrStdout(), transitDelayStatsOutput{
				Supported: false,
				Reason:    "Open Data Hub GTFS exposes current static GTFS and live GTFS-RT feeds, but this CLI has no historical GTFS-RT archive to compute probabilities from.",
				Requested: map[string]string{
					"from":    strings.TrimSpace(dsFrom),
					"to":      strings.TrimSpace(depTo),
					"time":    strings.TrimSpace(dsTime),
					"weekday": strings.TrimSpace(dsWeekday),
					"since":   strings.TrimSpace(dsSince),
				},
				AvailableNow: []string{
					"odh gtfs realtime --dataset sta-time-tables --feed trip-updates",
					"odh transit trip --from <stop> --to <stop> --date YYYY-MM-DD --time HH:MM",
					"odh transit departures --stop <stop> --date YYYY-MM-DD --around HH:MM",
				},
				NextStep: "add an explicit archive collector for GTFS-RT trip-updates before reporting delay probability or usual delay minutes",
				Format:   format,
			})
		},
	}
	delayStatsCmd.Flags().StringVar(&dsFrom, "from", "", "origin stop query")
	delayStatsCmd.Flags().StringVar(&depTo, "to", "", "destination stop query")
	delayStatsCmd.Flags().StringVar(&dsTime, "time", "", "origin departure time HH:MM")
	delayStatsCmd.Flags().StringVar(&dsWeekday, "weekday", "", "weekday filter, for example saturday")
	delayStatsCmd.Flags().StringVar(&dsSince, "since", "", "archive range, for example 90d")
	delayStatsCmd.Flags().StringVar(&dsFormat, "format", "table", "output format: json, table, or markdown")
	delayStatsCmd.Flags().BoolVar(&dsJSON, "json", false, "shortcut for --format json")

	cmd.AddCommand(stopsCmd)
	cmd.AddCommand(departuresCmd)
	cmd.AddCommand(tripCmd)
	cmd.AddCommand(delayStatsCmd)
	return cmd
}

func (s transitStopSelector) validate(queryFlag, idFlag string) error {
	query := strings.TrimSpace(s.Query)
	id := strings.TrimSpace(s.ID)
	switch {
	case query == "" && id == "":
		return fmt.Errorf("--%s or --%s is required", queryFlag, idFlag)
	case query != "" && id != "":
		return fmt.Errorf("use either --%s or --%s, not both", queryFlag, idFlag)
	default:
		return nil
	}
}

func (s transitStopSelector) display() string {
	if strings.TrimSpace(s.ID) != "" {
		return strings.TrimSpace(s.ID)
	}
	return strings.TrimSpace(s.Query)
}

func appendTransitStopMatchWarning(warnings []string, label, idFlag string, selector transitStopSelector, matchMode string, count int) []string {
	if count <= 10 || matchMode != "query" {
		return warnings
	}
	return append(warnings, fmt.Sprintf("%s query %q matched %d stops; use odh transit stops search and rerun with --%s if results are noisy", label, selector.display(), count, idFlag))
}

type transitTimeQuery struct {
	Date       time.Time
	AroundText string
	Around     int
	Window     time.Duration
	HasAround  bool
}

type transitDeparturesResult struct {
	Stops         []gtfsStop         `json:"stops"`
	StopMatchMode string             `json:"stop_match_mode"`
	Departures    []transitDeparture `json:"departures"`
}

type transitTripResult struct {
	FromStops     []gtfsStop         `json:"from_stops"`
	FromMatchMode string             `json:"from_match_mode"`
	ToStops       []gtfsStop         `json:"to_stops"`
	ToMatchMode   string             `json:"to_match_mode"`
	Matches       []transitTripMatch `json:"matches"`
}

type transitStopsSearchOutput struct {
	Dataset string          `json:"dataset"`
	Query   string          `json:"query"`
	Archive gtfsArchiveInfo `json:"archive"`
	Count   int             `json:"count"`
	Stops   []gtfsStop      `json:"stops"`
	Format  string          `json:"-"`
}

type transitDeparturesOutput struct {
	Dataset       string             `json:"dataset"`
	StopQuery     string             `json:"stop_query,omitempty"`
	StopID        string             `json:"stop_id,omitempty"`
	StopMatchMode string             `json:"stop_match_mode"`
	Date          string             `json:"date"`
	Around        string             `json:"around"`
	Window        string             `json:"window"`
	Mode          string             `json:"mode"`
	Archive       gtfsArchiveInfo    `json:"archive"`
	MatchedStops  []gtfsStop         `json:"matched_stops"`
	Count         int                `json:"count"`
	Departures    []transitDeparture `json:"departures"`
	Warnings      []string           `json:"warnings,omitempty"`
	Format        string             `json:"-"`
}

type transitTripOutput struct {
	Dataset       string             `json:"dataset"`
	FromQuery     string             `json:"from_query,omitempty"`
	FromStopID    string             `json:"from_stop_id,omitempty"`
	FromMatchMode string             `json:"from_match_mode"`
	ToQuery       string             `json:"to_query,omitempty"`
	ToStopID      string             `json:"to_stop_id,omitempty"`
	ToMatchMode   string             `json:"to_match_mode"`
	Date          string             `json:"date"`
	Time          string             `json:"time"`
	Window        string             `json:"window"`
	Mode          string             `json:"mode"`
	Archive       gtfsArchiveInfo    `json:"archive"`
	FromStops     []gtfsStop         `json:"from_stops"`
	ToStops       []gtfsStop         `json:"to_stops"`
	Count         int                `json:"count"`
	Matches       []transitTripMatch `json:"matches"`
	Warnings      []string           `json:"warnings,omitempty"`
	Format        string             `json:"-"`
}

type transitDelayStatsOutput struct {
	Supported    bool              `json:"supported"`
	Reason       string            `json:"reason"`
	Requested    map[string]string `json:"requested"`
	AvailableNow []string          `json:"available_now"`
	NextStep     string            `json:"next_step"`
	Format       string            `json:"-"`
}

func writeTransitStopsSearchOutput(stdout io.Writer, result transitStopsSearchOutput) error {
	switch result.Format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "STOP_ID\tNAME\tPARENT_STATION\tLAT\tLON")
		for _, stop := range result.Stops {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%.6f\t%.6f\n", stop.ID, stop.Name, stop.ParentStation, stop.Lat, stop.Lon)
		}
		return tw.Flush()
	case "markdown":
		fmt.Fprintln(stdout, "| stop_id | name | parent_station | lat | lon |")
		fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- |")
		for _, stop := range result.Stops {
			fmt.Fprintf(stdout, "| %s | %s | %s | %.6f | %.6f |\n",
				escapeMarkdown(stop.ID),
				escapeMarkdown(stop.Name),
				escapeMarkdown(stop.ParentStation),
				stop.Lat,
				stop.Lon,
			)
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.Format)
	}
}

func writeTransitDeparturesOutput(stdout io.Writer, result transitDeparturesOutput) error {
	switch result.Format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "TIME\tROUTE\tHEADSIGN\tSTOP\tSTOP_ID\tTRIP_ID")
		for _, departure := range result.Departures {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
				departure.DepartureTime,
				firstNonEmpty(departure.RouteShortName, departure.RouteID),
				departure.Headsign,
				departure.StopName,
				departure.StopID,
				departure.TripID,
			)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		writePlainWarnings(stdout, result.Warnings)
		return nil
	case "markdown":
		fmt.Fprintln(stdout, "| time | route | headsign | stop | stop_id | trip_id |")
		fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- | --- |")
		for _, departure := range result.Departures {
			fmt.Fprintf(stdout, "| %s | %s | %s | %s | %s | %s |\n",
				escapeMarkdown(departure.DepartureTime),
				escapeMarkdown(firstNonEmpty(departure.RouteShortName, departure.RouteID)),
				escapeMarkdown(departure.Headsign),
				escapeMarkdown(departure.StopName),
				escapeMarkdown(departure.StopID),
				escapeMarkdown(departure.TripID),
			)
		}
		writeMarkdownWarnings(stdout, result.Warnings)
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.Format)
	}
}

func writeTransitTripOutput(stdout io.Writer, result transitTripOutput) error {
	switch result.Format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ROUTE\tFROM\tDEPART\tTO\tARRIVE\tHEADSIGN\tTRIP_ID")
		for _, match := range result.Matches {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				firstNonEmpty(match.RouteShortName, match.RouteID),
				match.From.StopName,
				match.From.DepartureTime,
				match.To.StopName,
				match.To.ArrivalTime,
				match.Headsign,
				match.TripID,
			)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		writePlainWarnings(stdout, result.Warnings)
		return nil
	case "markdown":
		fmt.Fprintln(stdout, "| route | from | depart | to | arrive | headsign | trip_id |")
		fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- | --- | --- |")
		for _, match := range result.Matches {
			fmt.Fprintf(stdout, "| %s | %s | %s | %s | %s | %s | %s |\n",
				escapeMarkdown(firstNonEmpty(match.RouteShortName, match.RouteID)),
				escapeMarkdown(match.From.StopName),
				escapeMarkdown(match.From.DepartureTime),
				escapeMarkdown(match.To.StopName),
				escapeMarkdown(match.To.ArrivalTime),
				escapeMarkdown(match.Headsign),
				escapeMarkdown(match.TripID),
			)
		}
		writeMarkdownWarnings(stdout, result.Warnings)
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.Format)
	}
}

func writeTransitDelayStatsOutput(stdout io.Writer, result transitDelayStatsOutput) error {
	switch result.Format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		fmt.Fprintf(stdout, "supported: %t\n", result.Supported)
		fmt.Fprintf(stdout, "reason: %s\n", result.Reason)
		fmt.Fprintf(stdout, "next_step: %s\n", result.NextStep)
		return nil
	case "markdown":
		fmt.Fprintf(stdout, "- supported: `%t`\n", result.Supported)
		fmt.Fprintf(stdout, "- reason: %s\n", escapeMarkdown(result.Reason))
		fmt.Fprintf(stdout, "- next_step: %s\n", escapeMarkdown(result.NextStep))
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.Format)
	}
}

func (r *Runner) fetchGTFSArchive(ctx context.Context, dataset, cacheDir string, refresh bool) (gtfsArchiveInfo, error) {
	dataset = strings.TrimSpace(dataset)
	if dataset == "" {
		dataset = defaultGTFSDataset
	}
	api, _ := r.Registry.Find("gtfs")
	path := fmt.Sprintf("/v1/dataset/%s/raw", url.PathEscape(dataset))
	endpoint, err := BuildURL(api.BaseURL, path, nil)
	if err != nil {
		return gtfsArchiveInfo{}, err
	}
	if strings.TrimSpace(cacheDir) == "" {
		cacheDir, err = defaultTransitCacheDir()
		if err != nil {
			return gtfsArchiveInfo{}, err
		}
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return gtfsArchiveInfo{}, err
	}
	cachePath := filepath.Join(cacheDir, sanitizeCacheName(dataset)+".zip")
	if !refresh && cacheFresh(cachePath, defaultGTFSCacheTTL) {
		return gtfsArchiveInfo{Dataset: dataset, Endpoint: endpoint, Path: cachePath, Cached: true}, nil
	}
	resp, err := r.Client.WithTimeout(gtfsDownloadTimeout).GetWithLimit(ctx, endpoint, maxGTFSArchiveBytes)
	if err != nil {
		return gtfsArchiveInfo{}, err
	}
	tempPath := cachePath + ".tmp"
	if err := os.WriteFile(tempPath, resp.Body, 0o644); err != nil {
		return gtfsArchiveInfo{}, err
	}
	if err := os.Rename(tempPath, cachePath); err != nil {
		return gtfsArchiveInfo{}, err
	}
	return gtfsArchiveInfo{Dataset: dataset, Endpoint: endpoint, Path: cachePath, Cached: false}, nil
}

func defaultTransitCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	if strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("could not determine cache directory")
	}
	return filepath.Join(base, "odh-cli", "gtfs"), nil
}

func sanitizeCacheName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "dataset"
	}
	return result
}

func cacheFresh(path string, ttl time.Duration) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return false
	}
	return time.Since(info.ModTime()) <= ttl
}

func readGTFSStops(zipPath string) ([]gtfsStop, error) {
	reader, closeFn, err := openGTFSZip(zipPath)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	rows, err := readCSVRows(reader, "stops.txt")
	if err != nil {
		return nil, err
	}
	stops := make([]gtfsStop, 0, len(rows))
	for _, row := range rows {
		stops = append(stops, gtfsStop{
			ID:            row["stop_id"],
			Name:          row["stop_name"],
			Lat:           parseFloat(row["stop_lat"]),
			Lon:           parseFloat(row["stop_lon"]),
			ParentStation: row["parent_station"],
		})
	}
	return stops, nil
}

func findTransitDepartures(zipPath string, selector transitStopSelector, query transitTimeQuery, routeTypes map[string]struct{}, limit int) (transitDeparturesResult, error) {
	reader, closeFn, err := openGTFSZip(zipPath)
	if err != nil {
		return transitDeparturesResult{}, err
	}
	defer closeFn()
	stops, stopByID, err := loadGTFSStops(reader)
	if err != nil {
		return transitDeparturesResult{}, err
	}
	matchedStops, matchMode, err := selectGTFSStops(stops, selector, 50)
	if err != nil {
		return transitDeparturesResult{}, err
	}
	matchedStopIDs := stopIDSet(matchedStops)
	activeServices, err := loadActiveGTFSServiceIDs(reader, query.Date)
	if err != nil {
		return transitDeparturesResult{}, err
	}
	routes, err := loadGTFSRoutes(reader)
	if err != nil {
		return transitDeparturesResult{}, err
	}
	trips, err := loadGTFSActiveTrips(reader, activeServices, routes, routeTypes)
	if err != nil {
		return transitDeparturesResult{}, err
	}
	stopTimes, err := readCSVRows(reader, "stop_times.txt")
	if err != nil {
		return transitDeparturesResult{}, err
	}
	departures := make([]transitDeparture, 0)
	for _, row := range stopTimes {
		if _, ok := matchedStopIDs[row["stop_id"]]; !ok {
			continue
		}
		trip, ok := trips[row["trip_id"]]
		if !ok || !query.matches(row["departure_time"]) {
			continue
		}
		route := routes[trip.RouteID]
		stop := stopByID[row["stop_id"]]
		departures = append(departures, makeTransitDeparture(row, trip, route, stop))
	}
	sortTransitDepartures(departures)
	if len(departures) > limit {
		departures = departures[:limit]
	}
	return transitDeparturesResult{Stops: matchedStops, StopMatchMode: matchMode, Departures: departures}, nil
}

func findTransitTripMatches(zipPath string, fromSelector, toSelector transitStopSelector, query transitTimeQuery, routeTypes map[string]struct{}, limit int) (transitTripResult, error) {
	reader, closeFn, err := openGTFSZip(zipPath)
	if err != nil {
		return transitTripResult{}, err
	}
	defer closeFn()
	stops, stopByID, err := loadGTFSStops(reader)
	if err != nil {
		return transitTripResult{}, err
	}
	fromStops, fromMode, err := selectGTFSStops(stops, fromSelector, 50)
	if err != nil {
		return transitTripResult{}, err
	}
	toStops, toMode, err := selectGTFSStops(stops, toSelector, 50)
	if err != nil {
		return transitTripResult{}, err
	}
	fromIDs := stopIDSet(fromStops)
	toIDs := stopIDSet(toStops)
	activeServices, err := loadActiveGTFSServiceIDs(reader, query.Date)
	if err != nil {
		return transitTripResult{}, err
	}
	routes, err := loadGTFSRoutes(reader)
	if err != nil {
		return transitTripResult{}, err
	}
	trips, err := loadGTFSActiveTrips(reader, activeServices, routes, routeTypes)
	if err != nil {
		return transitTripResult{}, err
	}
	stopTimes, err := readCSVRows(reader, "stop_times.txt")
	if err != nil {
		return transitTripResult{}, err
	}
	byTrip := map[string][]gtfsStopTime{}
	for _, row := range stopTimes {
		if _, ok := trips[row["trip_id"]]; !ok {
			continue
		}
		byTrip[row["trip_id"]] = append(byTrip[row["trip_id"]], gtfsStopTime{
			TripID:        row["trip_id"],
			ArrivalTime:   row["arrival_time"],
			DepartureTime: row["departure_time"],
			StopID:        row["stop_id"],
			StopSequence:  parseInt(row["stop_sequence"]),
		})
	}
	matches := make([]transitTripMatch, 0)
	for tripID, times := range byTrip {
		sort.Slice(times, func(i, j int) bool {
			return times[i].StopSequence < times[j].StopSequence
		})
		for fromIndex, fromTime := range times {
			if _, ok := fromIDs[fromTime.StopID]; !ok || !query.matches(fromTime.DepartureTime) {
				continue
			}
			for _, toTime := range times[fromIndex+1:] {
				if _, ok := toIDs[toTime.StopID]; !ok {
					continue
				}
				trip := trips[tripID]
				route := routes[trip.RouteID]
				matches = append(matches, makeTransitTripMatch(trip, route, fromTime, toTime, stopByID))
				break
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].From.DepartureTime < matches[j].From.DepartureTime
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return transitTripResult{FromStops: fromStops, FromMatchMode: fromMode, ToStops: toStops, ToMatchMode: toMode, Matches: matches}, nil
}

func selectGTFSStops(stops []gtfsStop, selector transitStopSelector, queryLimit int) ([]gtfsStop, string, error) {
	id := strings.TrimSpace(selector.ID)
	if id == "" {
		return searchGTFSStops(stops, selector.Query, queryLimit), "query", nil
	}
	parentMatches := make([]gtfsStop, 0)
	var exactMatch *gtfsStop
	for _, stop := range stops {
		if stop.ID == id {
			stopCopy := stop
			exactMatch = &stopCopy
		}
		if stop.ParentStation == id {
			parentMatches = append(parentMatches, stop)
		}
	}
	if len(parentMatches) > 0 {
		sort.Slice(parentMatches, func(i, j int) bool {
			if parentMatches[i].Name != parentMatches[j].Name {
				return parentMatches[i].Name < parentMatches[j].Name
			}
			return parentMatches[i].ID < parentMatches[j].ID
		})
		return parentMatches, "parent-station", nil
	}
	if exactMatch != nil {
		return []gtfsStop{*exactMatch}, "stop-id", nil
	}
	return nil, "stop-id", fmt.Errorf("GTFS stop id %q was not found", id)
}

func openGTFSZip(path string) (*zip.Reader, func(), error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, func() {}, err
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		_ = file.Close()
		return nil, func() {}, err
	}
	return reader, func() { _ = file.Close() }, nil
}

func readCSVRows(reader *zip.Reader, name string) ([]map[string]string, error) {
	file, err := openZipMember(reader, name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	headers := records[0]
	for index, header := range headers {
		headers[index] = strings.TrimPrefix(header, "\ufeff")
	}
	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := map[string]string{}
		for index, header := range headers {
			if index < len(record) {
				row[header] = record[index]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func openZipMember(reader *zip.Reader, name string) (io.ReadCloser, error) {
	for _, file := range reader.File {
		if file.Name == name {
			return file.Open()
		}
	}
	return nil, fmt.Errorf("GTFS archive does not contain %s", name)
}

func loadGTFSStops(reader *zip.Reader) ([]gtfsStop, map[string]gtfsStop, error) {
	rows, err := readCSVRows(reader, "stops.txt")
	if err != nil {
		return nil, nil, err
	}
	stops := make([]gtfsStop, 0, len(rows))
	byID := map[string]gtfsStop{}
	for _, row := range rows {
		stop := gtfsStop{
			ID:            row["stop_id"],
			Name:          row["stop_name"],
			Lat:           parseFloat(row["stop_lat"]),
			Lon:           parseFloat(row["stop_lon"]),
			ParentStation: row["parent_station"],
		}
		stops = append(stops, stop)
		byID[stop.ID] = stop
	}
	return stops, byID, nil
}

func loadGTFSRoutes(reader *zip.Reader) (map[string]gtfsRoute, error) {
	rows, err := readCSVRows(reader, "routes.txt")
	if err != nil {
		return nil, err
	}
	routes := map[string]gtfsRoute{}
	for _, row := range rows {
		route := gtfsRoute{
			ID:        row["route_id"],
			ShortName: row["route_short_name"],
			LongName:  row["route_long_name"],
			Type:      row["route_type"],
		}
		routes[route.ID] = route
	}
	return routes, nil
}

func loadGTFSActiveTrips(reader *zip.Reader, activeServices map[string]struct{}, routes map[string]gtfsRoute, routeTypes map[string]struct{}) (map[string]gtfsTrip, error) {
	rows, err := readCSVRows(reader, "trips.txt")
	if err != nil {
		return nil, err
	}
	trips := map[string]gtfsTrip{}
	for _, row := range rows {
		if _, ok := activeServices[row["service_id"]]; !ok {
			continue
		}
		route := routes[row["route_id"]]
		if len(routeTypes) > 0 {
			if _, ok := routeTypes[route.Type]; !ok {
				continue
			}
		}
		trips[row["trip_id"]] = gtfsTrip{
			ID:          row["trip_id"],
			RouteID:     row["route_id"],
			ServiceID:   row["service_id"],
			Headsign:    row["trip_headsign"],
			DirectionID: row["direction_id"],
		}
	}
	return trips, nil
}

func loadActiveGTFSServiceIDs(reader *zip.Reader, date time.Time) (map[string]struct{}, error) {
	active := map[string]struct{}{}
	rows, err := readCSVRows(reader, "calendar.txt")
	if err != nil {
		return nil, err
	}
	dateText := date.Format("20060102")
	weekday := gtfsWeekdayColumn(date)
	for _, row := range rows {
		if row["start_date"] <= dateText && dateText <= row["end_date"] && row[weekday] == "1" {
			active[row["service_id"]] = struct{}{}
		}
	}
	exceptions, err := readCSVRows(reader, "calendar_dates.txt")
	if err != nil {
		return nil, err
	}
	for _, row := range exceptions {
		if row["date"] != dateText {
			continue
		}
		switch row["exception_type"] {
		case "1":
			active[row["service_id"]] = struct{}{}
		case "2":
			delete(active, row["service_id"])
		}
	}
	return active, nil
}

func gtfsWeekdayColumn(value time.Time) string {
	switch value.Weekday() {
	case time.Monday:
		return "monday"
	case time.Tuesday:
		return "tuesday"
	case time.Wednesday:
		return "wednesday"
	case time.Thursday:
		return "thursday"
	case time.Friday:
		return "friday"
	case time.Saturday:
		return "saturday"
	default:
		return "sunday"
	}
}

func searchGTFSStops(stops []gtfsStop, query string, limit int) []gtfsStop {
	terms := transitQueryAlternatives(query)
	if len(terms) == 0 {
		return nil
	}
	matches := make([]gtfsStop, 0)
	for _, stop := range stops {
		name := normalizeTransitText(stop.Name)
		matched := true
		for _, alternatives := range terms {
			termMatched := false
			for _, alternative := range alternatives {
				if transitTextContainsTerm(name, alternative) {
					termMatched = true
					break
				}
			}
			if !termMatched {
				matched = false
				break
			}
		}
		if matched {
			matches = append(matches, stop)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return stopSearchScore(matches[i], terms) < stopSearchScore(matches[j], terms)
	})
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func transitQueryAlternatives(query string) [][]string {
	aliases := map[string][]string{
		"auer":       {"auer", "ora"},
		"ora":        {"ora", "auer"},
		"brenner":    {"brenner", "brennero"},
		"brennero":   {"brennero", "brenner"},
		"bozen":      {"bozen", "bolzano"},
		"bolzano":    {"bolzano", "bozen"},
		"meran":      {"meran", "merano"},
		"merano":     {"merano", "meran"},
		"brixen":     {"brixen", "bressanone"},
		"bressanone": {"bressanone", "brixen"},
	}
	fields := strings.Fields(normalizeTransitText(query))
	result := make([][]string, 0, len(fields))
	for _, field := range fields {
		if values, ok := aliases[field]; ok {
			normalized := make([]string, 0, len(values))
			for _, value := range values {
				normalized = append(normalized, normalizeTransitText(value))
			}
			result = append(result, normalized)
			continue
		}
		result = append(result, []string{field})
	}
	return result
}

func stopSearchScore(stop gtfsStop, terms [][]string) int {
	name := normalizeTransitText(stop.Name)
	score := 100
	for _, alternatives := range terms {
		for _, term := range alternatives {
			if name == term {
				return 0
			}
			if strings.HasPrefix(name, term) {
				score = min(score, 10)
			} else if transitTextContainsTerm(name, term) {
				score = min(score, 20)
			}
		}
	}
	if strings.Contains(name, "stazione") || strings.Contains(name, "station") {
		score -= 5
	}
	return score
}

func transitTextContainsTerm(text, term string) bool {
	if term == "" {
		return true
	}
	if len([]rune(term)) <= 4 {
		for _, field := range strings.Fields(text) {
			if field == term {
				return true
			}
		}
		return false
	}
	return strings.Contains(text, term)
}

func stopIDSet(stops []gtfsStop) map[string]struct{} {
	result := map[string]struct{}{}
	for _, stop := range stops {
		result[stop.ID] = struct{}{}
	}
	return result
}

func makeTransitDeparture(row map[string]string, trip gtfsTrip, route gtfsRoute, stop gtfsStop) transitDeparture {
	return transitDeparture{
		TripID:         trip.ID,
		RouteID:        route.ID,
		RouteShortName: route.ShortName,
		RouteLongName:  route.LongName,
		RouteType:      route.Type,
		Headsign:       trip.Headsign,
		DirectionID:    trip.DirectionID,
		StopID:         stop.ID,
		StopName:       stop.Name,
		ArrivalTime:    row["arrival_time"],
		DepartureTime:  row["departure_time"],
		StopSequence:   parseInt(row["stop_sequence"]),
	}
}

func makeTransitTripMatch(trip gtfsTrip, route gtfsRoute, fromTime, toTime gtfsStopTime, stops map[string]gtfsStop) transitTripMatch {
	return transitTripMatch{
		TripID:         trip.ID,
		RouteID:        route.ID,
		RouteShortName: route.ShortName,
		RouteLongName:  route.LongName,
		RouteType:      route.Type,
		Headsign:       trip.Headsign,
		DirectionID:    trip.DirectionID,
		TransferCount:  0,
		From: transitStopOnTrip{
			StopID:        fromTime.StopID,
			StopName:      stops[fromTime.StopID].Name,
			ArrivalTime:   fromTime.ArrivalTime,
			DepartureTime: fromTime.DepartureTime,
			StopSequence:  fromTime.StopSequence,
		},
		To: transitStopOnTrip{
			StopID:        toTime.StopID,
			StopName:      stops[toTime.StopID].Name,
			ArrivalTime:   toTime.ArrivalTime,
			DepartureTime: toTime.DepartureTime,
			StopSequence:  toTime.StopSequence,
		},
	}
}

func sortTransitDepartures(departures []transitDeparture) {
	sort.Slice(departures, func(i, j int) bool {
		if departures[i].DepartureTime != departures[j].DepartureTime {
			return departures[i].DepartureTime < departures[j].DepartureTime
		}
		return departures[i].StopName < departures[j].StopName
	})
}

func parseTransitTimeQuery(dateText, aroundText, windowText string) (transitTimeQuery, error) {
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(dateText), time.Local)
	if err != nil {
		return transitTimeQuery{}, fmt.Errorf("invalid date %q; use YYYY-MM-DD", dateText)
	}
	window, err := time.ParseDuration(strings.TrimSpace(windowText))
	if err != nil {
		return transitTimeQuery{}, fmt.Errorf("invalid --window %q", windowText)
	}
	if window < 0 {
		return transitTimeQuery{}, fmt.Errorf("--window must not be negative")
	}
	query := transitTimeQuery{Date: date, Window: window}
	if strings.TrimSpace(aroundText) == "" {
		return query, nil
	}
	seconds, normalized, err := parseGTFSTimeOfDay(aroundText)
	if err != nil {
		return transitTimeQuery{}, err
	}
	query.Around = seconds
	query.AroundText = normalized
	query.HasAround = true
	return query, nil
}

func (q transitTimeQuery) matches(gtfsTime string) bool {
	if !q.HasAround {
		return true
	}
	seconds, _, err := parseGTFSTimeOfDay(gtfsTime)
	if err != nil {
		return false
	}
	delta := seconds - q.Around
	if delta < 0 {
		delta = -delta
	}
	return time.Duration(delta)*time.Second <= q.Window
}

func parseGTFSTimeOfDay(value string) (int, string, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, "", fmt.Errorf("invalid time %q; use HH:MM or HH:MM:SS", value)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 {
		return 0, "", fmt.Errorf("invalid hour in time %q", value)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, "", fmt.Errorf("invalid minute in time %q", value)
	}
	second := 0
	if len(parts) == 3 {
		second, err = strconv.Atoi(parts[2])
		if err != nil || second < 0 || second > 59 {
			return 0, "", fmt.Errorf("invalid second in time %q", value)
		}
	}
	return hour*3600 + minute*60 + second, fmt.Sprintf("%02d:%02d:%02d", hour, minute, second), nil
}

func transitModeRouteTypes(mode string) (map[string]struct{}, error) {
	switch normalizeTransitModeName(mode) {
	case "all":
		return nil, nil
	case "train":
		return map[string]struct{}{"2": {}}, nil
	case "bus":
		return map[string]struct{}{"3": {}}, nil
	case "cable-car":
		return map[string]struct{}{"7": {}}, nil
	default:
		return nil, fmt.Errorf("unsupported transit mode %q", mode)
	}
}

func normalizeTransitModeName(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "all", "any":
		return "all"
	case "train", "rail", "railway":
		return "train"
	case "bus":
		return "bus"
	case "cable", "cable-car", "cablecar":
		return "cable-car"
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}

func normalizeTransitText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacements := map[string]string{
		"ä": "ae", "ö": "oe", "ü": "ue", "ß": "ss",
		"à": "a", "á": "a", "è": "e", "é": "e", "ì": "i", "í": "i", "ò": "o", "ó": "o", "ù": "u", "ú": "u",
		"-": " ", "_": " ", ".": " ", ",": " ",
	}
	for old, replacement := range replacements {
		value = strings.ReplaceAll(value, old, replacement)
	}
	return strings.Join(strings.Fields(value), " ")
}

func parseFloat(value string) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return parsed
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}
