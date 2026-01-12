// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestGenerateCorrelationID(t *testing.T) {
	t.Parallel()

	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	if id1 == "" {
		t.Error("expected non-empty correlation ID")
	}
	if len(id1) != 8 {
		t.Errorf("expected 8-character correlation ID, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("expected unique correlation IDs")
	}
}

func TestGenerateRequestID(t *testing.T) {
	t.Parallel()

	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if id1 == "" {
		t.Error("expected non-empty request ID")
	}
	if len(id1) != 36 { // UUID format
		t.Errorf("expected 36-character request ID, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("expected unique request IDs")
	}
}

func TestCorrelationIDContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Without correlation ID
	id := CorrelationIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty correlation ID, got %s", id)
	}

	// With correlation ID
	ctx = ContextWithCorrelationID(ctx, "test-123")
	id = CorrelationIDFromContext(ctx)
	if id != "test-123" {
		t.Errorf("expected 'test-123', got '%s'", id)
	}
}

func TestContextWithNewCorrelationID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = ContextWithNewCorrelationID(ctx)

	id := CorrelationIDFromContext(ctx)
	if id == "" {
		t.Error("expected correlation ID to be generated")
	}
	if len(id) != 8 {
		t.Errorf("expected 8-character correlation ID, got %d", len(id))
	}
}

func TestRequestIDContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Without request ID
	id := RequestIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty request ID, got %s", id)
	}

	// With request ID
	ctx = ContextWithRequestID(ctx, "req-456")
	id = RequestIDFromContext(ctx)
	if id != "req-456" {
		t.Errorf("expected 'req-456', got '%s'", id)
	}
}

func TestContextWithNewRequestID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = ContextWithNewRequestID(ctx)

	id := RequestIDFromContext(ctx)
	if id == "" {
		t.Error("expected request ID to be generated")
	}
	if len(id) != 36 {
		t.Errorf("expected 36-character request ID, got %d", len(id))
	}
}

func TestContextWithLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	customLogger := zerolog.New(&buf).With().Str("custom", "field").Logger()

	ctx := context.Background()
	ctx = ContextWithLogger(ctx, customLogger)

	retrievedLogger := LoggerFromContext(ctx)
	retrievedLogger.Info().Msg("test")

	output := buf.String()
	if !strings.Contains(output, "custom") {
		t.Errorf("expected custom field in output: %s", output)
	}
}

func TestLoggerFromContext_NoLogger(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := LoggerFromContext(ctx)

	// Should return global logger without panic
	if logger.GetLevel() == zerolog.Disabled {
		t.Error("expected valid logger")
	}
}

func TestCtx(t *testing.T) {
	var buf bytes.Buffer
	SetLogger(zerolog.New(&buf))

	ctx := context.Background()
	ctx = ContextWithCorrelationID(ctx, "corr-123")
	ctx = ContextWithRequestID(ctx, "req-456")

	Ctx(ctx).Info().Msg("context test")

	output := buf.String()
	if !strings.Contains(output, "corr-123") {
		t.Errorf("expected correlation_id in output: %s", output)
	}
	if !strings.Contains(output, "req-456") {
		t.Errorf("expected request_id in output: %s", output)
	}
}

func TestCtxWith(t *testing.T) {
	var buf bytes.Buffer
	SetLogger(zerolog.New(&buf))

	ctx := context.Background()
	ctx = ContextWithCorrelationID(ctx, "corr-789")

	logger := CtxWith(ctx).Str("extra", "field").Logger()
	logger.Info().Msg("ctxwith test")

	output := buf.String()
	if !strings.Contains(output, "corr-789") {
		t.Errorf("expected correlation_id in output: %s", output)
	}
	if !strings.Contains(output, "extra") {
		t.Errorf("expected extra field in output: %s", output)
	}
}

func TestCtxShortcuts(t *testing.T) {
	var buf bytes.Buffer
	SetLogger(zerolog.New(&buf))
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	ctx := context.Background()
	ctx = ContextWithCorrelationID(ctx, "short-123")

	tests := []struct {
		name    string
		logFunc func()
		level   string
	}{
		{"CtxDebug", func() { CtxDebug(ctx).Msg("debug") }, "debug"},
		{"CtxInfo", func() { CtxInfo(ctx).Msg("info") }, "info"},
		{"CtxWarn", func() { CtxWarn(ctx).Msg("warn") }, "warn"},
		{"CtxError", func() { CtxError(ctx).Msg("error") }, "error"},
	}

	for _, tt := range tests {
		buf.Reset()
		tt.logFunc()
		output := buf.String()
		if !strings.Contains(output, tt.level) {
			t.Errorf("%s: expected level '%s' in output: %s", tt.name, tt.level, output)
		}
		if !strings.Contains(output, "short-123") {
			t.Errorf("%s: expected correlation_id in output: %s", tt.name, output)
		}
	}
}

func TestCtxErr(t *testing.T) {
	var buf bytes.Buffer
	SetLogger(zerolog.New(&buf))

	ctx := context.Background()
	ctx = ContextWithCorrelationID(ctx, "err-123")

	testErr := &testError{msg: "test error"}
	CtxErr(ctx, testErr).Msg("error with context")

	output := buf.String()
	if !strings.Contains(output, "err-123") {
		t.Errorf("expected correlation_id in output: %s", output)
	}
	if !strings.Contains(output, "test error") {
		t.Errorf("expected error in output: %s", output)
	}
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	SetLogger(zerolog.New(&buf))

	logger := WithComponent("sync")
	logger.Info().Msg("sync started")

	output := buf.String()
	if !strings.Contains(output, "sync") {
		t.Errorf("expected component in output: %s", output)
	}
}

func TestWithService(t *testing.T) {
	var buf bytes.Buffer
	SetLogger(zerolog.New(&buf))

	logger := WithService("api")
	logger.Info().Msg("api started")

	output := buf.String()
	if !strings.Contains(output, "api") {
		t.Errorf("expected service in output: %s", output)
	}
}
