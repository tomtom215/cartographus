// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
//
// errors.go - Common API error definitions
//
// This file contains sentinel errors for common API error conditions.
package api

import "errors"

// Common API errors
var (
	// ErrPlexNotEnabled indicates Plex integration is not enabled in config
	ErrPlexNotEnabled = errors.New("plex integration is not enabled")

	// ErrPlexTokenRequired indicates Plex token is required but not configured
	ErrPlexTokenRequired = errors.New("plex token is required")
)
