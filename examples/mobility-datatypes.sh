#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: CC0-1.0

set -euo pipefail

if [[ -z "${ODH_BIN:-}" && -x ./odh ]]; then
  ODH_BIN="./odh"
else
  ODH_BIN="${ODH_BIN:-odh}"
fi

"$ODH_BIN" mobility datatypes \
  --station-type TrafficSensor \
  --origin A22 \
  --limit 100
