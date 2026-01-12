// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"testing"
	"time"
)

func TestDefaultResilientReaderConfig(t *testing.T) {
	url := "nats://localhost:4222"
	cfg := DefaultResilientReaderConfig(url)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"NATSURL", cfg.NATSURL, url},
		{"CircuitBreakerName", cfg.CircuitBreakerName, "stream-reader"},
		{"MaxRequests", cfg.MaxRequests, uint32(3)},
		{"Interval", cfg.Interval, 30 * time.Second},
		{"Timeout", cfg.Timeout, 10 * time.Second},
		{"FailureThreshold", cfg.FailureThreshold, uint32(5)},
		{"HealthCheckInterval", cfg.HealthCheckInterval, 30 * time.Second},
		{"EnablePrimaryReader", cfg.EnablePrimaryReader, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.got)
			}
		})
	}
}

func TestResilientReaderConfig_CustomValues(t *testing.T) {
	cfg := ResilientReaderConfig{
		NATSURL:             "nats://custom:4223",
		CircuitBreakerName:  "custom-breaker",
		MaxRequests:         5,
		Interval:            1 * time.Minute,
		Timeout:             20 * time.Second,
		FailureThreshold:    10,
		HealthCheckInterval: 1 * time.Minute,
		EnablePrimaryReader: true,
	}

	if cfg.NATSURL != "nats://custom:4223" {
		t.Errorf("Expected custom NATS URL")
	}
	if cfg.CircuitBreakerName != "custom-breaker" {
		t.Errorf("Expected custom circuit breaker name")
	}
	if cfg.MaxRequests != 5 {
		t.Errorf("Expected MaxRequests=5, got %d", cfg.MaxRequests)
	}
	if cfg.Interval != 1*time.Minute {
		t.Errorf("Expected Interval=1m, got %v", cfg.Interval)
	}
	if cfg.Timeout != 20*time.Second {
		t.Errorf("Expected Timeout=20s, got %v", cfg.Timeout)
	}
	if cfg.FailureThreshold != 10 {
		t.Errorf("Expected FailureThreshold=10, got %d", cfg.FailureThreshold)
	}
	if cfg.HealthCheckInterval != 1*time.Minute {
		t.Errorf("Expected HealthCheckInterval=1m, got %v", cfg.HealthCheckInterval)
	}
	if cfg.EnablePrimaryReader != true {
		t.Errorf("Expected EnablePrimaryReader to be true")
	}
}

func TestReaderStats(t *testing.T) {
	now := time.Now()
	stats := ReaderStats{
		CurrentReader:       ReaderTypeFallback,
		CircuitBreakerState: "closed",
		PrimaryAvailable:    false,
		QueriesTotal:        100,
		ErrorsTotal:         5,
		LastQueryTime:       now,
	}

	if stats.CurrentReader != ReaderTypeFallback {
		t.Errorf("Expected ReaderTypeFallback, got %s", stats.CurrentReader)
	}
	if stats.CircuitBreakerState != "closed" {
		t.Errorf("Expected closed state, got %s", stats.CircuitBreakerState)
	}
	if stats.PrimaryAvailable != false {
		t.Errorf("Expected PrimaryAvailable=false, got %v", stats.PrimaryAvailable)
	}
	if stats.QueriesTotal != 100 {
		t.Errorf("Expected 100 queries, got %d", stats.QueriesTotal)
	}
	if stats.ErrorsTotal != 5 {
		t.Errorf("Expected 5 errors, got %d", stats.ErrorsTotal)
	}
	if stats.LastQueryTime != now {
		t.Errorf("Expected LastQueryTime=%v, got %v", now, stats.LastQueryTime)
	}
}

func TestQueryOptions(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	opts := QueryOptions{
		StartSeq:  1,
		EndSeq:    100,
		StartTime: startTime,
		EndTime:   now,
		Subject:   "playback.>",
		Limit:     500,
		JSONExtract: map[string]string{
			"username": "$.username",
			"title":    "$.title",
		},
	}

	if opts.StartSeq != 1 {
		t.Errorf("Expected StartSeq=1, got %d", opts.StartSeq)
	}
	if opts.EndSeq != 100 {
		t.Errorf("Expected EndSeq=100, got %d", opts.EndSeq)
	}
	if opts.StartTime != startTime {
		t.Errorf("Expected StartTime=%v, got %v", startTime, opts.StartTime)
	}
	if opts.EndTime != now {
		t.Errorf("Expected EndTime=%v, got %v", now, opts.EndTime)
	}
	if opts.Limit != 500 {
		t.Errorf("Expected Limit=500, got %d", opts.Limit)
	}
	if opts.Subject != "playback.>" {
		t.Errorf("Expected Subject='playback.>', got %s", opts.Subject)
	}
	if len(opts.JSONExtract) != 2 {
		t.Errorf("Expected 2 JSON extracts, got %d", len(opts.JSONExtract))
	}
}

func TestStreamMessage(t *testing.T) {
	now := time.Now()
	msg := StreamMessage{
		Sequence:  42,
		Subject:   "playback.start",
		Data:      []byte(`{"user":"test"}`),
		Timestamp: now,
		Headers: map[string][]string{
			"Nats-Msg-Id": {"unique-123"},
		},
	}

	if msg.Sequence != 42 {
		t.Errorf("Expected Sequence=42, got %d", msg.Sequence)
	}
	if msg.Subject != "playback.start" {
		t.Errorf("Expected Subject='playback.start', got %s", msg.Subject)
	}
	if string(msg.Data) != `{"user":"test"}` {
		t.Errorf("Expected JSON data, got %s", string(msg.Data))
	}
	if msg.Timestamp != now {
		t.Errorf("Expected Timestamp=%v, got %v", now, msg.Timestamp)
	}
	if msg.Headers["Nats-Msg-Id"][0] != "unique-123" {
		t.Errorf("Expected message ID header")
	}
}

func TestReaderTypeConstants(t *testing.T) {
	if ReaderTypeNatsJS != "natsjs" {
		t.Errorf("Expected ReaderTypeNatsJS='natsjs', got %s", ReaderTypeNatsJS)
	}
	if ReaderTypeFallback != "fallback" {
		t.Errorf("Expected ReaderTypeFallback='fallback', got %s", ReaderTypeFallback)
	}
}
