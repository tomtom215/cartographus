// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"io"
	"log/slog"

	"github.com/tomtom215/cartographus/internal/logging"
)

// closeWithLog closes a resource and logs any error
// Use this for cleanup operations where errors should be acknowledged but not fail the operation
func closeWithLog(closer io.Closer, logger *slog.Logger, resourceType string) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		if logger != nil {
			logger.Error("failed to close resource",
				"type", resourceType,
				"error", err)
		} else {
			// Fallback to logging if logger not available
			logging.Warn().Str("type", resourceType).Err(err).Msg("Failed to close resource")
		}
	}
}

// closeQuietly closes a resource and explicitly ignores any error
// Use this for cleanup operations in error paths where Close() errors are not actionable
// Satisfies errcheck linter by explicitly acknowledging the ignored error
func closeQuietly(closer io.Closer) {
	if closer != nil {
		_ = closer.Close() // Explicitly ignore error - cleanup is best-effort
	}
}
