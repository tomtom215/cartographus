// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// --- Test: NewSlogHandler ---

func TestNewSlogHandler(t *testing.T) {
	t.Parallel()

	handler := NewSlogHandler()

	if handler == nil {
		t.Fatal("NewSlogHandler() = nil, want non-nil")
	}

	if handler.attrs != nil {
		t.Errorf("NewSlogHandler().attrs = %v, want nil", handler.attrs)
	}

	if handler.groups != nil {
		t.Errorf("NewSlogHandler().groups = %v, want nil", handler.groups)
	}
}

// --- Test: NewSlogHandlerWithLogger ---

func TestNewSlogHandlerWithLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	handler := NewSlogHandlerWithLogger(logger)

	if handler == nil {
		t.Fatal("NewSlogHandlerWithLogger() = nil, want non-nil")
	}

	// Use the handler to log something and verify it goes to the buffer
	slogger := slog.New(handler)
	slogger.Info("test message")

	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("expected 'test message' in output: %s", buf.String())
	}
}

// --- Test: SlogHandler.Enabled ---

func TestSlogHandler_Enabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		zerologLevel zerolog.Level
		slogLevel    slog.Level
		want         bool
	}{
		{
			name:         "debug logger enables debug level",
			zerologLevel: zerolog.DebugLevel,
			slogLevel:    slog.LevelDebug,
			want:         true,
		},
		{
			name:         "info logger disables debug level",
			zerologLevel: zerolog.InfoLevel,
			slogLevel:    slog.LevelDebug,
			want:         false,
		},
		{
			name:         "info logger enables info level",
			zerologLevel: zerolog.InfoLevel,
			slogLevel:    slog.LevelInfo,
			want:         true,
		},
		{
			name:         "info logger enables warn level",
			zerologLevel: zerolog.InfoLevel,
			slogLevel:    slog.LevelWarn,
			want:         true,
		},
		{
			name:         "warn logger disables info level",
			zerologLevel: zerolog.WarnLevel,
			slogLevel:    slog.LevelInfo,
			want:         false,
		},
		{
			name:         "error logger disables warn level",
			zerologLevel: zerolog.ErrorLevel,
			slogLevel:    slog.LevelWarn,
			want:         false,
		},
		{
			name:         "trace logger enables all levels",
			zerologLevel: zerolog.TraceLevel,
			slogLevel:    slog.LevelDebug,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			logger := zerolog.New(nil).Level(tt.zerologLevel)
			handler := NewSlogHandlerWithLogger(logger)

			got := handler.Enabled(context.Background(), tt.slogLevel)
			if got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Test: SlogHandler.Handle ---

func TestSlogHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		level      slog.Level
		message    string
		wantLevel  string
		wantOutput string
	}{
		{
			name:       "debug level",
			level:      slog.LevelDebug,
			message:    "debug message",
			wantLevel:  "debug",
			wantOutput: "debug message",
		},
		{
			name:       "info level",
			level:      slog.LevelInfo,
			message:    "info message",
			wantLevel:  "info",
			wantOutput: "info message",
		},
		{
			name:       "warn level",
			level:      slog.LevelWarn,
			message:    "warn message",
			wantLevel:  "warn",
			wantOutput: "warn message",
		},
		{
			name:       "error level",
			level:      slog.LevelError,
			message:    "error message",
			wantLevel:  "error",
			wantOutput: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
			handler := NewSlogHandlerWithLogger(logger)

			record := slog.NewRecord(time.Now(), tt.level, tt.message, 0)
			err := handler.Handle(context.Background(), record)

			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.wantLevel) {
				t.Errorf("Handle() output missing level %q: %s", tt.wantLevel, output)
			}
			if !strings.Contains(output, tt.wantOutput) {
				t.Errorf("Handle() output missing message %q: %s", tt.wantOutput, output)
			}
		})
	}
}

