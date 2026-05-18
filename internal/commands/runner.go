// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/galjos/odh-cli/internal/apis"
	"github.com/galjos/odh-cli/internal/client"
	"github.com/galjos/odh-cli/internal/openapi"
	"github.com/galjos/odh-cli/internal/output"
)

const usageText = `odh is a JSON-first CLI for Open Data Hub APIs.

Usage:
  odh version [--format json|text]
  odh doctor [--network=false]
  odh apis
  odh datasets list [--domain tourism|mobility]
  odh datasets search <query>
  odh openapi <api>
  odh call <api> <path> [--param key=value ...]
  odh tourism poi [--limit n] [--seed value] [--fields fields]
  odh tourism types [--dataset poi|event|event-topic|accommodation|article|venue|tag]
  odh gtfs datasets
  odh gtfs realtime --dataset sta-time-tables --feed trip-updates
  odh transit stops search <query>
  odh transit departures (--stop query | --stop-id id) --date YYYY-MM-DD --around HH:MM
  odh transit trip (--from query | --from-stop-id id) (--to query | --to-stop-id id) --date YYYY-MM-DD --time HH:MM
  odh diagnostics ev-charging
  odh diagnostics parking-forecasts
  odh diagnostics tourism-events
  odh traffic zones [--format json|table|markdown]
  odh traffic categories [--format json|table|markdown]
  odh traffic today [--area area] [--type type] [--format json|table|markdown]
  odh traffic events [--area area] [--from date] [--to date]
  odh traffic search <text> [--today|--from date --to date]
  odh mobility types [--kind station|event|edge]
  odh mobility origins --station-type type
  odh mobility stations --station-type type [--origin origin]
  odh mobility datatypes --station-type type [--origin origin]
  odh mobility events --origin origin [--latest]
  odh mobility latest --station-type type --data-type type [--limit n] [--origin origin] [--active] [--fresh-within duration] [--sort newest|oldest|station|upstream] [--where expr]
  odh a22 status

Run "odh help" for examples.
`

// Runner owns command execution dependencies.
type Runner struct {
	Registry *apis.Registry
	Client   *client.Client
}

// NewDefaultRunner creates a runner for production use.
func NewDefaultRunner() *Runner {
	return &Runner{
		Registry: apis.DefaultRegistry(),
		Client:   client.New(30 * time.Second),
	}
}

// Run executes a command and returns a process exit code.
func (r *Runner) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if r.Registry == nil {
		r.Registry = apis.DefaultRegistry()
	}
	if r.Client == nil {
		r.Client = client.New(30 * time.Second)
	}
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, helpText())
		return 0
	case "version", "--version":
		return r.runVersion(args[1:], stdout, stderr)
	case "doctor":
		return r.runDoctor(ctx, args[1:], stdout, stderr)
	case "apis":
		return r.runAPIs(args[1:], stdout, stderr)
	case "datasets":
		return r.runDatasets(args[1:], stdout, stderr)
	case "openapi":
		return r.runOpenAPI(ctx, args[1:], stdout, stderr)
	case "call":
		return r.runCall(ctx, args[1:], stdout, stderr)
	case "tourism":
		return r.runTourism(ctx, args[1:], stdout, stderr)
	case "gtfs":
		return r.runGTFS(ctx, args[1:], stdout, stderr)
	case "transit":
		return r.runTransit(ctx, args[1:], stdout, stderr)
	case "diagnostics":
		return r.runDiagnostics(ctx, args[1:], stdout, stderr)
	case "traffic":
		return r.runTraffic(ctx, args[1:], stdout, stderr)
	case "mobility":
		return r.runMobility(ctx, args[1:], stdout, stderr)
	case "a22":
		return r.runA22(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}

func (r *Runner) runAPIs(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("apis", stderr)
	format := fs.String("format", "json", "output format: json or table")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "apis does not accept positional arguments")
		return 2
	}

	switch *format {
	case "json":
		if err := output.WriteJSON(stdout, map[string]any{"apis": r.Registry.List()}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "table":
		fmt.Fprintln(stdout, "NAME\tBASE URL\tOPENAPI")
		for _, api := range r.Registry.List() {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", api.Name, api.BaseURL, api.OpenAPIURL)
		}
	default:
		fmt.Fprintf(stderr, "unsupported format %q\n", *format)
		return 2
	}
	return 0
}

