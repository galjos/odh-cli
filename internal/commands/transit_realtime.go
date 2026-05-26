// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const gtfsRealtimeStaleAfter = 15 * time.Minute
const transitRealtimeSource = "Open Data Hub GTFS Realtime API"

type transitRealtimeFeedInfo struct {
	Source                  string `json:"source"`
	TripUpdatesEndpoint     string `json:"trip_updates_endpoint,omitempty"`
	ServiceAlertsEndpoint   string `json:"service_alerts_endpoint,omitempty"`
	TripUpdateEntityCount   int    `json:"trip_update_entity_count"`
	ServiceAlertEntityCount int    `json:"service_alert_entity_count"`
	MatchedTripUpdateCount  int    `json:"matched_trip_update_count"`
	MatchedAlertCount       int    `json:"matched_alert_count"`
	FeedTimestamp           string `json:"feed_timestamp,omitempty"`
	FeedTimestampUnix       int64  `json:"feed_timestamp_unix,omitempty"`
}

type transitLegRealtime struct {
	Status                string                 `json:"status"`
	DelaySeconds          *int                   `json:"delay_seconds,omitempty"`
	AdjustedDepartureTime string                 `json:"adjusted_departure_time,omitempty"`
	AdjustedArrivalTime   string                 `json:"adjusted_arrival_time,omitempty"`
	TripUpdateID          string                 `json:"trip_update_id,omitempty"`
	ScheduleRelationship  string                 `json:"schedule_relationship,omitempty"`
	StopTimeStatus        string                 `json:"stop_time_status,omitempty"`
	Alerts                []transitRealtimeAlert `json:"alerts,omitempty"`
}

type transitRealtimeAlert struct {
	ID          string `json:"id"`
	Cause       string `json:"cause,omitempty"`
	Effect      string `json:"effect,omitempty"`
	Header      string `json:"header,omitempty"`
	Description string `json:"description,omitempty"`
}

type transitTransferRealtime struct {
	FromLeg                int    `json:"from_leg"`
	ToLeg                  int    `json:"to_leg"`
	ScheduledBuffer        string `json:"scheduled_buffer"`
	ScheduledBufferSeconds int    `json:"scheduled_buffer_seconds"`
	AdjustedBuffer         string `json:"adjusted_buffer,omitempty"`
	AdjustedBufferSeconds  *int   `json:"adjusted_buffer_seconds,omitempty"`
	Status                 string `json:"status"`
}

type gtfsRealtimeTripUpdate struct {
	EntityID             string
	TripID               string
	RouteID              string
	ScheduleRelationship string
	StopUpdates          []gtfsRealtimeStopUpdate
}

type gtfsRealtimeStopUpdate struct {
	StopID               string
	StopSequence         int
	ScheduleRelationship string
	ArrivalDelay         *int
	DepartureDelay       *int
}

type gtfsRealtimeAlertRecord struct {
	Alert    transitRealtimeAlert
	Informed []gtfsRealtimeInformedEntity
}

type gtfsRealtimeInformedEntity struct {
	TripID  string
	RouteID string
	StopID  string
}

