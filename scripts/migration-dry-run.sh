#!/bin/bash
# migration-dry-run.sh
# Analyzes the codebase and reports all files that need changes for migration
# Does NOT modify any files - read-only analysis only
#
# Usage: ./scripts/migration-dry-run.sh [--verbose] [--output report.txt]

set -e

# Configuration - patterns to search for
OLD_REPO="github.com/tomtom215/cartographus"
NEW_REPO="github.com/tomtom215/cartographus"  # For display in report
OLD_IMAGE="ghcr.io/tomtom215/map"
OLD_DB="map.duckdb"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Parse arguments
VERBOSE=false
OUTPUT_FILE=""
JSON_OUTPUT=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -j|--json)
            JSON_OUTPUT=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--verbose] [--output report.txt] [--json]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose    Show all matching lines, not just file names"
            echo "  -o, --output     Write report to file"
            echo "  -j, --json       Output results in JSON format (for automation)"
            echo "  -h, --help       Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Initialize counters
declare -A CATEGORY_COUNTS

# Function to log output (to stdout and optionally to file)
log() {
    if [ -n "$OUTPUT_FILE" ]; then
        echo -e "$1" | tee -a "$OUTPUT_FILE"
    else
        echo -e "$1"
    fi
}

# Clear output file if specified
if [ -n "$OUTPUT_FILE" ]; then
    true > "$OUTPUT_FILE"
fi

# Header
log ""
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log "${BOLD}       CARTOGRAPHUS MIGRATION DRY-RUN ANALYSIS${NC}"
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log ""
log "Analyzing codebase for migration from:"
log "  ${CYAN}$OLD_REPO${NC} -> ${GREEN}$NEW_REPO${NC}"
log ""
log "Date: $(date)"
log ""

# ============================================================================
# ENVIRONMENT SNAPSHOT (for reproducibility)
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[0/9] ENVIRONMENT SNAPSHOT${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

log "${YELLOW}Git state:${NC}"
if command -v git &> /dev/null && [ -d .git ]; then
    GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    GIT_DIRTY=$(git status --porcelain 2>/dev/null | wc -l)
    log "  Branch: ${CYAN}$GIT_BRANCH${NC}"
    log "  Commit: ${CYAN}$GIT_COMMIT${NC}"
    if [ "$GIT_DIRTY" -gt 0 ]; then
        log "  Status: ${YELLOW}$GIT_DIRTY uncommitted changes${NC}"
    else
        log "  Status: ${GREEN}Clean${NC}"
    fi
else
    log "  ${YELLOW}Not a git repository${NC}"
fi
log ""

log "${YELLOW}Tool versions:${NC}"
if command -v go &> /dev/null; then
    GO_VERSION=$(go version 2>/dev/null | awk '{print $3}')
    log "  Go: ${CYAN}$GO_VERSION${NC}"
else
    log "  Go: ${RED}Not found${NC}"
fi
if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version 2>/dev/null)
    log "  Node: ${CYAN}$NODE_VERSION${NC}"
else
    log "  Node: ${YELLOW}Not found${NC}"
fi
if [ -f go.mod ]; then
    GO_MODULE=$(grep "^module " go.mod | awk '{print $2}')
    log "  Current module: ${CYAN}$GO_MODULE${NC}"
fi
log ""

# ============================================================================
# CATEGORY 1: DANGEROUS PATTERNS WARNING (DO NOT REPLACE THESE)
# ============================================================================
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log "${RED}${BOLD}[1/9] DANGEROUS PATTERNS - DO NOT REPLACE${NC}"
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log ""
log "${RED}${BOLD}⚠️  WARNING: The name 'map' is an extremely common programming term${NC}"
log "${RED}${BOLD}⚠️  A naive find-and-replace will DESTROY the codebase${NC}"
log ""

# Count dangerous patterns
log "${YELLOW}Counting dangerous 'map' patterns that must NOT be replaced:${NC}"
log ""

# Go map type declarations
GO_MAP_TYPE_COUNT=$(grep -r "map\[" --include="*.go" . 2>/dev/null | wc -l)
log "  ${RED}map[${NC} (Go map types):           ${BOLD}$GO_MAP_TYPE_COUNT${NC} occurrences"
CATEGORY_COUNTS["dangerous_go_map"]=$GO_MAP_TYPE_COUNT

# JavaScript/TypeScript .map() calls
JS_MAP_METHOD_COUNT=$(grep -r "\.map(" --include="*.ts" --include="*.js" . 2>/dev/null | wc -l)
log "  ${RED}.map(${NC} (JS/TS array method):     ${BOLD}$JS_MAP_METHOD_COUNT${NC} occurrences"
CATEGORY_COUNTS["dangerous_js_map"]=$JS_MAP_METHOD_COUNT

# Map-related variable and function names (excluding project references)
MAP_VARIABLE_COUNT=$(grep -riE "[A-Za-z_]map|map[A-Za-z_]" --include="*.go" --include="*.ts" . 2>/dev/null | grep -v "github.com/tomtom215/cartographus" | wc -l)
log "  ${RED}*map* / *Map*${NC} (variable names):  ${BOLD}$MAP_VARIABLE_COUNT${NC} occurrences"
CATEGORY_COUNTS["dangerous_map_vars"]=$MAP_VARIABLE_COUNT

# MapLibre/Mapbox references
MAPLIBRE_COUNT=$(grep -riE "maplibre|mapbox" --include="*.ts" --include="*.js" --include="*.go" --include="*.md" --include="*.css" . 2>/dev/null | wc -l)
log "  ${RED}MapLibre/Mapbox${NC} (UI library):   ${BOLD}$MAPLIBRE_COUNT${NC} occurrences"
CATEGORY_COUNTS["dangerous_maplibre"]=$MAPLIBRE_COUNT

