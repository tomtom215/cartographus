// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package logging provides centralized zerolog-based structured logging for Cartographus.
//
// This package implements a unified logging layer using zerolog, providing
// zero-allocation structured JSON logging for production and human-readable
// console output for development. It replaces mixed logging approaches with
// a consistent, high-performance logging interface.
//
// # Overview
//
// The package provides:
//   - Zero-allocation structured logging via zerolog
//   - JSON output format for production (machine-parseable)
//   - Console output format for development (human-readable)
//   - Global logger configuration via environment variables
//   - Context-aware logging with correlation ID propagation
//   - slog adapter for Suture v4 integration
//   - Security-focused logging with sensitive data filtering
//
// # Quick Start
//
//	import "github.com/tomtom215/cartographus/internal/logging"
//
//	// Initialize at application startup
//	logging.Init(logging.Config{
//	    Level:  "info",
//	    Format: "json",
//	    Caller: false,
//	})
//
//	// Log messages with structured fields
//	logging.Info().Str("user", "alice").Msg("Login successful")
//	logging.Error().Err(err).Int("code", 500).Msg("Request failed")
//
//	// Context-aware logging
//	logging.Ctx(ctx).Info().Str("request_id", reqID).Msg("Processing")
//
// # Configuration
//
// Environment Variables:
//
//	LOG_LEVEL   - Minimum log level: trace, debug, info, warn, error (default: info)
//	LOG_FORMAT  - Output format: json, console (default: json)
//	LOG_CALLER  - Include caller file:line: true, false (default: false)
//
// Programmatic Configuration:
//
//	logging.Init(logging.Config{
//	    Level:     "debug",    // trace, debug, info, warn, error, fatal
//	    Format:    "console",  // json or console
//	    Caller:    true,       // Include caller info
//	    Timestamp: true,       // Include timestamps
//	    Output:    os.Stderr,  // Output writer
//	})
//
// # Log Levels
//
// Supported log levels (from most to least verbose):
//
//	trace  - Very detailed diagnostic information
//	debug  - Detailed diagnostic information
//	info   - General operational information (default)
//	warn   - Warning conditions that should be addressed
//	error  - Error conditions requiring attention
//	fatal  - Fatal errors that terminate the program
//	panic  - Panic conditions that crash the program
//
// # Structured Logging Best Practices
//
// Always terminate log chains with .Msg() or .Send():
//
//	logging.Info().Str("key", "value").Msg("message")  // Correct
//	logging.Info().Str("key", "value")                 // WRONG - log not emitted
//
// Use structured fields instead of string formatting:
//
//	// Good - structured, searchable, efficient
//	logging.Info().
//	    Str("user", username).
//	    Int("count", itemCount).
//	    Dur("elapsed", duration).
//	    Msg("Items processed")
//
//	// Avoid - unstructured, harder to parse
//	logging.Info().Msgf("User %s processed %d items in %v", username, itemCount, duration)
//
// # Component Loggers
//
// Create component-specific loggers with default fields:
//
//	// Create a logger for the sync component
//	syncLogger := logging.With().Str("component", "sync").Logger()
//	syncLogger.Info().Msg("Starting sync")
//	syncLogger.Error().Err(err).Msg("Sync failed")
//
// # Context-Aware Logging
//
// Propagate request context through logging:
//
//	// Extract correlation ID from context
//	logger := logging.Ctx(ctx)
//	logger.Info().Msg("Processing request")
//
// # slog Adapter
//
// The package provides an slog adapter for libraries that require slog.Logger:
//
//	slogLogger := logging.NewSlogLogger()
//	// Use slogLogger with Suture or other slog-compatible libraries
//
// # Security Logging
//
// Security-relevant events should use structured fields:
//
//	logging.Warn().
//	    Str("event", "auth.failure").
//	    Str("ip", clientIP).
//	    Str("user", username).
//	    Str("reason", "invalid_password").
//	    Msg("Authentication failed")
//
// # Output Formats
//
// JSON Format (Production):
//
//	{"level":"info","time":"2025-01-03T10:30:00Z","message":"Server starting","port":3857}
//
// Console Format (Development):
//
//	10:30:00 INF Server starting port=3857
//
// # Thread Safety
//
// All exported functions are safe for concurrent use. The global logger
// is protected by sync.RWMutex for configuration changes.
//
// # Performance Characteristics
//
//   - Info/Debug/Warn/Error: ~150ns per call (zero allocations)
//   - With context fields: ~200ns per call
//   - JSON encoding: ~500ns per message
//   - Console encoding: ~800ns per message
//   - Memory allocation: 0 bytes for typical log calls
//
// # Testing
//
// Create test loggers that capture output:
//
//	var buf bytes.Buffer
//	logger := logging.NewTestLogger(&buf)
//	logger.Info().Msg("test message")
//	output := buf.String()
//
// # See Also
//
//   - github.com/rs/zerolog: Underlying logging library
//   - internal/middleware: Request ID middleware for correlation
//   - internal/audit: Security audit logging (uses this package internally)
package logging
