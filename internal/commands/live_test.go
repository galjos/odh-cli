// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestLiveTourismPOI(t *testing.T) {
	if os.Getenv("ODH_LIVE_TESTS") != "1" {
		t.Skip("set ODH_LIVE_TESTS=1 to run live Open Data Hub smoke tests")
	}
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"tourism", "poi", "--limit", "1", "--seed", "42", "--fields", "Detail.en.Title,GpsInfo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"Items"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestLiveMobilityLatest(t *testing.T) {
	if os.Getenv("ODH_LIVE_TESTS") != "1" {
		t.Skip("set ODH_LIVE_TESTS=1 to run live Open Data Hub smoke tests")
	}
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "latest", "--station-type", "EChargingStation", "--data-type", "number-available", "--limit", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"data"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}
