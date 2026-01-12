// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package main provides the Tautulli Maps HTTP server
//
// Tautulli Maps API provides geographic visualization and analytics
// for Plex Media Server playback activity via Tautulli integration.
//
// @title Tautulli Maps API
// @version 1.0
// @description Geographic visualization and analytics platform for Plex Media Server playback activity
// @description
// @description ## Features
// @description
// @description - **Interactive WebGL Map**: Visualize playback locations with clustering for 10,000+ points
// @description - **3D Globe View**: WebGL-accelerated 3D globe visualization
// @description - **32 Analytics Charts**: Powered by Apache ECharts across 5 themed sections
// @description - **Real-time Updates**: WebSocket-based live notifications
// @description - **Advanced Filtering**: 14+ filter dimensions (users, media types, platforms, codecs)
// @description - **Data Export**: CSV and GeoJSON export capabilities
// @description - **Progressive Web App**: Offline support with service worker
// @description
// @description ## Authentication
// @description
// @description Most endpoints require JWT authentication via HTTP-only cookie.
// @description Use `/api/v1/auth/login` to obtain a token, which will be automatically included in subsequent requests.
// @description
// @description ## Rate Limiting
// @description
// @description Default rate limit: 100 requests per minute per IP address.
// @description Rate limit headers are included in responses: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.
// @description
// @description ## Performance
// @description
// @description - API response times (p95): <100ms
// @description - Map rendering: 60 FPS
// @description - Data sync (10k events): <30s
// @description - In-memory caching: 5-minute TTL for analytics
// @description
// @description ## Error Responses
// @description
// @description All error responses follow this format:
// @description ```json
// @description {
// @description   "status": "error",
// @description   "data": null,
// @description   "error": {
// @description     "code": "ERROR_CODE",
// @description     "message": "Human-readable error message",
// @description     "details": {}
// @description   },
// @description   "metadata": {
// @description     "timestamp": "2025-11-18T12:34:56Z"
// @description   }
// @description }
// @description ```
//
// @contact.name GitHub Repository
// @contact.url https://github.com/tomtom215/cartographus/issues
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:3857
// @BasePath /api/v1
// @schemes http https
//
// @securityDefinitions.apikey BearerAuth
// @in cookie
// @name token
// @description JWT token stored in HTTP-only cookie. Obtain via /api/v1/auth/login endpoint.
//
// @tag.name Core
// @tag.description Core API endpoints for health checks, statistics, and system status
//
// @tag.name Analytics
// @tag.description Analytics and visualization data endpoints providing insights into playback patterns, user behavior, and media consumption
//
// @tag.name Export
// @tag.description Data export endpoints supporting CSV and GeoJSON formats for external analysis
//
// @tag.name Auth
// @tag.description Authentication and session management endpoints
//
// @tag.name Realtime
// @tag.description Real-time WebSocket connections for live playback notifications and statistics updates
//
// @tag.name Admin
// @tag.description Administrative operations requiring authentication (sync management, system configuration)
//
// @x-logo {"url": "https://raw.githubusercontent.com/tomtom215/cartographus/main/web/public/icons/icon.svg", "altText": "Tautulli Maps Logo"}
package main
