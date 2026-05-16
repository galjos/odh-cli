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

func TestLiveTourismTypes(t *testing.T) {
	if os.Getenv("ODH_LIVE_TESTS") != "1" {
		t.Skip("set ODH_LIVE_TESTS=1 to run live Open Data Hub smoke tests")
	}
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"tourism", "types", "--dataset", "event", "--limit", "3"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"dataset": "event"`) || !strings.Contains(stdout.String(), `"items"`) {
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

func TestLiveMobilityStations(t *testing.T) {
	if os.Getenv("ODH_LIVE_TESTS") != "1" {
		t.Skip("set ODH_LIVE_TESTS=1 to run live Open Data Hub smoke tests")
	}
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "stations", "--station-type", "ParkingStation", "--limit", "3"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"station_type": "ParkingStation"`) || !strings.Contains(stdout.String(), `"stations"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestLiveMobilityDiscovery(t *testing.T) {
	if os.Getenv("ODH_LIVE_TESTS") != "1" {
		t.Skip("set ODH_LIVE_TESTS=1 to run live Open Data Hub smoke tests")
	}
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"mobility", "datatypes", "--station-type", "TrafficSensor", "--origin", "A22", "--limit", "100"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"datatypes"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestLiveA22Status(t *testing.T) {
	if os.Getenv("ODH_LIVE_TESTS") != "1" {
		t.Skip("set ODH_LIVE_TESTS=1 to run live Open Data Hub smoke tests")
	}
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"a22", "status", "--limit", "10"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"warnings"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestLiveTrafficToday(t *testing.T) {
	if os.Getenv("ODH_LIVE_TESTS") != "1" {
		t.Skip("set ODH_LIVE_TESTS=1 to run live Open Data Hub smoke tests")
	}
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"traffic", "today", "--source", "odh", "--area", "bozen-unterland", "--limit", "10", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"source": "odh"`) || !strings.Contains(stdout.String(), `"events"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestLiveDoctor(t *testing.T) {
	if os.Getenv("ODH_LIVE_TESTS") != "1" {
		t.Skip("set ODH_LIVE_TESTS=1 to run live Open Data Hub smoke tests")
	}
	runner := NewDefaultRunner()
	var stdout, stderr bytes.Buffer
	code := runner.Run(context.Background(), []string{"doctor", "--timeout", "10s"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run exit = %d, stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"ok": true`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}
