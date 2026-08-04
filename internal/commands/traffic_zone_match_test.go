// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"strings"
	"testing"
)

// isolatedTrafficZonePoint returns a reference point whose nearest neighbour is
// far enough away that a test coordinate offset from it can only match that one
// point, so the assertions below survive a regenerated table.
func isolatedTrafficZonePoint(t *testing.T) trafficZonePoint {
	t.Helper()
	best := trafficZonePoint{}
	bestIsolation := 0.0
	for _, point := range trafficZonePoints {
		isolation := 0.0
		for _, other := range trafficZonePoints {
			if other == point {
				continue
			}
			distance := haversineKM(point.Lat, point.Lon, other.Lat, other.Lon)
			if isolation == 0 || distance < isolation {
				isolation = distance
			}
		}
		if isolation > bestIsolation {
			bestIsolation, best = isolation, point
		}
	}
	if bestIsolation <= 2*trafficZoneGuardKM {
		t.Fatalf("no reference point is isolated enough to test the guard: %.2f km", bestIsolation)
	}
	return best
}

func TestNearestTrafficZoneGuardsByDistance(t *testing.T) {
	anchor := isolatedTrafficZonePoint(t)
	// One degree of latitude is a fixed distance, so offsetting north gives an
	// exact distance from the anchor without depending on longitude.
	const kmPerDegreeLat = 111.194926644
	offsetLat := func(km float64) []float64 {
		return []float64{anchor.Lon, anchor.Lat + km/kmPerDegreeLat}
	}

	if got := nearestTrafficZone([]float64{anchor.Lon, anchor.Lat}); got != anchor.ZoneID {
		t.Fatalf("a reference point must match its own zone: got %q, want %q", got, anchor.ZoneID)
	}
	if got := nearestTrafficZone(offsetLat(trafficZoneGuardKM - 0.01)); got != anchor.ZoneID {
		t.Fatalf("just inside the guard must match: got %q, want %q", got, anchor.ZoneID)
	}
	if got := nearestTrafficZone(offsetLat(trafficZoneGuardKM + 0.01)); got != "" {
		t.Fatalf("just outside the guard must be unassignable: got %q", got)
	}
	if got := nearestTrafficZone(offsetLat(50)); got != "" {
		t.Fatalf("far outside the guard must be unassignable: got %q", got)
	}
	if got := nearestTrafficZone(nil); got != "" {
		t.Fatalf("missing coordinates must be unassignable: got %q", got)
	}
	if got := nearestTrafficZone([]float64{11.25}); got != "" {
		t.Fatalf("a partial coordinate pair must be unassignable: got %q", got)
	}
}

// The generator drops cells that more than one zone claimed; a duplicate cell
// in the committed table would mean an announcement's zone depends on slice
// order rather than on the data.
func TestTrafficZonePointsAreUniqueAndKnown(t *testing.T) {
	if len(trafficZonePoints) < 500 {
		t.Fatalf("reference table looks truncated: %d points", len(trafficZonePoints))
	}
	known := map[string]struct{}{}
	for _, zone := range knownTrafficZones() {
		known[zone.ZoneID] = struct{}{}
	}
	seen := make(map[[2]float64]string, len(trafficZonePoints))
	for _, point := range trafficZonePoints {
		if _, ok := known[point.ZoneID]; !ok {
			t.Fatalf("unknown zone %q at %.3f,%.3f", point.ZoneID, point.Lat, point.Lon)
		}
		key := [2]float64{point.Lat, point.Lon}
		if existing, exists := seen[key]; exists {
			t.Fatalf("cell %.3f,%.3f is claimed by zones %q and %q", point.Lat, point.Lon, existing, point.ZoneID)
		}
		seen[key] = point.ZoneID
		// Zone 7 is "Ausserhalb Südtirol" and runs the Brenner axis from Modena
		// to Austria; zones 1-6 are South Tyrol districts and must stay inside it.
		if point.Lat < 44 || point.Lat > 48 || point.Lon < 9 || point.Lon > 14 {
			t.Fatalf("point %.3f,%.3f is outside the PROVINCE_BZ bulletin's range", point.Lat, point.Lon)
		}
		if point.ZoneID != "7" && (point.Lat < 46.1 || point.Lat > 47.2 || point.Lon < 10.3 || point.Lon > 12.6) {
			t.Fatalf("zone %s point %.3f,%.3f is outside South Tyrol", point.ZoneID, point.Lat, point.Lon)
		}
	}
}

// --zone-id and --area narrow independently, as they do for --source odh.
// Unioning them let --zone-id 1 --area pustertal return zone 6 rows.
func TestIntersectZoneIDsNarrowsRatherThanWidens(t *testing.T) {
	if got := intersectZoneIDs(nil, nil); len(got) != 0 {
		t.Fatalf("an unfiltered query must not narrow by zone: %#v", got)
	}
	if got := intersectZoneIDs([]string{"3"}, nil); strings.Join(got, ",") != "3" {
		t.Fatalf("an empty gate must not constrain: %#v", got)
	}
	if got := intersectZoneIDs(nil, []string{"6"}); strings.Join(got, ",") != "6" {
		t.Fatalf("an empty gate must not constrain: %#v", got)
	}
	if got := intersectZoneIDs([]string{"6", "3"}, []string{"1", "3"}); strings.Join(got, ",") != "3" {
		t.Fatalf("want only the zone both gates accept: %#v", got)
	}
	if got := intersectZoneIDs([]string{"1"}, []string{"6"}); len(got) != 0 {
		t.Fatalf("disjoint gates must accept nothing: %#v", got)
	}
}

func TestContentTrafficAreaKeywordWarningOnlyForKeywordAliases(t *testing.T) {
	if got := contentTrafficAreaKeywordWarning(trafficArea{Name: "pustertal", ZoneIDs: []string{"6"}}); got != "" {
		t.Fatalf("a zone-only alias needs no keyword warning: %q", got)
	}
	got := contentTrafficAreaKeywordWarning(trafficArea{Name: "kaltern", ZoneIDs: []string{"3"}, Keywords: []string{"Kaltern"}})
	if !strings.Contains(got, "area kaltern narrows to zone 3 (Bozen-Unterland) only with --source content") {
		t.Fatalf("unexpected keyword warning: %q", got)
	}
	if !strings.Contains(got, "Treat this result as the whole zone, not as kaltern.") {
		t.Fatalf("keyword warning must say the result is wider than the alias: %q", got)
	}
}
