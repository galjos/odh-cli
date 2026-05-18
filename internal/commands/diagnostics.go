// SPDX-FileCopyrightText: 2026 Josef M. Gallmetzer
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

	"github.com/galjos/odh-cli/internal/output"
)

const (
	defaultDiagnosticFreshWithin = "24h"
	defaultParkingFreshWithin    = "2h"
)

func (r *Runner) runDiagnostics(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: odh diagnostics <ev-charging|parking-forecasts|tourism-events>")
		return 2
	}
	switch args[0] {
	case "ev-charging", "ev", "charging":
		return r.runDiagnosticsEVCharging(ctx, args[1:], stdout, stderr)
	case "parking-forecasts", "parking-forecast", "parking":
		return r.runDiagnosticsParkingForecasts(ctx, args[1:], stdout, stderr)
	case "tourism-events", "events":
		return r.runDiagnosticsTourismEvents(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown diagnostics subcommand %q\n", args[0])
		return 2
	}
}

func (r *Runner) runDiagnosticsEVCharging(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("diagnostics ev-charging", stderr)
	origin := fs.String("origin", "", "optional sorigin filter, for example ALPERIA")
	freshWithin := fs.String("fresh-within", defaultDiagnosticFreshWithin, "freshness window for current availability rows")
	limit := fs.Int("limit", 10, "maximum filtered rows to include")
	requestLimit := fs.Int("request-limit", 10000, "raw upstream rows to inspect before local filtering")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "diagnostics ev-charging does not accept positional arguments")
		return 2
	}
	result, err := r.fetchFilteredMobilityLatest(ctx, mobilityLatestFilter{
		StationType:  "EChargingStation",
		DataType:     "number-available",
		Origin:       *origin,
		ActiveOnly:   true,
		FreshWithin:  *freshWithin,
		Sort:         "newest",
		Limit:        *limit,
		RequestLimit: *requestLimit,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
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
	if err := output.WriteJSON(stdout, map[string]any{
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
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (r *Runner) runDiagnosticsParkingForecasts(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("diagnostics parking-forecasts", stderr)
	origin := fs.String("origin", "", "optional sorigin filter, for example Municipality Merano")
	forecastMinutes := fs.Int("forecast-minutes", 60, "parking forecast horizon in minutes")
	freshWithin := fs.String("fresh-within", defaultParkingFreshWithin, "freshness window for current and forecast rows")
	limit := fs.Int("limit", 10, "maximum filtered rows to include per feed")
	requestLimit := fs.Int("request-limit", 10000, "raw upstream rows to inspect before local filtering")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "diagnostics parking-forecasts does not accept positional arguments")
		return 2
	}
	if *forecastMinutes <= 0 {
		fmt.Fprintln(stderr, "--forecast-minutes must be greater than zero")
		return 2
	}
	current, err := r.fetchFilteredMobilityLatest(ctx, mobilityLatestFilter{
		StationType:  "ParkingStation",
		DataType:     "free",
		Origin:       *origin,
		ActiveOnly:   true,
		FreshWithin:  *freshWithin,
		Sort:         "newest",
		Limit:        *limit,
		RequestLimit: *requestLimit,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	forecastType := fmt.Sprintf("parking-forecast-%d", *forecastMinutes)
	forecast, err := r.fetchFilteredMobilityLatest(ctx, mobilityLatestFilter{
		StationType:  "ParkingStation",
		DataType:     forecastType,
		Origin:       *origin,
		ActiveOnly:   true,
		FreshWithin:  *freshWithin,
		Sort:         "newest",
		Limit:        *limit,
		RequestLimit: *requestLimit,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
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
	if err := output.WriteJSON(stdout, map[string]any{
		"domain":           "parking-forecasts",
		"source":           "Open Data Hub Mobility API",
		"verdict":          verdict,
		"origin":           strings.TrimSpace(*origin),
		"fresh_within":     strings.TrimSpace(*freshWithin),
		"forecast_minutes": *forecastMinutes,
		"current":          current,
		"forecast":         forecast,
		"warnings":         warnings,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (r *Runner) runDiagnosticsTourismEvents(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("diagnostics tourism-events", stderr)
	dateText := fs.String("date", time.Now().Format("2006-01-02"), "date used for local active-today checks")
	onlyActive := fs.Bool("only-active", true, "request upstream onlyactive=true")
	limit := fs.Int("limit", 20, "number of upstream events to inspect")
	page := fs.Int("page", 1, "page number")
	params := paramValues{}
	fs.Var(&params, "param", "additional query parameter as key=value; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "diagnostics tourism-events does not accept positional arguments")
		return 2
	}
	day, err := time.Parse("2006-01-02", strings.TrimSpace(*dateText))
	if err != nil {
		fmt.Fprintln(stderr, "--date must use YYYY-MM-DD")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}
	if *page < 1 {
		fmt.Fprintln(stderr, "--page must be greater than zero")
		return 2
	}
	api, _ := r.Registry.Find("tourism")
	values := params.Values()
	values.Set("pagenumber", strconv.Itoa(*page))
	values.Set("pagesize", strconv.Itoa(*limit))
	if *onlyActive {
		values.Set("onlyactive", "true")
	}
	requestURL, err := BuildURL(api.BaseURL, "/v1/EventShort", values)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	events := summarizeTourismEvents(extractItemsMaps(value), day)
	warnings := tourismEventWarnings(events, *onlyActive)
	verdict := "usable"
	if len(warnings) > 0 {
		verdict = "usable_with_caveats"
	}
	activeCount := countTourismEventsByStatus(events, "active")
	if len(events) == 0 || activeCount == 0 {
		verdict = "unavailable"
	}
	if err := output.WriteJSON(stdout, map[string]any{
		"domain":       "tourism-events",
		"source":       "Open Data Hub Tourism API",
		"verdict":      verdict,
		"date":         day.Format("2006-01-02"),
		"only_active":  *onlyActive,
		"endpoint":     requestURL,
		"count":        len(events),
		"active_count": activeCount,
		"events":       events,
		"warnings":     warnings,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
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