# Files with "map" in the filename
MAP_FILENAME_COUNT=$(find . -type f \( -name "*map*" -o -name "*Map*" \) 2>/dev/null | grep -v node_modules | grep -v ".git" | wc -l)
log "  ${RED}Files named *map*${NC}:              ${BOLD}$MAP_FILENAME_COUNT${NC} files"
CATEGORY_COUNTS["dangerous_map_files"]=$MAP_FILENAME_COUNT

TOTAL_DANGEROUS=$((GO_MAP_TYPE_COUNT + JS_MAP_METHOD_COUNT + MAP_VARIABLE_COUNT + MAPLIBRE_COUNT))
log ""
log "  ${RED}${BOLD}TOTAL DANGEROUS PATTERNS:       $TOTAL_DANGEROUS+ occurrences${NC}"
log ""

log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${GREEN}Counting SAFE patterns (these ARE project references to replace):${NC}"
log ""

# Safe: github.com/tomtom215/cartographus (full URL)
SAFE_GITHUB_URL_COUNT=$(grep -r "github.com/tomtom215/cartographus" . 2>/dev/null | grep -v ".git/" | wc -l)
log "  ${GREEN}github.com/tomtom215/cartographus${NC}:       ${BOLD}$SAFE_GITHUB_URL_COUNT${NC} occurrences"
CATEGORY_COUNTS["safe_github_url"]=$SAFE_GITHUB_URL_COUNT

# Safe: ghcr.io/tomtom215/map
SAFE_GHCR_COUNT=$(grep -r "ghcr.io/tomtom215/map" --include="*.yml" --include="*.yaml" --include="*.md" . 2>/dev/null | wc -l)
log "  ${GREEN}ghcr.io/tomtom215/map${NC}:          ${BOLD}$SAFE_GHCR_COUNT${NC} occurrences"
CATEGORY_COUNTS["safe_ghcr"]=$SAFE_GHCR_COUNT

# Safe: map.duckdb
SAFE_DUCKDB_COUNT=$(grep -r "map\.duckdb" --include="*.go" --include="*.md" --include="*.yml" --include="*.yaml" --include="*.env*" . 2>/dev/null | wc -l)
log "  ${GREEN}map.duckdb${NC}:                     ${BOLD}$SAFE_DUCKDB_COUNT${NC} occurrences"
CATEGORY_COUNTS["safe_duckdb"]=$SAFE_DUCKDB_COUNT

TOTAL_SAFE=$((SAFE_GITHUB_URL_COUNT + SAFE_GHCR_COUNT + SAFE_DUCKDB_COUNT))
log ""
log "  ${GREEN}${BOLD}TOTAL SAFE PATTERNS:            $TOTAL_SAFE occurrences${NC}"
log ""

log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""
log "${RED}${BOLD}RATIO: ${TOTAL_DANGEROUS}+ dangerous vs ${TOTAL_SAFE} safe${NC}"
log ""
if [ "$TOTAL_DANGEROUS" -gt "$TOTAL_SAFE" ]; then
    RATIO=$((TOTAL_DANGEROUS / TOTAL_SAFE))
    log "${RED}${BOLD}⚠️  There are ${RATIO}x MORE dangerous patterns than safe ones!${NC}"
    log "${RED}${BOLD}⚠️  NEVER use blind 'sed s/map/cartographus/g' commands!${NC}"
fi
log ""
log "${YELLOW}See docs/MIGRATION_PLAN.md section 'CRITICAL: Safe Replacement Patterns'${NC}"
log "${YELLOW}for the ONLY commands that are safe to run.${NC}"
log ""

