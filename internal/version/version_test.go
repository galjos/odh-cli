// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package version

import "testing"

func TestCurrentHasMetadata(t *testing.T) {
	info := Current()
	if info.Version == "" || info.Commit == "" || info.Date == "" || info.GoOS == "" || info.GoArch == "" {
		t.Fatalf("missing version metadata: %#v", info)
	}
}

func TestShortenCommit(t *testing.T) {
	got := shortenCommit("1234567890abcdef")
	if got != "1234567890ab" {
		t.Fatalf("shortenCommit = %q", got)
	}
}
