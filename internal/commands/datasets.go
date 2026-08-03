// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/galjos/odh-cli/internal/output"
	"github.com/spf13/cobra"
)

type datasetEntry struct {
	ID          string   `json:"id"`
	Domain      string   `json:"domain"`
	API         string   `json:"api"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Commands    []string `json:"commands,omitempty"`
	Endpoints   []string `json:"endpoints,omitempty"`
	Keywords    []string `json:"-"`
}

type datasetGuideEntry struct {
	Dataset   datasetEntry `json:"dataset"`
	Discovery []string     `json:"discovery"`
	Verify    []string     `json:"verify"`
	Caveats   []string     `json:"caveats,omitempty"`
}

func (r *Runner) newDatasetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "datasets",
		Short: "Search and list Open Data Hub datasets",
		RunE:  requireSubcommand,
	}

	var listDomain string
	var listFormat string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List known datasets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries := filterDatasetsByDomain(datasetCatalog(), listDomain)
			return writeDatasetEntries(cmd, entries, listFormat)
		},
	}
	listCmd.Flags().StringVar(&listDomain, "domain", "", "optional domain filter, for example tourism or mobility")
	listCmd.Flags().StringVar(&listFormat, "format", "json", "output format: json or table")

	var searchDomain string
	var searchFormat string
	var searchJSON bool
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the curated dataset catalog",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&searchFormat, searchJSON)
			query := strings.Join(args, " ")
			entries := filterDatasetsByDomain(datasetCatalog(), searchDomain)
			entries = filterDatasetsByQuery(entries, query)
			return writeDatasetEntries(cmd, entries, searchFormat)
		},
	}
	searchCmd.Flags().StringVar(&searchDomain, "domain", "", "optional domain filter, for example tourism or mobility")
	searchCmd.Flags().StringVar(&searchFormat, "format", "json", "output format: json or table")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "shortcut for --format json")

	var guideDomain string
	var guideFormat string
	var guideLimit int
	guideCmd := &cobra.Command{
		Use:   "guide <query>",
		Short: "Suggest discovery and verification commands for a dataset question",
		Long: `Suggest a discovery-first command path for a data question.

The guide is based on the curated dataset catalog. It returns matching datasets,
the discovery commands to run before guessing upstream vocabulary, verification
commands that check whether open rows exist, and caveats that should be carried
into agent answers.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if guideLimit < 0 {
				return usageErrorf("--limit must not be negative")
			}
			query := strings.Join(args, " ")
			entries := filterDatasetsByDomain(datasetCatalog(), guideDomain)
			entries = filterDatasetsByQuery(entries, query)
			if guideLimit > 0 && len(entries) > guideLimit {
				entries = entries[:guideLimit]
			}
			guides := make([]datasetGuideEntry, 0, len(entries))
			for _, entry := range entries {
				guides = append(guides, datasetGuideFor(entry))
			}
			return writeDatasetGuide(cmd, query, guides, guideFormat)
		},
	}
	guideCmd.Flags().StringVar(&guideDomain, "domain", "", "optional domain filter, for example tourism or mobility")
	guideCmd.Flags().StringVar(&guideFormat, "format", "json", "output format: json or table")
	guideCmd.Flags().IntVar(&guideLimit, "limit", 3, "maximum number of matching datasets to guide; 0 means no limit")

	cmd.AddCommand(listCmd)
	cmd.AddCommand(searchCmd)
	cmd.AddCommand(guideCmd)
	return cmd
}

