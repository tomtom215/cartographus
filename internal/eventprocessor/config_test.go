// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"os"
	"testing"
	"time"
)

func TestDefaultNATSConfig(t *testing.T) {
	cfg := DefaultNATSConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Enabled", cfg.Enabled, false},
		{"URL", cfg.URL, "nats://127.0.0.1:4222"},
		{"EmbeddedServer", cfg.EmbeddedServer, true},
		{"StoreDir", cfg.StoreDir, "/data/nats/jetstream"},
		{"MaxMemory", cfg.MaxMemory, int64(1 << 30)},
		{"MaxStore", cfg.MaxStore, int64(10 << 30)},
		{"StreamRetentionDays", cfg.StreamRetentionDays, 7},
		{"BatchSize", cfg.BatchSize, 1000},
		{"FlushInterval", cfg.FlushInterval, 5 * time.Second},
		{"SubscribersCount", cfg.SubscribersCount, 4},
		{"DurableName", cfg.DurableName, "media-processor"},
		{"QueueGroup", cfg.QueueGroup, "processors"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("DefaultNATSConfig().%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestLoadNATSConfig(t *testing.T) {
	// Save original env values
	origEnv := make(map[string]string)
	envVars := []string{
		"NATS_ENABLED",
		"NATS_URL",
		"NATS_EMBEDDED",
		"NATS_STORE_DIR",
		"NATS_MAX_MEMORY",
		"NATS_MAX_STORE",
		"NATS_RETENTION_DAYS",
		"NATS_BATCH_SIZE",
		"NATS_FLUSH_INTERVAL",
		"NATS_SUBSCRIBERS",
		"NATS_DURABLE_NAME",
		"NATS_QUEUE_GROUP",
	}
	for _, key := range envVars {
		origEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}

	// Restore original env after test
	defer func() {
		for key, value := range origEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("defaults when env vars not set", func(t *testing.T) {
		cfg := LoadNATSConfig()
		if cfg.URL != "nats://127.0.0.1:4222" {
			t.Errorf("Expected default URL, got %s", cfg.URL)
		}
		if cfg.Enabled {
			t.Error("Expected Enabled=false by default")
		}
	})

	t.Run("loads from environment", func(t *testing.T) {
		os.Setenv("NATS_ENABLED", "true")
		os.Setenv("NATS_URL", "nats://custom:4222")
		os.Setenv("NATS_EMBEDDED", "false")
		os.Setenv("NATS_STORE_DIR", "/custom/path")
		os.Setenv("NATS_MAX_MEMORY", "2147483648")
		os.Setenv("NATS_MAX_STORE", "21474836480")
		os.Setenv("NATS_RETENTION_DAYS", "14")
		os.Setenv("NATS_BATCH_SIZE", "500")
		os.Setenv("NATS_FLUSH_INTERVAL", "10s")
		os.Setenv("NATS_SUBSCRIBERS", "8")
		os.Setenv("NATS_DURABLE_NAME", "custom-processor")
		os.Setenv("NATS_QUEUE_GROUP", "custom-group")

		cfg := LoadNATSConfig()

		if !cfg.Enabled {
			t.Error("Expected Enabled=true")
		}
		if cfg.URL != "nats://custom:4222" {
			t.Errorf("Expected custom URL, got %s", cfg.URL)
		}
		if cfg.EmbeddedServer {
			t.Error("Expected EmbeddedServer=false")
		}
		if cfg.StoreDir != "/custom/path" {
			t.Errorf("Expected custom StoreDir, got %s", cfg.StoreDir)
		}
		if cfg.MaxMemory != 2147483648 {
			t.Errorf("Expected MaxMemory=2GB, got %d", cfg.MaxMemory)
		}
		if cfg.MaxStore != 21474836480 {
			t.Errorf("Expected MaxStore=20GB, got %d", cfg.MaxStore)
		}
		if cfg.StreamRetentionDays != 14 {
			t.Errorf("Expected RetentionDays=14, got %d", cfg.StreamRetentionDays)
		}
		if cfg.BatchSize != 500 {
			t.Errorf("Expected BatchSize=500, got %d", cfg.BatchSize)
		}
		if cfg.FlushInterval != 10*time.Second {
			t.Errorf("Expected FlushInterval=10s, got %v", cfg.FlushInterval)
		}
		if cfg.SubscribersCount != 8 {
			t.Errorf("Expected SubscribersCount=8, got %d", cfg.SubscribersCount)
		}
		if cfg.DurableName != "custom-processor" {
			t.Errorf("Expected DurableName=custom-processor, got %s", cfg.DurableName)
		}
		if cfg.QueueGroup != "custom-group" {
			t.Errorf("Expected QueueGroup=custom-group, got %s", cfg.QueueGroup)
		}
	})

	t.Run("handles invalid values gracefully", func(t *testing.T) {
		os.Setenv("NATS_MAX_MEMORY", "invalid")
		os.Setenv("NATS_BATCH_SIZE", "not-a-number")
		os.Setenv("NATS_FLUSH_INTERVAL", "bad-duration")

		cfg := LoadNATSConfig()

		// Should fall back to defaults
		if cfg.MaxMemory != int64(1<<30) {
			t.Errorf("Expected default MaxMemory on invalid input, got %d", cfg.MaxMemory)
		}
		if cfg.BatchSize != 1000 {
			t.Errorf("Expected default BatchSize on invalid input, got %d", cfg.BatchSize)
		}
		if cfg.FlushInterval != 5*time.Second {
			t.Errorf("Expected default FlushInterval on invalid input, got %v", cfg.FlushInterval)
		}
	})
}

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Expected Host=127.0.0.1, got %s", cfg.Host)
	}
	if cfg.Port != 4222 {
		t.Errorf("Expected Port=4222, got %d", cfg.Port)
	}
	if cfg.StoreDir != "/data/nats/jetstream" {
		t.Errorf("Expected StoreDir=/data/nats/jetstream, got %s", cfg.StoreDir)
	}
	if cfg.JetStreamMaxMem != int64(1<<30) {
		t.Errorf("Expected JetStreamMaxMem=1GB, got %d", cfg.JetStreamMaxMem)
	}
	if cfg.JetStreamMaxStore != int64(10<<30) {
		t.Errorf("Expected JetStreamMaxStore=10GB, got %d", cfg.JetStreamMaxStore)
	}
	if cfg.EnableClustering {
		t.Error("Expected EnableClustering=false")
	}
}

func TestDefaultPublisherConfig(t *testing.T) {
	url := "nats://test:4222"
	cfg := DefaultPublisherConfig(url)

	if cfg.URL != url {
		t.Errorf("Expected URL=%s, got %s", url, cfg.URL)
	}
	if cfg.MaxReconnects != -1 {
		t.Errorf("Expected MaxReconnects=-1 (unlimited), got %d", cfg.MaxReconnects)
	}
	if cfg.ReconnectWait != 2*time.Second {
		t.Errorf("Expected ReconnectWait=2s, got %v", cfg.ReconnectWait)
	}
	if cfg.ReconnectBuffer != 8*1024*1024 {
		t.Errorf("Expected ReconnectBuffer=8MB, got %d", cfg.ReconnectBuffer)
	}
	if !cfg.EnableTrackMsgID {
		t.Error("Expected EnableTrackMsgID=true")
	}
}

func TestDefaultSubscriberConfig(t *testing.T) {
	url := "nats://test:4222"
	cfg := DefaultSubscriberConfig(url)

	if cfg.URL != url {
		t.Errorf("Expected URL=%s, got %s", url, cfg.URL)
	}
	if cfg.DurableName != "media-processor" {
		t.Errorf("Expected DurableName=media-processor, got %s", cfg.DurableName)
	}
	if cfg.QueueGroup != "processors" {
		t.Errorf("Expected QueueGroup=processors, got %s", cfg.QueueGroup)
	}
	if cfg.SubscribersCount != 4 {
		t.Errorf("Expected SubscribersCount=4, got %d", cfg.SubscribersCount)
	}
	if cfg.AckWaitTimeout != 30*time.Second {
		t.Errorf("Expected AckWaitTimeout=30s, got %v", cfg.AckWaitTimeout)
	}
	if cfg.MaxDeliver != 5 {
		t.Errorf("Expected MaxDeliver=5, got %d", cfg.MaxDeliver)
	}
	if cfg.MaxAckPending != 1000 {
		t.Errorf("Expected MaxAckPending=1000, got %d", cfg.MaxAckPending)
	}
}

func TestDefaultStreamConfig(t *testing.T) {
	cfg := DefaultStreamConfig()

	if cfg.Name != "MEDIA_EVENTS" {
		t.Errorf("Expected Name=MEDIA_EVENTS, got %s", cfg.Name)
	}
	if len(cfg.Subjects) != 4 {
		t.Errorf("Expected 4 subjects, got %d", len(cfg.Subjects))
	}
	if cfg.MaxAge != 7*24*time.Hour {
		t.Errorf("Expected MaxAge=7 days, got %v", cfg.MaxAge)
	}
	if cfg.MaxBytes != 10*1024*1024*1024 {
		t.Errorf("Expected MaxBytes=10GB, got %d", cfg.MaxBytes)
	}
	if cfg.DuplicateWindow != 2*time.Minute {
		t.Errorf("Expected DuplicateWindow=2m, got %v", cfg.DuplicateWindow)
	}
	if cfg.Replicas != 1 {
		t.Errorf("Expected Replicas=1, got %d", cfg.Replicas)
	}
}

func TestDefaultAppenderConfig(t *testing.T) {
	cfg := DefaultAppenderConfig()

	if cfg.Table != "playback_events" {
		t.Errorf("Expected Table=playback_events, got %s", cfg.Table)
	}
	if cfg.BatchSize != 1000 {
		t.Errorf("Expected BatchSize=1000, got %d", cfg.BatchSize)
	}
	if cfg.FlushInterval != 5*time.Second {
		t.Errorf("Expected FlushInterval=5s, got %v", cfg.FlushInterval)
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	name := "test-breaker"
	cfg := DefaultCircuitBreakerConfig(name)

	if cfg.Name != name {
		t.Errorf("Expected Name=%s, got %s", name, cfg.Name)
	}
	if cfg.MaxRequests != 3 {
		t.Errorf("Expected MaxRequests=3, got %d", cfg.MaxRequests)
	}
	if cfg.Interval != 30*time.Second {
		t.Errorf("Expected Interval=30s, got %v", cfg.Interval)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Expected Timeout=10s, got %v", cfg.Timeout)
	}
	if cfg.FailureThreshold != 5 {
		t.Errorf("Expected FailureThreshold=5, got %d", cfg.FailureThreshold)
	}
}
