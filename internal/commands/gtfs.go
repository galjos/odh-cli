// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/galjos/odh-cli/internal/output"
	"github.com/spf13/cobra"
)

const defaultGTFSDataset = "sta-time-tables"

type gtfsDataset struct {
	ID          string         `json:"id"`
	Description string         `json:"description,omitempty"`
	Origin      string         `json:"origin,omitempty"`
	License     string         `json:"license,omitempty"`
	Endpoint    string         `json:"endpoint,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Realtime    map[string]any `json:"realtime,omitempty"`
}

func (r *Runner) newGTFSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gtfs",
		Short: "Public transport GTFS commands",
		RunE:  requireSubcommand,
	}

	var datasetsFormat string
	datasetsCmd := &cobra.Command{
		Use:   "datasets",
		Short: "List GTFS datasets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, _ := r.Registry.Find("gtfs")
			requestURL, err := BuildURL(api.BaseURL, "/v1/dataset", nil)
			if err != nil {
				return err
			}
			value, err := r.fetchJSONValue(cmd.Context(), requestURL)
			if err != nil {
				return err
			}
			datasets := normalizeGTFSDatasets(value)
			switch strings.ToLower(strings.TrimSpace(datasetsFormat)) {
			case "json", "":
				return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
					"endpoint": requestURL,
					"count":    len(datasets),
					"datasets": datasets,
				})
			case "table":
				fmt.Fprintln(cmd.OutOrStdout(), "ID\tORIGIN\tDESCRIPTION\tREALTIME")
				for _, dataset := range datasets {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%t\n", dataset.ID, dataset.Origin, dataset.Description, len(dataset.Realtime) > 0)
				}
				return nil
			default:
				return fmt.Errorf("unsupported format %q", datasetsFormat)
			}
		},
	}
	datasetsCmd.Flags().StringVar(&datasetsFormat, "format", "json", "output format: json or table")

	var realtimeDataset string
	var realtimeFeed string
	var realtimeLimit int
	var realtimeTripID string
	var realtimeRouteID string
	var realtimeRaw bool
	realtimeCmd := &cobra.Command{
		Use:   "realtime",
		Short: "Inspect GTFS-RT realtime feeds",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(realtimeDataset) == "" {
				return fmt.Errorf("--dataset is required")
			}
			normalizedFeed, err := normalizeGTFSFeed(realtimeFeed)
			if err != nil {
				return err
			}
			if realtimeLimit < 0 {
				return fmt.Errorf("--limit must not be negative")
			}

			api, _ := r.Registry.Find("gtfs")
			path := fmt.Sprintf("/v1/realtime/%s/%s", url.PathEscape(strings.TrimSpace(realtimeDataset)), normalizedFeed)
			requestURL, err := BuildURL(api.BaseURL, path, nil)
			if err != nil {
				return err
			}
			if realtimeRaw && strings.TrimSpace(realtimeTripID) == "" && strings.TrimSpace(realtimeRouteID) == "" && realtimeLimit == 0 {
				return r.fetchJSONCobra(cmd.Context(), requestURL, cmd.OutOrStdout())
			}
			value, err := r.fetchJSONValue(cmd.Context(), requestURL)
			if err != nil {
				return err
			}
			object, _ := value.(map[string]any)
			entities := mapsFromList(asAnySlice(object["entity"]))
			entities = filterGTFSRealtimeEntities(entities, realtimeTripID, realtimeRouteID)
			total := len(entities)
			if realtimeLimit > 0 && len(entities) > realtimeLimit {
				entities = entities[:realtimeLimit]
			}
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"dataset":      strings.TrimSpace(realtimeDataset),
				"feed":         normalizedFeed,
				"endpoint":     requestURL,
				"entity_count": total,
				"count":        len(entities),
				"header":       object["header"],
				"entities":     entities,
			})
		},
	}
	realtimeCmd.Flags().StringVar(&realtimeDataset, "dataset", defaultGTFSDataset, "GTFS dataset id")
	realtimeCmd.Flags().StringVar(&realtimeFeed, "feed", "trip-updates", "feed: trip-updates, vehicle-positions, or service-alerts")
	realtimeCmd.Flags().IntVar(&realtimeLimit, "limit", 20, "maximum entities to include; use 0 for all")
	realtimeCmd.Flags().StringVar(&realtimeTripID, "trip-id", "", "optional GTFS trip_id filter for trip-updates")
	realtimeCmd.Flags().StringVar(&realtimeRouteID, "route-id", "", "optional route_id filter for trip-updates or vehicle-positions")
	realtimeCmd.Flags().BoolVar(&realtimeRaw, "raw", false, "write the upstream JSON feed without wrapping or filtering")

	cmd.AddCommand(datasetsCmd)
	cmd.AddCommand(realtimeCmd)
	return cmd
}

func normalizeGTFSDatasets(value any) []gtfsDataset {
	object, _ := value.(map[string]any)
	datasets := make([]gtfsDataset, 0, len(object))
	for id, raw := range object {
		item, _ := raw.(map[string]any)
		metadata, _ := item["metadata"].(map[string]any)
		realtime, _ := item["realtime"].(map[string]any)
		datasets = append(datasets, gtfsDataset{
			ID:          id,
			Description: asString(item["description"]),
			Origin:      asString(item["origin"]),
			License:     asString(item["license"]),
			Endpoint:    asString(item["endpoint"]),
			Metadata:    metadata,
			Realtime:    realtime,
		})
	}
	sort.Slice(datasets, func(i, j int) bool {
		return datasets[i].ID < datasets[j].ID
	})
	return datasets
}

func normalizeGTFSFeed(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "trip", "trips", "trip-update", "trip-updates", "trip_updates":
		return "trip-updates", nil
	case "vehicle", "vehicles", "vehicle-position", "vehicle-positions", "vehicle_positions":
		return "vehicle-positions", nil
	case "alert", "alerts", "service-alert", "service-alerts", "service_alerts":
		return "service-alerts", nil
	default:
		return "", fmt.Errorf("unsupported GTFS realtime feed %q", value)
	}
}

func filterGTFSRealtimeEntities(entities []map[string]any, tripID, routeID string) []map[string]any {
	tripID = strings.TrimSpace(tripID)
	routeID = strings.TrimSpace(routeID)
	if tripID == "" && routeID == "" {
		return entities
	}
	filtered := make([]map[string]any, 0, len(entities))
	for _, entity := range entities {
		trip := gtfsRealtimeTripObject(entity)
		if tripID != "" && asString(trip["trip_id"]) != tripID {
			continue
		}
		if routeID != "" && asString(trip["route_id"]) != routeID {
			continue
		}
		filtered = append(filtered, entity)
	}
	return filtered
}

func gtfsRealtimeTripObject(entity map[string]any) map[string]any {
	for _, key := range []string{"trip_update", "vehicle"} {
		parent, _ := entity[key].(map[string]any)
		trip, _ := parent["trip"].(map[string]any)
		if len(trip) > 0 {
			return trip
		}
	}
	return nil
}

func asAnySlice(value any) []any {
	items, _ := value.([]any)
	return items
}
