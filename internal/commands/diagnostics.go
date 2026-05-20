// SPDX-FileCopyrightText: 2026 Josef M. Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/galjos/odh-cli/internal/output"
	"github.com/spf13/cobra"
)

const (
	defaultDiagnosticFreshWithin = "24h"
	defaultParkingFreshWithin    = "2h"
)

func (r *Runner) newDiagnosticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Data quality and reliability checks",
		RunE:  requireSubcommand,
	}

	var evOrigin string
	var evFreshWithin string
	var evLimit int
	var evRequestLimit int
	evCmd := &cobra.Command{
		Use:     "ev-charging",
		Aliases: []string{"ev", "charging"},
		Short:   "Check EV charging data reliability",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := r.fetchFilteredMobilityLatest(cmd.Context(), mobilityLatestFilter{
				StationType:  "EChargingStation",
				DataType:     "number-available",
				Origin:       evOrigin,
				ActiveOnly:   true,
				FreshWithin:  evFreshWithin,
				Sort:         "newest",
				Limit:        evLimit,
				RequestLimit: evRequestLimit,
			})
			if err != nil {
				return err
			}
			verdict := "usable"
			warnings := append([]string{}, result.Warnings...)
			if result.Count == 0 {
				verdict = "unavailable"
				warnings = append(warnings, "no fresh active EV availability rows found in Open Data Hub; do not report stale rows as current charger availability")
			}
			if result.RawCount == 0 {
				warnings = append(warnings, "Open Data Hub returned no EV availability rows from the inspected endpoint")
			}
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"domain":              "ev-charging",
				"source":              "Open Data Hub Mobility API",
				"verdict":             verdict,
				"station_type":        result.StationType,
				"data_type":           result.DataType,
				"origin":              result.Origin,
				"active_only":         true,
				"fresh_within":        result.FreshWithin,
				"raw_count":           result.RawCount,
				"current_count":       result.Count,
				"measurements":        result.Measurements,
				"warnings":            warnings,
				"recommended_command": recommendedMobilityLatestCommand(result),
			})
		},
	}
	evCmd.Flags().StringVar(&evOrigin, "origin", "", "optional sorigin filter, for example ALPERIA")
	evCmd.Flags().StringVar(&evFreshWithin, "fresh-within", defaultDiagnosticFreshWithin, "freshness window for current availability rows")
	evCmd.Flags().IntVar(&evLimit, "limit", 10, "maximum filtered rows to include")
	evCmd.Flags().IntVar(&evRequestLimit, "request-limit", 10000, "raw upstream rows to inspect before local filtering")

	var parkOrigin string
	var parkMinutes int
	var parkFreshWithin string
	var parkLimit int
	var parkRequestLimit int
	parkCmd := &cobra.Command{
		Use:     "parking-forecasts",
		Aliases: []string{"parking-forecast", "parking"},
		Short:   "Check parking forecast reliability",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if parkMinutes <= 0 {
				return fmt.Errorf("--forecast-minutes must be greater than zero")
			}
			current, err := r.fetchFilteredMobilityLatest(cmd.Context(), mobilityLatestFilter{
				StationType:  "ParkingStation",
				DataType:     "free",
				Origin:       parkOrigin,
				ActiveOnly:   true,
				FreshWithin:  parkFreshWithin,
				Sort:         "newest",
				Limit:        parkLimit,
				RequestLimit: parkRequestLimit,
			})
			if err != nil {
				return err
			}
			forecastType := fmt.Sprintf("parking-forecast-%d", parkMinutes)
			forecast, err := r.fetchFilteredMobilityLatest(cmd.Context(), mobilityLatestFilter{
				StationType:  "ParkingStation",
				DataType:     forecastType,
				Origin:       parkOrigin,
				ActiveOnly:   true,
				FreshWithin:  parkFreshWithin,
				Sort:         "newest",
				Limit:        parkLimit,
				RequestLimit: parkRequestLimit,
			})
			if err != nil {
				return err
			}
			var verdict string
			warnings := make([]string, 0)
			warnings = appendPrefixedWarnings(warnings, "current occupancy", current.Warnings)
			warnings = appendPrefixedWarnings(warnings, "forecast", forecast.Warnings)
			switch {
			case current.Count == 0:
				verdict = "unavailable"
				warnings = append(warnings, "no fresh active parking occupancy rows found; do not report parking availability as current")
			case forecast.Count == 0:
				verdict = "current_only"
				warnings = append(warnings, "no fresh parking forecast rows found; treat forecast values as unavailable instead of stale")
			default:
				verdict = "usable_with_forecast"
			}
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"domain":           "parking-forecasts",
				"source":           "Open Data Hub Mobility API",
				"verdict":          verdict,
				"origin":           strings.TrimSpace(parkOrigin),
				"fresh_within":     strings.TrimSpace(parkFreshWithin),
				"forecast_minutes": parkMinutes,
				"current":          current,
				"forecast":         forecast,
				"warnings":         warnings,
			})
		},
	}
	parkCmd.Flags().StringVar(&parkOrigin, "origin", "", "optional sorigin filter, for example Municipality Merano")
	parkCmd.Flags().IntVar(&parkMinutes, "forecast-minutes", 60, "parking forecast horizon in minutes")
	parkCmd.Flags().StringVar(&parkFreshWithin, "fresh-within", defaultParkingFreshWithin, "freshness window for current and forecast rows")
	parkCmd.Flags().IntVar(&parkLimit, "limit", 10, "maximum filtered rows to include per feed")
	parkCmd.Flags().IntVar(&parkRequestLimit, "request-limit", 10000, "raw upstream rows to inspect before local filtering")

	var tourDate string
	var tourOnlyActive bool
	var tourLimit int
	var tourPage int
	var tourParams []string
	tourCmd := &cobra.Command{
		Use:     "tourism-events",
		Aliases: []string{"events"},
		Short:   "Check tourism event data reliability",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			day, err := time.Parse("2006-01-02", strings.TrimSpace(tourDate))
			if err != nil {
				return fmt.Errorf("--date must use YYYY-MM-DD")
			}
			if tourLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			if tourPage < 1 {
				return fmt.Errorf("--page must be greater than zero")
			}
			api, _ := r.Registry.Find("tourism")
			values := url.Values{}
			values.Set("pagenumber", strconv.Itoa(tourPage))
			values.Set("pagesize", strconv.Itoa(tourLimit))
			if tourOnlyActive {
				values.Set("onlyactive", "true")
			}
			for _, p := range tourParams {
				key, value, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}
			requestURL, err := BuildURL(api.BaseURL, "/v1/EventShort", values)
			if err != nil {
				return err
			}
			value, err := r.fetchJSONValue(cmd.Context(), requestURL)
			if err != nil {
				return err
			}
			events := summarizeTourismEvents(extractItemsMaps(value), day)
			warnings := tourismEventWarnings(events, tourOnlyActive)
			verdict := "usable"
			if len(warnings) > 0 {
				verdict = "usable_with_caveats"
			}
			activeCount := countTourismEventsByStatus(events, "active")
			if len(events) == 0 || activeCount == 0 {
				verdict = "unavailable"
			}
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"domain":       "tourism-events",
				"source":       "Open Data Hub Tourism API",
				"verdict":      verdict,
				"date":         day.Format("2006-01-02"),
				"only_active":  tourOnlyActive,
				"endpoint":     requestURL,
				"count":        len(events),
				"active_count": activeCount,
				"events":       events,
				"warnings":     warnings,
			})
		},
	}
	tourCmd.Flags().StringVar(&tourDate, "date", time.Now().Format("2006-01-02"), "date used for local active-today checks")
	tourCmd.Flags().BoolVar(&tourOnlyActive, "only-active", true, "request upstream onlyactive=true")
	tourCmd.Flags().IntVar(&tourLimit, "limit", 20, "number of upstream events to inspect")
	tourCmd.Flags().IntVar(&tourPage, "page", 1, "page number")
	tourCmd.Flags().StringArrayVar(&tourParams, "param", nil, "additional query parameter as key=value; repeatable; values may contain commas")

	cmd.AddCommand(evCmd)
	cmd.AddCommand(parkCmd)
	cmd.AddCommand(tourCmd)
	return cmd
}

