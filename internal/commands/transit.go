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
	"time"

	"github.com/galjos/odh-cli/internal/output"
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

func (r *Runner) runTransit(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: odh transit <stops|departures|trip|delay-stats>")
		return 2
	}
	switch args[0] {
	case "stops":
		return r.runTransitStops(ctx, args[1:], stdout, stderr)
	case "departures":
		return r.runTransitDepartures(ctx, args[1:], stdout, stderr)
	case "trip":
		return r.runTransitTrip(ctx, args[1:], stdout, stderr)
	case "delay-stats", "delay-probability":
		return r.runTransitDelayStats(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown transit subcommand %q\n", args[0])
		return 2
	}
}

func (r *Runner) runTransitStops(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "search" {
		fmt.Fprintln(stderr, "usage: odh transit stops search <query>")
		return 2
	}
	flagArgs, queryParts, err := splitTransitStopsSearchArgs(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fs := newFlagSet("transit stops search", stderr)
	dataset := fs.String("dataset", defaultGTFSDataset, "GTFS dataset id")
	limit := fs.Int("limit", 20, "maximum stops to return")
	cacheDir := fs.String("cache-dir", "", "directory for cached GTFS archives")
	refresh := fs.Bool("refresh", false, "refresh cached GTFS archive")
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}
	query := strings.TrimSpace(strings.Join(queryParts, " "))
	if query == "" {
		fmt.Fprintln(stderr, "usage: odh transit stops search <query>")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}
	archive, err := r.fetchGTFSArchive(ctx, *dataset, *cacheDir, *refresh)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	stops, err := readGTFSStops(archive.Path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	matches := searchGTFSStops(stops, query, *limit)
	if err := output.WriteJSON(stdout, map[string]any{
		"dataset": *dataset,
		"query":   query,
		"archive": archive,
		"count":   len(matches),
		"stops":   matches,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func splitTransitStopsSearchArgs(args []string) ([]string, []string, error) {
	flagArgs := make([]string, 0, len(args))
	queryParts := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dataset" || arg == "--limit" || arg == "--cache-dir":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("%s requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "--dataset=") || strings.HasPrefix(arg, "--limit=") || strings.HasPrefix(arg, "--cache-dir="):
			flagArgs = append(flagArgs, arg)
		case arg == "--refresh":
			flagArgs = append(flagArgs, arg)
		case strings.HasPrefix(arg, "-"):
			return nil, nil, fmt.Errorf("unknown flag %q", arg)
		default:
			queryParts = append(queryParts, arg)
		}
	}
	return flagArgs, queryParts, nil
}

func (r *Runner) runTransitDepartures(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("transit departures", stderr)
	dataset := fs.String("dataset", defaultGTFSDataset, "GTFS dataset id")
	stopQuery := fs.String("stop", "", "stop search query")
	dateText := fs.String("date", time.Now().Format("2006-01-02"), "service date YYYY-MM-DD")
	around := fs.String("around", "", "departure time HH:MM to search around")
	windowText := fs.String("window", defaultTransitWindow.String(), "time window around --around, for example 15m")
	mode := fs.String("mode", "all", "mode filter: all, train, bus, or cable-car")
	limit := fs.Int("limit", 20, "maximum departures to return")
	cacheDir := fs.String("cache-dir", "", "directory for cached GTFS archives")
	refresh := fs.Bool("refresh", false, "refresh cached GTFS archive")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "transit departures does not accept positional arguments")
		return 2
	}
	if strings.TrimSpace(*stopQuery) == "" {
		fmt.Fprintln(stderr, "--stop is required")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}
	query, err := parseTransitTimeQuery(*dateText, *around, *windowText)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	routeTypes, err := transitModeRouteTypes(*mode)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	archive, err := r.fetchGTFSArchive(ctx, *dataset, *cacheDir, *refresh)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	result, err := findTransitDepartures(archive.Path, *stopQuery, query, routeTypes, *limit)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	warnings := appendTransitStopMatchWarning(nil, "stop", *stopQuery, len(result.Stops))
	if err := output.WriteJSON(stdout, map[string]any{
		"dataset":       *dataset,
		"stop_query":    *stopQuery,
		"date":          query.Date.Format("2006-01-02"),
		"around":        query.AroundText,
		"window":        query.Window.String(),
		"mode":          normalizeTransitModeName(*mode),
		"archive":       archive,
		"matched_stops": result.Stops,
		"count":         len(result.Departures),
		"departures":    result.Departures,
		"warnings":      warnings,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (r *Runner) runTransitTrip(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("transit trip", stderr)
	dataset := fs.String("dataset", defaultGTFSDataset, "GTFS dataset id")
	fromQuery := fs.String("from", "", "origin stop query")
	toQuery := fs.String("to", "", "destination stop query")
	dateText := fs.String("date", time.Now().Format("2006-01-02"), "service date YYYY-MM-DD")
	timeText := fs.String("time", "", "origin departure time HH:MM")
	windowText := fs.String("window", defaultTransitWindow.String(), "time window around --time, for example 15m")
	mode := fs.String("mode", "all", "mode filter: all, train, bus, or cable-car")
	limit := fs.Int("limit", 20, "maximum direct trip matches to return")
	cacheDir := fs.String("cache-dir", "", "directory for cached GTFS archives")
	refresh := fs.Bool("refresh", false, "refresh cached GTFS archive")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "transit trip does not accept positional arguments")
		return 2
	}
	if strings.TrimSpace(*fromQuery) == "" {
		fmt.Fprintln(stderr, "--from is required")
		return 2
	}
	if strings.TrimSpace(*toQuery) == "" {
		fmt.Fprintln(stderr, "--to is required")
		return 2
	}
	if strings.TrimSpace(*timeText) == "" {
		fmt.Fprintln(stderr, "--time is required")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}
	query, err := parseTransitTimeQuery(*dateText, *timeText, *windowText)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	routeTypes, err := transitModeRouteTypes(*mode)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	archive, err := r.fetchGTFSArchive(ctx, *dataset, *cacheDir, *refresh)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	result, err := findTransitTripMatches(archive.Path, *fromQuery, *toQuery, query, routeTypes, *limit)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	warnings := make([]string, 0)
	if len(result.Matches) == 0 {
		warnings = append(warnings, "no direct GTFS trip matched; this command does not perform transfer routing")
	}
	warnings = appendTransitStopMatchWarning(warnings, "from", *fromQuery, len(result.FromStops))
	warnings = appendTransitStopMatchWarning(warnings, "to", *toQuery, len(result.ToStops))
	warnings = append(warnings, "historical delay probability is not available from the live GTFS API without an archived GTFS-RT snapshot dataset")
	if err := output.WriteJSON(stdout, map[string]any{
		"dataset":    *dataset,
		"from_query": *fromQuery,
		"to_query":   *toQuery,
		"date":       query.Date.Format("2006-01-02"),
		"time":       query.AroundText,
		"window":     query.Window.String(),
		"mode":       normalizeTransitModeName(*mode),
		"archive":    archive,
		"from_stops": result.FromStops,
		"to_stops":   result.ToStops,
		"count":      len(result.Matches),
		"matches":    result.Matches,
		"warnings":   warnings,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (r *Runner) runTransitDelayStats(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("transit delay-stats", stderr)
	fromQuery := fs.String("from", "", "origin stop query")
	toQuery := fs.String("to", "", "destination stop query")
	timeText := fs.String("time", "", "origin departure time HH:MM")
	weekday := fs.String("weekday", "", "weekday filter, for example saturday")
	since := fs.String("since", "", "archive range, for example 90d")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "transit delay-stats does not accept positional arguments")
		return 2
	}
	if err := output.WriteJSON(stdout, map[string]any{
		"supported": false,
		"reason":    "Open Data Hub GTFS exposes current static GTFS and live GTFS-RT feeds, but this CLI has no historical GTFS-RT archive to compute probabilities from.",
		"requested": map[string]string{
			"from":    strings.TrimSpace(*fromQuery),
			"to":      strings.TrimSpace(*toQuery),
			"time":    strings.TrimSpace(*timeText),
			"weekday": strings.TrimSpace(*weekday),
			"since":   strings.TrimSpace(*since),
		},
		"available_now": []string{
			"odh gtfs realtime --dataset sta-time-tables --feed trip-updates",
			"odh transit trip --from <stop> --to <stop> --date YYYY-MM-DD --time HH:MM",
			"odh transit departures --stop <stop> --date YYYY-MM-DD --around HH:MM",
		},
		"next_step": "add an explicit archive collector for GTFS-RT trip-updates before reporting delay probability or usual delay minutes",
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func appendTransitStopMatchWarning(warnings []string, label, query string, count int) []string {
	if count <= 10 {
		return warnings
	}
	return append(warnings, fmt.Sprintf("%s query %q matched %d stops; use odh transit stops search and a more specific stop name if results are noisy", label, strings.TrimSpace(query), count))
}

type transitTimeQuery struct {
	Date       time.Time
	AroundText string
	Around     int
	Window     time.Duration
	HasAround  bool
}

type transitDeparturesResult struct {
	Stops      []gtfsStop         `json:"stops"`
	Departures []transitDeparture `json:"departures"`
}

type transitTripResult struct {
	FromStops []gtfsStop         `json:"from_stops"`
	ToStops   []gtfsStop         `json:"to_stops"`
	Matches   []transitTripMatch `json:"matches"`
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

func findTransitDepartures(zipPath, stopQuery string, query transitTimeQuery, routeTypes map[string]struct{}, limit int) (transitDeparturesResult, error) {
	reader, closeFn, err := openGTFSZip(zipPath)
	if err != nil {
		return transitDeparturesResult{}, err
	}
	defer closeFn()
	stops, stopByID, err := loadGTFSStops(reader)
	if err != nil {
		return transitDeparturesResult{}, err
	}
	matchedStops := searchGTFSStops(stops, stopQuery, 50)
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
	return transitDeparturesResult{Stops: matchedStops, Departures: departures}, nil
}

func findTransitTripMatches(zipPath, fromQuery, toQuery string, query transitTimeQuery, routeTypes map[string]struct{}, limit int) (transitTripResult, error) {
	reader, closeFn, err := openGTFSZip(zipPath)
	if err != nil {
		return transitTripResult{}, err
	}
	defer closeFn()
	stops, stopByID, err := loadGTFSStops(reader)
	if err != nil {
		return transitTripResult{}, err
	}
	fromStops := searchGTFSStops(stops, fromQuery, 50)
	toStops := searchGTFSStops(stops, toQuery, 50)
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
	return transitTripResult{FromStops: fromStops, ToStops: toStops, Matches: matches}, nil
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
