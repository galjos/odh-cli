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
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the curated dataset catalog",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			entries := filterDatasetsByDomain(datasetCatalog(), searchDomain)
			entries = filterDatasetsByQuery(entries, query)
			return writeDatasetEntries(cmd, entries, searchFormat)
		},
	}
	searchCmd.Flags().StringVar(&searchDomain, "domain", "", "optional domain filter, for example tourism or mobility")
	searchCmd.Flags().StringVar(&searchFormat, "format", "json", "output format: json or table")

	cmd.AddCommand(listCmd)
	cmd.AddCommand(searchCmd)
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
			Keywords:  []string{"parking", "car park", "capacity", "free spaces"},
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
			Description: "A22 current event checks, traffic sensor discovery, and traffic forecast caveats.",
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
			Description: "Opinionated Open Data Hub PROVINCE_BZ traffic-event view with zone filters, text search, type filters, date filtering, deduplication, and stale-record warnings.",
			Commands: []string{
				"odh traffic zones",
				"odh traffic categories",
				"odh traffic today --area ueberetsch-unterland --type roadworks --format table",
				"odh traffic today --zone-id 6 --type closure --json",
				"odh traffic events --area bozen-unterland --from 2026-05-16 --to 2026-05-16 --format json",
				"odh traffic search \"road closed badia\" --today --zone-id 6 --json",
				"odh traffic today --near 46.42,11.25 --radius 15km --json",
			},
			Endpoints: []string{"/v2/flat,event/PROVINCE_BZ/{from}/{to}"},
			Keywords:  []string{"traffic", "roadworks", "closure", "roadblock", "category", "zone", "zone-id", "unterland", "ueberetsch", "pustertal", "province_bz"},
		},
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries
}
