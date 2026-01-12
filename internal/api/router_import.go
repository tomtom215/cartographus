// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package api

import (
	"net/http"
)

// RegisterImportRoutes adds import-related routes to the given ServeMux.
// This function is called from the main router setup when NATS is enabled.
//
// Routes added:
//   - POST   /api/v1/import/tautulli    - Start import
//   - GET    /api/v1/import/status      - Get import status
//   - DELETE /api/v1/import             - Stop import
//   - DELETE /api/v1/import/progress    - Clear saved progress
//   - POST   /api/v1/import/validate    - Validate database file
func (router *Router) RegisterImportRoutes(mux *http.ServeMux, handlers *ImportHandlers) {
	// Start import (authenticated, rate limited)
	mux.HandleFunc("/api/v1/import/tautulli",
		router.middleware.CORS(
			router.middleware.RateLimit(
				router.middleware.Authenticate(
					router.wrap(handlers.HandleStartImport),
				),
			),
		),
	)

	// Get import status (authenticated)
	mux.HandleFunc("/api/v1/import/status",
		router.middleware.CORS(
			router.middleware.Authenticate(
				router.wrap(handlers.HandleGetImportStatus),
			),
		),
	)

	// Stop import (authenticated)
	mux.HandleFunc("/api/v1/import/stop",
		router.middleware.CORS(
			router.middleware.Authenticate(
				router.wrap(handlers.HandleStopImport),
			),
		),
	)

	// Clear progress (authenticated)
	mux.HandleFunc("/api/v1/import/progress",
		router.middleware.CORS(
			router.middleware.Authenticate(
				router.wrap(handlers.HandleClearProgress),
			),
		),
	)

	// Validate database (authenticated, rate limited)
	mux.HandleFunc("/api/v1/import/validate",
		router.middleware.CORS(
			router.middleware.RateLimit(
				router.middleware.Authenticate(
					router.wrap(handlers.HandleValidateDatabase),
				),
			),
		),
	)
}
