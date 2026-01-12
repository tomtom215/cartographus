#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
set -e

echo "Running Cartographus test suite..."

echo "Step 1: Linting Go code..."
go vet ./...
gofmt -l .

echo "Step 2: Running Go tests..."
go test -v -race -coverprofile=coverage.txt ./...

echo "Step 3: Linting TypeScript..."
cd web
npx tsc --noEmit
cd ..

echo "All tests passed!"
