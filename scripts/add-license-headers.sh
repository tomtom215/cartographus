#!/bin/bash
# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
#
# Script to add license headers to all source files
# Improvements over original:
#   - Preserves file permissions and ownership
#   - Adds blank line after header for Go (staticcheck ST1000)
#   - Dry-run mode for preview
#   - Permission verification before/after
#   - Skips minified files

# Don't use set -e as ((var++)) returns 1 when var is 0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_FILES=0
MODIFIED_FILES=0
SKIPPED_FILES=0
ERROR_FILES=0

# Options
DRY_RUN=false
VERIFY_ONLY=false
VERBOSE=false

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Permission tracking files
PERMISSIONS_BEFORE=""
PERMISSIONS_AFTER=""

# License header templates
# Note: Go files need blank line after header for staticcheck ST1000
HEADER_GO="// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

"

HEADER_SLASH="// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
"

HEADER_HASH="# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
"

HEADER_CSS="/*
 * Cartographus - Media Server Analytics and Geographic Visualization
 * Copyright 2026 Tom F. (tomtom215)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 * https://github.com/tomtom215/cartographus
 */
"

HEADER_HTML="<!--
  Cartographus - Media Server Analytics and Geographic Visualization
  Copyright 2026 Tom F. (tomtom215)
  SPDX-License-Identifier: AGPL-3.0-or-later
  https://github.com/tomtom215/cartographus
-->
"

# Function to display usage
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Add AGPL-3.0-or-later license headers to all source files.

OPTIONS:
    -n, --dry-run     Preview changes without modifying files
    -v, --verbose     Show all files being processed
    --verify          Only verify permissions (no modifications)
    -h, --help        Show this help message

EXAMPLES:
    $(basename "$0")              # Add headers to all files
    $(basename "$0") --dry-run    # Preview what would change
    $(basename "$0") --verify     # Check current permissions
EOF
    exit 0
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            --verify)
                VERIFY_ONLY=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                usage
                ;;
        esac
    done
}

# Function to check if file already has license header
has_license_header() {
    local file="$1"
    grep -q "SPDX-License-Identifier" "$file" 2>/dev/null
}

# Function to capture permissions for a file
capture_permissions() {
    local file="$1"
    local output_file="$2"
    stat -c "%a %U:%G %n" "$file" >> "$output_file" 2>/dev/null
}

# Function to add header while preserving permissions
add_header_safe() {
    local file="$1"
    local header="$2"
    local preserve_first_line="${3:-false}"
    local first_line_pattern="${4:-}"

    if has_license_header "$file"; then
        ((SKIPPED_FILES++))
        [[ "$VERBOSE" == "true" ]] && echo -e "${YELLOW}[skip]${NC} $file (already has header)"
        return
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        ((MODIFIED_FILES++))
        echo -e "${BLUE}[dry-run]${NC} Would modify: $file"
        return
    fi

    # Capture original permissions
    local original_perms
    original_perms=$(stat -c "%a" "$file")

    local temp_file
    temp_file=$(mktemp)

    if [[ "$preserve_first_line" == "true" ]] && [[ -n "$first_line_pattern" ]] && head -1 "$file" | grep -q "$first_line_pattern"; then
        # Preserve first line (shebang or DOCTYPE)
        head -1 "$file" > "$temp_file"
        echo "" >> "$temp_file"
        echo -n "$header" >> "$temp_file"
        tail -n +2 "$file" >> "$temp_file"
    else
        echo -n "$header" > "$temp_file"
        cat "$file" >> "$temp_file"
    fi

    # Move temp file to original location
    mv "$temp_file" "$file"

    # Restore original permissions
    chmod "$original_perms" "$file"

    # Verify permissions were preserved
    local new_perms
    new_perms=$(stat -c "%a" "$file")
    if [[ "$original_perms" != "$new_perms" ]]; then
        echo -e "${RED}[ERROR]${NC} Permission mismatch for $file: was $original_perms, now $new_perms"
        ((ERROR_FILES++))
        return
    fi

    ((MODIFIED_FILES++))
    echo -e "${GREEN}[+]${NC} $file"
}

# Function to add header to Go files (with blank line for staticcheck ST1000)
add_go_header() {
    local file="$1"
    add_header_safe "$file" "$HEADER_GO" "false" ""
}

# Function to add header to TypeScript/JavaScript files
add_slash_header() {
    local file="$1"
    add_header_safe "$file" "$HEADER_SLASH" "false" ""
}

# Function to add header to files with # comments (Shell, YAML, etc.)
add_hash_header() {
    local file="$1"
    local preserve_shebang="${2:-false}"

    if [[ "$preserve_shebang" == "true" ]]; then
        add_header_safe "$file" "$HEADER_HASH" "true" "^#!"
    else
        add_header_safe "$file" "$HEADER_HASH" "false" ""
    fi
}