func TestSlogHandler_Handle_WithAttributes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
	handler := NewSlogHandlerWithLogger(logger)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	record.AddAttrs(
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	)

	err := handler.Handle(context.Background(), record)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "key1") || !strings.Contains(output, "value1") {
		t.Errorf("Handle() output missing key1:value1: %s", output)
	}
	if !strings.Contains(output, "key2") || !strings.Contains(output, "42") {
		t.Errorf("Handle() output missing key2:42: %s", output)
	}
}

func TestSlogHandler_Handle_WithPreConfiguredAttributes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
	handler := NewSlogHandlerWithLogger(logger)

	// Add pre-configured attributes
	handlerWithAttrs := handler.WithAttrs([]slog.Attr{
		slog.String("service", "test-service"),
	})

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handlerWithAttrs.Handle(context.Background(), record)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "service") || !strings.Contains(output, "test-service") {
		t.Errorf("Handle() output missing pre-configured attribute: %s", output)
	}
}

func TestSlogHandler_Handle_UnknownLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
	handler := NewSlogHandlerWithLogger(logger)

	// Use a custom level that doesn't match any case
	record := slog.NewRecord(time.Now(), slog.Level(100), "unknown level message", 0)
	err := handler.Handle(context.Background(), record)

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	// Should default to info level
	output := buf.String()
	if !strings.Contains(output, "unknown level message") {
		t.Errorf("Handle() output missing message: %s", output)
	}
}

// --- Test: SlogHandler.WithAttrs ---

func TestSlogHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	handler := NewSlogHandler()

	// Add first set of attributes
	attrs1 := []slog.Attr{
		slog.String("key1", "value1"),
	}
	handler1 := handler.WithAttrs(attrs1).(*SlogHandler)

	if len(handler1.attrs) != 1 {
		t.Errorf("WithAttrs() attrs length = %d, want 1", len(handler1.attrs))
	}

	// Add second set of attributes
	attrs2 := []slog.Attr{
		slog.String("key2", "value2"),
		slog.Int("key3", 3),
	}
	handler2 := handler1.WithAttrs(attrs2).(*SlogHandler)

	if len(handler2.attrs) != 3 {
		t.Errorf("WithAttrs() chained attrs length = %d, want 3", len(handler2.attrs))
	}

	// Verify original handler is not modified
	if len(handler.attrs) != 0 {
		t.Error("WithAttrs() should not modify original handler")
	}
}

func TestSlogHandler_WithAttrs_Empty(t *testing.T) {
	t.Parallel()

	handler := NewSlogHandler()
	handler1 := handler.WithAttrs([]slog.Attr{})

	if handler1 == nil {
		t.Fatal("WithAttrs([]) = nil, want non-nil")
	}
}

// --- Test: SlogHandler.WithGroup ---

func TestSlogHandler_WithGroup(t *testing.T) {
	t.Parallel()

	handler := NewSlogHandler()

	// Add first group
	handler1 := handler.WithGroup("group1").(*SlogHandler)
	if len(handler1.groups) != 1 || handler1.groups[0] != "group1" {
		t.Errorf("WithGroup() groups = %v, want ['group1']", handler1.groups)
	}

	// Add second group
	handler2 := handler1.WithGroup("group2").(*SlogHandler)
	if len(handler2.groups) != 2 || handler2.groups[1] != "group2" {
		t.Errorf("WithGroup() chained groups = %v, want ['group1', 'group2']", handler2.groups)
	}

	// Verify original handler is not modified
	if len(handler.groups) != 0 {
		t.Error("WithGroup() should not modify original handler")
	}
}

func TestSlogHandler_WithGroup_Empty(t *testing.T) {
	t.Parallel()

	handler := NewSlogHandler()
	handler1 := handler.WithGroup("")

	// Empty group name should return the same handler
	if handler1 != handler {
		t.Error("WithGroup('') should return same handler")
	}
}

