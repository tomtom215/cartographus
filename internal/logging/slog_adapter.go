// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package logging

import (
	"context"
	"log/slog"

	"github.com/rs/zerolog"
)

// SlogHandler implements slog.Handler using zerolog as the backend.
// This adapter enables libraries that require slog.Logger (like sutureslog)
// to use zerolog for actual logging.
//
// Usage:
//
//	handler := logging.NewSlogHandler()
//	slogger := slog.New(handler)
//	// Now slogger writes to zerolog
type SlogHandler struct {
	logger zerolog.Logger
	attrs  []slog.Attr
	groups []string
}

// NewSlogHandler creates a new slog.Handler that wraps the global zerolog logger.
func NewSlogHandler() *SlogHandler {
	return &SlogHandler{
		logger: Logger(),
		attrs:  nil,
		groups: nil,
	}
}

// NewSlogHandlerWithLogger creates a new slog.Handler with a specific zerolog logger.
//
//nolint:gocritic // zerolog.Logger is designed to be passed by value
func NewSlogHandlerWithLogger(logger zerolog.Logger) *SlogHandler {
	return &SlogHandler{
		logger: logger,
		attrs:  nil,
		groups: nil,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.logger.GetLevel() <= slogToZerologLevel(level)
}

// Handle handles the Record.
//
//nolint:gocritic // slog.Record is passed by value per slog.Handler interface
func (h *SlogHandler) Handle(_ context.Context, record slog.Record) error {
	var event *zerolog.Event

	switch record.Level {
	case slog.LevelDebug:
		event = h.logger.Debug()
	case slog.LevelInfo:
		event = h.logger.Info()
	case slog.LevelWarn:
		event = h.logger.Warn()
	case slog.LevelError:
		event = h.logger.Error()
	default:
		event = h.logger.Info()
	}

	// Add pre-configured attributes
	for _, attr := range h.attrs {
		event = addAttr(event, attr, h.groups)
	}

	// Add record attributes
	record.Attrs(func(attr slog.Attr) bool {
		event = addAttr(event, attr, h.groups)
		return true
	})

	event.Msg(record.Message)
	return nil
}

// WithAttrs returns a new Handler with the given attributes.
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &SlogHandler{
		logger: h.logger,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

// WithGroup returns a new Handler with the given group name.
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &SlogHandler{
		logger: h.logger,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// addAttr adds a slog attribute to a zerolog event.
func addAttr(event *zerolog.Event, attr slog.Attr, groups []string) *zerolog.Event {
	key := attr.Key
	if len(groups) > 0 {
		for _, g := range groups {
			key = g + "." + key
		}
	}

	switch attr.Value.Kind() {
	case slog.KindString:
		return event.Str(key, attr.Value.String())
	case slog.KindInt64:
		return event.Int64(key, attr.Value.Int64())
	case slog.KindUint64:
		return event.Uint64(key, attr.Value.Uint64())
	case slog.KindFloat64:
		return event.Float64(key, attr.Value.Float64())
	case slog.KindBool:
		return event.Bool(key, attr.Value.Bool())
	case slog.KindDuration:
		return event.Dur(key, attr.Value.Duration())
	case slog.KindTime:
		return event.Time(key, attr.Value.Time())
	case slog.KindAny:
		return event.Interface(key, attr.Value.Any())
	case slog.KindGroup:
		// Handle group attributes recursively
		groupAttrs := attr.Value.Group()
		for _, ga := range groupAttrs {
			event = addAttr(event, ga, append(groups, attr.Key))
		}
		return event
	default:
		return event.Interface(key, attr.Value.Any())
	}
}

// slogToZerologLevel converts slog.Level to zerolog.Level.
func slogToZerologLevel(level slog.Level) zerolog.Level {
	switch {
	case level < slog.LevelDebug:
		return zerolog.TraceLevel
	case level < slog.LevelInfo:
		return zerolog.DebugLevel
	case level < slog.LevelWarn:
		return zerolog.InfoLevel
	case level < slog.LevelError:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}

// NewSlogLogger creates an slog.Logger backed by zerolog.
// This is a convenience function for creating slog loggers compatible with
// libraries like sutureslog.
//
//	slogger := logging.NewSlogLogger()
//	sutureHandler := &sutureslog.Handler{Logger: slogger}
func NewSlogLogger() *slog.Logger {
	return slog.New(NewSlogHandler())
}

// NewSlogLoggerWithLevel creates an slog.Logger with a specific level.
func NewSlogLoggerWithLevel(level string) *slog.Logger {
	zl := parseLevel(level)
	logger := Logger().Level(zl)
	return slog.New(NewSlogHandlerWithLogger(logger))
}
