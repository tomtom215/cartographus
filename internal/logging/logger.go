// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package logging provides centralized zerolog-based logging for Cartographus.
//
// This package replaces the mixed logging approach (standard log, slog, custom loggers)
// with a unified zerolog implementation that provides:
//
//   - Zero-allocation structured logging
//   - JSON output for production, console output for development
//   - Context-aware logging with correlation ID propagation
//   - Global logger configuration via environment variables
//   - Backward-compatible APIs for existing custom loggers
//
// # Quick Start
//
//	import "github.com/tomtom215/cartographus/internal/logging"
//
//	// Initialize at application startup
//	logging.Init(logging.Config{
//	    Level:  "info",
//	    Format: "json",
//	})
//
//	// Log messages
//	logging.Info().Msg("Server starting")
//	logging.Error().Err(err).Msg("Operation failed")
//
//	// With context (correlation ID)
//	logging.Ctx(ctx).Info().Str("user", userID).Msg("Request processed")
//
// # Configuration
//
// Environment Variables:
//   - LOG_LEVEL: debug, info, warn, error (default: info)
//   - LOG_FORMAT: json, console (default: json)
//   - LOG_CALLER: true/false - include caller info (default: false)
//
// # Best Practices
//
// Always terminate log chains with .Msg() or .Send():
//
//	logging.Info().Str("key", "value").Msg("message")  // Correct
//	logging.Info().Str("key", "value")                 // WRONG - log not emitted
//
// Use structured fields instead of string formatting:
//
//	logging.Info().Str("user", u).Int("count", n).Msg("processed")  // Correct
//	logging.Info().Msgf("processed %d items for %s", n, u)          // Avoid
package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Config holds logging configuration.
type Config struct {
	// Level is the minimum log level: trace, debug, info, warn, error, fatal, panic.
	// Default: info
	Level string

	// Format is the output format: json or console.
	// Default: json (recommended for production)
	Format string

	// Caller includes caller file and line number in logs.
	// Default: false (reduces performance overhead)
	Caller bool

	// Timestamp enables timestamps in log output.
	// Default: true
	Timestamp bool

	// Output is the writer for log output.
	// Default: os.Stderr
	Output io.Writer
}

// DefaultConfig returns the default logging configuration.
func DefaultConfig() Config {
	return Config{
		Level:     "info",
		Format:    "json",
		Caller:    false,
		Timestamp: true,
		Output:    os.Stderr,
	}
}

var (
	// log is the global logger instance.
	log zerolog.Logger

	// mu protects concurrent initialization.
	mu sync.RWMutex
)

//nolint:gochecknoinits // init ensures logging works before explicit Init() call
func init() {
	cfg := DefaultConfig()

	// Check for FUZZ_MODE to suppress logging during fuzz testing
	// This prevents noisy log output that pollutes fuzz test results
	if os.Getenv("FUZZ_MODE") == "1" {
		cfg.Level = "fatal" // Only log fatal errors during fuzzing
	}

	// Initialize with defaults to ensure logging works before explicit Init()
	initLogger(cfg)
}

// Init initializes the global logger with the given configuration.
// This should be called early in application startup, typically from main().
// It is safe to call multiple times; subsequent calls reconfigure the logger.
func Init(cfg Config) {
	mu.Lock()
	defer mu.Unlock()
	initLogger(cfg)
}

// initLogger configures the global logger (must be called with mu held).
func initLogger(cfg Config) {
	// Apply defaults for empty values
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "json"
	}
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}

	// Set global level
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Configure time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Configure field names for consistency
	zerolog.TimestampFieldName = "time"
	zerolog.LevelFieldName = "level"
	zerolog.MessageFieldName = "message"
	zerolog.ErrorFieldName = "error"
	zerolog.CallerFieldName = "caller"

	// Create output writer based on format
	output := cfg.Output
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        cfg.Output,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}
	}

	// Build logger context
	ctx := zerolog.New(output)

	// Add timestamp if enabled (default: true)
	if cfg.Timestamp {
		ctx = ctx.With().Timestamp().Logger()
	}

	// Add caller info if enabled
	if cfg.Caller {
		ctx = ctx.With().Caller().Logger()
	}

	log = ctx
}

// parseLevel converts a string level to zerolog.Level.
func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	case "disabled":
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

// Logger returns the global logger instance.
// Use this to access the underlying zerolog.Logger directly.
func Logger() zerolog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return log
}

