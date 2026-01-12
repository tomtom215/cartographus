# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
.PHONY: help build build-frontend build-backend test test-e2e test-all lint clean docker-build docker-run install-deps swagger-gen swagger-serve release-snapshot release install-goreleaser build-binaries-local verify-templates sync-templates pre-commit setup setup-quick setup-verify setup-extensions

help:
	@echo "Cartographus - Available commands:"
	@echo ""
	@echo "Session Setup (Claude Code Web):"
	@echo "  make setup                - Full session setup (env + extensions + build)"
	@echo "  make setup-quick          - Quick setup (env vars only)"
	@echo "  make setup-verify         - Verify current setup"
	@echo "  make setup-extensions     - Install DuckDB extensions only"
	@echo ""
	@echo "Build Commands:
	@echo "  make build                - Build frontend and backend"
	@echo "  make build-frontend       - Build frontend only"
	@echo "  make build-backend        - Build backend only"
	@echo "  make build-binaries-local - Build all platform binaries locally (requires goreleaser)"
	@echo ""
	@echo "Test Commands:"
	@echo "  make test                 - Run unit tests"
	@echo "  make test-e2e             - Run E2E tests with Playwright"
	@echo "  make test-all             - Run all tests (unit + E2E)"
	@echo "  make lint                 - Run linters"
	@echo ""
	@echo "Release Commands:"
	@echo "  make release-snapshot     - Build snapshot release (pre-release testing)"
	@echo "  make release              - Build official release (requires git tag)"
	@echo "  make install-goreleaser   - Install goreleaser CLI"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-build         - Build Docker image"
	@echo "  make docker-run           - Run with Docker Compose"
	@echo "  make docker-logs          - View Docker logs"
	@echo "  make docker-stop          - Stop Docker containers"
	@echo ""
	@echo "Validation Commands:"
	@echo "  make verify-templates     - Check HTML templates are in sync"
	@echo "  make sync-templates       - Update production template from development"
	@echo "  make pre-commit           - Run all pre-commit checks"
	@echo ""
	@echo "Other Commands:"
	@echo "  make swagger-gen          - Generate OpenAPI/Swagger documentation"
	@echo "  make swagger-serve        - Serve Swagger UI (requires swagger-gen first)"
	@echo "  make clean                - Clean build artifacts"
	@echo "  make install-deps         - Install dependencies"

install-deps:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing frontend dependencies..."
	cd web && npm ci

