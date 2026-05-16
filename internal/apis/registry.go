// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package apis

import (
	"fmt"
	"sort"
	"strings"
)

// API describes one public Open Data Hub API surface known to the CLI.
type API struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	BaseURL     string   `json:"base_url"`
	OpenAPIURL  string   `json:"openapi_url,omitempty"`
	DocsURL     string   `json:"docs_url,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Public      bool     `json:"public"`
}

// Registry stores APIs by name and alias.
type Registry struct {
	apis    []API
	aliases map[string]int
}

// NewRegistry builds a registry and validates names and aliases.
func NewRegistry(entries []API) (*Registry, error) {
	aliases := make(map[string]int, len(entries))
	for i, api := range entries {
		if strings.TrimSpace(api.Name) == "" {
			return nil, fmt.Errorf("api at index %d has empty name", i)
		}
		if strings.TrimSpace(api.BaseURL) == "" {
			return nil, fmt.Errorf("api %q has empty base URL", api.Name)
		}
		keys := append([]string{api.Name}, api.Aliases...)
		for _, key := range keys {
			normalized := NormalizeName(key)
			if normalized == "" {
				return nil, fmt.Errorf("api %q has empty alias", api.Name)
			}
			if _, exists := aliases[normalized]; exists {
				return nil, fmt.Errorf("duplicate api name or alias %q", key)
			}
			aliases[normalized] = i
		}
	}
	return &Registry{apis: append([]API(nil), entries...), aliases: aliases}, nil
}

// DefaultRegistry returns the public API set used by the CLI.
func DefaultRegistry() *Registry {
	registry, err := NewRegistry([]API{
		{
			Name:        "tourism",
			Title:       "Open Data Hub Tourism API",
			Description: "Content API for tourism, points of interest, events, gastronomy, accommodation, and related datasets.",
			BaseURL:     "https://tourism.opendatahub.com",
			OpenAPIURL:  "https://tourism.opendatahub.com/swagger/v1/swagger.json",
			DocsURL:     "https://tourism.opendatahub.com/swagger/index.html",
			Aliases:     []string{"content"},
			Public:      true,
		},
		{
			Name:        "mobility",
			Title:       "Open Data Hub Timeseries / Mobility API",
			Description: "Time series API for mobility stations, events, data types, and latest or historical measurements.",
			BaseURL:     "https://mobility.api.opendatahub.com",
			OpenAPIURL:  "https://mobility.api.opendatahub.com/v2/apispec",
			DocsURL:     "https://docs.opendatahub.com/en/latest/howto/mobility/getstarted.html",
			Public:      true,
		},
		{
			Name:        "gtfs",
			Title:       "Open Data Hub GTFS API",
			Description: "Public transport GTFS static datasets and GTFS-RT realtime feeds.",
			BaseURL:     "https://gtfs.api.opendatahub.com",
			OpenAPIURL:  "https://gtfs.api.opendatahub.com/v1/apispec",
			DocsURL:     "https://opendatahub.com/api/",
			Public:      true,
		},
		{
			Name:        "gbfs",
			Title:       "Open Data Hub GBFS API",
			Description: "GBFS bike-sharing and mobility feed access surface.",
			BaseURL:     "https://gbfs.api.opendatahub.com",
			DocsURL:     "https://opendatahub.com/api/",
			Public:      true,
		},
		{
			Name:        "transmodel",
			Title:       "Open Data Hub Transmodel API",
			Description: "Transmodel wrapper APIs for NeTEx and SIRI related data.",
			BaseURL:     "https://transmodel.api.opendatahub.com",
			OpenAPIURL:  "https://transmodel.api.opendatahub.com/apispec",
			DocsURL:     "https://opendatahub.com/api/",
			Public:      true,
		},
		{
			Name:        "alpinebits",
			Title:       "Open Data Hub AlpineBits",
			Description: "AlpineBits hotel and destination data integration surface.",
			BaseURL:     "https://alpinebits.opendatahub.com",
			DocsURL:     "https://alpinebits.opendatahub.com",
			Public:      true,
		},
	})
	if err != nil {
		panic(err)
	}
	return registry
}

// NormalizeName normalizes API names and aliases for lookup.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Find returns an API by canonical name or alias.
func (r *Registry) Find(name string) (API, bool) {
	if r == nil {
		return API{}, false
	}
	index, ok := r.aliases[NormalizeName(name)]
	if !ok {
		return API{}, false
	}
	return r.apis[index], true
}

// List returns APIs sorted by canonical name.
func (r *Registry) List() []API {
	if r == nil {
		return nil
	}
	apis := append([]API(nil), r.apis...)
	sort.Slice(apis, func(i, j int) bool {
		return apis[i].Name < apis[j].Name
	})
	return apis
}

// Names returns the sorted canonical API names.
func (r *Registry) Names() []string {
	apis := r.List()
	names := make([]string, 0, len(apis))
	for _, api := range apis {
		names = append(names, api.Name)
	}
	return names
}