func (r *Runner) fetchFilteredMobilityLatest(ctx context.Context, filter mobilityLatestFilter) (mobilityLatestResult, error) {
	if filter.Limit < 1 {
		return mobilityLatestResult{}, fmt.Errorf("--limit must be greater than zero")
	}
	if filter.RequestLimit < 1 {
		filter.RequestLimit = max(filter.Limit, 1000)
	}
	freshDuration, err := parseFreshWithin(filter.FreshWithin)
	if err != nil {
		return mobilityLatestResult{}, err
	}
	filter.FreshDuration = freshDuration
	filter.Now = time.Now()
	api, _ := r.Registry.Find("mobility")
	path := fmt.Sprintf("/v2/flat,node/%s/%s/latest", url.PathEscape(filter.StationType), url.PathEscape(filter.DataType))
	values := url.Values{}
	values.Set("limit", strconv.Itoa(filter.RequestLimit))
	requestURL, err := BuildURL(api.BaseURL, path, values)
	if err != nil {
		return mobilityLatestResult{}, err
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		return mobilityLatestResult{}, err
	}
	filter.Endpoint = requestURL
	return filterMobilityLatest(extractDataList(value), filter), nil
}

type tourismEventSummary struct {
	ID                  string `json:"id,omitempty"`
	Title               string `json:"title,omitempty"`
	StartDate           string `json:"start_date,omitempty"`
	EndDate             string `json:"end_date,omitempty"`
	ActiveTodayUpstream any    `json:"active_today_upstream,omitempty"`
	ActiveUpstream      any    `json:"active_upstream,omitempty"`
	DateStatus          string `json:"date_status"`
	LocationStatus      string `json:"location_status"`
}

