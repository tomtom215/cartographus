// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
)

// mockCloser implements io.Closer for testing
type mockCloser struct {
	closed bool
	err    error
}

func (m *mockCloser) Close() error {
	m.closed = true
	return m.err
}

func TestCloseWithLog(t *testing.T) {
	// Note: Subtests should NOT call t.Parallel() to avoid resource contention
	// when combined with parallel database tests using the race detector.

	t.Run("nil closer does not panic", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		// Should not panic
		closeWithLog(nil, logger, "test")

		if buf.Len() > 0 {
			t.Errorf("Expected no log output for nil closer, got: %s", buf.String())
		}
	})

	t.Run("successful close does not log", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		closer := &mockCloser{err: nil}
		closeWithLog(closer, logger, "test resource")

		if !closer.closed {
			t.Error("Expected closer to be closed")
		}
		if buf.Len() > 0 {
			t.Errorf("Expected no log output for successful close, got: %s", buf.String())
		}
	})

	t.Run("error during close is logged", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		closeErr := errors.New("close failed: connection reset")
		closer := &mockCloser{err: closeErr}
		closeWithLog(closer, logger, "database connection")

		if !closer.closed {
			t.Error("Expected closer to be closed")
		}
		logOutput := buf.String()
		if !strings.Contains(logOutput, "failed to close resource") {
			t.Errorf("Expected log to contain 'failed to close resource', got: %s", logOutput)
		}
		if !strings.Contains(logOutput, "database connection") {
			t.Errorf("Expected log to contain resource type 'database connection', got: %s", logOutput)
		}
		if !strings.Contains(logOutput, "close failed: connection reset") {
			t.Errorf("Expected log to contain error message, got: %s", logOutput)
		}
	})

	t.Run("nil logger falls back to zerolog", func(t *testing.T) {
		closeErr := errors.New("close failed")
		closer := &mockCloser{err: closeErr}

		// Should not panic with nil logger, falls back to zerolog
		closeWithLog(closer, nil, "test resource")

		if !closer.closed {
			t.Error("Expected closer to be closed")
		}
	})

	t.Run("multiple close calls", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		closer := &mockCloser{err: nil}
		closeWithLog(closer, logger, "first")
		closeWithLog(closer, logger, "second")

		// mockCloser doesn't prevent double close, but real resources might error
		if buf.Len() > 0 {
			t.Errorf("Expected no log output for successful closes, got: %s", buf.String())
		}
	})
}

func TestCloseQuietly(t *testing.T) {
	t.Run("nil closer does not panic", func(t *testing.T) {
		// Should not panic
		closeQuietly(nil)
	})

	t.Run("successful close is silent", func(t *testing.T) {
		closer := &mockCloser{err: nil}
		closeQuietly(closer)

		if !closer.closed {
			t.Error("Expected closer to be closed")
		}
	})

	t.Run("error during close is ignored", func(t *testing.T) {
		closer := &mockCloser{err: errors.New("close failed")}
		closeQuietly(closer)

		if !closer.closed {
			t.Error("Expected closer to be closed even with error")
		}
		// No assertion on error since it's intentionally ignored
	})

	t.Run("works with various io.Closer implementations", func(t *testing.T) {
		// Test with strings.Reader wrapped in NopCloser
		reader := strings.NewReader("test data")
		nopCloser := io.NopCloser(reader)
		closeQuietly(nopCloser) // Should not panic
	})
}

// Test various io.Closer implementations
func TestCloseWithLog_IoCloserTypes(t *testing.T) {
	t.Run("bytes.Buffer wrapped in NopCloser", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		dataBuffer := bytes.NewBuffer([]byte("test data"))
		nopCloser := io.NopCloser(dataBuffer)

		closeWithLog(nopCloser, logger, "buffer")

		if buf.Len() > 0 {
			t.Errorf("NopCloser should never error, got: %s", buf.String())
		}
	})
}

// Benchmark tests
func BenchmarkCloseWithLog_Success(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	closer := &mockCloser{err: nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		closer.closed = false
		closeWithLog(closer, logger, "benchmark")
	}
}

func BenchmarkCloseWithLog_WithError(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	closer := &mockCloser{err: errors.New("error")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		closer.closed = false
		closeWithLog(closer, logger, "benchmark")
	}
}

func BenchmarkCloseQuietly(b *testing.B) {
	closer := &mockCloser{err: nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		closer.closed = false
		closeQuietly(closer)
	}
}
