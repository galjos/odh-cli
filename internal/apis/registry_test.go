// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package apis

import "testing"

func TestDefaultRegistryFindsAliases(t *testing.T) {
	registry := DefaultRegistry()

	api, ok := registry.Find("content")
	if !ok {
		t.Fatal("expected content alias to resolve")
	}
	if api.Name != "tourism" {
		t.Fatalf("expected content alias to resolve tourism, got %q", api.Name)
	}
}

func TestRegistryNamesAreSorted(t *testing.T) {
	registry := DefaultRegistry()
	names := registry.Names()
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("names are not sorted: %v", names)
		}
	}
}

func TestRegistryRejectsDuplicateAlias(t *testing.T) {
	_, err := NewRegistry([]API{
		{Name: "one", BaseURL: "https://example.com", Aliases: []string{"same"}},
		{Name: "two", BaseURL: "https://example.org", Aliases: []string{"same"}},
	})
	if err == nil {
		t.Fatal("expected duplicate alias error")
	}
}