func TestSlogHandler_WithGroup_KeyPrefix(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
	handler := NewSlogHandlerWithLogger(logger)

	// Add group and log
	groupHandler := handler.WithGroup("prefix")
	slogger := slog.New(groupHandler)
	slogger.Info("test", "key", "value")

	output := buf.String()
	// The key should be prefixed with group name
	if !strings.Contains(output, "prefix.key") {
		t.Errorf("WithGroup() should prefix keys: %s", output)
	}
}

// --- Test: addAttr ---

func TestAddAttr_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		attr     slog.Attr
		wantKeys []string
	}{
		{
			name:     "string",
			attr:     slog.String("str", "value"),
			wantKeys: []string{"str", "value"},
		},
		{
			name:     "int64",
			attr:     slog.Int64("int", 42),
			wantKeys: []string{"int", "42"},
		},
		{
			name:     "uint64",
			attr:     slog.Uint64("uint", 100),
			wantKeys: []string{"uint", "100"},
		},
		{
			name:     "float64",
			attr:     slog.Float64("float", 3.14),
			wantKeys: []string{"float", "3.14"},
		},
		{
			name:     "bool true",
			attr:     slog.Bool("flag", true),
			wantKeys: []string{"flag", "true"},
		},
		{
			name:     "bool false",
			attr:     slog.Bool("disabled", false),
			wantKeys: []string{"disabled", "false"},
		},
		{
			name:     "duration",
			attr:     slog.Duration("elapsed", time.Second),
			wantKeys: []string{"elapsed"},
		},
		{
			name:     "time",
			attr:     slog.Time("created", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			wantKeys: []string{"created"},
		},
		{
			name:     "any",
			attr:     slog.Any("data", map[string]int{"a": 1}),
			wantKeys: []string{"data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
			handler := NewSlogHandlerWithLogger(logger)

			record := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
			record.AddAttrs(tt.attr)
			_ = handler.Handle(context.Background(), record)

			output := buf.String()
			for _, key := range tt.wantKeys {
				if !strings.Contains(output, key) {
					t.Errorf("output missing %q: %s", key, output)
				}
			}
		})
	}
}

func TestAddAttr_Group(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
	handler := NewSlogHandlerWithLogger(logger)

	// Create a group attribute
	groupAttr := slog.Group("request",
		slog.String("method", "GET"),
		slog.Int("status", 200),
	)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	record.AddAttrs(groupAttr)
	_ = handler.Handle(context.Background(), record)

	output := buf.String()
	// Group attributes should be prefixed
	if !strings.Contains(output, "request.method") {
		t.Errorf("output missing request.method: %s", output)
	}
	if !strings.Contains(output, "request.status") {
		t.Errorf("output missing request.status: %s", output)
	}
}

func TestAddAttr_NestedGroups(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
	handler := NewSlogHandlerWithLogger(logger)

	// First level group
	handler1 := handler.WithGroup("level1")
	// Second level group
	handler2 := handler1.WithGroup("level2")

	slogger := slog.New(handler2)
	slogger.Info("test", "key", "value")

	output := buf.String()
	// Key should have both group prefixes (prepended in order, so outer groups come first)
	// The addAttr function prepends groups in order, resulting in level2.level1.key
	if !strings.Contains(output, "level2.level1.key") {
		t.Errorf("output should have nested group prefix: %s", output)
	}
}

// --- Test: slogToZerologLevel ---

func TestSlogToZerologLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		slogLvl  slog.Level
		wantZlog zerolog.Level
	}{
		{
			name:     "debug",
			slogLvl:  slog.LevelDebug,
			wantZlog: zerolog.DebugLevel,
		},
		{
			name:     "info",
			slogLvl:  slog.LevelInfo,
			wantZlog: zerolog.InfoLevel,
		},
		{
			name:     "warn",
			slogLvl:  slog.LevelWarn,
			wantZlog: zerolog.WarnLevel,
		},
		{
			name:     "error",
			slogLvl:  slog.LevelError,
			wantZlog: zerolog.ErrorLevel,
		},
		{
			name:     "below debug (trace equivalent)",
			slogLvl:  slog.Level(-8), // Below debug
			wantZlog: zerolog.TraceLevel,
		},
		{
			name:     "above error",
			slogLvl:  slog.Level(12), // Above error
			wantZlog: zerolog.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := slogToZerologLevel(tt.slogLvl)
			if got != tt.wantZlog {
				t.Errorf("slogToZerologLevel(%v) = %v, want %v", tt.slogLvl, got, tt.wantZlog)
			}
		})
	}
}