func summarizeTourismEvents(records []map[string]any, day time.Time) []tourismEventSummary {
	events := make([]tourismEventSummary, 0, len(records))
	for _, record := range records {
		events = append(events, tourismEventSummary{
			ID:                  firstNonEmptyString(record["Id"], record["ID"], record["id"]),
			Title:               tourismEventTitle(record),
			StartDate:           asString(record["StartDate"]),
			EndDate:             asString(record["EndDate"]),
			ActiveTodayUpstream: record["ActiveToday"],
			ActiveUpstream:      record["Active"],
			DateStatus:          tourismEventDateStatus(record, day),
			LocationStatus:      tourismEventLocationStatus(record),
		})
	}
	return events
}

func tourismEventWarnings(events []tourismEventSummary, onlyActive bool) []string {
	warnings := make([]string, 0)
	missingGPS := 0
	activeTodayFalse := 0
	outsideDate := 0
	unknownDate := 0
	for _, event := range events {
		if event.LocationStatus == "missing-gps" {
			missingGPS++
		}
		if value, ok := event.ActiveTodayUpstream.(bool); ok && !value {
			activeTodayFalse++
		}
		switch event.DateStatus {
		case "future", "expired":
			outsideDate++
		case "unknown":
			unknownDate++
		}
	}
	if onlyActive && activeTodayFalse > 0 {
		warnings = append(warnings, fmt.Sprintf("upstream onlyactive=true returned %d rows with ActiveToday=false", activeTodayFalse))
	}
	if onlyActive && outsideDate > 0 {
		warnings = append(warnings, fmt.Sprintf("local date checks marked %d upstream active rows as future or expired", outsideDate))
	}
	if unknownDate > 0 {
		warnings = append(warnings, fmt.Sprintf("%d rows had missing or unparseable StartDate/EndDate", unknownDate))
	}
	if missingGPS > 0 {
		warnings = append(warnings, fmt.Sprintf("%d rows have missing GpsInfo; location or radius claims are weak for those events", missingGPS))
	}
	if len(events) == 0 {
		warnings = append(warnings, "Open Data Hub returned no tourism event rows for the inspected request")
	}
	return warnings
}

func countTourismEventsByStatus(events []tourismEventSummary, status string) int {
	count := 0
	for _, event := range events {
		if event.DateStatus == status {
			count++
		}
	}
	return count
}

func tourismEventTitle(record map[string]any) string {
	titleMap, ok := record["EventTitle"].(map[string]any)
	if !ok {
		return ""
	}
	return firstNonEmptyString(titleMap["en"], titleMap["de"], titleMap["it"], titleMap["nl"])
}

func tourismEventDateStatus(record map[string]any, day time.Time) string {
	start := parseODHTime(asString(record["StartDate"]))
	end := parseODHTime(asString(record["EndDate"]))
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24*time.Hour - time.Nanosecond)
	switch {
	case start == nil && end == nil:
		return "unknown"
	case end != nil && end.Before(dayStart):
		return "expired"
	case start != nil && start.After(dayEnd):
		return "future"
	default:
		return "active"
	}
}

func tourismEventLocationStatus(record map[string]any) string {
	gps := record["GpsInfo"]
	if gps == nil {
		return "missing-gps"
	}
	if object, ok := gps.(map[string]any); ok && len(object) == 0 {
		return "missing-gps"
	}
	return "geocoded"
}

func extractItemsMaps(value any) []map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		if items, ok := typed["Items"].([]any); ok {
			return mapsFromList(items)
		}
		if items, ok := typed["data"].([]any); ok {
			return mapsFromList(items)
		}
	case []any:
		return mapsFromList(typed)
	}
	return nil
}

func appendPrefixedWarnings(warnings []string, prefix string, values []string) []string {
	for _, value := range values {
		warnings = append(warnings, prefix+": "+value)
	}
	return warnings
}

func recommendedMobilityLatestCommand(result mobilityLatestResult) string {
	parts := []string{
		"odh", "mobility", "latest",
		"--station-type", shellQuote(result.StationType),
		"--data-type", shellQuote(result.DataType),
	}
	if strings.TrimSpace(result.Origin) != "" {
		parts = append(parts, "--origin", shellQuote(result.Origin))
	}
	if result.ActiveOnly {
		parts = append(parts, "--active")
	}
	if strings.TrimSpace(result.FreshWithin) != "" {
		parts = append(parts, "--fresh-within", shellQuote(result.FreshWithin))
	}
	if strings.TrimSpace(result.Sort) != "" {
		parts = append(parts, "--sort", shellQuote(result.Sort))
	}
	return strings.Join(parts, " ")
}

func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		text := strings.TrimSpace(asString(value))
		if text != "" {
			return text
		}
	}
	return ""
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.ContainsAny(value, " \t\n'\"") {
		return strconv.Quote(value)
	}
	return value
}
