// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"strings"
)

// trafficZonePoint is a rounded coordinate at which a PROVINCE_BZ zone was
// observed. The table of these lives in traffic_zone_points.go and is generated
// by scripts/generate-traffic-zone-points.go.
type trafficZonePoint struct {
	Lat    float64
	Lon    float64
	ZoneID string
}

// trafficZoneGuardKM bounds how far an announcement may sit from the nearest
// reference point before its zone counts as unknown.
//
// Leave-one-out over the reference table, matching each point against the
// nearest other point: the zone agrees for 1024 of 1043 points whose nearest
// neighbour is within 2 km (98.2%), and for only 34 of 41 between 2 and 3 km
// (82.9%). The agreement rate is flat below the bound and breaks above it.
const trafficZoneGuardKM = 2.0

// nearestTrafficZone returns the zone of the closest reference point to the
// given [lon, lat] pair, or "" when there are no coordinates or every reference
// point is farther than trafficZoneGuardKM.
//
// The scan is linear over the whole table. At ~1100 points that is tens of
// microseconds per announcement against a page of at most a few thousand, which
// stays far below the cost of the HTTP request that fetched them.
func nearestTrafficZone(coordinates []float64) string {
	if len(coordinates) < 2 {
		return ""
	}
	lon, lat := coordinates[0], coordinates[1]
	nearest := ""
	nearestKM := trafficZoneGuardKM
	for _, point := range trafficZonePoints {
		distance := haversineKM(lat, lon, point.Lat, point.Lon)
		if distance <= nearestKM {
			nearestKM = distance
			nearest = point.ZoneID
		}
	}
	return nearest
}

// intersectZoneIDs reports the zones accepted by both gates. An empty gate
// constrains nothing, so it yields the other one.
func intersectZoneIDs(zoneIDs, areaZoneIDs []string) []string {
	switch {
	case len(zoneIDs) == 0:
		return areaZoneIDs
	case len(areaZoneIDs) == 0:
		return zoneIDs
	}
	both := make([]string, 0, len(zoneIDs))
	for _, value := range zoneIDs {
		if containsString(areaZoneIDs, value) {
			both = append(both, value)
		}
	}
	return both
}

// contentTrafficZoneInferenceWarning states that the zone behind an --area or
// --zone-id filter was inferred from coordinates, because the Announcement
// record has no zone field to read.
func contentTrafficZoneInferenceWarning(query trafficQuery, area trafficArea, zoneIDs []string) string {
	flags := make([]string, 0, 2)
	if strings.TrimSpace(query.ZoneID) != "" {
		flags = append(flags, "--zone-id")
	}
	if area.Name != "" {
		flags = append(flags, "--area")
	}
	return fmt.Sprintf("%s with --source content is geographic inference, not a field: Announcement records carry no zone, so each one was placed in zone %s by nearest of %d reference coordinates derived from historical PROVINCE_BZ Mobility event rows, accepted only within %.1f km. No inferred zone is written into the events; zone_id, zone and zone_it stay empty there.",
		strings.Join(flags, " and "), strings.Join(zoneIDs, "/"), len(trafficZonePoints), trafficZoneGuardKM)
}

// contentTrafficUnassignableWarning separates "could not be placed" from "did
// not match", so a thin result is not read as an empty road network.
func contentTrafficUnassignableWarning(count int) string {
	if count == 0 {
		return ""
	}
	noun := "announcements"
	if count == 1 {
		noun = "announcement"
	}
	// Not "matching": these are the records whose zone could NOT be determined,
	// so whether they belong to the requested zone is exactly what is unknown.
	return fmt.Sprintf("%d %s passed every other filter but could not be placed in any zone: they carry no coordinates, or lie farther than %.1f km from every reference coordinate. They are excluded here, and whether any of them belongs to the zone you asked for is unknown",
		count, noun, trafficZoneGuardKM)
}

// contentTrafficAreaKeywordWarning fires for area aliases that narrow by place
// keywords as well as by zone under --source odh. Only the zone survives here,
// so the result is wider than the alias name promises.
func contentTrafficAreaKeywordWarning(area trafficArea) string {
	if area.Name == "" || len(area.Keywords) == 0 || len(area.ZoneIDs) == 0 {
		return ""
	}
	names := make([]string, 0, len(area.ZoneIDs))
	for _, zone := range knownTrafficZones() {
		if containsString(area.ZoneIDs, zone.ZoneID) {
			names = append(names, zone.Name)
		}
	}
	return fmt.Sprintf("area %s narrows to zone %s (%s) only with --source content; under --source odh it also filters on place names (%s), which this source cannot match. Treat this result as the whole zone, not as %s.",
		area.Name, strings.Join(area.ZoneIDs, "/"), strings.Join(names, ", "), strings.Join(area.Keywords, ", "), area.Name)
}