func (r *Runner) runOpenAPI(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("openapi", stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: odh openapi <api>")
		return 2
	}
	api, ok := r.Registry.Find(fs.Arg(0))
	if !ok {
		fmt.Fprintf(stderr, "unknown api %q; known APIs: %s\n", fs.Arg(0), strings.Join(r.Registry.Names(), ", "))
		return 2
	}
	if api.OpenAPIURL == "" {
		fmt.Fprintf(stderr, "api %q does not have a known OpenAPI URL\n", api.Name)
		return 1
	}
	resp, err := r.Client.Get(ctx, api.OpenAPIURL)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoded, _, err := openapi.ToJSON(resp.Body)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	_, err = stdout.Write(encoded)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (r *Runner) runCall(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	apiName, path, params, err := parseCallArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr, "usage: odh call <api> <path> [--param key=value ...]")
		return 2
	}
	api, ok := r.Registry.Find(apiName)
	if !ok {
		fmt.Fprintf(stderr, "unknown api %q; known APIs: %s\n", apiName, strings.Join(r.Registry.Names(), ", "))
		return 2
	}
	requestURL, err := BuildURL(api.BaseURL, path, params.Values())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return r.fetchJSON(ctx, requestURL, stdout, stderr)
}

func parseCallArgs(args []string) (string, string, paramValues, error) {
	var params paramValues
	positionals := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--param":
			if i+1 >= len(args) {
				return "", "", nil, errors.New("--param requires key=value")
			}
			i++
			if err := params.Set(args[i]); err != nil {
				return "", "", nil, err
			}
		case strings.HasPrefix(arg, "--param="):
			if err := params.Set(strings.TrimPrefix(arg, "--param=")); err != nil {
				return "", "", nil, err
			}
		case strings.HasPrefix(arg, "-"):
			return "", "", nil, fmt.Errorf("unknown flag %q", arg)
		default:
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) != 2 {
		return "", "", nil, errors.New("call requires exactly <api> and <path>")
	}
	return positionals[0], positionals[1], params, nil
}

func (r *Runner) runTourism(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: odh tourism <subcommand>")
		return 2
	}
	switch args[0] {
	case "poi":
		return r.runTourismPOI(ctx, args[1:], stdout, stderr)
	case "types":
		return r.runTourismTypes(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown tourism subcommand %q\n", args[0])
		return 2
	}
}

