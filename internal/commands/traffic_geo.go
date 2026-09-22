// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func parseNearRadius(near, radius string) (float64, float64, float64, error) {
	latText, lonText, ok := strings.Cut(strings.TrimSpace(near), ",")
	if !ok {
		return 0, 0, 0, fmt.Errorf("--near must use lat,lon")
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(latText), 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid latitude in --near: %w", err)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(lonText), 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid longitude in --near: %w", err)
	}
	radiusKM, err := parseRadiusKM(radius)
	if err != nil {
		return 0, 0, 0, err
	}
	return lat, lon, radiusKM, nil
}

func parseRadiusKM(value string) (float64, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimSuffix(value, "km")
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid radius %q", value)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("radius must be greater than zero")
	}
	return parsed, nil
}

func extractCoordinates(value any) []float64 {
	geometry, _ := value.(map[string]any)
	coordinates, _ := geometry["coordinates"].([]any)
	if len(coordinates) < 2 {
		return nil
	}
	lon, lonOK := numberValue(coordinates[0])
	lat, latOK := numberValue(coordinates[1])
	if !lonOK || !latOK {
		return nil
	}
	return []float64{lon, lat}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKM = 6371.0
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func startOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func endOfDay(value time.Time) time.Time {
	return startOfDay(value).Add(24*time.Hour - time.Nanosecond)
}

func normalizeRoad(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	value = strings.ReplaceAll(value, "LS / SP", "LS/SP")
	value = strings.ReplaceAll(value, "LS/ SP", "LS/SP")
	value = strings.ReplaceAll(value, "LS /SP", "LS/SP")
	return value
}

func compactRoadToken(value string) string {
	value = normalizeRoad(value)
	replacer := strings.NewReplacer(" ", "", "/", "", "-", "")
	return replacer.Replace(value)
}

func normalizeAreaAlias(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacements := map[string]string{
		"ü": "ue",
		"ö": "oe",
		"ä": "ae",
		"ß": "ss",
		"_": "-",
		" ": "-",
	}
	for old, newValue := range replacements {
		value = strings.ReplaceAll(value, old, newValue)
	}
	value = strings.Trim(value, "-")
	return value
}

func normalizeTrafficSearchText(value string) string {
	value = strings.ToLower(cleanTrafficText(value))
	replacer := strings.NewReplacer(
		"ä", "ae",
		"ö", "oe",
		"ü", "ue",
		"ß", "ss",
		"à", "a",
		"á", "a",
		"è", "e",
		"é", "e",
		"ì", "i",
		"í", "i",
		"ò", "o",
		"ó", "o",
		"ù", "u",
		"ú", "u",
		"/", " ",
		"-", " ",
		"_", " ",
		".", " ",
		",", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"?", " ",
		"!", " ",
	)
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func cleanTrafficText(value string) string {
	value = strings.ReplaceAll(value, "\\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}

func textContainsPass(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "pass") || strings.Contains(value, "joch")
}
