// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	var out bytes.Buffer
	err := WriteJSON(&out, map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}
	if got := out.String(); !strings.Contains(got, `"ok": true`) {
		t.Fatalf("unexpected output %q", got)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatal("expected trailing newline")
	}
}

func TestWriteRawJSONRejectsInvalidJSON(t *testing.T) {
	var out bytes.Buffer
	err := WriteRawJSON(&out, []byte(`not json`))
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}
