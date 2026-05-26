// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
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
	poiCmd.Flags().StringArrayVar(&poiParams, "param", nil, "additional query parameter as key=value; repeatable; values may contain commas")

	var typesDataset string
	var typesLimit int
	var typesPage int
	var typesSeed string
	var typesParams []string
	var typesFormat string
	var typesJSON bool
	typesCmd := &cobra.Command{
		Use:   "types",
		Short: "Discover Tourism taxonomy values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyJSONShortcut(&typesFormat, typesJSON)
			format, err := normalizeOutputFormat(typesFormat)
			if err != nil {
				return err
			}
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
			return writeTourismTypesOutput(cmd.OutOrStdout(), tourismTypesOutput{
				Source:       "Open Data Hub Tourism API",
				SourceDetail: "Tourism taxonomy/type endpoint",
				Dataset:      normalizedDataset,
				Endpoint:     requestURL,
				Count:        len(items),
				Items:        items,
				Format:       format,
			})
		},
	}
	typesCmd.Flags().StringVar(&typesDataset, "dataset", "poi", "taxonomy dataset: poi, event, event-topic, accommodation, article, venue, or tag")
	typesCmd.Flags().IntVar(&typesLimit, "limit", 100, "number of type records to request")
	typesCmd.Flags().IntVar(&typesPage, "page", 1, "page number")
	typesCmd.Flags().StringVar(&typesSeed, "seed", "", "stable randomization seed")
	typesCmd.Flags().StringArrayVar(&typesParams, "param", nil, "additional query parameter as key=value; repeatable; values may contain commas")
	typesCmd.Flags().StringVar(&typesFormat, "format", "table", "output format: json, table, or markdown")
	typesCmd.Flags().BoolVar(&typesJSON, "json", false, "shortcut for --format json")

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

type tourismTypesOutput struct {
	Source       string `json:"source"`
	SourceDetail string `json:"source_detail"`
	Dataset      string `json:"dataset"`
	Endpoint     string `json:"endpoint"`
	Count        int    `json:"count"`
	Items        []any  `json:"items"`
	Format       string `json:"-"`
}

type tourismTypeRow struct {
	ID     string
	Key    string
	Type   string
	Parent string
	EN     string
	DE     string
	IT     string
}

func writeTourismTypesOutput(stdout io.Writer, result tourismTypesOutput) error {
	switch result.Format {
	case "", "json":
		return output.WriteJSON(stdout, result)
	case "table":
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tKEY\tTYPE\tPARENT\tEN\tDE\tIT")
		for _, item := range result.Items {
			row := summarizeTourismTypeRow(item)
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				row.ID,
				row.Key,
				row.Type,
				row.Parent,
				compactText(row.EN, 48),
				compactText(row.DE, 48),
				compactText(row.IT, 48),
			)
		}
		return tw.Flush()
	case "markdown":
		fmt.Fprintln(stdout, "| id | key | type | parent | en | de | it |")
		fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- | --- | --- |")
		for _, item := range result.Items {
			row := summarizeTourismTypeRow(item)
			fmt.Fprintf(stdout, "| %s | %s | %s | %s | %s | %s | %s |\n",
				escapeMarkdown(row.ID),
				escapeMarkdown(row.Key),
				escapeMarkdown(row.Type),
				escapeMarkdown(row.Parent),
				escapeMarkdown(compactText(row.EN, 48)),
				escapeMarkdown(compactText(row.DE, 48)),
				escapeMarkdown(compactText(row.IT, 48)),
			)
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", result.Format)
	}
}

func summarizeTourismTypeRow(item any) tourismTypeRow {
	record, _ := item.(map[string]any)
	descriptions, _ := record["TypeDesc"].(map[string]any)
	return tourismTypeRow{
		ID:     asString(record["Id"]),
		Key:    asString(record["Key"]),
		Type:   asString(record["Type"]),
		Parent: asString(record["Parent"]),
		EN:     asString(descriptions["en"]),
		DE:     asString(descriptions["de"]),
		IT:     asString(descriptions["it"]),
	}
}