# ============================================================================
# CATEGORY 2: Go Module and Import Paths
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[2/9] GO MODULE AND IMPORT PATHS${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

# Check go.mod
log "${YELLOW}go.mod module declaration:${NC}"
if grep -q "module $OLD_REPO" go.mod 2>/dev/null; then
    log "  ${RED}NEEDS UPDATE:${NC} go.mod"
    grep "module $OLD_REPO" go.mod | sed 's/^/    /'
    CATEGORY_COUNTS["go_mod"]=1
else
    log "  ${GREEN}OK${NC} - go.mod already updated or not found"
    CATEGORY_COUNTS["go_mod"]=0
fi
log ""

# Check Go imports
log "${YELLOW}Go files with old import paths:${NC}"
GO_IMPORT_FILES=$(grep -rl "\"$OLD_REPO" --include="*.go" . 2>/dev/null | sort || true)
GO_IMPORT_COUNT=$(echo "$GO_IMPORT_FILES" | grep -c "." || echo 0)

if [ "$GO_IMPORT_COUNT" -gt 0 ]; then
    log "  ${RED}Found $GO_IMPORT_COUNT files with old imports${NC}"
    CATEGORY_COUNTS["go_imports"]=$GO_IMPORT_COUNT

    if [ "$VERBOSE" = true ]; then
        log ""
        log "  Files:"
        echo "$GO_IMPORT_FILES" | while read -r file; do
            [ -n "$file" ] && log "    - $file"
        done
    else
        log ""
        log "  Sample files (first 10):"
        echo "$GO_IMPORT_FILES" | head -10 | while read -r file; do
            [ -n "$file" ] && log "    - $file"
        done
        if [ "$GO_IMPORT_COUNT" -gt 10 ]; then
            log "    ... and $((GO_IMPORT_COUNT - 10)) more"
        fi
    fi
else
    log "  ${GREEN}OK${NC} - No Go files with old imports"
    CATEGORY_COUNTS["go_imports"]=0
fi
log ""

# ============================================================================
# CATEGORY 3: Docker Image References
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[3/9] DOCKER IMAGE REFERENCES${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

log "${YELLOW}Files with old Docker image references (${OLD_IMAGE}):${NC}"
DOCKER_FILES=$(grep -rl "$OLD_IMAGE" --include="*.yml" --include="*.yaml" --include="*.md" --include="*.xml" --include="Dockerfile*" . 2>/dev/null | sort || true)
DOCKER_COUNT=$(echo "$DOCKER_FILES" | grep -c "." || echo 0)

if [ "$DOCKER_COUNT" -gt 0 ]; then
    log "  ${RED}Found $DOCKER_COUNT files with old Docker image${NC}"
    CATEGORY_COUNTS["docker"]=$DOCKER_COUNT

    log ""
    log "  Files:"
    echo "$DOCKER_FILES" | while read -r file; do
        [ -n "$file" ] && log "    - $file"
    done

    if [ "$VERBOSE" = true ]; then
        log ""
        log "  Matching lines:"
        echo "$DOCKER_FILES" | while read -r file; do
            [ -n "$file" ] && grep -n "$OLD_IMAGE" "$file" 2>/dev/null | sed "s|^|    $file:|"
        done
    fi
else
    log "  ${GREEN}OK${NC} - No Docker image references found"
    CATEGORY_COUNTS["docker"]=0
fi
log ""

# ============================================================================
# CATEGORY 4: GitHub Repository URLs
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[4/9] GITHUB REPOSITORY URLs${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

log "${YELLOW}Files with old GitHub URLs (excluding Go imports):${NC}"

# Find in documentation and config files (not Go files, those are counted separately)
GITHUB_URL_FILES=$(grep -rl "$OLD_REPO" --include="*.md" --include="*.yml" --include="*.yaml" --include="*.json" --include="*.ts" --include="*.html" --include="*.xml" --include="*.txt" --include="*.sh" . 2>/dev/null | sort || true)
GITHUB_URL_COUNT=$(echo "$GITHUB_URL_FILES" | grep -c "." || echo 0)

if [ "$GITHUB_URL_COUNT" -gt 0 ]; then
    log "  ${RED}Found $GITHUB_URL_COUNT non-Go files with old GitHub URLs${NC}"
    CATEGORY_COUNTS["github_urls"]=$GITHUB_URL_COUNT

    if [ "$VERBOSE" = true ]; then
        log ""
        log "  Files:"
        echo "$GITHUB_URL_FILES" | while read -r file; do
            [ -n "$file" ] && log "    - $file"
        done
    else
        log ""
        log "  By file type:"
        for ext in md yml yaml json ts html xml txt sh; do
            count=$(echo "$GITHUB_URL_FILES" | grep -c "\.$ext$" || echo 0)
            [ "$count" -gt 0 ] && log "    - *.$ext: $count files"
        done
    fi
else
    log "  ${GREEN}OK${NC} - No GitHub URL references found"
    CATEGORY_COUNTS["github_urls"]=0
fi
log ""

# ============================================================================
# CATEGORY 5: Database Path References
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[5/9] DATABASE PATH REFERENCES${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

log "${YELLOW}Files with old database path (${OLD_DB}):${NC}"
DB_FILES=$(grep -rl "$OLD_DB" --include="*.go" --include="*.yml" --include="*.yaml" --include="*.md" --include="*.env*" --include="Dockerfile*" . 2>/dev/null | sort || true)
DB_COUNT=$(echo "$DB_FILES" | grep -c "." || echo 0)

if [ "$DB_COUNT" -gt 0 ]; then
    log "  ${RED}Found $DB_COUNT files with old database path${NC}"
    CATEGORY_COUNTS["database"]=$DB_COUNT

    log ""
    log "  Files:"
    echo "$DB_FILES" | while read -r file; do
        [ -n "$file" ] && log "    - $file"
    done

    if [ "$VERBOSE" = true ]; then
        log ""
        log "  Matching lines:"
        echo "$DB_FILES" | while read -r file; do
            [ -n "$file" ] && grep -n "$OLD_DB" "$file" 2>/dev/null | sed "s|^|    $file:|"
        done
    fi
else
    log "  ${GREEN}OK${NC} - No database path references found"
    CATEGORY_COUNTS["database"]=0
fi
log ""

# ============================================================================
# CATEGORY 6: License References
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[6/9] LICENSE REFERENCES${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

log "${YELLOW}LICENSE file check:${NC}"
if [ -f "LICENSE" ]; then
    if grep -q "MIT License" LICENSE 2>/dev/null; then
        log "  ${RED}NEEDS UPDATE:${NC} LICENSE file is MIT (needs AGPL-3.0)"
        CATEGORY_COUNTS["license_file"]=1
    elif grep -q "GNU AFFERO GENERAL PUBLIC LICENSE" LICENSE 2>/dev/null; then
        log "  ${GREEN}OK${NC} - LICENSE file is already AGPL-3.0"
        CATEGORY_COUNTS["license_file"]=0
    else
        log "  ${YELLOW}WARNING:${NC} LICENSE file exists but type unknown"
        CATEGORY_COUNTS["license_file"]=1
    fi
else
    log "  ${RED}MISSING:${NC} No LICENSE file found"
    CATEGORY_COUNTS["license_file"]=1
fi
log ""

log "${YELLOW}Files with MIT License references:${NC}"
MIT_FILES=$(grep -rl "MIT License\|License-MIT\|License: MIT" --include="*.md" --include="*.go" --include="*.ts" . 2>/dev/null | grep -v "^./LICENSE$" | sort || true)
MIT_COUNT=$(echo "$MIT_FILES" | grep -c "." || echo 0)

if [ "$MIT_COUNT" -gt 0 ]; then
    log "  ${RED}Found $MIT_COUNT files with MIT license references${NC}"
    CATEGORY_COUNTS["license_refs"]=$MIT_COUNT

    log ""
    log "  Files:"
    echo "$MIT_FILES" | while read -r file; do
        [ -n "$file" ] && log "    - $file"
    done
else
    log "  ${GREEN}OK${NC} - No MIT license references found"
    CATEGORY_COUNTS["license_refs"]=0
fi
log ""

# ============================================================================
# CATEGORY 7: CI/CD Workflow Files
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[7/9] CI/CD WORKFLOW ANALYSIS${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

log "${YELLOW}Workflows using self-hosted runners with PR triggers:${NC}"
WORKFLOW_DIR=".github/workflows"
VULNERABLE_WORKFLOWS=0

if [ -d "$WORKFLOW_DIR" ]; then
    shopt -s nullglob
    for workflow in "$WORKFLOW_DIR"/*.yml "$WORKFLOW_DIR"/*.yaml; do
        [ -f "$workflow" ] || continue

        # Check if workflow has pull_request trigger AND self-hosted runner
        if grep -q "pull_request" "$workflow" 2>/dev/null; then
            if grep -q "self-hosted" "$workflow" 2>/dev/null; then
                log "  ${RED}VULNERABLE:${NC} $(basename "$workflow")"
                log "    - Has pull_request trigger"
                log "    - Uses self-hosted runner"
                VULNERABLE_WORKFLOWS=$((VULNERABLE_WORKFLOWS + 1))
            fi
        fi
    done
    shopt -u nullglob

    if [ "$VULNERABLE_WORKFLOWS" -eq 0 ]; then
        log "  ${GREEN}OK${NC} - No vulnerable workflow configurations found"
    fi
    CATEGORY_COUNTS["workflows"]=$VULNERABLE_WORKFLOWS
else
    log "  ${YELLOW}WARNING:${NC} No .github/workflows directory found"
    CATEGORY_COUNTS["workflows"]=0
fi
log ""

log "${YELLOW}Workflows with dangerous permissions for PRs:${NC}"
PERMISSION_ISSUES=0

if [ -d "$WORKFLOW_DIR" ]; then
    shopt -s nullglob
    for workflow in "$WORKFLOW_DIR"/*.yml "$WORKFLOW_DIR"/*.yaml; do
        [ -f "$workflow" ] || continue

        if grep -q "pull_request" "$workflow" 2>/dev/null; then
            issues=""
            if grep -q "packages: write" "$workflow" 2>/dev/null; then
                issues="$issues packages:write"
            fi
            if grep -q "id-token: write" "$workflow" 2>/dev/null; then
                issues="$issues id-token:write"
            fi
            if [ -n "$issues" ]; then
                log "  ${RED}RISKY:${NC} $(basename "$workflow") has$issues"
                PERMISSION_ISSUES=$((PERMISSION_ISSUES + 1))
            fi
        fi
    done
    shopt -u nullglob

    if [ "$PERMISSION_ISSUES" -eq 0 ]; then
        log "  ${GREEN}OK${NC} - No dangerous permission configurations found"
    fi
else
    log "  ${YELLOW}WARNING:${NC} No .github/workflows directory found"
fi
log ""

# ============================================================================
# CATEGORY 8: Sensitive Data Check
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[8/9] SENSITIVE DATA CHECK${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

log "${YELLOW}Checking for potential secrets or sensitive data:${NC}"
SENSITIVE_ISSUES=0

# Check for .env files (should not be committed)
ENV_FILES=$(find . -name ".env" -o -name ".env.local" -o -name ".env.production" 2>/dev/null | grep -v node_modules || true)
if [ -n "$ENV_FILES" ]; then
    log "  ${RED}WARNING:${NC} Found .env files that may contain secrets:"
    echo "$ENV_FILES" | while read -r file; do
        [ -n "$file" ] && log "    - $file"
    done
    SENSITIVE_ISSUES=$((SENSITIVE_ISSUES + 1))
else
    log "  ${GREEN}OK${NC} - No .env files found"
fi
log ""

# Check for hardcoded API keys (basic patterns)
log "${YELLOW}Checking for potential hardcoded credentials:${NC}"
CREDENTIAL_PATTERNS="api_key.*=.*['\"][a-zA-Z0-9]{20,}|password.*=.*['\"][^'\"]{8,}|secret.*=.*['\"][a-zA-Z0-9]{20,}|token.*=.*['\"][a-zA-Z0-9]{20,}"

# Only check non-test, non-example files
CREDENTIAL_FILES=$(grep -rliE "$CREDENTIAL_PATTERNS" --include="*.go" --include="*.ts" --include="*.yml" --include="*.yaml" . 2>/dev/null | grep -v "_test.go" | grep -v "example" | grep -v "testdata" | grep -v "node_modules" | sort || true)

if [ -n "$CREDENTIAL_FILES" ]; then
    CRED_COUNT=$(echo "$CREDENTIAL_FILES" | grep -c "." || echo 0)
    log "  ${YELLOW}REVIEW:${NC} Found $CRED_COUNT files with potential credentials (may be false positives)"
    log "  Files to review:"
    echo "$CREDENTIAL_FILES" | head -10 | while read -r file; do
        [ -n "$file" ] && log "    - $file"
    done
    if [ "$CRED_COUNT" -gt 10 ]; then
        log "    ... and $((CRED_COUNT - 10)) more"
    fi
    CATEGORY_COUNTS["sensitive"]=$CRED_COUNT
else
    log "  ${GREEN}OK${NC} - No obvious credential patterns found"
    CATEGORY_COUNTS["sensitive"]=0
fi
log ""

# Check for internal URLs that shouldn't be public
log "${YELLOW}Checking for internal/private URLs:${NC}"
INTERNAL_PATTERNS="192\.168\.|10\.\d+\.|172\.(1[6-9]|2[0-9]|3[01])\.|localhost:[0-9]{4,5}|\.internal\.|\.local[:/]"
INTERNAL_FILES=$(grep -rlE "$INTERNAL_PATTERNS" --include="*.go" --include="*.md" --include="*.yml" --include="*.yaml" . 2>/dev/null | grep -v "_test" | grep -v "testdata" | grep -v "node_modules" | sort || true)

if [ -n "$INTERNAL_FILES" ]; then
    INT_COUNT=$(echo "$INTERNAL_FILES" | grep -c "." || echo 0)
    log "  ${YELLOW}REVIEW:${NC} Found $INT_COUNT files with internal URLs (review before public release)"
    if [ "$VERBOSE" = true ]; then
        echo "$INTERNAL_FILES" | while read -r file; do
            [ -n "$file" ] && log "    - $file"
        done
    else
        log "  First 5 files:"
        echo "$INTERNAL_FILES" | head -5 | while read -r file; do
            [ -n "$file" ] && log "    - $file"
        done
    fi
else
    log "  ${GREEN}OK${NC} - No internal URLs found"
fi
log ""

# ============================================================================
# CATEGORY 9: Additional Migration Checks
# ============================================================================
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log "${BLUE}[9/9] ADDITIONAL MIGRATION CHECKS${NC}"
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

# Check package.json for old references
log "${YELLOW}Package.json repository references:${NC}"
if [ -f "package.json" ]; then
    if grep -q "$OLD_REPO" package.json 2>/dev/null; then
        log "  ${RED}NEEDS UPDATE:${NC} package.json contains old repo URL"
        CATEGORY_COUNTS["package_json"]=1
    else
        log "  ${GREEN}OK${NC} - package.json has no old references"
        CATEGORY_COUNTS["package_json"]=0
    fi
elif [ -f "web/package.json" ]; then
    if grep -q "$OLD_REPO" web/package.json 2>/dev/null; then
        log "  ${RED}NEEDS UPDATE:${NC} web/package.json contains old repo URL"
        CATEGORY_COUNTS["package_json"]=1
    else
        log "  ${GREEN}OK${NC} - web/package.json has no old references"
        CATEGORY_COUNTS["package_json"]=0
    fi
else
    log "  ${YELLOW}INFO:${NC} No package.json found"
    CATEGORY_COUNTS["package_json"]=0
fi
log ""

# Check Makefile references
log "${YELLOW}Makefile repository references:${NC}"
if [ -f "Makefile" ]; then
    if grep -q "$OLD_REPO\|tomtom215/map" Makefile 2>/dev/null; then
        log "  ${RED}NEEDS UPDATE:${NC} Makefile contains old repo references"
        CATEGORY_COUNTS["makefile"]=1
    else
        log "  ${GREEN}OK${NC} - Makefile has no old references"
        CATEGORY_COUNTS["makefile"]=0
    fi
else
    log "  ${YELLOW}INFO:${NC} No Makefile found"
    CATEGORY_COUNTS["makefile"]=0
fi
log ""

# Check GoReleaser config
log "${YELLOW}GoReleaser configuration:${NC}"
GORELEASER_FILES=$(find . -maxdepth 2 -name ".goreleaser*.yml" -o -name ".goreleaser*.yaml" 2>/dev/null | head -5)
if [ -n "$GORELEASER_FILES" ]; then
    GORELEASER_ISSUES=0
    while IFS= read -r file; do
        [ -z "$file" ] && continue
        if grep -q "$OLD_REPO\|tomtom215/map" "$file" 2>/dev/null; then
            log "  ${RED}NEEDS UPDATE:${NC} $file contains old references"
            GORELEASER_ISSUES=$((GORELEASER_ISSUES + 1))
        fi
    done <<< "$GORELEASER_FILES"
    if [ "$GORELEASER_ISSUES" -eq 0 ]; then
        log "  ${GREEN}OK${NC} - GoReleaser config has no old references"
    fi
    CATEGORY_COUNTS["goreleaser"]=$GORELEASER_ISSUES
else
    log "  ${YELLOW}INFO:${NC} No GoReleaser config found"
    CATEGORY_COUNTS["goreleaser"]=0
fi
log ""

# Check GitHub templates (issue/PR templates, CODEOWNERS)
log "${YELLOW}GitHub repository configuration:${NC}"
GITHUB_CONFIG_ISSUES=0
if [ -d ".github" ]; then
    # Check CODEOWNERS
    if [ -f ".github/CODEOWNERS" ] || [ -f "CODEOWNERS" ]; then
        log "  ${GREEN}OK${NC} - CODEOWNERS file exists"
    else
        log "  ${YELLOW}MISSING:${NC} CODEOWNERS file (recommended for public repos)"
        GITHUB_CONFIG_ISSUES=$((GITHUB_CONFIG_ISSUES + 1))
    fi

    # Check issue templates
    if [ -d ".github/ISSUE_TEMPLATE" ] || [ -f ".github/ISSUE_TEMPLATE.md" ]; then
        log "  ${GREEN}OK${NC} - Issue templates exist"
    else
        log "  ${YELLOW}MISSING:${NC} Issue templates (recommended for public repos)"
        GITHUB_CONFIG_ISSUES=$((GITHUB_CONFIG_ISSUES + 1))
    fi

    # Check PR template
    if [ -f ".github/PULL_REQUEST_TEMPLATE.md" ] || [ -f ".github/pull_request_template.md" ]; then
        log "  ${GREEN}OK${NC} - PR template exists"
    else
        log "  ${YELLOW}MISSING:${NC} PR template (recommended for public repos)"
        GITHUB_CONFIG_ISSUES=$((GITHUB_CONFIG_ISSUES + 1))
    fi

    # Check SECURITY.md
    if [ -f "SECURITY.md" ] || [ -f ".github/SECURITY.md" ]; then
        log "  ${GREEN}OK${NC} - SECURITY.md exists"
    else
        log "  ${YELLOW}MISSING:${NC} SECURITY.md (required for public repos)"
        GITHUB_CONFIG_ISSUES=$((GITHUB_CONFIG_ISSUES + 1))
    fi

    # Check CONTRIBUTING.md
    if [ -f "CONTRIBUTING.md" ]; then
        log "  ${GREEN}OK${NC} - CONTRIBUTING.md exists"
    else
        log "  ${YELLOW}MISSING:${NC} CONTRIBUTING.md (recommended for public repos)"
        GITHUB_CONFIG_ISSUES=$((GITHUB_CONFIG_ISSUES + 1))
    fi
else
    log "  ${YELLOW}WARNING:${NC} No .github directory found"
    GITHUB_CONFIG_ISSUES=5
fi
CATEGORY_COUNTS["github_config"]=$GITHUB_CONFIG_ISSUES
log ""

# Check for large files that might slow cloning
log "${YELLOW}Large files check (>5MB, may slow clones):${NC}"
LARGE_FILES=$(find . -type f -size +5M -not -path "./.git/*" -not -path "./node_modules/*" -not -path "./vendor/*" 2>/dev/null | head -20)
if [ -n "$LARGE_FILES" ]; then
    LARGE_COUNT=$(echo "$LARGE_FILES" | wc -l)
    log "  ${YELLOW}REVIEW:${NC} Found $LARGE_COUNT large files"
    echo "$LARGE_FILES" | while read -r file; do
        [ -z "$file" ] && continue
        SIZE=$(du -h "$file" 2>/dev/null | cut -f1)
        log "    - $file ($SIZE)"
    done
    log "  Consider using Git LFS for large binary files"
    CATEGORY_COUNTS["large_files"]=$LARGE_COUNT
else
    log "  ${GREEN}OK${NC} - No large files found"
    CATEGORY_COUNTS["large_files"]=0
fi
log ""

# Check for binary files that shouldn't be committed
log "${YELLOW}Binary/compiled files check:${NC}"
BINARY_FILES=$(find . -type f \( -name "*.exe" -o -name "*.dll" -o -name "*.so" -o -name "*.dylib" -o -name "*.a" -o -name "*.o" -o -name "*.pyc" -o -name "*.class" \) -not -path "./.git/*" -not -path "./node_modules/*" -not -path "./vendor/*" 2>/dev/null | head -20)
# Also check for the main binary if it exists
if [ -f "cartographus" ] && file cartographus 2>/dev/null | grep -q "ELF\|executable"; then
    BINARY_FILES="$BINARY_FILES
./cartographus"
fi
if [ -n "$BINARY_FILES" ] && [ "$BINARY_FILES" != "" ]; then
    BINARY_COUNT=$(echo "$BINARY_FILES" | grep -c "." || echo 0)
    if [ "$BINARY_COUNT" -gt 0 ]; then
        log "  ${YELLOW}REVIEW:${NC} Found $BINARY_COUNT binary files (should not be committed)"
        echo "$BINARY_FILES" | head -10 | while read -r file; do
            [ -n "$file" ] && log "    - $file"
        done
        log "  Add these to .gitignore before public release"
        CATEGORY_COUNTS["binary_files"]=$BINARY_COUNT
    else
        log "  ${GREEN}OK${NC} - No unexpected binary files found"
        CATEGORY_COUNTS["binary_files"]=0
    fi
else
    log "  ${GREEN}OK${NC} - No unexpected binary files found"
    CATEGORY_COUNTS["binary_files"]=0
fi
log ""

# Check Swagger/OpenAPI files for old references
log "${YELLOW}Swagger/OpenAPI documentation:${NC}"
SWAGGER_FILES=$(find . -name "swagger*.json" -o -name "swagger*.yml" -o -name "swagger*.yaml" -o -name "openapi*.json" -o -name "openapi*.yml" -o -name "openapi*.yaml" 2>/dev/null | grep -v node_modules | head -10)
if [ -n "$SWAGGER_FILES" ]; then
    SWAGGER_ISSUES=0
    while IFS= read -r file; do
        [ -z "$file" ] && continue
        if grep -q "$OLD_REPO\|tomtom215/map" "$file" 2>/dev/null; then
            log "  ${RED}NEEDS UPDATE:${NC} $file contains old references"
            SWAGGER_ISSUES=$((SWAGGER_ISSUES + 1))
        else
            log "  ${GREEN}OK:${NC} $file"
        fi
    done <<< "$SWAGGER_FILES"
    CATEGORY_COUNTS["swagger"]=$SWAGGER_ISSUES
else
    log "  ${YELLOW}INFO:${NC} No Swagger/OpenAPI files found (may need regeneration)"
    CATEGORY_COUNTS["swagger"]=0
fi
log ""

# Check for license headers in source files
log "${YELLOW}License headers in source files:${NC}"
# Sample check - look at first 10 Go files
GO_FILES_SAMPLE=$(find . -name "*.go" -not -path "./vendor/*" -not -name "*_test.go" 2>/dev/null | head -10)
FILES_WITH_HEADER=0
FILES_WITHOUT_HEADER=0
if [ -n "$GO_FILES_SAMPLE" ]; then
    while IFS= read -r file; do
        [ -z "$file" ] && continue
        # Check first 10 lines for license/copyright comment
        if head -10 "$file" 2>/dev/null | grep -qiE "copyright|license|spdx"; then
            FILES_WITH_HEADER=$((FILES_WITH_HEADER + 1))
        else
            FILES_WITHOUT_HEADER=$((FILES_WITHOUT_HEADER + 1))
        fi
    done <<< "$GO_FILES_SAMPLE"

    if [ "$FILES_WITHOUT_HEADER" -gt "$FILES_WITH_HEADER" ]; then
        log "  ${YELLOW}INFO:${NC} Most Go files lack license headers (optional but recommended)"
        log "  Consider adding SPDX headers: // SPDX-License-Identifier: AGPL-3.0-or-later"
    else
        log "  ${GREEN}OK${NC} - License headers present in sampled files"
    fi
else
    log "  ${YELLOW}INFO:${NC} No Go files found to check"
fi
CATEGORY_COUNTS["license_headers"]=0
log ""

# Check CHANGELOG.md
log "${YELLOW}CHANGELOG.md status:${NC}"
if [ -f "CHANGELOG.md" ]; then
    if grep -q "$OLD_REPO\|tomtom215/map" CHANGELOG.md 2>/dev/null; then
        log "  ${RED}NEEDS UPDATE:${NC} CHANGELOG.md contains old repo references"
        CATEGORY_COUNTS["changelog"]=1
    else
        log "  ${GREEN}OK${NC} - CHANGELOG.md has no old references"
        CATEGORY_COUNTS["changelog"]=0
    fi
else
    log "  ${YELLOW}MISSING:${NC} CHANGELOG.md (recommended for public repos)"
    CATEGORY_COUNTS["changelog"]=1
fi
log ""

# ============================================================================
# SUMMARY
# ============================================================================
log ""
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log "${BOLD}                         SUMMARY${NC}"
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log ""

# ============================================================================
# DANGEROUS vs SAFE PATTERN SUMMARY (Prominent Warning)
# ============================================================================
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log "${RED}${BOLD}             ⚠️  REPLACEMENT SAFETY ANALYSIS ⚠️${NC}"
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log ""

TOTAL_DANGEROUS_SUM=$((${CATEGORY_COUNTS["dangerous_go_map"]:-0} + ${CATEGORY_COUNTS["dangerous_js_map"]:-0} + ${CATEGORY_COUNTS["dangerous_map_vars"]:-0} + ${CATEGORY_COUNTS["dangerous_maplibre"]:-0}))
TOTAL_SAFE_SUM=$((${CATEGORY_COUNTS["safe_github_url"]:-0} + ${CATEGORY_COUNTS["safe_ghcr"]:-0} + ${CATEGORY_COUNTS["safe_duckdb"]:-0}))

log "${RED}  DANGEROUS patterns (DO NOT replace):   ${BOLD}${TOTAL_DANGEROUS_SUM}+${NC}"
log "    - Go map[] types:                    ${CATEGORY_COUNTS["dangerous_go_map"]:-0}"
log "    - JS/TS .map() methods:              ${CATEGORY_COUNTS["dangerous_js_map"]:-0}"
log "    - *map* variable names:              ${CATEGORY_COUNTS["dangerous_map_vars"]:-0}"
log "    - MapLibre/Mapbox refs:              ${CATEGORY_COUNTS["dangerous_maplibre"]:-0}"
log "    - Files with 'map' in name:          ${CATEGORY_COUNTS["dangerous_map_files"]:-0}"
log ""
log "${GREEN}  SAFE patterns (OK to replace):         ${BOLD}${TOTAL_SAFE_SUM}${NC}"
log "    - github.com/tomtom215/cartographus:          ${CATEGORY_COUNTS["safe_github_url"]:-0}"
log "    - ghcr.io/tomtom215/map:             ${CATEGORY_COUNTS["safe_ghcr"]:-0}"
log "    - map.duckdb:                        ${CATEGORY_COUNTS["safe_duckdb"]:-0}"
log ""

if [ "$TOTAL_DANGEROUS_SUM" -gt 0 ] && [ "$TOTAL_SAFE_SUM" -gt 0 ]; then
    RATIO=$((TOTAL_DANGEROUS_SUM / TOTAL_SAFE_SUM))
    log "${RED}${BOLD}  ⚠️  RATIO: ${RATIO}x more dangerous patterns than safe ones!${NC}"
    log ""
fi

log "${YELLOW}  ╔════════════════════════════════════════════════════════════╗${NC}"
log "${YELLOW}  ║  NEVER run: sed 's/map/cartographus/g'                     ║${NC}"
log "${YELLOW}  ║  ONLY use safe commands from docs/MIGRATION_PLAN.md       ║${NC}"
log "${YELLOW}  ╚════════════════════════════════════════════════════════════╝${NC}"
log ""

log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""

# Calculate totals
TOTAL_FILES=0
CRITICAL_ISSUES=0

log "${BOLD}Files requiring changes (SAFE patterns only):${NC}"
log ""

if [ "${CATEGORY_COUNTS["go_mod"]:-0}" -gt 0 ]; then
    log "  ${RED}go.mod:${NC}                    ${CATEGORY_COUNTS["go_mod"]} file"
    TOTAL_FILES=$((TOTAL_FILES + CATEGORY_COUNTS["go_mod"]))
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + 1))
fi

if [ "${CATEGORY_COUNTS["go_imports"]:-0}" -gt 0 ]; then
    log "  ${RED}Go imports:${NC}                ${CATEGORY_COUNTS["go_imports"]} files"
    TOTAL_FILES=$((TOTAL_FILES + CATEGORY_COUNTS["go_imports"]))
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + 1))
fi

if [ "${CATEGORY_COUNTS["docker"]:-0}" -gt 0 ]; then
    log "  ${RED}Docker images:${NC}             ${CATEGORY_COUNTS["docker"]} files"
    TOTAL_FILES=$((TOTAL_FILES + CATEGORY_COUNTS["docker"]))
fi

if [ "${CATEGORY_COUNTS["github_urls"]:-0}" -gt 0 ]; then
    log "  ${RED}GitHub URLs:${NC}               ${CATEGORY_COUNTS["github_urls"]} files"
    TOTAL_FILES=$((TOTAL_FILES + CATEGORY_COUNTS["github_urls"]))
fi

if [ "${CATEGORY_COUNTS["database"]:-0}" -gt 0 ]; then
    log "  ${RED}Database paths:${NC}            ${CATEGORY_COUNTS["database"]} files"
    TOTAL_FILES=$((TOTAL_FILES + CATEGORY_COUNTS["database"]))
fi

if [ "${CATEGORY_COUNTS["license_file"]:-0}" -gt 0 ]; then
    log "  ${RED}LICENSE file:${NC}              Needs replacement"
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + 1))
fi

if [ "${CATEGORY_COUNTS["license_refs"]:-0}" -gt 0 ]; then
    log "  ${RED}License references:${NC}        ${CATEGORY_COUNTS["license_refs"]} files"
    TOTAL_FILES=$((TOTAL_FILES + CATEGORY_COUNTS["license_refs"]))
fi

if [ "${CATEGORY_COUNTS["workflows"]:-0}" -gt 0 ]; then
    log "  ${RED}Vulnerable workflows:${NC}      ${CATEGORY_COUNTS["workflows"]} files"
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + CATEGORY_COUNTS["workflows"]))
fi

log ""
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""
log "  ${BOLD}Total files to modify:${NC}       ~$TOTAL_FILES"
log "  ${BOLD}Critical issues:${NC}             $CRITICAL_ISSUES"
log ""

if [ "$CRITICAL_ISSUES" -gt 0 ]; then
    log "${RED}${BOLD}ACTION REQUIRED:${NC} There are $CRITICAL_ISSUES critical issues to address"
else
    log "${GREEN}${BOLD}READY:${NC} No critical issues found"
fi

log ""
log "${BOLD}───────────────────────────────────────────────────────────────${NC}"
log ""
log "${BOLD}Next steps:${NC}"
log "  1. Review this report carefully"
log "  2. Run security scan: gitleaks detect --source . --verbose"
log "  3. Address any sensitive data issues"
log "  4. Update CI/CD workflows for hybrid runner architecture"
log "  5. When ready, run the migration script: ./scripts/migration-script.sh"
log ""

if [ -n "$OUTPUT_FILE" ]; then
    log "Report saved to: $OUTPUT_FILE"
fi

log ""
log "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
log ""

# Generate JSON output if requested
if [ "$JSON_OUTPUT" = true ]; then
    JSON_FILE="${OUTPUT_FILE:-migration-dry-run.json}"
    if [ "$JSON_FILE" = "$OUTPUT_FILE" ] && [ -n "$OUTPUT_FILE" ]; then
        JSON_FILE="${OUTPUT_FILE%.txt}.json"
    fi

    cat > "$JSON_FILE" << JSONEOF
{
  "timestamp": "$(date -Iseconds)",
  "git": {
    "branch": "${GIT_BRANCH:-unknown}",
    "commit": "${GIT_COMMIT:-unknown}",
    "dirty_files": ${GIT_DIRTY:-0}
  },
  "migration": {
    "old_repo": "$OLD_REPO",
    "new_repo": "$NEW_REPO",
    "old_image": "$OLD_IMAGE",
    "old_db": "$OLD_DB"
  },
  "dangerous_patterns": {
    "go_map_types": ${CATEGORY_COUNTS["dangerous_go_map"]:-0},
    "js_map_methods": ${CATEGORY_COUNTS["dangerous_js_map"]:-0},
    "map_variable_names": ${CATEGORY_COUNTS["dangerous_map_vars"]:-0},
    "maplibre_mapbox_refs": ${CATEGORY_COUNTS["dangerous_maplibre"]:-0},
    "files_with_map_name": ${CATEGORY_COUNTS["dangerous_map_files"]:-0},
    "total_dangerous": ${TOTAL_DANGEROUS_SUM:-0},
    "warning": "NEVER replace these patterns - they are NOT project references"
  },
  "safe_patterns": {
    "github_url": ${CATEGORY_COUNTS["safe_github_url"]:-0},
    "ghcr_image": ${CATEGORY_COUNTS["safe_ghcr"]:-0},
    "duckdb_path": ${CATEGORY_COUNTS["safe_duckdb"]:-0},
    "total_safe": ${TOTAL_SAFE_SUM:-0}
  },
  "categories": {
    "go_mod": ${CATEGORY_COUNTS["go_mod"]:-0},
    "go_imports": ${CATEGORY_COUNTS["go_imports"]:-0},
    "docker": ${CATEGORY_COUNTS["docker"]:-0},
    "github_urls": ${CATEGORY_COUNTS["github_urls"]:-0},
    "database": ${CATEGORY_COUNTS["database"]:-0},
    "license_file": ${CATEGORY_COUNTS["license_file"]:-0},
    "license_refs": ${CATEGORY_COUNTS["license_refs"]:-0},
    "workflows": ${CATEGORY_COUNTS["workflows"]:-0},
    "sensitive": ${CATEGORY_COUNTS["sensitive"]:-0},
    "package_json": ${CATEGORY_COUNTS["package_json"]:-0},
    "makefile": ${CATEGORY_COUNTS["makefile"]:-0},
    "goreleaser": ${CATEGORY_COUNTS["goreleaser"]:-0},
    "github_config": ${CATEGORY_COUNTS["github_config"]:-0},
    "large_files": ${CATEGORY_COUNTS["large_files"]:-0},
    "binary_files": ${CATEGORY_COUNTS["binary_files"]:-0},
    "swagger": ${CATEGORY_COUNTS["swagger"]:-0},
    "changelog": ${CATEGORY_COUNTS["changelog"]:-0}
  },
  "summary": {
    "total_files": $TOTAL_FILES,
    "critical_issues": $CRITICAL_ISSUES,
    "dangerous_to_safe_ratio": "$((${TOTAL_DANGEROUS_SUM:-0} / (${TOTAL_SAFE_SUM:-1}))):1",
    "ready_for_migration": $([ "$CRITICAL_ISSUES" -eq 0 ] && echo "true" || echo "false")
  }
}
JSONEOF

    echo "JSON report saved to: $JSON_FILE"
fi

# Exit with error code if critical issues found
if [ "$CRITICAL_ISSUES" -gt 0 ]; then
    exit 1
fi
exit 0
