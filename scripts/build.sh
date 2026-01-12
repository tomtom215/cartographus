#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
set -e

echo "Building Cartographus..."

echo "Step 1: Building frontend..."
cd web
npm ci
npm run build
cd ..

echo "Step 2: Building backend..."
CGO_ENABLED=1 go build -o cartographus ./cmd/server

echo "Build complete! Binary: ./cartographus"
echo "Frontend assets: ./web/dist"