# Function to add header to CSS files
add_css_header() {
    local file="$1"
    add_header_safe "$file" "$HEADER_CSS" "false" ""
}

# Function to add header to HTML files
add_html_header() {
    local file="$1"
    add_header_safe "$file" "$HEADER_HTML" "true" "<!DOCTYPE"
}

# Capture all permissions before modifications
capture_all_permissions() {
    local output_file="$1"
    echo "# Permissions captured at $(date -Iseconds)" > "$output_file"

    # Find all relevant source files
    find "$PROJECT_ROOT" -type f \( \
        -name "*.go" -o \
        -name "*.ts" -o \
        -name "*.js" -o \
        -name "*.css" -o \
        -name "*.sh" -o \
        -name "*.yaml" -o \
        -name "*.yml" -o \
        -name "*.html" -o \
        -name "*.tmpl" -o \
        -name "Dockerfile" -o \
        -name "Makefile" \
    \) -not -path "*/vendor/*" \
       -not -path "*/.git/*" \
       -not -path "*/node_modules/*" \
       -not -name "*.min.css" \
       -not -name "*.min.js" \
       -print0 2>/dev/null | while IFS= read -r -d '' file; do
        capture_permissions "$file" "$output_file"
    done

    # Also capture Dockerfile and Makefile explicitly
    [[ -f "$PROJECT_ROOT/Dockerfile" ]] && capture_permissions "$PROJECT_ROOT/Dockerfile" "$output_file"
    [[ -f "$PROJECT_ROOT/Makefile" ]] && capture_permissions "$PROJECT_ROOT/Makefile" "$output_file"
}

# Verify permissions match
verify_permissions() {
    local before_file="$1"
    local after_file="$2"

    echo -e "\n${YELLOW}=== Verifying Permissions ===${NC}"

    # Compare permissions (ignoring the timestamp comment line)
    local diff_output
    diff_output=$(diff <(grep -v "^#" "$before_file" | sort) <(grep -v "^#" "$after_file" | sort))

    if [[ -z "$diff_output" ]]; then
        echo -e "${GREEN}[OK]${NC} All permissions preserved correctly"
        return 0
    else
        echo -e "${RED}[ERROR]${NC} Permission differences detected:"
        echo "$diff_output"
        return 1
    fi
}

# Process Go files
process_go_files() {
    echo -e "\n${YELLOW}=== Processing Go files ===${NC}"
    while IFS= read -r -d '' file; do
        ((TOTAL_FILES++))
        add_go_header "$file"
    done < <(find "$PROJECT_ROOT" -type f -name "*.go" \
        -not -path "*/vendor/*" \
        -not -path "*/.git/*" \
        -print0 2>/dev/null)
}

# Process TypeScript files
process_ts_files() {
    echo -e "\n${YELLOW}=== Processing TypeScript files ===${NC}"
    while IFS= read -r -d '' file; do
        ((TOTAL_FILES++))
        add_slash_header "$file"
    done < <(find "$PROJECT_ROOT" -type f -name "*.ts" \
        -not -path "*/node_modules/*" \
        -not -path "*/.git/*" \
        -not -name "*.min.ts" \
        -print0 2>/dev/null)
}

# Process JavaScript files
process_js_files() {
    echo -e "\n${YELLOW}=== Processing JavaScript files ===${NC}"
    while IFS= read -r -d '' file; do
        ((TOTAL_FILES++))
        add_slash_header "$file"
    done < <(find "$PROJECT_ROOT" -type f -name "*.js" \
        -not -path "*/node_modules/*" \
        -not -path "*/.git/*" \
        -not -name "*.min.js" \
        -print0 2>/dev/null)
}

# Process CSS files (skip minified)
process_css_files() {
    echo -e "\n${YELLOW}=== Processing CSS files ===${NC}"
    while IFS= read -r -d '' file; do
        ((TOTAL_FILES++))
        add_css_header "$file"
    done < <(find "$PROJECT_ROOT" -type f -name "*.css" \
        -not -path "*/node_modules/*" \
        -not -path "*/.git/*" \
        -not -name "*.min.css" \
        -print0 2>/dev/null)
}

# Process Shell scripts
process_shell_files() {
    echo -e "\n${YELLOW}=== Processing Shell scripts ===${NC}"
    while IFS= read -r -d '' file; do
        ((TOTAL_FILES++))
        add_hash_header "$file" "true"
    done < <(find "$PROJECT_ROOT" -type f -name "*.sh" \
        -not -path "*/.git/*" \
        -print0 2>/dev/null)
}

# Process YAML files
process_yaml_files() {
    echo -e "\n${YELLOW}=== Processing YAML files ===${NC}"
    while IFS= read -r -d '' file; do
        ((TOTAL_FILES++))
        add_hash_header "$file" "false"
    done < <(find "$PROJECT_ROOT" -type f \( -name "*.yaml" -o -name "*.yml" \) \
        -not -path "*/.git/*" \
        -print0 2>/dev/null)
}

