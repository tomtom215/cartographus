# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# Multi-stage Dockerfile for Cartographus
# Optimized with cross-compilation for fast ARM64 builds (3-5min vs 15min)
# Supports linux/amd64, linux/arm64
# Note: linux/arm/v7 is NOT supported due to DuckDB lacking pre-built armv7 binaries
#
# Security: Uses Debian 13 (Trixie) base images to address:
# - CVE-2023-2953 (libldap): Fixed in OpenLDAP 2.6.10+dfsg-1
# - CVE-2025-6020 (linux-pam): Fixed in linux-pam 1.7.0-5
# - CVE-2023-45853 (zlib): False positive - MiniZip not built in Debian binaries

# Stage 1: Build frontend (fast with caching, consolidated verification)
# Supports two modes:
# 1. Standard build (default): Builds frontend inside container
# 2. CI mode (USE_PREBUILT_FRONTEND=true): Uses pre-built artifacts from host
FROM node:25-trixie-slim AS frontend-builder

ARG USE_PREBUILT_FRONTEND=false

WORKDIR /build/web

# Install librsvg for icon generation (only needed for standard build)
RUN if [ "$USE_PREBUILT_FRONTEND" != "true" ]; then \
    apt-get update && apt-get install -y --no-install-recommends \
        librsvg2-bin \
        && rm -rf /var/lib/apt/lists/*; \
    fi

# Standard build path: Build frontend from source
RUN if [ "$USE_PREBUILT_FRONTEND" != "true" ]; then \
    echo "=== Building frontend from source ==="; \
    fi

COPY web/package.json web/package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm \
    if [ "$USE_PREBUILT_FRONTEND" != "true" ]; then \
        npm ci --production=false; \
    fi

COPY web/ ./

# Generate PNG icons from SVG source (only for standard build)
RUN if [ "$USE_PREBUILT_FRONTEND" != "true" ]; then \
    cd public && \
    rsvg-convert -w 192 -h 192 icon.svg -o icon-192.png && \
    rsvg-convert -w 512 -h 512 icon.svg -o icon-512.png && \
    echo "Generated PNG icons from SVG source" && \
    ls -lh icon-*.png && \
    cd ..; \
    fi

# Build frontend with verification (only for standard build)
RUN if [ "$USE_PREBUILT_FRONTEND" != "true" ]; then \
    echo "=== Building TypeScript bundle ===" && \
    npm run build && \
    echo "✅ TypeScript build completed"; \
    fi

# Consolidated verification (fewer layers, faster builds, same safety)
# Order: verify esbuild output -> copy public assets -> verify all files
RUN if [ "$USE_PREBUILT_FRONTEND" != "true" ]; then \
    echo "=== Verifying build output ===" && \
    test -f dist/index.js || (echo "❌ dist/index.js not created" && exit 1) && \
    test -d dist/chunks || (echo "❌ dist/chunks not created" && exit 1) && \
    cp -rv public/* dist/ && \
    test -f dist/styles.css && test -f dist/index.html && \
    test -f dist/manifest.json && test -f dist/service-worker.js && \
    INDEX_SIZE=$(stat -c%s dist/index.js 2>/dev/null || stat -f%z dist/index.js) && \
    STYLES_SIZE=$(stat -c%s dist/styles.css 2>/dev/null || stat -f%z dist/styles.css) && \
    HTML_SIZE=$(stat -c%s dist/index.html 2>/dev/null || stat -f%z dist/index.html) && \
    test "$STYLES_SIZE" -gt 10000 && test "$HTML_SIZE" -gt 10000 && test "$INDEX_SIZE" -gt 1000 && \
    echo "✅ Frontend build verified: index.js (${INDEX_SIZE}B), styles.css (${STYLES_SIZE}B)"; \
    else \
    echo "=== Using pre-built frontend artifacts ===" && \
    test -d dist || (echo "❌ Pre-built dist/ directory not found" && exit 1) && \
    test -f dist/index.js || (echo "❌ Pre-built dist/index.js not found" && exit 1) && \
    test -d dist/chunks || (echo "❌ Pre-built dist/chunks not found" && exit 1) && \
    echo "✅ Pre-built esbuild artifacts verified" && \
    echo "=== Copying public assets to dist/ ===" && \
    cp -rv public/* dist/ && \
    test -f dist/styles.css && test -f dist/index.html && \
    test -f dist/manifest.json && test -f dist/service-worker.js && \
    echo "✅ Public assets copied and verified"; \
    fi

# Stage 2: Build backend with CROSS-COMPILATION (10-20x faster for ARM64)
# Uses build platform (AMD64) to compile for target platform (ARM64)
# This avoids slow QEMU emulation for Go compilation
FROM --platform=$BUILDPLATFORM golang:1.25.5-trixie AS backend-builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Install cross-compilation toolchains for ARM64
# This allows AMD64 runners to build ARM64 binaries without QEMU emulation
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    build-essential \
    gcc-aarch64-linux-gnu \
    g++-aarch64-linux-gnu \
    file \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (arch-agnostic, cached globally)
# id= ensures cache persists across GitHub Actions runs with type=gha
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go mod download && \
    go mod verify && \
    echo "✅ Go modules downloaded"

# Copy all source code
COPY . .

# Ensure go.mod is synchronized with source code
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod tidy && \
    echo "✅ Go modules synchronized"

# Cross-compile binary for target architecture
# CRITICAL OPTIMIZATION: This runs on AMD64 (fast) even when building ARM64
# Uses architecture-specific build cache to avoid cross-arch cache poisoning
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild-${TARGETARCH},target=/root/.cache/go-build \
    set -ex; \
    echo "🔨 Cross-compiling for ${TARGETOS}/${TARGETARCH} on ${BUILDPLATFORM}"; \
    \
    # Set cross-compiler for ARM64
    if [ "$TARGETARCH" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc; \
        export CXX=aarch64-linux-gnu-g++; \
        echo "Using ARM64 cross-compiler: $CC"; \
    fi; \
    \
    # Build with parallelization and architecture-specific settings
    # Tags: wal (BadgerDB WAL for event durability), nats (event-driven architecture)
    CGO_ENABLED=1 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -tags "wal,nats" \
    -ldflags="-w -s" \
    -p=$(nproc) \
    -o cartographus \
    ./cmd/server && \
    \
    # Verify binary architecture matches target
    file cartographus && \
    echo "✅ Binary built: $(ls -lh cartographus | awk '{print $5}') for ${TARGETARCH}"

# Stage 3: Final runtime image
FROM debian:trixie-slim

# DuckDB version - MUST match go.mod duckdb-go-bindings version
# Single source of truth: scripts/duckdb-version.sh
ARG DUCKDB_VERSION=v1.4.3

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 1000 cartographus \
    && useradd -u 1000 -g cartographus -s /bin/sh -m cartographus

# Pre-install DuckDB extensions to avoid runtime downloads
# This ensures the app starts reliably without network dependencies
# Extensions are installed to /home/cartographus/.duckdb/extensions/ (user home)
ARG TARGETARCH
RUN set -ex; \
    PLATFORM="linux_amd64"; \
    if [ "$TARGETARCH" = "arm64" ]; then PLATFORM="linux_arm64"; fi; \
    EXTENSIONS_DIR="/home/cartographus/.duckdb/extensions/${DUCKDB_VERSION}/${PLATFORM}"; \
    mkdir -p "$EXTENSIONS_DIR"; \
    MAIN_REPO="https://extensions.duckdb.org/${DUCKDB_VERSION}/${PLATFORM}"; \
    COMMUNITY_REPO="https://community-extensions.duckdb.org/${DUCKDB_VERSION}/${PLATFORM}"; \
    echo "=== Pre-installing DuckDB extensions (${PLATFORM}) ==="; \
    # Core extensions from main repository
    for ext in httpfs spatial inet icu json sqlite_scanner; do \
        echo "Downloading ${ext}..."; \
        curl -sSfL --retry 5 --retry-delay 3 --retry-all-errors \
            "${MAIN_REPO}/${ext}.duckdb_extension.gz" \
            -o "${EXTENSIONS_DIR}/${ext}.duckdb_extension.gz" && \
        gunzip -f "${EXTENSIONS_DIR}/${ext}.duckdb_extension.gz" && \
        echo "  OK: ${ext} ($(du -h "${EXTENSIONS_DIR}/${ext}.duckdb_extension" | cut -f1))"; \
    done; \
    # Community extensions (all required - installed in every build)
    for ext in h3 rapidfuzz datasketches; do \
        echo "Downloading ${ext} (community)..."; \
        curl -sSfL --retry 5 --retry-delay 3 --retry-all-errors \
            "${COMMUNITY_REPO}/${ext}.duckdb_extension.gz" \
            -o "${EXTENSIONS_DIR}/${ext}.duckdb_extension.gz" && \
        gunzip -f "${EXTENSIONS_DIR}/${ext}.duckdb_extension.gz" && \
        echo "  OK: ${ext} ($(du -h "${EXTENSIONS_DIR}/${ext}.duckdb_extension" | cut -f1))"; \
    done; \
    # Fix ownership for cartographus user
    chown -R cartographus:cartographus /home/cartographus/.duckdb; \
    echo "=== Extensions installed: $(du -sh "$EXTENSIONS_DIR" | cut -f1) total ==="; \
    ls -la "$EXTENSIONS_DIR/"

WORKDIR /app

COPY --from=backend-builder /build/cartographus /app/cartographus
COPY --from=backend-builder /build/internal/templates /app/internal/templates
COPY --from=frontend-builder /build/web/dist /app/web/dist

# Verify all files were copied to runtime image
RUN echo "=== Verifying runtime image files ===" && \
    test -f /app/cartographus || (echo "❌ ERROR: /app/cartographus binary not found" && exit 1) && \
    test -d /app/web/dist || (echo "❌ ERROR: /app/web/dist directory not found" && exit 1) && \
    test -f /app/web/dist/styles.css || (echo "❌ ERROR: /app/web/dist/styles.css not found in runtime image" && exit 1) && \
    test -f /app/web/dist/index.html || (echo "❌ ERROR: /app/web/dist/index.html not found in runtime image" && exit 1) && \
    test -f /app/web/dist/index.js || (echo "❌ ERROR: /app/web/dist/index.js not found in runtime image" && exit 1) && \
    test -d /app/web/dist/chunks || (echo "❌ ERROR: /app/web/dist/chunks directory not found in runtime image" && exit 1) && \
    test -f /app/web/dist/manifest.json || (echo "❌ ERROR: /app/web/dist/manifest.json not found in runtime image" && exit 1) && \
    test -f /app/internal/templates/index.html.tmpl || (echo "❌ ERROR: /app/internal/templates/index.html.tmpl not found in runtime image" && exit 1) && \
    echo "✅ All files verified in runtime image" && \
    echo "" && \
    echo "=== Runtime image web/dist contents ===" && \
    ls -lh /app/web/dist/ && \
    echo "" && \
    echo "=== Code-split chunks in runtime image ===" && \
    ls -lh /app/web/dist/chunks/ && \
    echo ""

RUN mkdir -p /data && \
    chown -R cartographus:cartographus /app /data

USER cartographus

EXPOSE 3857

VOLUME ["/data"]

ENV DUCKDB_PATH=/data/cartographus.duckdb \
    HTTP_PORT=3857 \
    HTTP_HOST=0.0.0.0 \
    LOG_LEVEL=info

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:3857/api/v1/health/ready || exit 1

ENTRYPOINT ["/app/cartographus"]
