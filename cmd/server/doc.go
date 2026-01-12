// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package main is the entry point for the Cartographus server application.

Cartographus is a self-hosted media server analytics platform that visualizes
playback activity from Plex, Jellyfin, and Emby on interactive maps. It provides
geographic visualization of streaming sessions, 47+ analytics charts across
6 themed dashboards, and real-time monitoring with security detection.

# Application Architecture

The server implements a layered architecture with Suture v4 process supervision:

	RootSupervisor ("cartographus")
	├── DataSupervisor ("data-layer")
	│   └── WAL services (optional, -tags wal)
	├── MessagingSupervisor ("messaging-layer")
	│   ├── WebSocket Hub (real-time updates)
	│   ├── Sync Manager (media server sync)
	│   └── NATS components (optional, -tags nats)
	└── APISupervisor ("api-layer")
	    └── HTTP Server (227 REST endpoints)

Component initialization order:

 1. Configuration: Koanf v2 with environment variables and config files
 2. Logging: zerolog with JSON/console output modes
 3. Database: DuckDB with spatial, H3, INET, and community extensions
 4. Authentication: JWT, Basic Auth, or no-auth mode
 5. Sync Manager: Multi-server (Plex, Jellyfin, Emby) + optional Tautulli
 6. WebSocket Hub: Real-time playback notifications
 7. Detection Engine: Security anomaly detection (5 rules)
 8. Backup Manager: Scheduled backups with retention policies
 9. Supervisor Tree: Suture v4 process supervision
 10. HTTP Server: Chi router with middleware stack

# Configuration

Configuration is loaded via Koanf v2 with layered sources (highest priority wins):

	Priority: Environment variables > Config file > Defaults

Core environment variables:

	# Server
	PORT=3857                    # HTTP server port (EPSG:3857 reference)
	LOG_LEVEL=info               # trace, debug, info, warn, error
	LOG_FORMAT=json              # json or console

	# Authentication (choose one mode)
	AUTH_MODE=jwt                # jwt, basic, or none
	JWT_SECRET=<32+ chars>       # Required for JWT mode
	ADMIN_USERNAME=admin
	ADMIN_PASSWORD=<password>

	# Media Servers (enable one or more)
	PLEX_ENABLED=true
	PLEX_URL=http://localhost:32400
	PLEX_TOKEN=<token>

	JELLYFIN_ENABLED=false
	JELLYFIN_URL=http://localhost:8096
	JELLYFIN_API_KEY=<api-key>

	EMBY_ENABLED=false
	EMBY_URL=http://localhost:8096
	EMBY_API_KEY=<api-key>

	# Optional Tautulli (for migration or enhanced metadata)
	TAUTULLI_ENABLED=false
	TAUTULLI_URL=http://localhost:8181
	TAUTULLI_API_KEY=<api-key>

See .env.example for complete configuration reference.

# Standalone Mode (v2.0+)

Cartographus can run WITHOUT Tautulli, connecting directly to media servers:

	# Plex direct connection
	export PLEX_ENABLED=true PLEX_URL=http://plex:32400 PLEX_TOKEN=xxx
	./cartographus

	# Jellyfin direct connection
	export JELLYFIN_ENABLED=true JELLYFIN_URL=http://jellyfin:8096 JELLYFIN_API_KEY=xxx
	./cartographus

	# Multiple servers simultaneously
	export PLEX_ENABLED=true JELLYFIN_ENABLED=true ...
	./cartographus

# Build Tags

Optional build tags enable additional functionality:

	go build ./cmd/server                    # Standard build
	go build -tags wal ./cmd/server          # Enable BadgerDB WAL durability
	go build -tags nats ./cmd/server         # Enable NATS JetStream events
	go build -tags "wal,nats" ./cmd/server   # Enable both

Build tags affect supervisor tree composition:
  - wal: Adds WALRetryLoopService and WALCompactorService to data layer
  - nats: Adds NATSComponentsService to messaging layer

# Signal Handling

The server handles graceful shutdown on SIGINT and SIGTERM:

 1. Stops accepting new HTTP connections
 2. Broadcasts shutdown to WebSocket clients
 3. Waits for in-flight requests (10s timeout)
 4. Stops sync manager and closes media server connections
 5. Flushes pending writes and closes database
 6. Reports any services that failed to stop

# Usage Examples

Development (no auth):

	export AUTH_MODE=none
	export PLEX_ENABLED=true PLEX_URL=http://localhost:32400 PLEX_TOKEN=xxx
	go run ./cmd/server

Production (JWT + Plex):

	export AUTH_MODE=jwt
	export JWT_SECRET=$(openssl rand -base64 32)
	export ADMIN_USERNAME=admin ADMIN_PASSWORD=secure-password
	export PLEX_ENABLED=true PLEX_URL=http://plex:32400 PLEX_TOKEN=xxx
	./cartographus

Docker:

	docker run -d \
	  -e PLEX_ENABLED=true \
	  -e PLEX_URL=http://plex:32400 \
	  -e PLEX_TOKEN=xxx \
	  -e AUTH_MODE=none \
	  -p 3857:3857 \
	  ghcr.io/tomtom215/cartographus

# Port 3857

The default port 3857 references EPSG:3857 (Web Mercator projection),
the coordinate system used by web mapping libraries like MapLibre GL JS.

# API Documentation

Swagger documentation is available at /swagger/index.html when the server
is running. The API provides 227 endpoints organized into categories:

  - Core: Health checks, server info, statistics
  - Analytics: 45+ analytics endpoints with filtering
  - Spatial: Geographic queries, clustering, H3 aggregation
  - Tautulli Proxy: Pass-through to Tautulli API (60+ endpoints)
  - WebSocket: Real-time playback notifications
  - Export: CSV, GeoJSON, GeoParquet export
  - Admin: Sync management, backup operations

# See Also

  - internal/config: Configuration management
  - internal/supervisor: Process supervision
  - internal/api: HTTP handlers and routing
  - internal/sync: Media server synchronization
  - CLAUDE.md: Development and AI assistant guidelines
  - docs/DEVELOPMENT.md: Development workflow
*/
package main
