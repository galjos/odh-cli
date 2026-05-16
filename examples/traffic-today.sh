#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: CC0-1.0

set -euo pipefail

ODH_BIN="${ODH_BIN:-odh}"

"$ODH_BIN" traffic today \
  --area ueberetsch-unterland \
  --type roadworks \
  --format table \
  --limit 100