func (r *Runner) annotateTransitJourneysRealtime(ctx context.Context, dataset string, serviceDate time.Time, journeys []transitJourney, minTransfer time.Duration, warnings []string) (*transitRealtimeFeedInfo, []string) {
	info := &transitRealtimeFeedInfo{Source: transitRealtimeSource}
	if len(journeys) == 0 {
		return info, warnings
	}

	now := time.Now()
	if serviceDate.Format("2006-01-02") != now.Format("2006-01-02") {
		warnings = append(warnings, fmt.Sprintf("GTFS-RT is a current live feed; realtime annotations may not match non-current service date %s", serviceDate.Format("2006-01-02")))
	}

	tripEntities, tripHeader, endpoint, err := r.fetchGTFSRealtimeEntities(ctx, dataset, "trip-updates")
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("could not fetch GTFS-RT trip-updates: %v", err))
	} else {
		info.TripUpdatesEndpoint = endpoint
		info.TripUpdateEntityCount = len(tripEntities)
		setTransitRealtimeFeedTimestamp(info, tripHeader)
	}

	alertEntities, alertHeader, endpoint, err := r.fetchGTFSRealtimeEntities(ctx, dataset, "service-alerts")
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("could not fetch GTFS-RT service-alerts: %v", err))
	} else {
		info.ServiceAlertsEndpoint = endpoint
		info.ServiceAlertEntityCount = len(alertEntities)
		setTransitRealtimeFeedTimestamp(info, alertHeader)
	}
	if info.FeedTimestampUnix > 0 {
		age := now.Sub(time.Unix(info.FeedTimestampUnix, 0))
		if age < 0 {
			age = -age
		}
		if age > gtfsRealtimeStaleAfter {
			warnings = append(warnings, fmt.Sprintf("GTFS-RT feed timestamp is %s old; treat realtime annotations as stale", compactDuration(age)))
		}
	}

	updatesByTrip := indexGTFSRealtimeTripUpdates(tripEntities)
	alerts := parseGTFSRealtimeAlerts(alertEntities)
	matchedTrips := map[string]struct{}{}
	matchedAlerts := map[string]struct{}{}

	for journeyIndex := range journeys {
		for legIndex := range journeys[journeyIndex].Legs {
			leg := &journeys[journeyIndex].Legs[legIndex]
			realtime := &transitLegRealtime{Status: "no-update"}
			if update, ok := updatesByTrip[leg.TripID]; ok {
				matchedTrips[leg.TripID] = struct{}{}
				applyTripUpdateToTransitLeg(realtime, *leg, update)
			}
			for _, alert := range alerts {
				if !gtfsRealtimeAlertMatchesLeg(alert, *leg) {
					continue
				}
				realtime.Alerts = append(realtime.Alerts, alert.Alert)
				matchedAlerts[alert.Alert.ID] = struct{}{}
				if realtime.Status == "no-update" {
					realtime.Status = "alert"
				}
			}
			leg.Realtime = realtime
		}
		journeys[journeyIndex].RealtimeTransfers = buildTransitRealtimeTransfers(journeys[journeyIndex], minTransfer)
		for _, transfer := range journeys[journeyIndex].RealtimeTransfers {
			switch transfer.Status {
			case "missed":
				warnings = append(warnings, fmt.Sprintf("realtime annotation marks journey %d transfer %d->%d as missed", journeyIndex+1, transfer.FromLeg, transfer.ToLeg))
			case "tight":
				warnings = append(warnings, fmt.Sprintf("realtime annotation marks journey %d transfer %d->%d as tight", journeyIndex+1, transfer.FromLeg, transfer.ToLeg))
			}
		}
	}

	info.MatchedTripUpdateCount = len(matchedTrips)
	info.MatchedAlertCount = len(matchedAlerts)
	if info.TripUpdateEntityCount > 0 && info.MatchedTripUpdateCount == 0 {
		warnings = append(warnings, fmt.Sprintf("GTFS-RT trip-updates returned %d entities, but none matched the returned journey trip IDs", info.TripUpdateEntityCount))
	}
	return info, warnings
}

func (r *Runner) fetchGTFSRealtimeEntities(ctx context.Context, dataset, feed string) ([]map[string]any, map[string]any, string, error) {
	api, ok := r.Registry.Find("gtfs")
	if !ok {
		return nil, nil, "", fmt.Errorf("gtfs API is not configured")
	}
	dataset = strings.TrimSpace(dataset)
	if dataset == "" {
		dataset = defaultGTFSDataset
	}
	path := fmt.Sprintf("/v1/realtime/%s/%s", url.PathEscape(strings.TrimSpace(dataset)), feed)
	endpoint, err := BuildURL(api.BaseURL, path, nil)
	if err != nil {
		return nil, nil, "", err
	}
	value, err := r.fetchJSONValue(ctx, endpoint)
	if err != nil {
		return nil, nil, endpoint, err
	}
	object, _ := value.(map[string]any)
	header, _ := object["header"].(map[string]any)
	return mapsFromList(asAnySlice(object["entity"])), header, endpoint, nil
}

func setTransitRealtimeFeedTimestamp(info *transitRealtimeFeedInfo, header map[string]any) {
	if info == nil || len(header) == 0 {
		return
	}
	timestamp, ok := realtimeInt64Value(header["timestamp"])
	if !ok || timestamp <= 0 {
		return
	}
	if timestamp < info.FeedTimestampUnix {
		return
	}
	info.FeedTimestampUnix = timestamp
	info.FeedTimestamp = time.Unix(timestamp, 0).UTC().Format(time.RFC3339)
}