func writeDatasetEntries(cmd *cobra.Command, entries []datasetEntry, format string) error {
	switch format {
	case "json":
		return output.WriteJSON(cmd.OutOrStdout(), map[string]any{"count": len(entries), "datasets": entries})
	case "table":
		fmt.Fprintln(cmd.OutOrStdout(), "ID\tDOMAIN\tAPI\tTITLE")
		for _, entry := range entries {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", entry.ID, entry.Domain, entry.API, entry.Title)
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func writeDatasetGuide(cmd *cobra.Command, query string, guides []datasetGuideEntry, format string) error {
	switch format {
	case "json":
		return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
			"source":   "odh curated dataset catalog",
			"query":    query,
			"count":    len(guides),
			"matches":  guides,
			"warnings": datasetGuideWarnings(),
		})
	case "table":
		fmt.Fprintln(cmd.OutOrStdout(), "ID\tTITLE\tFIRST_DISCOVERY\tFIRST_VERIFY")
		for _, guide := range guides {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n",
				guide.Dataset.ID,
				guide.Dataset.Title,
				firstString(guide.Discovery),
				firstString(guide.Verify),
			)
		}
		if len(guides) > 0 {
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Run --format json for the full discovery path and caveats.")
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func filterDatasetsByDomain(entries []datasetEntry, domain string) []datasetEntry {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return entries
	}
	filtered := make([]datasetEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Domain == domain {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func filterDatasetsByQuery(entries []datasetEntry, query string) []datasetEntry {
	terms := strings.Fields(strings.ToLower(query))
	filtered := make([]datasetEntry, 0, len(entries))
	for _, entry := range entries {
		haystack := strings.ToLower(strings.Join(append([]string{
			entry.ID,
			entry.Domain,
			entry.API,
			entry.Title,
			entry.Description,
		}, entry.Keywords...), " "))
		matched := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matched = false
				break
			}
		}
		if matched {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func datasetGuideFor(entry datasetEntry) datasetGuideEntry {
	guidance := datasetGuideEntry{Dataset: entry}
	switch entry.ID {
	case "gtfs.transit":
		guidance.Discovery = []string{
			"odh datasets search train",
			"odh gtfs datasets --json",
			"odh transit stops search <place> --limit 10",
		}
		guidance.Verify = []string{
			"odh transit departures --stop <stop-name-or-id> --date <YYYY-MM-DD> --around <HH:MM> --json",
			"odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5 --json",
		}
		guidance.Caveats = []string{
			"Static GTFS routing is timetable-based; --with-realtime annotates static journeys but does not reroute live.",
			"GTFS-RT is a current snapshot, not a historical delay archive.",
		}
	case "mobility.a22":
		guidance.Discovery = []string{
			"odh datasets search a22",
			"odh mobility origins --station-type TrafficSensor --limit 1000 --json",
			"odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 1000 --json",
		}
		guidance.Verify = []string{
			"odh a22 status --limit 10 --json",
		}
		guidance.Caveats = []string{
			"TrafficForecast rows are forecast data, not proof of current incidents.",
			"Open Data Hub does not provide a historical A22 incident archive through this CLI.",
		}
	case "mobility.charging":
		guidance.Discovery = []string{
			"odh mobility origins --station-type EChargingStation --limit 1000 --json",
			"odh mobility datatypes --station-type EChargingStation --limit 1000 --json",
			"odh mobility datatypes --station-type EChargingPlug --limit 1000 --json",
		}
		guidance.Verify = []string{
			"odh diagnostics ev-charging --fresh-within 24h",
			"odh mobility latest --station-type EChargingStation --data-type number-available --active --fresh-within 24h --sort newest --limit 5 --json",
		}
		guidance.Caveats = []string{
			"Open Data Hub exposes EV availability data, not public per-kWh tariff data.",
			"A catalogued datatype is not proof that fresh open measurements exist; verify with latest rows or diagnostics.",
		}
	case "mobility.parking":
		guidance.Discovery = []string{
			"odh mobility origins --station-type ParkingStation --limit 1000 --json",
			"odh mobility datatypes --station-type ParkingStation --limit 1000 --json",
		}
		guidance.Verify = []string{
			"odh diagnostics parking-forecasts --fresh-within 2h",
			"odh mobility latest --station-type ParkingStation --active --fresh-within 2h --sort newest --limit 5 --json",
		}
		guidance.Caveats = []string{
			"Parking forecasts may be stale or unavailable even when current occupancy is fresh.",
			"Report current and forecast freshness separately.",
		}
	case "mobility.traffic-events":
		guidance.Discovery = []string{
			"odh traffic zones --json",
			"odh traffic categories --json",
		}
		guidance.Verify = []string{
			"odh traffic today --area <area> --type <category> --json",
			"odh traffic search <text> --today --json",
			"odh traffic today --source content --json",
		}
		guidance.Caveats = []string{
			"Open Data Hub PROVINCE_BZ is a public bulletin feed, not a complete live road bulletin.",
			"Stale open-ended rows are hidden by default; carry warnings into answers.",
			"--source content reads the Content API bulletin the province still updates, but cannot answer --zone-id, --area or --road and rejects them.",
		}
	case "tourism.events":
		guidance.Discovery = []string{
			"odh tourism types --dataset event --json",
			"odh tourism types --dataset event-topic --json",
		}
		guidance.Verify = []string{
			"odh diagnostics tourism-events --date <YYYY-MM-DD>",
			"odh call tourism /v1/EventShort --param pagenumber=1 --param pagesize=5",
		}
		guidance.Caveats = []string{
			"Tourism API active flags may not match the user's requested date; run diagnostics for date-sensitive event answers.",
			"Verify coordinates before making local geography claims.",
		}
	case "tourism.poi":
		guidance.Discovery = []string{
			"odh tourism types --dataset poi --json",
			"odh tourism types --dataset tag --json",
		}
		guidance.Verify = []string{
			"odh tourism poi --limit 5 --fields Detail.en.Title,GpsInfo",
		}
		guidance.Caveats = []string{
			"Tourism POI records may have multilingual fields and sparse coordinates; inspect returned fields before answering.",
		}
	default:
		guidance.Discovery = append([]string{}, entry.Commands...)
		guidance.Verify = append([]string{}, entry.Commands...)
		guidance.Caveats = []string{
			"Start with discovery commands, then verify open rows before answering from the catalog alone.",
			"Prefer returned source and warning fields over inferred provenance.",
		}
	}
	return guidance
}

func datasetGuideWarnings() []string {
	return []string{
		"This guide is a curated starting path, not proof that open rows exist.",
		"Run the discovery commands before guessing upstream station types, origins, datatypes, zone IDs, or taxonomies.",
		"Run the verification commands and carry source, freshness, and warning fields into the final answer.",
	}
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func datasetCatalog() []datasetEntry {
	entries := []datasetEntry{
		{
			ID:          "tourism.poi",
			Domain:      "tourism",
			API:         "tourism",
			Title:       "Tourism POIs and activities",
			Description: "Points of interest, activities, routes, and related tourism objects.",
			Commands: []string{
				"odh tourism poi --limit 5 --fields Detail.en.Title,GpsInfo",
				"odh tourism types --dataset poi",
			},
			Endpoints: []string{"/v1/ODHActivityPoi", "/v1/ODHActivityPoiTypes"},
			Keywords:  []string{"activity", "activities", "route", "routes", "hiking", "bike", "culture"},
		},
		{
			ID:          "tourism.events",
			Domain:      "tourism",
			API:         "tourism",
			Title:       "Tourism events",
			Description: "Event and event-short content with event type and topic taxonomies.",
			Commands: []string{
				"odh call tourism /v1/EventShort --param pagenumber=1 --param pagesize=5",
				"odh tourism types --dataset event",
				"odh tourism types --dataset event-topic",
			},
			Endpoints: []string{"/v1/EventShort", "/v1/EventShortTypes", "/v1/EventTopics"},
			Keywords:  []string{"event", "events", "calendar", "topic", "topics"},
		},
		{
			ID:          "tourism.accommodation",
			Domain:      "tourism",
			API:         "tourism",
			Title:       "Accommodation",
			Description: "Accommodation records, rooms, availability, features, and accommodation types.",
			Commands: []string{
				"odh call tourism /v1/Accommodation --param pagenumber=1 --param pagesize=5",
				"odh tourism types --dataset accommodation",
			},
			Endpoints: []string{"/v1/Accommodation", "/v1/AccommodationTypes", "/v1/AccommodationFeatures"},
			Keywords:  []string{"hotel", "hotels", "room", "rooms", "availability", "lodging"},
		},
		{
			ID:          "tourism.weather",
			Domain:      "tourism",
			API:         "tourism",
			Title:       "Weather and snow reports",
			Description: "Weather bulletin, forecasts, real-time weather, measuring points, and snow reports.",
			Commands: []string{
				"odh call tourism /v1/Weather --param pagenumber=1 --param pagesize=1",
				"odh call tourism /v1/Weather/SnowReport --param pagenumber=1 --param pagesize=5",
			},
			Endpoints: []string{"/v1/Weather", "/v1/Weather/Forecast", "/v1/Weather/Realtime", "/v1/Weather/SnowReport"},
			Keywords:  []string{"forecast", "snow", "temperature", "meteo", "weather"},
		},
		{
			ID:          "mobility.station-types",
			Domain:      "mobility",
			API:         "mobility",
			Title:       "Mobility station types",
			Description: "Discovery endpoint for available Mobility station, event, and edge types.",
			Commands: []string{
				"odh mobility types --kind station",
				"odh mobility types --kind event",
			},
			Endpoints: []string{"/v2/flat", "/v2/flat,event", "/v2/flat,edge"},
			Keywords:  []string{"station", "stations", "types", "origins"},
		},
		{
			ID:          "gtfs.transit",
			Domain:      "mobility",
			API:         "gtfs",
			Title:       "Public transport GTFS and realtime feeds",
			Description: "Static GTFS datasets, STA train and bus timetables, GTFS-RT trip updates, vehicle positions, and service alerts.",
			Commands: []string{
				"odh gtfs datasets",
				"odh gtfs realtime --dataset sta-time-tables --feed trip-updates --limit 5",
				"odh transit stops search auer",
				"odh transit departures --stop \"Ora, Stazione di Ora\" --date 2026-05-16 --around 14:05",
				"odh transit trip --from auer --to brenner --date 2026-05-16 --time 14:05 --mode train",
				"odh transit journey --from auer --to brenner --date 2026-05-16 --time 14:05 --max-transfers 2",
			},
			Endpoints: []string{"/v1/dataset", "/v1/dataset/{datasetId}/raw", "/v1/realtime/{datasetId}/{feedType}"},
			Keywords:  []string{"gtfs", "gtfs-rt", "transit", "public transport", "train", "trains", "bus", "delay", "delays", "trip updates", "sta", "timetable", "schedule", "connection"},
		},
		{
			ID:          "mobility.parking",
			Domain:      "mobility",
			API:         "mobility",
			Title:       "Parking stations",
			Description: "Parking station metadata and latest parking measurements.",
			Commands: []string{
				"odh mobility stations --station-type ParkingStation --limit 5",
				"odh mobility datatypes --station-type ParkingStation",
			},
			Endpoints: []string{"/v2/flat/ParkingStation", "/v2/flat/ParkingStation/*/latest"},
			Keywords:  []string{"parking", "car park", "capacity", "free spaces", "availability", "forecast", "forecasts"},
		},
		{
			ID:          "mobility.charging",
			Domain:      "mobility",
			API:         "mobility",
			Title:       "E-charging stations",
			Description: "Electric charging station metadata and availability measurements.",
			Commands: []string{
				"odh mobility stations --station-type EChargingStation --limit 5",
				"odh mobility latest --station-type EChargingStation --data-type number-available --limit 5",
			},
			Endpoints: []string{"/v2/flat/EChargingStation", "/v2/flat,node/EChargingStation/number-available/latest"},
			Keywords:  []string{"charging", "charger", "electric", "ev", "available"},
		},
		{
			ID:          "mobility.a22",
			Domain:      "mobility",
			API:         "mobility",
			Title:       "A22 traffic diagnostics",
			Description: "A22 event feed checks, traffic sensor discovery, and traffic forecast caveats.",
			Commands: []string{
				"odh a22 status --limit 10",
				"odh mobility origins --station-type TrafficSensor",
				"odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 1000",
			},
			Endpoints: []string{"/v2/flat,event/A22/latest", "/v2/flat/TrafficForecast/forecast/latest"},
			Keywords:  []string{"traffic", "a22", "autobrennero", "forecast", "incident", "sensor"},
		},
		{
			ID:          "mobility.traffic-events",
			Domain:      "mobility",
			API:         "mobility",
			Title:       "Traffic events and roadworks",
			Description: "Opinionated Open Data Hub PROVINCE_BZ traffic-event view with zone filters, text search, type filters, date filtering, deduplication, and stale-record warnings. --source content answers the same commands from the Content API Announcement bulletin.",
			Commands: []string{
				"odh traffic zones",
				"odh traffic categories",
				"odh traffic today --area ueberetsch-unterland --type roadworks --format table",
				"odh traffic today --zone-id 6 --type closure --json",
				"odh traffic events --area bozen-unterland --from 2026-05-16 --to 2026-05-16 --format json",
				"odh traffic search \"road closed badia\" --today --zone-id 6 --json",
				"odh traffic today --near 46.42,11.25 --radius 15km --json",
				"odh traffic today --source content --json",
			},
			Endpoints: []string{"/v2/flat,event/PROVINCE_BZ/{from}/{to}", "/v1/Announcement?source=PROVINCE_BZ"},
			Keywords:  []string{"traffic", "roadworks", "closure", "roadblock", "category", "zone", "zone-id", "unterland", "ueberetsch", "pustertal", "province_bz", "announcement", "bulletin"},
		},
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries
}
