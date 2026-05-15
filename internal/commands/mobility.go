// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/galjos/odh-cli/internal/output"
)

func (r *Runner) runMobilityTypes(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("mobility types", stderr)
	kind := fs.String("kind", "station", "type kind: station, event, or edge")
	limit := fs.Int("limit", 200, "maximum number of types to request")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "mobility types does not accept positional arguments")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}

	path, normalizedKind, err := mobilityTypesPath(*kind)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	api, _ := r.Registry.Find("mobility")
	values := url.Values{}
	values.Set("limit", strconv.Itoa(*limit))
	requestURL, err := BuildURL(api.BaseURL, path, values)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	items := extractDataList(value)
	if err := output.WriteJSON(stdout, map[string]any{
		"kind":  normalizedKind,
		"count": len(items),
		"types": items,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
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

func (r *Runner) runMobilityDatatypes(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("mobility datatypes", stderr)
	stationType := fs.String("station-type", "", "station type, for example TrafficSensor")
	representation := fs.String("representation", "flat", "API representation")
	origin := fs.String("origin", "", "optional sorigin filter, for example A22")
	limit := fs.Int("limit", 1000, "maximum station/datatype records to inspect")
	params := paramValues{}
	fs.Var(&params, "param", "additional query parameter as key=value; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "mobility datatypes does not accept positional arguments")
		return 2
	}
	if strings.TrimSpace(*stationType) == "" {
		fmt.Fprintln(stderr, "--station-type is required")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}

	api, _ := r.Registry.Find("mobility")
	path := fmt.Sprintf("/v2/%s/%s/*", url.PathEscape(*representation), url.PathEscape(*stationType))
	path = strings.ReplaceAll(path, "%2C", ",")
	values := params.Values()
	values.Set("limit", strconv.Itoa(*limit))
	requestURL, err := BuildURL(api.BaseURL, path, values)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	records := extractDataList(value)
	summary := summarizeDatatypes(records, *origin)
	if err := output.WriteJSON(stdout, map[string]any{
		"station_type": *stationType,
		"origin":       strings.TrimSpace(*origin),
		"record_count": len(records),
		"count":        len(summary),
		"datatypes":    summary,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (r *Runner) runMobilityEvents(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("mobility events", stderr)
	origin := fs.String("origin", "", "event origin, for example A22")
	latest := fs.Bool("latest", true, "request latest events")
	representation := fs.String("representation", "flat", "API representation")
	limit := fs.Int("limit", 20, "maximum events to request")
	params := paramValues{}
	fs.Var(&params, "param", "additional query parameter as key=value; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "mobility events does not accept positional arguments")
		return 2
	}
	if strings.TrimSpace(*origin) == "" {
		fmt.Fprintln(stderr, "--origin is required")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}

	api, _ := r.Registry.Find("mobility")
	path := fmt.Sprintf("/v2/%s,event/%s", url.PathEscape(*representation), url.PathEscape(*origin))
	path = strings.ReplaceAll(path, "%2C", ",")
	if *latest {
		path += "/latest"
	}
	values := params.Values()
	values.Set("limit", strconv.Itoa(*limit))
	requestURL, err := BuildURL(api.BaseURL, path, values)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	events := extractDataList(value)
	if err := output.WriteJSON(stdout, map[string]any{
		"origin": *origin,
		"latest": *latest,
		"count":  len(events),
		"events": events,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (r *Runner) runA22(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: odh a22 <subcommand>")
		return 2
	}
	switch args[0] {
	case "status":
		return r.runA22Status(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown a22 subcommand %q\n", args[0])
		return 2
	}
}

func (r *Runner) runA22Status(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("a22 status", stderr)
	limit := fs.Int("limit", 20, "maximum records to request from each A22 feed")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "a22 status does not accept positional arguments")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}

	api, _ := r.Registry.Find("mobility")
	events, eventsURL, err := r.fetchListFromMobility(ctx, api.BaseURL, "/v2/flat,event/A22/latest", *limit)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	forecasts, forecastsURL, err := r.fetchListFromMobility(ctx, api.BaseURL, "/v2/flat/TrafficForecast/forecast/latest", *limit)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
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

	if err := output.WriteJSON(stdout, map[string]any{
		"source": "Open Data Hub Mobility API",
		"events": map[string]any{
			"endpoint": eventsURL,
			"count":    len(events),
			"items":    events,
		},
		"forecast": map[string]any{
			"endpoint": forecastsURL,
			"count":    len(forecasts),
			"items":    forecasts,
		},
		"warnings": warnings,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (r *Runner) fetchListFromMobility(ctx context.Context, baseURL, path string, limit int) ([]map[string]any, string, error) {
	values := url.Values{}
	values.Set("limit", strconv.Itoa(limit))
	requestURL, err := BuildURL(baseURL, path, values)
	if err != nil {
		return nil, "", err
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		return nil, requestURL, err
	}
	return extractDataList(value), requestURL, nil
}

func (r *Runner) fetchJSONValue(ctx context.Context, requestURL string) (any, error) {
	resp, err := r.Client.Get(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(resp.Body, &value); err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}
	return value, nil
}

type datatypeSummary struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Unit         string   `json:"unit,omitempty"`
	StationCount int      `json:"station_count"`
	Origins      []string `json:"origins,omitempty"`
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