// --- Test: NewSlogLogger ---

func TestNewSlogLogger(t *testing.T) {
	// Not parallel because it uses global logger state

	var buf bytes.Buffer
	SetLogger(zerolog.New(&buf).Level(zerolog.TraceLevel))

	slogger := NewSlogLogger()
	if slogger == nil {
		t.Fatal("NewSlogLogger() = nil, want non-nil")
	}

	slogger.Info("test from slog")

	output := buf.String()
	if !strings.Contains(output, "test from slog") {
		t.Errorf("NewSlogLogger() should write to global logger: %s", output)
	}
}

// --- Test: NewSlogLoggerWithLevel ---

func TestNewSlogLoggerWithLevel(t *testing.T) {
	// Not parallel because it uses global logger state

	var buf bytes.Buffer
	SetLogger(zerolog.New(&buf))

	tests := []struct {
		name         string
		level        string
		debugEnabled bool
		infoEnabled  bool
	}{
		{
			name:         "debug level enables all",
			level:        "debug",
			debugEnabled: true,
			infoEnabled:  true,
		},
		{
			name:         "info level disables debug",
			level:        "info",
			debugEnabled: false,
			infoEnabled:  true,
		},
		{
			name:         "warn level disables info",
			level:        "warn",
			debugEnabled: false,
			infoEnabled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			slogger := NewSlogLoggerWithLevel(tt.level)
			if slogger == nil {
				t.Fatal("NewSlogLoggerWithLevel() = nil")
			}

			handler := slogger.Handler()

			debugEnabled := handler.Enabled(context.Background(), slog.LevelDebug)
			if debugEnabled != tt.debugEnabled {
				t.Errorf("debug enabled = %v, want %v", debugEnabled, tt.debugEnabled)
			}

			infoEnabled := handler.Enabled(context.Background(), slog.LevelInfo)
			if infoEnabled != tt.infoEnabled {
				t.Errorf("info enabled = %v, want %v", infoEnabled, tt.infoEnabled)
			}
		})
	}
}

// --- Test: Integration with slog ---

func TestSlogHandler_FullIntegration(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
	handler := NewSlogHandlerWithLogger(logger)
	slogger := slog.New(handler)

	// Create a child logger with context
	childLogger := slogger.With("component", "test")

	// Log with various levels and attributes
	childLogger.Debug("debug message", "debug_key", "debug_value")
	childLogger.Info("info message", "info_key", 123)
	childLogger.Warn("warn message", "warn_key", true)
	childLogger.Error("error message", "error_key", 3.14)

	output := buf.String()

	// Verify all messages are present
	expected := []string{
		"debug message", "debug_key", "debug_value",
		"info message", "info_key", "123",
		"warn message", "warn_key", "true",
		"error message", "error_key", "3.14",
		"component", "test",
	}

	for _, e := range expected {
		if !strings.Contains(output, e) {
			t.Errorf("output missing %q: %s", e, output)
		}
	}
}

func TestSlogHandler_ContextPassing(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.TraceLevel)
	handler := NewSlogHandlerWithLogger(logger)

	// Context is passed to Handle but not used (for now)
	// Use a typed key to satisfy revive linter
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "test-value")

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test with context", 0)
	err := handler.Handle(ctx, record)

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	// Just verify it doesn't panic and logs the message
	if !strings.Contains(buf.String(), "test with context") {
		t.Errorf("Handle() should log message: %s", buf.String())
	}
}
