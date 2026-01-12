// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.Level != "info" {
		t.Errorf("expected default level 'info', got '%s'", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected default format 'json', got '%s'", cfg.Format)
	}
	if cfg.Caller {
		t.Error("expected default caller to be false")
	}
	if !cfg.Timestamp {
		t.Error("expected default timestamp to be true")
	}
}

func TestInit(t *testing.T) {
	var buf bytes.Buffer

	Init(Config{
		Level:     "debug",
		Format:    "json",
		Timestamp: true,
		Output:    &buf,
	})

	Info().Msg("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, `"level":"info"`) {
		t.Errorf("expected output to contain level, got: %s", output)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zerolog.Level
	}{
		{"trace", zerolog.TraceLevel},
		{"debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"warning", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"fatal", zerolog.FatalLevel},
		{"panic", zerolog.PanicLevel},
		{"disabled", zerolog.Disabled},
		{"TRACE", zerolog.TraceLevel},
		{"DEBUG", zerolog.DebugLevel},
		{"INFO", zerolog.InfoLevel},
		{"invalid", zerolog.InfoLevel}, // default
		{"", zerolog.InfoLevel},        // empty
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	// Set up test logger
	SetLogger(zerolog.New(&buf).With().Timestamp().Logger())
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	tests := []struct {
		name    string
		logFunc func()
		level   string
	}{
		{"Trace", func() { Trace().Msg("trace msg") }, "trace"},
		{"Debug", func() { Debug().Msg("debug msg") }, "debug"},
		{"Info", func() { Info().Msg("info msg") }, "info"},
		{"Warn", func() { Warn().Msg("warn msg") }, "warn"},
		{"Error", func() { Error().Msg("error msg") }, "error"},
	}

	for _, tt := range tests {
		buf.Reset()
		tt.logFunc()
		output := buf.String()
		if !strings.Contains(output, tt.level) {
			t.Errorf("%s: expected level '%s' in output: %s", tt.name, tt.level, output)
		}
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer

	SetLogger(zerolog.New(&buf).With().Timestamp().Logger())

	logger := With().Str("component", "test").Logger()
	logger.Info().Msg("component message")

	output := buf.String()
	if !strings.Contains(output, "component") {
		t.Errorf("expected 'component' field in output: %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("expected 'test' value in output: %s", output)
	}
}

func TestNewTestLogger(t *testing.T) {
	var buf bytes.Buffer

	logger := NewTestLogger(&buf)
	logger.Info().Str("key", "value").Msg("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected 'test message' in output: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("expected 'key' in output: %s", output)
	}
	if !strings.Contains(output, "value") {
		t.Errorf("expected 'value' in output: %s", output)
	}
}

func TestSetLevelString(t *testing.T) {
	// Save original level
	originalLevel := GetLevel()
	defer SetLevel(originalLevel)

	SetLevelString("debug")
	if GetLevel() != zerolog.DebugLevel {
		t.Errorf("expected DebugLevel, got %v", GetLevel())
	}

	SetLevelString("error")
	if GetLevel() != zerolog.ErrorLevel {
		t.Errorf("expected ErrorLevel, got %v", GetLevel())
	}
}

func TestIsLevelEnabled(t *testing.T) {
	// Save original level
	originalLevel := GetLevel()
	defer SetLevel(originalLevel)

	SetLevel(zerolog.InfoLevel)

	if !IsLevelEnabled(zerolog.InfoLevel) {
		t.Error("expected InfoLevel to be enabled")
	}
	if !IsLevelEnabled(zerolog.WarnLevel) {
		t.Error("expected WarnLevel to be enabled")
	}
	if IsLevelEnabled(zerolog.DebugLevel) {
		t.Error("expected DebugLevel to be disabled")
	}
}

func TestConsoleFormat(t *testing.T) {
	var buf bytes.Buffer

	Init(Config{
		Level:     "info",
		Format:    "console",
		Timestamp: false,
		Output:    &buf,
	})

	Info().Msg("console test")

	output := buf.String()
	// Console format should not contain JSON syntax
	if strings.Contains(output, `"level"`) {
		t.Errorf("expected console format (not JSON): %s", output)
	}
}

func TestPrintFunctions(t *testing.T) {
	var buf bytes.Buffer

	SetLogger(zerolog.New(&buf))

	Print("print test")
	if !strings.Contains(buf.String(), "print test") {
		t.Errorf("expected 'print test' in output: %s", buf.String())
	}

	buf.Reset()
	Printf("formatted %s", "test")
	if !strings.Contains(buf.String(), "formatted test") {
		t.Errorf("expected 'formatted test' in output: %s", buf.String())
	}
}

func TestErr(t *testing.T) {
	var buf bytes.Buffer

	SetLogger(zerolog.New(&buf))

	testErr := &testError{msg: "test error"}
	Err(testErr).Msg("error occurred")

	output := buf.String()
	if !strings.Contains(output, "test error") {
		t.Errorf("expected error in output: %s", output)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