func (r *Runner) runTourismPOI(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("tourism poi", stderr)
	limit := fs.Int("limit", 1, "number of POIs to request")
	page := fs.Int("page", 1, "page number")
	seed := fs.String("seed", "", "stable randomization seed")
	fields := fs.String("fields", "", "comma-separated fields")
	params := paramValues{}
	fs.Var(&params, "param", "additional query parameter as key=value; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "tourism poi does not accept positional arguments")
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
	if strings.TrimSpace(*seed) != "" {
		values.Set("seed", *seed)
	}
	if strings.TrimSpace(*fields) != "" {
		values.Set("fields", *fields)
	}
	requestURL, err := BuildURL(api.BaseURL, "/v1/ODHActivityPoi", values)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return r.fetchJSON(ctx, requestURL, stdout, stderr)
}

func (r *Runner) runTourismTypes(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("tourism types", stderr)
	dataset := fs.String("dataset", "poi", "taxonomy dataset: poi, event, event-topic, accommodation, article, venue, or tag")
	limit := fs.Int("limit", 100, "number of type records to request")
	page := fs.Int("page", 1, "page number")
	seed := fs.String("seed", "", "stable randomization seed")
	params := paramValues{}
	fs.Var(&params, "param", "additional query parameter as key=value; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "tourism types does not accept positional arguments")
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
	endpoint, normalizedDataset, err := tourismTypesEndpoint(*dataset)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	api, _ := r.Registry.Find("tourism")
	values := params.Values()
	values.Set("pagenumber", strconv.Itoa(*page))
	values.Set("pagesize", strconv.Itoa(*limit))
	if strings.TrimSpace(*seed) != "" {
		values.Set("seed", *seed)
	}
	requestURL, err := BuildURL(api.BaseURL, endpoint, values)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	items := extractItemsList(value)
	if err := output.WriteJSON(stdout, map[string]any{
		"dataset":  normalizedDataset,
		"endpoint": requestURL,
		"count":    len(items),
		"items":    items,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func tourismTypesEndpoint(dataset string) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(dataset)) {
	case "", "poi", "pois", "activity", "activity-poi", "activity-pois":
		return "/v1/ODHActivityPoiTypes", "poi", nil
	case "event", "events", "event-type", "event-types":
		return "/v1/EventShortTypes", "event", nil
	case "event-topic", "event-topics", "topic", "topics":
		return "/v1/EventTopics", "event-topic", nil
	case "accommodation", "accommodations", "accommodation-type", "accommodation-types":
		return "/v1/AccommodationTypes", "accommodation", nil
	case "article", "articles", "article-type", "article-types":
		return "/v1/ArticleTypes", "article", nil
	case "venue", "venues", "venue-type", "venue-types":
		return "/v1/VenueTypes", "venue", nil
	case "tag", "tags", "odh-tag", "odh-tags":
		return "/v1/ODHTag", "tag", nil
	default:
		return "", "", fmt.Errorf("unsupported tourism types dataset %q", dataset)
	}
}

func (r *Runner) runMobility(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: odh mobility <subcommand>")
		return 2
	}
	switch args[0] {
	case "types":
		return r.runMobilityTypes(ctx, args[1:], stdout, stderr)
	case "origins":
		return r.runMobilityOrigins(ctx, args[1:], stdout, stderr)
	case "stations":
		return r.runMobilityStations(ctx, args[1:], stdout, stderr)
	case "datatypes":
		return r.runMobilityDatatypes(ctx, args[1:], stdout, stderr)
	case "events":
		return r.runMobilityEvents(ctx, args[1:], stdout, stderr)
	case "latest":
		return r.runMobilityLatest(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown mobility subcommand %q\n", args[0])
		return 2
	}
}

func (r *Runner) runMobilityLatest(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("mobility latest", stderr)
	stationType := fs.String("station-type", "", "station type, for example EChargingStation")
	dataType := fs.String("data-type", "", "data type, for example number-available")
	representation := fs.String("representation", "flat,node", "API representation")
	limit := fs.Int("limit", 5, "number of measurements to request")
	requestLimit := fs.Int("request-limit", 0, "raw upstream rows to request before local filtering")
	offset := fs.Int("offset", 0, "pagination offset")
	origin := fs.String("origin", "", "optional sorigin filter, for example ALPERIA")
	active := fs.Bool("active", false, "keep only active stations")
	freshWithin := fs.String("fresh-within", "", "keep only rows with mvalidtime within this age, for example 24h or 7d")
	sortMode := fs.String("sort", "upstream", "local sort: upstream, newest, oldest, or station")
	where := fs.String("where", "", "Open Data Hub where filter")
	params := paramValues{}
	fs.Var(&params, "param", "additional query parameter as key=value; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "mobility latest does not accept positional arguments")
		return 2
	}
	if strings.TrimSpace(*stationType) == "" {
		fmt.Fprintln(stderr, "--station-type is required")
		return 2
	}
	if strings.TrimSpace(*dataType) == "" {
		fmt.Fprintln(stderr, "--data-type is required")
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "--limit must be greater than zero")
		return 2
	}
	if *requestLimit < 0 {
		fmt.Fprintln(stderr, "--request-limit must not be negative")
		return 2
	}
	if *offset < 0 {
		fmt.Fprintln(stderr, "--offset must not be negative")
		return 2
	}
	sortModeValue, err := normalizeMobilityLatestSort(*sortMode)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	freshDuration, err := parseFreshWithin(*freshWithin)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	api, _ := r.Registry.Find("mobility")
	path := fmt.Sprintf("/v2/%s/%s/%s/latest", url.PathEscape(*representation), url.PathEscape(*stationType), url.PathEscape(*dataType))
	path = strings.ReplaceAll(path, "%2C", ",")
	values := params.Values()
	localProcessing := mobilityLatestNeedsLocalProcessing(*origin, *active, freshDuration, sortModeValue)
	upstreamLimit := *limit
	if localProcessing {
		upstreamLimit = *requestLimit
		if upstreamLimit == 0 {
			upstreamLimit = max(*limit, 1000)
		}
		if upstreamLimit < *limit {
			upstreamLimit = *limit
		}
	}
	values.Set("limit", strconv.Itoa(upstreamLimit))
	values.Set("offset", strconv.Itoa(*offset))
	if strings.TrimSpace(*where) != "" {
		values.Set("where", *where)
	}
	requestURL, err := BuildURL(api.BaseURL, path, values)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if localProcessing {
		value, err := r.fetchJSONValue(ctx, requestURL)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		records := extractDataList(value)
		result := filterMobilityLatest(records, mobilityLatestFilter{
			StationType:   *stationType,
			DataType:      *dataType,
			Origin:        *origin,
			ActiveOnly:    *active,
			FreshWithin:   *freshWithin,
			FreshDuration: freshDuration,
			Sort:          sortModeValue,
			Limit:         *limit,
			RequestLimit:  upstreamLimit,
			Endpoint:      requestURL,
			Now:           time.Now(),
		})
		if err := output.WriteJSON(stdout, result); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	return r.fetchJSON(ctx, requestURL, stdout, stderr)
}

func (r *Runner) fetchJSON(ctx context.Context, requestURL string, stdout, stderr io.Writer) int {
	resp, err := r.Client.Get(ctx, requestURL)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := output.WriteRawJSON(stdout, resp.Body); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// BuildURL joins a base URL, path, and query parameters.
func BuildURL(baseURL, path string, params url.Values) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", errors.New("base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid base URL %q", baseURL)
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		absolute, err := url.Parse(path)
		if err != nil {
			return "", err
		}
		parsed = absolute
	} else {
		joined, err := url.JoinPath(parsed.String(), path)
		if err != nil {
			return "", err
		}
		parsed, err = url.Parse(joined)
		if err != nil {
			return "", err
		}
	}
	query := parsed.Query()
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

type paramValues []string

func (p *paramValues) String() string {
	return strings.Join(*p, ",")
}

func (p *paramValues) Set(value string) error {
	if !strings.Contains(value, "=") {
		return fmt.Errorf("parameter %q must use key=value", value)
	}
	key, _, _ := strings.Cut(value, "=")
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("parameter %q has empty key", value)
	}
	*p = append(*p, value)
	return nil
}

func (p *paramValues) Values() url.Values {
	values := url.Values{}
	for _, item := range *p {
		key, value, _ := strings.Cut(item, "=")
		values.Add(key, value)
	}
	return values
}

func helpText() string {
	return usageText + `
Examples:
  odh version
  odh doctor
  odh apis
  odh datasets search parking
  odh openapi mobility
  odh call tourism /v1/ODHActivityPoi --param pagenumber=1 --param pagesize=1 --param seed=42
  odh tourism types --dataset event --limit 10
  odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
  odh gtfs datasets
  odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5
  odh transit stops search auer
  odh transit departures --stop "Ora, Stazione di Ora" --date 2026-05-16 --around 14:05
  odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train
  odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday
  odh diagnostics ev-charging
  odh diagnostics parking-forecasts --origin "Municipality Merano"
  odh diagnostics tourism-events --date 2026-05-18
  odh traffic zones
  odh traffic categories
  odh traffic today --area ueberetsch-unterland --type roadworks --format table
  odh traffic search "road closed badia" --today --zone-id 6 --json
  odh mobility types --kind event
  odh mobility origins --station-type TrafficSensor
  odh mobility stations --station-type ParkingStation --limit 5
  odh mobility datatypes --station-type TrafficSensor --origin A22
  odh mobility events --origin A22 --latest
  odh mobility latest --station-type EChargingStation --data-type number-available --limit 5
  odh a22 status
`
}
