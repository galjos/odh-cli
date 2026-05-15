#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Josef Gallmetzer
#
# SPDX-License-Identifier: CC0-1.0

set -euo pipefail

odh tourism poi --limit 1 --seed 42 --fields Detail.en.Title,GpsInfo