func indexGTFSRealtimeTripUpdates(entities []map[string]any) map[string]gtfsRealtimeTripUpdate {
	index := make(map[string]gtfsRealtimeTripUpdate)
	for _, entity := range entities {
		update, ok := parseGTFSRealtimeTripUpdate(entity)
		if !ok || update.TripID == "" {
			continue
		}
		index[update.TripID] = update
	}
	return index
}

func parseGTFSRealtimeTripUpdate(entity map[string]any) (gtfsRealtimeTripUpdate, bool) {
	rawUpdate, _ := entity["trip_update"].(map[string]any)
	if len(rawUpdate) == 0 {
		return gtfsRealtimeTripUpdate{}, false
	}
	trip, _ := rawUpdate["trip"].(map[string]any)
	update := gtfsRealtimeTripUpdate{
		EntityID:             asString(entity["id"]),
		TripID:               asString(trip["trip_id"]),
		RouteID:              asString(trip["route_id"]),
		ScheduleRelationship: asString(trip["schedule_relationship"]),
	}
	for _, raw := range asAnySlice(rawUpdate["stop_time_update"]) {
		item, _ := raw.(map[string]any)
		if len(item) == 0 {
			continue
		}
		stopUpdate := gtfsRealtimeStopUpdate{
			StopID:               asString(item["stop_id"]),
			ScheduleRelationship: asString(item["schedule_relationship"]),
		}
		if sequence, ok := realtimeIntValue(item["stop_sequence"]); ok {
			stopUpdate.StopSequence = sequence
		}
		if arrival, _ := item["arrival"].(map[string]any); len(arrival) > 0 {
			if delay, ok := realtimeIntValue(arrival["delay"]); ok {
				stopUpdate.ArrivalDelay = intPtr(delay)
			}
		}
		if departure, _ := item["departure"].(map[string]any); len(departure) > 0 {
			if delay, ok := realtimeIntValue(departure["delay"]); ok {
				stopUpdate.DepartureDelay = intPtr(delay)
			}
		}
		update.StopUpdates = append(update.StopUpdates, stopUpdate)
	}
	return update, true
}

func applyTripUpdateToTransitLeg(realtime *transitLegRealtime, leg transitJourneyLeg, update gtfsRealtimeTripUpdate) {
	realtime.Status = "updated"
	realtime.TripUpdateID = update.EntityID
	realtime.ScheduleRelationship = update.ScheduleRelationship
	if strings.EqualFold(update.ScheduleRelationship, "CANCELED") || strings.EqualFold(update.ScheduleRelationship, "CANCELLED") {
		realtime.Status = "cancelled"
	}

	fromUpdate, fromOK := bestGTFSRealtimeStopUpdate(update.StopUpdates, leg.From.StopID, leg.From.StopSequence)
	toUpdate, toOK := bestGTFSRealtimeStopUpdate(update.StopUpdates, leg.To.StopID, leg.To.StopSequence)
	if fromOK {
		realtime.StopTimeStatus = firstNonEmpty(realtime.StopTimeStatus, fromUpdate.ScheduleRelationship)
		if fromUpdate.DepartureDelay != nil {
			realtime.DelaySeconds = fromUpdate.DepartureDelay
			if adjusted := adjustedGTFSTimeText(leg.From.DepartureTime, *fromUpdate.DepartureDelay); adjusted != "" {
				realtime.AdjustedDepartureTime = adjusted
			}
		}
	}
	if toOK {
		realtime.StopTimeStatus = firstNonEmpty(realtime.StopTimeStatus, toUpdate.ScheduleRelationship)
		if toUpdate.ArrivalDelay != nil {
			realtime.DelaySeconds = toUpdate.ArrivalDelay
			if adjusted := adjustedGTFSTimeText(leg.To.ArrivalTime, *toUpdate.ArrivalDelay); adjusted != "" {
				realtime.AdjustedArrivalTime = adjusted
			}
		}
	}
	if realtime.DelaySeconds == nil {
		if delay, ok := fallbackGTFSRealtimeDelay(update.StopUpdates); ok {
			realtime.DelaySeconds = intPtr(delay)
		}
	}
}

