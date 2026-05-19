// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/galjos/odh-cli/internal/output"
	"github.com/spf13/cobra"
)

func (r *Runner) newTourismCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tourism",
		Short: "Curated Tourism API commands",
		RunE:  requireSubcommand,
	}

	var poiLimit int
	var poiPage int
	var poiSeed string
	var poiFields string
	var poiParams []string
	poiCmd := &cobra.Command{
		Use:   "poi",
		Short: "Query Tourism POIs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if poiLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			if poiPage < 1 {
				return fmt.Errorf("--page must be greater than zero")
			}
			api, _ := r.Registry.Find("tourism")
			values := url.Values{}
			values.Set("pagenumber", strconv.Itoa(poiPage))
			values.Set("pagesize", strconv.Itoa(poiLimit))
			if strings.TrimSpace(poiSeed) != "" {
				values.Set("seed", poiSeed)
			}
			if strings.TrimSpace(poiFields) != "" {
				values.Set("fields", poiFields)
			}
			for _, p := range poiParams {
				key, value, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}
			requestURL, err := BuildURL(api.BaseURL, "/v1/ODHActivityPoi", values)
			if err != nil {
				return err
			}
			return r.fetchJSONCobra(cmd.Context(), requestURL, cmd.OutOrStdout())
		},
	}
	poiCmd.Flags().IntVar(&poiLimit, "limit", 1, "number of POIs to request")
	poiCmd.Flags().IntVar(&poiPage, "page", 1, "page number")
	poiCmd.Flags().StringVar(&poiSeed, "seed", "", "stable randomization seed")
	poiCmd.Flags().StringVar(&poiFields, "fields", "", "comma-separated fields")
	poiCmd.Flags().StringSliceVar(&poiParams, "param", nil, "additional query parameter as key=value; repeatable")

	var typesDataset string
	var typesLimit int
	var typesPage int
	var typesSeed string
	var typesParams []string
	typesCmd := &cobra.Command{
		Use:   "types",
		Short: "Discover Tourism taxonomy values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if typesLimit < 1 {
				return fmt.Errorf("--limit must be greater than zero")
			}
			if typesPage < 1 {
				return fmt.Errorf("--page must be greater than zero")
			}
			endpoint, normalizedDataset, err := tourismTypesEndpoint(typesDataset)
			if err != nil {
				return err
			}
			api, _ := r.Registry.Find("tourism")
			values := url.Values{}
			values.Set("pagenumber", strconv.Itoa(typesPage))
			values.Set("pagesize", strconv.Itoa(typesLimit))
			if strings.TrimSpace(typesSeed) != "" {
				values.Set("seed", typesSeed)
			}
			for _, p := range typesParams {
				key, value, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}
			requestURL, err := BuildURL(api.BaseURL, endpoint, values)
			if err != nil {
				return err
			}
			value, err := r.fetchJSONValueCached(cmd.Context(), requestURL, 24*time.Hour)
			if err != nil {
				return err
			}
			items := extractItemsList(value)
			return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"dataset":  normalizedDataset,
				"endpoint": requestURL,
				"count":    len(items),
				"items":    items,
			})
		},
	}
	typesCmd.Flags().StringVar(&typesDataset, "dataset", "poi", "taxonomy dataset: poi, event, event-topic, accommodation, article, venue, or tag")
	typesCmd.Flags().IntVar(&typesLimit, "limit", 100, "number of type records to request")
	typesCmd.Flags().IntVar(&typesPage, "page", 1, "page number")
	typesCmd.Flags().StringVar(&typesSeed, "seed", "", "stable randomization seed")
	typesCmd.Flags().StringSliceVar(&typesParams, "param", nil, "additional query parameter as key=value; repeatable")

	cmd.AddCommand(poiCmd)
	cmd.AddCommand(typesCmd)
	return cmd
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
