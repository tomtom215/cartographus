#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# =============================================================================
# DuckDB Extensions Installer
# =============================================================================
# Installs bundled DuckDB extensions to the user's home directory.
# This script is included in binary releases to provide zero-dependency setup.
#
# Usage:
#   ./scripts/install-extensions.sh
#
# The script copies pre-bundled extensions from:
#   ./duckdb-extensions/{version}/{platform}/
# To:
#   ~/.duckdb/extensions/{version}/{platform}/
#
# =============================================================================

set -euo pipefail

# Colors for output
if [[ -t 1 ]]; then
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    RED='\033[0;31m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    GREEN=''
    YELLOW=''
    RED=''
    BOLD=''
    NC=''
fi

echo ""
echo "${BOLD}DuckDB Extensions Installer${NC}"
echo "============================"
echo ""

# Find the script's directory (where the release was extracted)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASE_DIR="$(dirname "$SCRIPT_DIR")"

# Check if extensions are bundled
BUNDLED_DIR="$RELEASE_DIR/duckdb-extensions"
if [[ ! -d "$BUNDLED_DIR" ]]; then
    echo "${RED}Error: Bundled extensions not found at: $BUNDLED_DIR${NC}"
    echo "Make sure you're running this script from the extracted release directory."
    exit 1
fi

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    linux)  OS_NAME="linux" ;;
    darwin) OS_NAME="osx" ;;
    *)
        echo "${RED}Error: Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)  ARCH_NAME="amd64" ;;
    aarch64|arm64) ARCH_NAME="arm64" ;;
    *)
        echo "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

PLATFORM="${OS_NAME}_${ARCH_NAME}"
echo "Detected platform: ${BOLD}$PLATFORM${NC}"

# Find the DuckDB version from bundled extensions
DUCKDB_VERSION=""
for version_dir in "$BUNDLED_DIR"/v*; do
    if [[ -d "$version_dir" ]]; then
        DUCKDB_VERSION=$(basename "$version_dir")
        break
    fi
done

if [[ -z "$DUCKDB_VERSION" ]]; then
    echo "${RED}Error: Could not determine DuckDB version from bundled extensions${NC}"
    exit 1
fi

echo "DuckDB version: ${BOLD}$DUCKDB_VERSION${NC}"

# Source directory for this platform
SOURCE_DIR="$BUNDLED_DIR/$DUCKDB_VERSION/$PLATFORM"
if [[ ! -d "$SOURCE_DIR" ]]; then
    echo "${RED}Error: No extensions found for platform $PLATFORM${NC}"
    echo "Available platforms:"
    ls -1 "$BUNDLED_DIR/$DUCKDB_VERSION/" 2>/dev/null || echo "  (none)"
    exit 1
fi

# Target directory in user's home
TARGET_DIR="$HOME/.duckdb/extensions/$DUCKDB_VERSION/$PLATFORM"

echo ""
echo "Source: $SOURCE_DIR"
echo "Target: $TARGET_DIR"
echo ""

# Create target directory
mkdir -p "$TARGET_DIR"

# Copy extensions
echo "Installing extensions..."
INSTALLED=0
SKIPPED=0

for ext_file in "$SOURCE_DIR"/*.duckdb_extension; do
    if [[ ! -f "$ext_file" ]]; then
        continue
    fi

    ext_name=$(basename "$ext_file")
    target_file="$TARGET_DIR/$ext_name"

    if [[ -f "$target_file" ]]; then
        # Compare file sizes to detect if update is needed
        source_size=$(stat -c%s "$ext_file" 2>/dev/null || stat -f%z "$ext_file" 2>/dev/null)
        target_size=$(stat -c%s "$target_file" 2>/dev/null || stat -f%z "$target_file" 2>/dev/null)

        if [[ "$source_size" == "$target_size" ]]; then
            echo "  ${YELLOW}~${NC} $ext_name (already installed)"
            SKIPPED=$((SKIPPED + 1))
            continue
        fi
    fi

    cp "$ext_file" "$target_file"
    echo "  ${GREEN}+${NC} $ext_name"
    INSTALLED=$((INSTALLED + 1))
done

echo ""
echo "============================"
echo "${GREEN}Installation complete!${NC}"
echo ""
echo "  Installed: $INSTALLED extensions"
echo "  Skipped:   $SKIPPED extensions (already present)"
echo "  Location:  $TARGET_DIR"
echo ""
echo "You can now run Cartographus:"
echo "  ./cartographus"
echo ""