func bestGTFSRealtimeStopUpdate(updates []gtfsRealtimeStopUpdate, stopID string, sequence int) (gtfsRealtimeStopUpdate, bool) {
	for _, update := range updates {
		if update.StopID != "" && update.StopID == stopID {
			return update, true
		}
	}
	if sequence > 0 {
		for _, update := range updates {
			if update.StopSequence == sequence {
				return update, true
			}
		}
	}
	return gtfsRealtimeStopUpdate{}, false
}

func fallbackGTFSRealtimeDelay(updates []gtfsRealtimeStopUpdate) (int, bool) {
	for i := len(updates) - 1; i >= 0; i-- {
		if updates[i].ArrivalDelay != nil {
			return *updates[i].ArrivalDelay, true
		}
		if updates[i].DepartureDelay != nil {
			return *updates[i].DepartureDelay, true
		}
	}
	return 0, false
}

func parseGTFSRealtimeAlerts(entities []map[string]any) []gtfsRealtimeAlertRecord {
	alerts := make([]gtfsRealtimeAlertRecord, 0, len(entities))
	for _, entity := range entities {
		rawAlert, _ := entity["alert"].(map[string]any)
		if len(rawAlert) == 0 {
			continue
		}
		alert := gtfsRealtimeAlertRecord{
			Alert: transitRealtimeAlert{
				ID:          asString(entity["id"]),
				Cause:       asString(rawAlert["cause"]),
				Effect:      asString(rawAlert["effect"]),
				Header:      gtfsRealtimeTranslatedText(rawAlert["header_text"]),
				Description: gtfsRealtimeTranslatedText(rawAlert["description_text"]),
			},
		}
		for _, raw := range asAnySlice(rawAlert["informed_entity"]) {
			item, _ := raw.(map[string]any)
			if len(item) == 0 {
				continue
			}
			trip, _ := item["trip"].(map[string]any)
			alert.Informed = append(alert.Informed, gtfsRealtimeInformedEntity{
				TripID:  firstNonEmpty(asString(item["trip_id"]), asString(trip["trip_id"])),
				RouteID: firstNonEmpty(asString(item["route_id"]), asString(trip["route_id"])),
				StopID:  asString(item["stop_id"]),
			})
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func gtfsRealtimeAlertMatchesLeg(alert gtfsRealtimeAlertRecord, leg transitJourneyLeg) bool {
	for _, informed := range alert.Informed {
		if informed.TripID != "" && informed.TripID == leg.TripID {
			return true
		}
		if informed.RouteID != "" && informed.RouteID == leg.RouteID {
			return true
		}
		if informed.StopID != "" && (informed.StopID == leg.From.StopID || informed.StopID == leg.To.StopID) {
			return true
		}
	}
	return false
}

func gtfsRealtimeTranslatedText(value any) string {
	object, _ := value.(map[string]any)
	translations := mapsFromList(asAnySlice(object["translation"]))
	for _, language := range []string{"en", "de", "it"} {
		for _, translation := range translations {
			if strings.EqualFold(asString(translation["language"]), language) {
				if text := strings.TrimSpace(asString(translation["text"])); text != "" {
					return text
				}
			}
		}
	}
	for _, translation := range translations {
		if text := strings.TrimSpace(asString(translation["text"])); text != "" {
			return text
		}
	}
	return ""
}

func buildTransitRealtimeTransfers(journey transitJourney, minTransfer time.Duration) []transitTransferRealtime {
	if len(journey.Legs) < 2 {
		return nil
	}
	transfers := make([]transitTransferRealtime, 0, len(journey.Legs)-1)
	minTransferSeconds := int(minTransfer.Seconds())
	for index := 0; index < len(journey.Legs)-1; index++ {
		fromLeg := journey.Legs[index]
		toLeg := journey.Legs[index+1]
		scheduledArrival, _, arrivalErr := parseGTFSTimeOfDay(fromLeg.To.ArrivalTime)
		scheduledDeparture, _, departureErr := parseGTFSTimeOfDay(toLeg.From.DepartureTime)
		if arrivalErr != nil || departureErr != nil {
			transfers = append(transfers, transitTransferRealtime{FromLeg: index + 1, ToLeg: index + 2, Status: "unknown"})
			continue
		}
		scheduledBuffer := scheduledDeparture - scheduledArrival
		transfer := transitTransferRealtime{
			FromLeg:                index + 1,
			ToLeg:                  index + 2,
			ScheduledBuffer:        formatDelaySeconds(scheduledBuffer),
			ScheduledBufferSeconds: scheduledBuffer,
			Status:                 "unknown",
		}
		adjustedArrival := scheduledArrival
		adjustedDeparture := scheduledDeparture
		adjusted := false
		if fromLeg.Realtime != nil && fromLeg.Realtime.AdjustedArrivalTime != "" {
			if seconds, _, err := parseGTFSTimeOfDay(fromLeg.Realtime.AdjustedArrivalTime); err == nil {
				adjustedArrival = seconds
				adjusted = true
			}
		}
		if toLeg.Realtime != nil && toLeg.Realtime.AdjustedDepartureTime != "" {
			if seconds, _, err := parseGTFSTimeOfDay(toLeg.Realtime.AdjustedDepartureTime); err == nil {
				adjustedDeparture = seconds
				adjusted = true
			}
		}
		if adjusted {
			adjustedBuffer := adjustedDeparture - adjustedArrival
			transfer.AdjustedBufferSeconds = intPtr(adjustedBuffer)
			transfer.AdjustedBuffer = formatDelaySeconds(adjustedBuffer)
			switch {
			case adjustedBuffer < 0:
				transfer.Status = "missed"
			case adjustedBuffer < minTransferSeconds:
				transfer.Status = "tight"
			default:
				transfer.Status = "ok"
			}
		}
		transfers = append(transfers, transfer)
	}
	return transfers
}

func adjustedGTFSTimeText(schedule string, delay int) string {
	seconds, _, err := parseGTFSTimeOfDay(schedule)
	if err != nil {
		return ""
	}
	adjusted := seconds + delay
	if adjusted < 0 {
		return ""
	}
	return formatGTFSTimeOfDay(adjusted)
}

func formatGTFSTimeOfDay(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	remainingSeconds := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, remainingSeconds)
}

func realtimeAdjustedDeparture(leg transitJourneyLeg) string {
	if leg.Realtime == nil {
		return ""
	}
	return leg.Realtime.AdjustedDepartureTime
}

func realtimeAdjustedArrival(leg transitJourneyLeg) string {
	if leg.Realtime == nil {
		return ""
	}
	return leg.Realtime.AdjustedArrivalTime
}

func realtimeLegStatus(leg transitJourneyLeg) string {
	if leg.Realtime == nil || strings.TrimSpace(leg.Realtime.Status) == "" {
		return "-"
	}
	return leg.Realtime.Status
}

func realtimeDelayText(leg transitJourneyLeg) string {
	if leg.Realtime == nil || leg.Realtime.DelaySeconds == nil {
		return "-"
	}
	return formatDelaySeconds(*leg.Realtime.DelaySeconds)
}

func realtimeTransferSummary(transfers []transitTransferRealtime) string {
	if len(transfers) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(transfers))
	for _, transfer := range transfers {
		text := transfer.Status
		if transfer.AdjustedBuffer != "" {
			text += " " + transfer.AdjustedBuffer
		}
		parts = append(parts, fmt.Sprintf("%d->%d %s", transfer.FromLeg, transfer.ToLeg, text))
	}
	return strings.Join(parts, "; ")
}

func formatDelaySeconds(seconds int) string {
	if seconds == 0 {
		return "0s"
	}
	sign := ""
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	duration := time.Duration(seconds) * time.Second
	if duration >= time.Hour && duration%time.Hour == 0 {
		return fmt.Sprintf("%s%dh", sign, int(duration/time.Hour))
	}
	if duration >= time.Hour && duration%time.Minute == 0 {
		hours := int(duration / time.Hour)
		minutes := int((duration % time.Hour) / time.Minute)
		return fmt.Sprintf("%s%dh%dm", sign, hours, minutes)
	}
	if duration%time.Minute == 0 {
		return fmt.Sprintf("%s%dm", sign, int(duration/time.Minute))
	}
	return sign + duration.String()
}

func realtimeIntValue(value any) (int, bool) {
	result, ok := realtimeInt64Value(value)
	if !ok || int64(int(result)) != result {
		return 0, false
	}
	return int(result), true
}

func realtimeInt64Value(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	case json.Number:
		result, err := typed.Int64()
		return result, err == nil
	case string:
		result, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return result, err == nil
	default:
		return 0, false
	}
}

func intPtr(value int) *int {
	return &value
}
