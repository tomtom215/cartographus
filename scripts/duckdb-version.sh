#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# DuckDB Version - Single Source of Truth
# ===========================================
# This file defines the DuckDB version used throughout the project.
# All scripts and CI workflows should source this file.
#
# IMPORTANT: When updating the version, also update:
#   - internal/database/database_extensions.go (duckdbVersion const)
#   - go.mod (duckdb-go dependency if needed)
#
# The version must match the duckdb-go-bindings library version.
# See: https://github.com/marcboeker/go-duckdb

export DUCKDB_VERSION="v1.4.3"
