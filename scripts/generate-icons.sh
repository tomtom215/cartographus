#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# Icon Generator Script for Cartographus
# This script converts the SVG icon to PNG formats required for the PWA manifest
# and Apple touch icons.
#
# Prerequisites:
#   - ImageMagick (for convert command)
#   - rsvg-convert (librsvg2-bin package) - preferred for better quality
#
# Usage:
#   ./generate-icons.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLIC_DIR="$SCRIPT_DIR/../web/public"
SVG_SOURCE="$PUBLIC_DIR/icon.svg"
OUTPUT_DIR="$PUBLIC_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Cartographus - Icon Generator"
echo "=============================="
echo

# Check if source SVG exists
if [ ! -f "$SVG_SOURCE" ]; then
    echo -e "${RED}Error: Source SVG not found at $SVG_SOURCE${NC}"
    exit 1
fi

# Check for rsvg-convert (preferred)
if command -v rsvg-convert &> /dev/null; then
    CONVERTER="rsvg"
    echo -e "${GREEN}Using rsvg-convert for high-quality rendering${NC}"
elif command -v convert &> /dev/null; then
    CONVERTER="imagemagick"
    echo -e "${YELLOW}Using ImageMagick convert (rsvg-convert recommended for better quality)${NC}"
else
    echo -e "${RED}Error: Neither rsvg-convert nor ImageMagick convert found${NC}"
    echo "Please install one of the following:"
    echo "  - librsvg2-bin (Debian/Ubuntu): sudo apt-get install librsvg2-bin"
    echo "  - ImageMagick: sudo apt-get install imagemagick"
    exit 1
fi

# Generate icons
echo
echo "Generating icons..."

# Function to generate a PNG from SVG
generate_icon() {
    local size=$1
    local output_file="$OUTPUT_DIR/icon-${size}.png"

    if [ "$CONVERTER" = "rsvg" ]; then
        rsvg-convert -w "$size" -h "$size" "$SVG_SOURCE" -o "$output_file"
    else
        convert -background none -resize "${size}x${size}" "$SVG_SOURCE" "$output_file"
    fi

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} Generated $output_file"
    else
        echo -e "${RED}✗${NC} Failed to generate $output_file"
        return 1
    fi
}

# Generate required sizes
generate_icon 192
generate_icon 512

# Generate favicon.ico
generate_favicon() {
    local output_file="$OUTPUT_DIR/favicon.ico"

    # Generate multiple sizes for favicon
    if [ "$CONVERTER" = "rsvg" ]; then
        # Generate temporary PNGs at different sizes
        rsvg-convert -w 16 -h 16 "$SVG_SOURCE" -o "/tmp/favicon-16.png"
        rsvg-convert -w 32 -h 32 "$SVG_SOURCE" -o "/tmp/favicon-32.png"
        rsvg-convert -w 48 -h 48 "$SVG_SOURCE" -o "/tmp/favicon-48.png"

        # Use ImageMagick to combine into ICO (if available)
        if command -v convert &> /dev/null; then
            convert /tmp/favicon-16.png /tmp/favicon-32.png /tmp/favicon-48.png "$output_file"
            rm -f /tmp/favicon-16.png /tmp/favicon-32.png /tmp/favicon-48.png
        else
            # Fallback: copy 32x32 as ico
            cp /tmp/favicon-32.png "$output_file"
            rm -f /tmp/favicon-16.png /tmp/favicon-32.png /tmp/favicon-48.png
        fi
    else
        convert -background none -resize "32x32" "$SVG_SOURCE" "$output_file"
    fi

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} Generated $output_file"
    else
        echo -e "${RED}✗${NC} Failed to generate $output_file"
        return 1
    fi
}

# Generate Open Graph image
generate_og_image() {
    local output_file="$OUTPUT_DIR/og-image.png"

    if [ "$CONVERTER" = "rsvg" ]; then
        # OG images should be 1200x630 for best display
        # We'll create a centered icon on a dark background
        rsvg-convert -w 400 -h 400 "$SVG_SOURCE" -o "/tmp/og-icon.png"

        if command -v convert &> /dev/null; then
            # Create a dark background with the icon centered
            convert -size 1200x630 xc:'#1a1a2e' /tmp/og-icon.png -gravity center -composite "$output_file"
            rm -f /tmp/og-icon.png
        else
            # Fallback: just use larger icon
            rsvg-convert -w 630 -h 630 "$SVG_SOURCE" -o "$output_file"
        fi
    else
        # Create a simple OG image with the icon
        convert -size 1200x630 xc:'#1a1a2e' \
            \( "$SVG_SOURCE" -background none -resize 400x400 \) \
            -gravity center -composite "$output_file"
    fi

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} Generated $output_file (1200x630 for social sharing)"
    else
        echo -e "${YELLOW}!${NC} Could not generate og-image.png - manual creation recommended"
        return 1
    fi
}

# Generate favicon.ico
echo
echo "Generating favicon.ico..."
generate_favicon

# Generate OG image for social sharing
echo
echo "Generating Open Graph image..."
generate_og_image

# Optional: Generate additional useful sizes
echo
read -p "Generate additional sizes (apple icons)? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    generate_icon 16    # Small icon
    generate_icon 32    # Medium icon
    generate_icon 180   # Apple touch icon (iPhone)
    generate_icon 167   # Apple touch icon (iPad)
fi

echo
echo -e "${GREEN}Icon generation complete!${NC}"
echo
echo "Generated icons are located in: $OUTPUT_DIR"
echo "  - icon-192.png   (PWA manifest)"
echo "  - icon-512.png   (PWA manifest)"
echo "  - favicon.ico    (Browser tab icon)"
echo "  - og-image.png   (Social media sharing)"
echo
echo "Next steps:"
echo "  1. Verify the generated icons look correct"
echo "  2. manifest.json references the PNG icons"
echo "  3. index.html references favicon.ico and og-image.png"
echo "  4. Commit the generated files to the repository"