// SetLogger replaces the global logger instance.
// This is useful for testing or specialized configurations.
//
//nolint:gocritic // zerolog.Logger is designed to be passed by value
func SetLogger(l zerolog.Logger) {
	mu.Lock()
	defer mu.Unlock()
	log = l
}

// With creates a child logger with additional context.
// Use this to create component-specific loggers with default fields.
//
//	syncLogger := logging.With().Str("component", "sync").Logger()
//	syncLogger.Info().Msg("Sync started")
func With() zerolog.Context {
	mu.RLock()
	defer mu.RUnlock()
	return log.With()
}

// Level creates a child logger with the specified minimum level.
//
//	debugLogger := logging.Level(zerolog.DebugLevel)
//	debugLogger.Debug().Msg("Verbose output")
func Level(level zerolog.Level) zerolog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return log.Level(level)
}

// Output duplicates the current logger and sets the output.
//
//	fileLogger := logging.Output(file)
func Output(w io.Writer) zerolog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return log.Output(w)
}

// Trace starts a new message with trace level.
//
//	logging.Trace().Str("detail", "verbose").Msg("Tracing")
func Trace() *zerolog.Event {
	mu.RLock()
	defer mu.RUnlock()
	return log.Trace()
}

// Debug starts a new message with debug level.
//
//	logging.Debug().Str("config", path).Msg("Loaded config")
func Debug() *zerolog.Event {
	mu.RLock()
	defer mu.RUnlock()
	return log.Debug()
}

// Info starts a new message with info level.
//
//	logging.Info().Msg("Server starting")
func Info() *zerolog.Event {
	mu.RLock()
	defer mu.RUnlock()
	return log.Info()
}

// Warn starts a new message with warning level.
//
//	logging.Warn().Err(err).Msg("Connection retry")
func Warn() *zerolog.Event {
	mu.RLock()
	defer mu.RUnlock()
	return log.Warn()
}

// Error starts a new message with error level.
//
//	logging.Error().Err(err).Str("user", uid).Msg("Auth failed")
func Error() *zerolog.Event {
	mu.RLock()
	defer mu.RUnlock()
	return log.Error()
}

// Fatal starts a new message with fatal level.
// The os.Exit(1) function is called after the message is logged.
//
//	logging.Fatal().Err(err).Msg("Cannot start server")
func Fatal() *zerolog.Event {
	mu.RLock()
	defer mu.RUnlock()
	return log.Fatal()
}

// Panic starts a new message with panic level.
// Panic() is called after the message is logged.
//
//	logging.Panic().Err(err).Msg("Unrecoverable error")
func Panic() *zerolog.Event {
	mu.RLock()
	defer mu.RUnlock()
	return log.Panic()
}

// Err starts a new message with error level and adds the error.
// This is a convenience method equivalent to Error().Err(err).
//
//	logging.Err(err).Msg("Operation failed")
func Err(err error) *zerolog.Event {
	mu.RLock()
	defer mu.RUnlock()
	return log.Err(err)
}

// Print sends a log event at info level.
// Arguments are handled like fmt.Print.
//
// Deprecated: Use structured logging instead.
func Print(v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	log.Info().Msg(fmt.Sprint(v...))
}

// Printf sends a log event at info level.
// Arguments are handled like fmt.Printf.
//
// Deprecated: Use structured logging instead.
func Printf(format string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	log.Info().Msgf(format, v...)
}

// GetLevel returns the current global log level.
func GetLevel() zerolog.Level {
	return zerolog.GlobalLevel()
}

// SetLevel updates the global log level.
func SetLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// SetLevelString updates the global log level from a string.
func SetLevelString(level string) {
	zerolog.SetGlobalLevel(parseLevel(level))
}

// IsLevelEnabled returns true if the given level is enabled.
func IsLevelEnabled(level zerolog.Level) bool {
	return zerolog.GlobalLevel() <= level
}

// NewTestLogger creates a logger that writes to the provided writer.
// This is useful for testing to capture log output.
//
//	var buf bytes.Buffer
//	logger := logging.NewTestLogger(&buf)
//	logger.Info().Msg("test")
//	output := buf.String()
func NewTestLogger(w io.Writer) zerolog.Logger {
	return zerolog.New(w).With().Timestamp().Logger()
}

// NewConsoleTestLogger creates a console-formatted logger for testing.
// Useful for visual inspection during test development.
func NewConsoleTestLogger(w io.Writer) zerolog.Logger {
	return zerolog.New(zerolog.ConsoleWriter{
		Out:        w,
		TimeFormat: "15:04:05",
		NoColor:    true,
	}).With().Timestamp().Logger()
}
