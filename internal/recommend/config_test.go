// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package recommend

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("weights sum to approximately 1", func(t *testing.T) {
		sum := cfg.Weights.EASE + cfg.Weights.ALS + cfg.Weights.UserCF +
			cfg.Weights.ItemCF + cfg.Weights.Content + cfg.Weights.CoVisit +
			cfg.Weights.SASRec + cfg.Weights.FPMC + cfg.Weights.Popularity + cfg.Weights.Recency

		if sum < 0.99 || sum > 1.01 {
			t.Errorf("weights sum = %f, want ~1.0", sum)
		}
	})

	t.Run("EASE config has valid defaults", func(t *testing.T) {
		if cfg.EASE.Lambda <= 0 {
			t.Errorf("EASE.Lambda = %f, want > 0", cfg.EASE.Lambda)
		}
		if cfg.EASE.MaxItems <= 0 {
			t.Errorf("EASE.MaxItems = %d, want > 0", cfg.EASE.MaxItems)
		}
	})

	t.Run("ALS config has valid defaults", func(t *testing.T) {
		if cfg.ALS.Factors <= 0 {
			t.Errorf("ALS.Factors = %d, want > 0", cfg.ALS.Factors)
		}
		if cfg.ALS.Iterations <= 0 {
			t.Errorf("ALS.Iterations = %d, want > 0", cfg.ALS.Iterations)
		}
	})

	t.Run("training config has valid defaults", func(t *testing.T) {
		if cfg.Training.Interval <= 0 {
			t.Errorf("Training.Interval = %v, want > 0", cfg.Training.Interval)
		}
		if cfg.Training.Timeout <= 0 {
			t.Errorf("Training.Timeout = %v, want > 0", cfg.Training.Timeout)
		}
	})

	t.Run("limits config has valid defaults", func(t *testing.T) {
		if cfg.Limits.DefaultK <= 0 {
			t.Errorf("Limits.DefaultK = %d, want > 0", cfg.Limits.DefaultK)
		}
		if cfg.Limits.MaxK < cfg.Limits.DefaultK {
			t.Errorf("Limits.MaxK = %d, want >= DefaultK (%d)", cfg.Limits.MaxK, cfg.Limits.DefaultK)
		}
	})

	t.Run("seed is set for determinism", func(t *testing.T) {
		if cfg.Seed == 0 {
			t.Error("Seed = 0, want non-zero for determinism")
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	validConfig := func() *Config {
		return DefaultConfig()
	}

	tests := []struct {
		name      string
		modify    func(*Config)
		wantError bool
	}{
		{
			name:      "valid default config",
			modify:    func(c *Config) {},
			wantError: false,
		},
		{
			name:      "negative EASE lambda",
			modify:    func(c *Config) { c.EASE.Lambda = -1 },
			wantError: true,
		},
		{
			name:      "zero EASE max items",
			modify:    func(c *Config) { c.EASE.MaxItems = 0 },
			wantError: true,
		},
		{
			name:      "zero ALS factors",
			modify:    func(c *Config) { c.ALS.Factors = 0 },
			wantError: true,
		},
		{
			name:      "negative ALS lambda",
			modify:    func(c *Config) { c.ALS.Lambda = -0.1 },
			wantError: true,
		},
		{
			name:      "MMR lambda > 1",
			modify:    func(c *Config) { c.Diversity.MMRLambda = 1.5 },
			wantError: true,
		},
		{
			name:      "MMR lambda < 0",
			modify:    func(c *Config) { c.Diversity.MMRLambda = -0.5 },
			wantError: true,
		},
		{
			name:      "zero training timeout",
			modify:    func(c *Config) { c.Training.Timeout = 0 },
			wantError: true,
		},
		{
			name:      "zero max candidates",
			modify:    func(c *Config) { c.Limits.MaxCandidates = 0 },
			wantError: true,
		},
		{
			name:      "MaxK less than DefaultK",
			modify:    func(c *Config) { c.Limits.MaxK = 5; c.Limits.DefaultK = 10 },
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if tt.wantError && err == nil {
				t.Error("Validate() = nil, want error")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestAlgorithmWeights_Normalize(t *testing.T) {
	tests := []struct {
		name    string
		weights AlgorithmWeights
	}{
		{
			name: "already normalized",
			weights: AlgorithmWeights{
				EASE: 0.1, ALS: 0.1, UserCF: 0.1, ItemCF: 0.1, Content: 0.1,
				CoVisit: 0.1, SASRec: 0.1, FPMC: 0.1, Popularity: 0.1, Recency: 0.1,
			},
		},
		{
			name: "unequal weights",
			weights: AlgorithmWeights{
				EASE: 0.5, ALS: 0.3, Content: 0.2,
			},
		},
		{
			name:    "all zeros returns equal weights",
			weights: AlgorithmWeights{},
		},
		{
			name: "large values",
			weights: AlgorithmWeights{
				EASE: 100, ALS: 200, Content: 300,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := tt.weights.Normalize()

			sum := normalized.EASE + normalized.ALS + normalized.UserCF +
				normalized.ItemCF + normalized.Content + normalized.CoVisit +
				normalized.SASRec + normalized.FPMC + normalized.Popularity + normalized.Recency +
				normalized.BPR + normalized.TimeAwareCF + normalized.MultiHopItemCF + normalized.MarkovChain

			if sum < 0.99 || sum > 1.01 {
				t.Errorf("normalized weights sum = %f, want ~1.0", sum)
			}
		})
	}
}

func TestAlgorithmWeights_ToMap(t *testing.T) {
	weights := AlgorithmWeights{
		EASE:       0.15,
		ALS:        0.15,
		UserCF:     0.10,
		ItemCF:     0.10,
		Content:    0.15,
		CoVisit:    0.10,
		SASRec:     0.10,
		FPMC:       0.05,
		Popularity: 0.05,
		Recency:    0.05,
	}

	m := weights.ToMap()

	tests := []struct {
		key      string
		expected float64
	}{
		{"ease", 0.15},
		{"als", 0.15},
		{"user_cf", 0.10},
		{"item_cf", 0.10},
		{"content", 0.15},
		{"covisit", 0.10},
		{"sasrec", 0.10},
		{"fpmc", 0.05},
		{"popularity", 0.05},
		{"recency", 0.05},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if m[tt.key] != tt.expected {
				t.Errorf("ToMap()[%q] = %f, want %f", tt.key, m[tt.key], tt.expected)
			}
		})
	}
}

func TestConfig_Clone(t *testing.T) {
	original := DefaultConfig()
	original.EASE.Lambda = 999
	original.Training.Interval = 48 * time.Hour

	clone := original.Clone()

	t.Run("clone has same values", func(t *testing.T) {
		if clone.EASE.Lambda != original.EASE.Lambda {
			t.Errorf("clone.EASE.Lambda = %f, want %f", clone.EASE.Lambda, original.EASE.Lambda)
		}
	})

	t.Run("clone is independent", func(t *testing.T) {
		clone.EASE.Lambda = 123
		if original.EASE.Lambda == clone.EASE.Lambda {
			t.Error("modifying clone affected original")
		}
	})
}

func TestConfig_MarshalJSON(t *testing.T) {
	cfg := DefaultConfig()

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Check that duration fields are strings
	t.Run("training interval is string", func(t *testing.T) {
		training, ok := parsed["training"].(map[string]interface{})
		if !ok {
			t.Fatal("training field not found or wrong type")
		}
		interval, ok := training["interval"].(string)
		if !ok {
			t.Error("training.interval is not a string")
		}
		if interval == "" {
			t.Error("training.interval is empty")
		}
	})
}
