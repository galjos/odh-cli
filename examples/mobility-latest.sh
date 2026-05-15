#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: CC0-1.0

set -euo pipefail

odh mobility latest \
  --station-type EChargingStation \
  --data-type number-available \
  --limit 5
