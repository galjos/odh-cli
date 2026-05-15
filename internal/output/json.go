// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON writes deterministic indented JSON with a trailing newline.
func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// NormalizeJSON pretty-prints a JSON document.
func NormalizeJSON(body []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	if err := WriteJSON(&buffer, value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// WriteRawJSON validates and writes a JSON response as deterministic JSON.
func WriteRawJSON(w io.Writer, body []byte) error {
	normalized, err := NormalizeJSON(body)
	if err != nil {
		return fmt.Errorf("response is not valid JSON: %w", err)
	}
	_, err = w.Write(normalized)
	return err
}
