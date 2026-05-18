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

run_odh mobility datatypes --station-type TrafficSensor --origin A22 --limit 1000 >"$tmpdir/a22-datatypes.json"
assert_json_filter "mobility datatypes summarizes A22 measurements" "$tmpdir/a22-datatypes.json" '.origin == "A22" and .count > 0 and (.datatypes | length > 0)'

run_odh mobility latest --station-type EChargingStation --data-type number-available --origin ALPERIA --active --fresh-within 24h --sort newest --request-limit 1000 --limit 5 >"$tmpdir/ev-availability.json"
assert_json_filter "mobility latest exposes filtered availability wrapper" "$tmpdir/ev-availability.json" '.station_type == "EChargingStation" and .data_type == "number-available" and .origin == "ALPERIA" and .active_only == true and .fresh_within == "24h" and .sort == "newest" and (.raw_count | type == "number") and (.measurements | type == "array") and (.warnings | type == "array")'

run_odh a22 status --limit 5 >"$tmpdir/a22-status.json"
assert_json_filter "a22 status separates events and forecast feeds" "$tmpdir/a22-status.json" '.events.count >= 0 and .forecast.count >= 0 and (.warnings | type == "array")'

run_odh transit delay-stats --from auer --to brenner --time 14:05 --weekday saturday >"$tmpdir/delay-stats.json"
assert_json_filter "transit delay-stats reports historical archive limitation" "$tmpdir/delay-stats.json" '.supported == false and (.reason | test("historical|archive|GTFS-RT"))'

printf '\nAgent eval smoke checks passed. Use evals/agent/tasks.json for manual agent scoring.\n'