build-frontend:
	@echo "Building frontend..."
	cd web && npm run build
	cp -r web/public/* web/dist/ 2>/dev/null || true

build-backend:
	@echo "Building backend with NATS and WAL support..."
	CGO_ENABLED=1 go build -tags "wal,nats" -o cartographus ./cmd/server

build: build-frontend build-backend
	@echo "Build complete!"

test:
	@echo "Running unit tests with NATS and WAL support..."
	go test -tags "wal,nats" -v -race ./...

test-e2e:
	@echo "Running E2E tests with Playwright..."
	cd web && npm install && npx playwright install --with-deps chromium
	npx playwright test

test-all: test test-e2e
	@echo "All tests complete!"

lint:
	@echo "Running linters..."
	go vet -tags "wal,nats" ./...
	gofmt -l .
	cd web && npx tsc --noEmit

clean:
	@echo "Cleaning build artifacts..."
	rm -f cartographus
	rm -rf web/dist
	rm -rf data/*.duckdb*
	rm -rf docs/

docker-build:
	@echo "Building Docker image..."
	docker build -t cartographus:latest .

docker-run:
	@echo "Starting with Docker Compose..."
	docker-compose up -d
	@echo "Access at http://localhost:3857"

docker-logs:
	docker-compose logs -f cartographus

docker-stop:
	docker-compose down

swagger-gen:
	@echo "Generating OpenAPI/Swagger documentation..."
	@command -v swag >/dev/null 2>&1 || { echo "Installing swag CLI..."; go install github.com/swaggo/swag/cmd/swag@latest; }
	swag init -g cmd/server/docs.go -o docs --parseInternal
	@echo "Swagger docs generated in docs/"
	@echo "View at http://localhost:3857/swagger/index.html (when server running)"

swagger-serve:
	@echo "Swagger docs available at http://localhost:3857/swagger/index.html"
	@echo "Start the server with 'make docker-run' or run the binary directly"

# Binary build and release targets

install-goreleaser:
	@echo "Checking for goreleaser..."
	@command -v goreleaser >/dev/null 2>&1 || { \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser/v2@latest; \
	}
	@goreleaser --version

release-snapshot: install-goreleaser build-frontend
	@echo "Building snapshot release for testing..."
	@echo "This creates local binaries without publishing to GitHub"
	goreleaser release --snapshot --clean --skip=announce
	@echo ""
	@echo "✅ Snapshot release built successfully!"
	@echo "📦 Binaries available in: dist/"
	@echo ""
	@echo "To test a binary:"
	@echo "  cd dist/cartographus_*_linux_amd64 && ./cartographus"

release: install-goreleaser build-frontend
	@echo "Building official release..."
	@if [ -z "$$(git describe --exact-match --tags 2>/dev/null)" ]; then \
		echo "❌ Error: No git tag found on current commit"; \
		echo "To create a release, first tag the commit:"; \
		echo "  git tag -a v1.0.0 -m 'Release v1.0.0'"; \
		echo "  git push origin v1.0.0"; \
		exit 1; \
	fi
	goreleaser release --clean
	@echo ""
	@echo "✅ Release complete! Artifacts published to GitHub"

build-binaries-local: install-goreleaser build-frontend
	@echo "Building binaries for all platforms..."
	@echo "Note: This requires cross-compilation toolchains"
	@echo "For native builds, use 'make release-snapshot' instead"
	goreleaser build --snapshot --clean
	@echo ""
	@echo "✅ Binaries built successfully!"
	@echo "📦 Available in: dist/"

# Template synchronization targets
# The production template (internal/templates/index.html.tmpl) must stay in sync
# with the development template (web/public/index.html) to prevent E2E test failures.

verify-templates:
	@echo "Verifying HTML templates are in sync..."
	@./scripts/sync-templates.sh --check

sync-templates:
	@echo "Syncing HTML templates..."
	@./scripts/sync-templates.sh --sync

# Pre-commit validation - run before every commit
# Ensures code quality and template synchronization

pre-commit: verify-templates
	@echo ""
	@echo "Running pre-commit checks..."
	@echo ""
	@echo "1. Formatting Go code..."
	@gofmt -s -w .
	@echo "2. Tidying Go modules..."
	@go mod tidy
	@echo "3. Building frontend..."
	@cd web && npm run build
	@echo "4. Running go vet..."
	@go vet -tags "wal,nats" ./...
	@echo "5. Running TypeScript check..."
	@cd web && npx tsc --noEmit
	@echo "6. Running unit tests..."
	@go test -tags "wal,nats" -race ./...
	@echo ""
	@echo "✅ All pre-commit checks passed!"
	@echo ""
	@echo "Ready to commit. Suggested workflow:"
	@echo "  git add ."
	@echo "  git diff --cached"
	@echo "  git commit -m 'type(scope): description'"

# =============================================================================
# Session Setup (Claude Code Web)
# =============================================================================
# These targets set up the development environment for Claude Code Web sessions.
# Run 'make setup' at the start of each session.

setup:
	@echo "Running full session setup..."
	@echo ""
	@echo "Setting environment variables..."
	@export GOTOOLCHAIN=local && \
	 export no_proxy="localhost,127.0.0.1" && \
	 export NO_PROXY="localhost,127.0.0.1" && \
	 export CGO_ENABLED=1 && \
	 echo "  GOTOOLCHAIN=local" && \
	 echo "  no_proxy=localhost,127.0.0.1" && \
	 echo "  CGO_ENABLED=1" && \
	 echo "" && \
	 echo "Installing DuckDB extensions..." && \
	 ./scripts/setup-duckdb-extensions.sh && \
	 echo "" && \
	 echo "Building binary..." && \
	 CGO_ENABLED=1 go build -tags "wal,nats" -o cartographus ./cmd/server && \
	 echo "" && \
	 echo "✅ Setup complete!" && \
	 ls -lh cartographus
	@echo ""
	@echo "IMPORTANT: Environment variables do not persist from make."
	@echo "Run this in your shell:"
	@echo "  export GOTOOLCHAIN=local"
	@echo "  export no_proxy=\"localhost,127.0.0.1\""
	@echo "  export NO_PROXY=\"localhost,127.0.0.1\""
	@echo ""
	@echo "Or use: source scripts/session-setup.sh"

setup-quick:
	@echo "Quick setup - printing environment commands..."
	@echo ""
	@echo "Run these commands in your shell:"
	@echo "  export GOTOOLCHAIN=local"
	@echo "  export no_proxy=\"localhost,127.0.0.1\""
	@echo "  export NO_PROXY=\"localhost,127.0.0.1\""
	@echo "  export CGO_ENABLED=1"
	@echo ""
	@echo "Or use: source scripts/session-setup.sh --quick"

setup-verify:
	@echo "Verifying session setup..."
	@./scripts/session-setup.sh --verify

setup-extensions:
	@echo "Installing DuckDB extensions..."
	@./scripts/setup-duckdb-extensions.sh
