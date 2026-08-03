// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"testing"
	"time"
)

// A caller that asks for more time than the default must get it: --timeout 10m
// was previously cut back to the 2m archive default, so raising the budget made
// no difference and a cold-cache fetch stayed unfixable.
func TestGTFSArchiveBudgetHonoursALongerDeadline(t *testing.T) {
	if got := gtfsArchiveBudget(context.Background()); got != gtfsDownloadTimeout {
		t.Fatalf("no deadline: got %s, want the %s default", got, gtfsDownloadTimeout)
	}

	long, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if got := gtfsArchiveBudget(long); got <= gtfsDownloadTimeout {
		t.Fatalf("10m deadline: got %s, want more than the %s default", got, gtfsDownloadTimeout)
	}

	// A shorter deadline must be reported as-is, not as the default: the progress
	// line used to promise "up to 2m0s" on a --timeout 3s run.
	short, cancelShort := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShort()
	if got := gtfsArchiveBudget(short); got > 5*time.Second || got <= 0 {
		t.Fatalf("5s deadline: got %s, want at most 5s", got)
	}
}
