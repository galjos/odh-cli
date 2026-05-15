// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"encoding/json"
	"testing"
)

func TestToJSONAcceptsJSON(t *testing.T) {
	body, format, err := ToJSON([]byte(`{"openapi":"3.0.1","info":{"title":"x"}}`))
	if err != nil {
		t.Fatalf("ToJSON returned error: %v", err)
	}
	if format != "json" {
		t.Fatalf("unexpected format %q", format)
	}
	assertJSONField(t, body, "openapi", "3.0.1")
}

func TestToJSONConvertsYAML(t *testing.T) {
	body, format, err := ToJSON([]byte("openapi: 3.0.1\ninfo:\n  title: Test API\n"))
	if err != nil {
		t.Fatalf("ToJSON returned error: %v", err)
	}
	if format != "yaml" {
		t.Fatalf("unexpected format %q", format)
	}
	assertJSONField(t, body, "openapi", "3.0.1")
}

func assertJSONField(t *testing.T, body []byte, key, want string) {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, string(body))
	}
	if got := decoded[key]; got != want {
		t.Fatalf("field %q = %#v, want %#v", key, got, want)
	}
}
