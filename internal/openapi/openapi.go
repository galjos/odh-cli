// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ToJSON converts a JSON or YAML OpenAPI document into deterministic JSON.
func ToJSON(body []byte) ([]byte, string, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, "", errors.New("empty OpenAPI document")
	}
	if json.Valid(trimmed) {
		var value any
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return nil, "", err
		}
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, "", err
		}
		return append(encoded, '\n'), "json", nil
	}

	var value any
	if err := yaml.Unmarshal(trimmed, &value); err != nil {
		return nil, "", fmt.Errorf("openapi document is neither JSON nor YAML: %w", err)
	}
	normalized := normalizeYAML(value)
	encoded, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return append(encoded, '\n'), "yaml", nil
}

func normalizeYAML(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, value := range typed {
			result[key] = normalizeYAML(value)
		}
		return result
	case map[any]any:
		result := make(map[string]any, len(typed))
		for key, value := range typed {
			result[fmt.Sprint(key)] = normalizeYAML(value)
		}
		return result
	case []any:
		for i, item := range typed {
			typed[i] = normalizeYAML(item)
		}
		return typed
	case string:
		return strings.TrimRight(typed, "\n")
	default:
		return value
	}
}
