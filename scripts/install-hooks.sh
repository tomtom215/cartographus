#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# =============================================================================
# Git Hooks Installation Script
# =============================================================================
# Configures git to use the project's custom hooks from .githooks/
#
# Usage:
#   ./scripts/install-hooks.sh
#
# To uninstall:
#   git config --unset core.hooksPath
#
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Installing git hooks for Cartographus..."
echo ""

# Check if we're in a git repository
if [[ ! -d "$PROJECT_ROOT/.git" ]]; then
    echo "Error: Not a git repository"
    exit 1
fi

# Configure git to use our hooks directory
git config core.hooksPath .githooks

# Make hooks executable
chmod +x "$PROJECT_ROOT/.githooks/"* 2>/dev/null || true

echo "Git hooks installed successfully!"
echo ""
echo "Hooks location: .githooks/"
echo "Available hooks:"
for hook in "$PROJECT_ROOT/.githooks/"*; do
    if [[ -f "$hook" ]]; then
        echo "  - $(basename "$hook")"
    fi
done
echo ""
echo "The pre-commit hook will now run automatically before each commit."
echo ""
echo "To bypass hooks (not recommended):"
echo "  git commit --no-verify"
echo ""
echo "To uninstall hooks:"
echo "  git config --unset core.hooksPath"
