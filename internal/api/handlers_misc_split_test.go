// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliLibraries_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetLibrariesFunc: func(ctx context.Context) (*tautulli.TautulliLibraries, error) {
			return &tautulli.TautulliLibraries{
				Response: tautulli.TautulliLibrariesResponse{
					Result: "success",
					Data:   []tautulli.TautulliLibraryDetail{},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/libraries", nil)
	w := httptest.NewRecorder()

	handler.TautulliLibraries(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