# Process HTML files
process_html_files() {
    echo -e "\n${YELLOW}=== Processing HTML files ===${NC}"
    while IFS= read -r -d '' file; do
        ((TOTAL_FILES++))
        add_html_header "$file"
    done < <(find "$PROJECT_ROOT" -type f -name "*.html" \
        -not -path "*/node_modules/*" \
        -not -path "*/.git/*" \
        -print0 2>/dev/null)
}

# Process Dockerfile
process_dockerfile() {
    echo -e "\n${YELLOW}=== Processing Dockerfile ===${NC}"
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    if [[ -f "$dockerfile" ]]; then
        ((TOTAL_FILES++))
        add_hash_header "$dockerfile" "false"
    fi
}

# Process Makefile
process_makefile() {
    echo -e "\n${YELLOW}=== Processing Makefile ===${NC}"
    local makefile="$PROJECT_ROOT/Makefile"
    if [[ -f "$makefile" ]]; then
        ((TOTAL_FILES++))
        add_hash_header "$makefile" "false"
    fi
}

# Process Go template files (.tmpl)
process_tmpl_files() {
    echo -e "\n${YELLOW}=== Processing Go template files ===${NC}"
    while IFS= read -r -d '' file; do
        ((TOTAL_FILES++))
        add_html_header "$file"
    done < <(find "$PROJECT_ROOT" -type f -name "*.tmpl" \
        -not -path "*/.git/*" \
        -print0 2>/dev/null)
}

# Main execution
main() {
    parse_args "$@"

    echo -e "${YELLOW}======================================${NC}"
    echo -e "${YELLOW}  Cartographus License Header Script  ${NC}"
    echo -e "${YELLOW}======================================${NC}"
    echo ""
    echo "Project root: $PROJECT_ROOT"
    echo "License: AGPL-3.0-or-later"
    echo "Copyright: 2026 Tom F. (tomtom215)"

    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "Mode: ${BLUE}DRY RUN${NC} (no files will be modified)"
    fi
    echo ""

    # Create temporary files for permission tracking
    PERMISSIONS_BEFORE=$(mktemp)
    PERMISSIONS_AFTER=$(mktemp)

    # Capture permissions before
    echo -e "${YELLOW}=== Capturing permissions before ===${NC}"
    capture_all_permissions "$PERMISSIONS_BEFORE"
    local before_count
    before_count=$(grep -c -v "^#" "$PERMISSIONS_BEFORE" || echo "0")
    echo -e "${GREEN}[OK]${NC} Captured permissions for $before_count files"

    if [[ "$VERIFY_ONLY" == "true" ]]; then
        echo -e "\n${YELLOW}=== Current File Permissions ===${NC}"
        echo "Shell scripts:"
        find "$PROJECT_ROOT/scripts" -name "*.sh" -exec stat -c "  %a %n" {} \; 2>/dev/null | head -20
        echo ""
        echo "Permissions saved to: $PERMISSIONS_BEFORE"
        exit 0
    fi

    # Process all file types
    process_go_files
    process_ts_files
    process_js_files
    process_css_files
    process_shell_files
    process_yaml_files
    process_html_files
    process_dockerfile
    process_makefile
    process_tmpl_files

    # Capture permissions after (only if not dry-run)
    if [[ "$DRY_RUN" != "true" ]]; then
        echo -e "\n${YELLOW}=== Capturing permissions after ===${NC}"
        capture_all_permissions "$PERMISSIONS_AFTER"

        # Verify permissions
        verify_permissions "$PERMISSIONS_BEFORE" "$PERMISSIONS_AFTER"
    fi

    # Summary
    echo ""
    echo -e "${YELLOW}======================================${NC}"
    echo -e "${YELLOW}           Summary                    ${NC}"
    echo -e "${YELLOW}======================================${NC}"
    echo -e "Total files processed: ${TOTAL_FILES}"

    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "Files that would be modified: ${BLUE}${MODIFIED_FILES}${NC}"
    else
        echo -e "Files modified:        ${GREEN}${MODIFIED_FILES}${NC}"
    fi

    echo -e "Files skipped:         ${YELLOW}${SKIPPED_FILES}${NC} (already had headers)"

    if [[ $ERROR_FILES -gt 0 ]]; then
        echo -e "Files with errors:     ${RED}${ERROR_FILES}${NC}"
    fi

    echo ""

    # Cleanup temp files
    if [[ "$DRY_RUN" != "true" ]]; then
        echo "Permission logs saved:"
        echo "  Before: $PERMISSIONS_BEFORE"
        echo "  After:  $PERMISSIONS_AFTER"
    else
        rm -f "$PERMISSIONS_BEFORE" "$PERMISSIONS_AFTER"
    fi

    # Return error code if there were issues
    if [[ $ERROR_FILES -gt 0 ]]; then
        exit 1
    fi
}

# Run main function
main "$@"
