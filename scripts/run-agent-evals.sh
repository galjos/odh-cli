#!/usr/bin/env sh

# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: MPL-2.0

set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tasks_file="$repo_root/evals/agent/tasks.json"

if [ ! -f "$tasks_file" ]; then
  echo "missing eval task file: $tasks_file" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for agent evals" >&2
  exit 1
fi

if [ -n "${ODH_EVAL_BIN:-}" ]; then
  odh_cmd="$ODH_EVAL_BIN"
else
  if ! command -v go >/dev/null 2>&1; then
    echo "go is required when ODH_EVAL_BIN is not set" >&2
    exit 1
  fi
  odh_cmd="go run ./cmd/odh"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT HUP INT TERM

pass() {
  printf 'ok - %s\n' "$1"
}

run_odh() {
  # shellcheck disable=SC2086
  $odh_cmd "$@"
}

assert_json_filter() {
  label="$1"
  file="$2"
  filter="$3"
  if jq -e "$filter" "$file" >/dev/null; then
    pass "$label"
  else
    echo "not ok - $label" >&2
    echo "failed jq filter: $filter" >&2
    echo "output:" >&2
    sed -n '1,120p' "$file" >&2
    exit 1
  fi
}

task_count="$(jq '.tasks | length' "$tasks_file")"
if [ "$task_count" -lt 5 ]; then
  echo "not ok - expected at least 5 agent eval tasks" >&2
  exit 1
fi
pass "loaded $task_count agent eval tasks"

run_odh traffic zones --json >"$tmpdir/traffic-zones.json"
assert_json_filter "traffic zones exposes upstream zone ids" "$tmpdir/traffic-zones.json" '.zones[] | select(.zone_id == "6" and .name == "Pustertal")'

run_odh traffic categories --json >"$tmpdir/traffic-categories.json"
assert_json_filter "traffic categories exposes stable type filters" "$tmpdir/traffic-categories.json" '.categories[] | select(.name == "closure" and (.aliases | index("gesperrt")))'
assert_json_filter "traffic categories includes roadworks" "$tmpdir/traffic-categories.json" '.categories[] | select(.name == "roadworks" and (.upstream_subtypes | index("BAUSTELLE")))'

run_odh traffic search badia --today --zone-id 6 --json >"$tmpdir/traffic-badia.json"
assert_json_filter "traffic search returns structured badia result" "$tmpdir/traffic-badia.json" '.source == "odh" and .zone_id == "6" and .search == "badia" and (.count | type == "number")'
assert_json_filter "traffic search keeps ODH caveat visible" "$tmpdir/traffic-badia.json" '.warnings[]? | contains("source is Open Data Hub PROVINCE_BZ")'

run_odh mobility origins --station-type TrafficSensor --limit 1000 >"$tmpdir/traffic-sensor-origins.json"
assert_json_filter "mobility origins discovers A22 traffic sensors" "$tmpdir/traffic-sensor-origins.json" '.origins[] | select(.name == "A22" and .station_count > 0)'

run_odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 1000 --json >"$tmpdir/a22-datatypes.json"
assert_json_filter "mobility datatypes summarizes A22 measurements" "$tmpdir/a22-datatypes.json" '.origin == "A22" and .count > 0 and (.datatypes | length > 0)'

run_odh mobility latest --station-type EChargingStation --data-type number-available --origin ALPERIA --active --fresh-within 24h --sort newest --request-limit 1000 --limit 5 >"$tmpdir/ev-availability.json"
assert_json_filter "mobility latest exposes filtered availability wrapper" "$tmpdir/ev-availability.json" '.station_type == "EChargingStation" and .data_type == "number-available" and .origin == "ALPERIA" and .active_only == true and .fresh_within == "24h" and .sort == "newest" and (.raw_count | type == "number") and (.measurements | type == "array") and (.warnings | type == "array")'

run_odh diagnostics ev-charging --origin ALPERIA --fresh-within 24h --request-limit 1000 --limit 5 >"$tmpdir/ev-diagnostics.json"
assert_json_filter "diagnostics reports EV availability caveats" "$tmpdir/ev-diagnostics.json" '.domain == "ev-charging" and (.verdict | type == "string") and (.raw_count | type == "number") and (.current_count | type == "number") and (.warnings | type == "array") and (.recommended_command | test("odh mobility latest"))'

run_odh diagnostics parking-forecasts --origin "Municipality Merano" --fresh-within 2h --forecast-minutes 60 --request-limit 1000 --limit 5 >"$tmpdir/parking-forecast-diagnostics.json"
assert_json_filter "diagnostics separates current parking from stale forecasts" "$tmpdir/parking-forecast-diagnostics.json" '.domain == "parking-forecasts" and (.verdict | type == "string") and .current.station_type == "ParkingStation" and .current.data_type == "free" and .forecast.data_type == "parking-forecast-60" and (.warnings | type == "array")'

run_odh diagnostics tourism-events --limit 5 >"$tmpdir/tourism-event-diagnostics.json"
assert_json_filter "diagnostics reports tourism event caveats" "$tmpdir/tourism-event-diagnostics.json" '.domain == "tourism-events" and (.verdict | type == "string") and (.events | type == "array") and (.warnings | type == "array")'

run_odh a22 status --limit 5 --json >"$tmpdir/a22-status.json"
assert_json_filter "a22 status separates events and forecast feeds" "$tmpdir/a22-status.json" '.events.count >= 0 and .forecast.count >= 0 and (.warnings | type == "array")'

run_odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday --json >"$tmpdir/delay-stats.json"
assert_json_filter "transit delay-stats reports historical archive limitation" "$tmpdir/delay-stats.json" '.supported == false and (.reason | test("historical|archive|GTFS-RT"))'

run_odh transit stops search auer --limit 1 --json >"$tmpdir/transit-stops.json"
transit_stop_id="$(jq -r '.stops[0].id // ""' "$tmpdir/transit-stops.json")"
if [ -z "$transit_stop_id" ]; then
  echo "not ok - transit stop search returned no stop id" >&2
  sed -n '1,120p' "$tmpdir/transit-stops.json" >&2
  exit 1
fi
run_odh transit departures --stop-id "$transit_stop_id" --date 2026-05-16 --around 14:05 --window 5m --mode train --limit 5 --json >"$tmpdir/transit-departures-by-id.json"
assert_json_filter "transit departures supports exact stop ids" "$tmpdir/transit-departures-by-id.json" '(.stop_match_mode == "stop-id" or .stop_match_mode == "parent-station") and (.stop_id | length > 0) and (.matched_stops | length >= 1) and (.departures | type == "array")'

printf '\nAgent eval smoke checks passed. Use evals/agent/tasks.json for manual agent scoring.\n'
