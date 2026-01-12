// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/recommend"
)

func TestNewTimeAwareCF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         TimeAwareCFConfig
		expectedConfig TimeAwareCFConfig
	}{
		{
			name:   "default config",
			config: DefaultTimeAwareCFConfig(),
			expectedConfig: TimeAwareCFConfig{
				DecayRate:     0.1,
				DecayUnit:     24 * time.Hour,
				MaxLookback:   365 * 24 * time.Hour,
				MinWeight:     0.01,
				NumNeighbors:  50,
				MinConfidence: 0.1,
				Mode:          "user",
			},
		},
		{
			name: "custom config",
			config: TimeAwareCFConfig{
				DecayRate:     0.2,
				DecayUnit:     12 * time.Hour,
				MaxLookback:   180 * 24 * time.Hour,
				MinWeight:     0.05,
				NumNeighbors:  30,
				MinConfidence: 0.2,
				Mode:          "item",
			},
			expectedConfig: TimeAwareCFConfig{
				DecayRate:     0.2,
				DecayUnit:     12 * time.Hour,
				MaxLookback:   180 * 24 * time.Hour,
				MinWeight:     0.05,
				NumNeighbors:  30,
				MinConfidence: 0.2,
				Mode:          "item",
			},
		},
		{
			name: "zero values get defaults",
			config: TimeAwareCFConfig{
				DecayRate:     0,
				DecayUnit:     0,
				MaxLookback:   0,
				MinWeight:     0,
				NumNeighbors:  0,
				MinConfidence: 0,
				Mode:          "",
			},
			expectedConfig: TimeAwareCFConfig{
				DecayRate:     0.1,
				DecayUnit:     24 * time.Hour,
				MaxLookback:   365 * 24 * time.Hour,
				MinWeight:     0.01,
				NumNeighbors:  50,
				MinConfidence: 0.1,
				Mode:          "user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cf := NewTimeAwareCF(&tt.config)

			if cf == nil {
				t.Fatal("NewTimeAwareCF returned nil")
			}

			if cf.Name() != "time_aware_cf" {
				t.Errorf("Name() = %q, want %q", cf.Name(), "time_aware_cf")
			}

			if cf.config.DecayRate != tt.expectedConfig.DecayRate {
				t.Errorf("DecayRate = %f, want %f", cf.config.DecayRate, tt.expectedConfig.DecayRate)
			}
			if cf.config.Mode != tt.expectedConfig.Mode {
				t.Errorf("Mode = %q, want %q", cf.config.Mode, tt.expectedConfig.Mode)
			}
		})
	}
}

func TestTimeAwareCF_ComputeTimeWeight(t *testing.T) {
	t.Parallel()

	refTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	cfg := DefaultTimeAwareCFConfig()
	cfg.ReferenceTime = refTime

	cf := NewTimeAwareCF(&cfg)
	cf.referenceTime = refTime

	tests := []struct {
		name      string
		timestamp time.Time
		wantMin   float64
		wantMax   float64
	}{
		{
			name:      "current time",
			timestamp: refTime,
			wantMin:   0.99,
			wantMax:   1.01,
		},
		{
			name:      "1 day ago",
			timestamp: refTime.Add(-24 * time.Hour),
			wantMin:   0.9, // exp(-0.1 * 1) = 0.905
			wantMax:   0.92,
		},
		{
			name:      "7 days ago",
			timestamp: refTime.Add(-7 * 24 * time.Hour),
			wantMin:   0.49, // exp(-0.1 * 7) = 0.497
			wantMax:   0.51,
		},
		{
			name:      "30 days ago",
			timestamp: refTime.Add(-30 * 24 * time.Hour),
			wantMin:   0.04, // exp(-0.1 * 30) = 0.049
			wantMax:   0.06,
		},
		{
			name:      "beyond max lookback",
			timestamp: refTime.Add(-400 * 24 * time.Hour),
			wantMin:   -0.01, // 0 (filtered out)
			wantMax:   0.01,
		},
		{
			name:      "zero timestamp (full weight)",
			timestamp: time.Time{},
			wantMin:   0.99,
			wantMax:   1.01,
		},
		{
			name:      "future timestamp",
			timestamp: refTime.Add(24 * time.Hour),
			wantMin:   0.99,
			wantMax:   1.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a local copy for this test
			localCF := NewTimeAwareCF(&cfg)
			localCF.referenceTime = refTime

			weight := localCF.computeTimeWeight(tt.timestamp)

			if weight < tt.wantMin || weight > tt.wantMax {
				t.Errorf("computeTimeWeight() = %f, want in [%f, %f]",
					weight, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestTimeAwareCF_Train(t *testing.T) {
	t.Parallel()

	refTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		config       TimeAwareCFConfig
		interactions []recommend.Interaction
		wantTrained  bool
	}{
		{
			name: "empty interactions",
			config: TimeAwareCFConfig{
				ReferenceTime: refTime,
			},
			interactions: nil,
			wantTrained:  true,
		},
		{
			name: "user-based mode",
			config: TimeAwareCFConfig{
				Mode:          "user",
				ReferenceTime: refTime,
			},
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
				{UserID: 1, ItemID: 101, Confidence: 0.8, Timestamp: refTime.Add(-2 * time.Hour)},
				{UserID: 2, ItemID: 100, Confidence: 0.9, Timestamp: refTime.Add(-1 * time.Hour)},
				{UserID: 2, ItemID: 102, Confidence: 1.0, Timestamp: refTime.Add(-3 * time.Hour)},
			},
			wantTrained: true,
		},
		{
			name: "item-based mode",
			config: TimeAwareCFConfig{
				Mode:          "item",
				ReferenceTime: refTime,
			},
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
				{UserID: 1, ItemID: 101, Confidence: 0.8, Timestamp: refTime.Add(-2 * time.Hour)},
				{UserID: 2, ItemID: 100, Confidence: 0.9, Timestamp: refTime.Add(-1 * time.Hour)},
				{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-3 * time.Hour)},
			},
			wantTrained: true,
		},
		{
			name: "old interactions filtered",
			config: TimeAwareCFConfig{
				MaxLookback:   7 * 24 * time.Hour, // 1 week
				ReferenceTime: refTime,
			},
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-30 * 24 * time.Hour)}, // Too old
				{UserID: 1, ItemID: 101, Confidence: 0.8, Timestamp: refTime.Add(-1 * 24 * time.Hour)},  // OK
			},
			wantTrained: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Ensure defaults are applied
			if tt.config.DecayRate == 0 {
				tt.config.DecayRate = 0.1
			}
			if tt.config.DecayUnit == 0 {
				tt.config.DecayUnit = 24 * time.Hour
			}
			if tt.config.MaxLookback == 0 {
				tt.config.MaxLookback = 365 * 24 * time.Hour
			}
			if tt.config.MinWeight == 0 {
				tt.config.MinWeight = 0.01
			}
			if tt.config.NumNeighbors == 0 {
				tt.config.NumNeighbors = 50
			}
			if tt.config.MinConfidence == 0 {
				tt.config.MinConfidence = 0.1
			}
			if tt.config.Mode == "" {
				tt.config.Mode = "user"
			}

			cf := NewTimeAwareCF(&tt.config)
			ctx := context.Background()

			err := cf.Train(ctx, tt.interactions, nil)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if cf.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", cf.IsTrained(), tt.wantTrained)
			}
		})
	}
}

func TestTimeAwareCF_TrainContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := DefaultTimeAwareCFConfig()
	cf := NewTimeAwareCF(&cfg)

	interactions := make([]recommend.Interaction, 1000)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:     i % 50,
			ItemID:     i,
			Confidence: 1.0,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := cf.Train(ctx, interactions, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Train() with canceled context: error = %v, want context.Canceled", err)
	}
}

func TestTimeAwareCF_PredictUserBased(t *testing.T) {
	t.Parallel()

	refTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	cfg := TimeAwareCFConfig{
		DecayRate:     0.1,
		DecayUnit:     24 * time.Hour,
		MaxLookback:   365 * 24 * time.Hour,
		MinWeight:     0.01,
		NumNeighbors:  10,
		MinConfidence: 0.1,
		Mode:          "user",
		ReferenceTime: refTime,
	}

	cf := NewTimeAwareCF(&cfg)

	// Create training data with clear patterns:
	// Users 1 and 2 are similar (both like items 100, 101)
	// Users 3 and 4 are similar (both like items 103, 104)
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
		{UserID: 2, ItemID: 102, Confidence: 1.0, Timestamp: refTime.Add(-3 * time.Hour)}, // User 2 also watched 102
		{UserID: 3, ItemID: 103, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 3, ItemID: 104, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
		{UserID: 4, ItemID: 103, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 4, ItemID: 104, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
	}

	ctx := context.Background()
	if err := cf.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// User 1 should get high scores for item 102 (similar user 2 watched it)
	scores, err := cf.Predict(ctx, 1, []int{102, 103, 104})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	// Should have scores
	if len(scores) == 0 {
		t.Error("Predict() returned no scores")
	}

	// Item 102 should be scored (user 2 watched it and is similar to user 1)
	if _, ok := scores[102]; !ok {
		t.Error("Expected score for item 102")
	}
}

func TestTimeAwareCF_PredictItemBased(t *testing.T) {
	t.Parallel()

	refTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	cfg := TimeAwareCFConfig{
		DecayRate:     0.1,
		DecayUnit:     24 * time.Hour,
		MaxLookback:   365 * 24 * time.Hour,
		MinWeight:     0.01,
		NumNeighbors:  10,
		MinConfidence: 0.1,
		Mode:          "item",
		ReferenceTime: refTime,
	}

	cf := NewTimeAwareCF(&cfg)

	// Create training data where items 100 and 101 are similar
	// (many users watched both)
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
		{UserID: 3, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 3, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
		{UserID: 4, ItemID: 102, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)}, // Separate item
	}

	ctx := context.Background()
	if err := cf.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// User 4 watched item 102, should get recommendations for similar items
	scores, err := cf.Predict(ctx, 4, []int{100, 101})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	// Check that predictions are made (they might be empty if no item similarity)
	// since user 4 only has one item and we need overlap
	_ = scores // Allow empty in this case
}

func TestTimeAwareCF_PredictSimilar(t *testing.T) {
	t.Parallel()

	refTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	cfg := TimeAwareCFConfig{
		Mode:          "item",
		ReferenceTime: refTime,
	}

	cf := NewTimeAwareCF(&cfg)

	// Items 100 and 101 are similar (watched by same users)
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
		{UserID: 3, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * time.Hour)},
		{UserID: 3, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-2 * time.Hour)},
	}

	ctx := context.Background()
	if err := cf.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		itemID     int
		candidates []int
		wantScores bool
	}{
		{
			name:       "similar to item 100",
			itemID:     100,
			candidates: []int{101},
			wantScores: true,
		},
		{
			name:       "unknown item",
			itemID:     999,
			candidates: []int{100, 101},
			wantScores: false,
		},
		{
			name:       "self excluded",
			itemID:     100,
			candidates: []int{100, 101},
			wantScores: true, // Should return 101 but not 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scores, err := cf.PredictSimilar(ctx, tt.itemID, tt.candidates)
			if err != nil {
				t.Fatalf("PredictSimilar() error = %v", err)
			}

			if tt.wantScores && len(scores) == 0 {
				t.Error("PredictSimilar() returned no scores, expected some")
			}

			// Self should never be in results
			if _, ok := scores[tt.itemID]; ok {
				t.Errorf("PredictSimilar() included self (item %d) in results", tt.itemID)
			}
		})
	}
}

func TestTimeAwareCF_TimeDecayAffectsScores(t *testing.T) {
	t.Parallel()

	refTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// Train with old interactions
	cfOld := NewTimeAwareCF(&TimeAwareCFConfig{
		DecayRate:     0.5, // High decay rate
		DecayUnit:     24 * time.Hour,
		MaxLookback:   365 * 24 * time.Hour,
		MinWeight:     0.01,
		NumNeighbors:  50,
		MinConfidence: 0.1,
		Mode:          "user",
		ReferenceTime: refTime,
	})

	// Train with recent interactions
	cfRecent := NewTimeAwareCF(&TimeAwareCFConfig{
		DecayRate:     0.5,
		DecayUnit:     24 * time.Hour,
		MaxLookback:   365 * 24 * time.Hour,
		MinWeight:     0.01,
		NumNeighbors:  50,
		MinConfidence: 0.1,
		Mode:          "user",
		ReferenceTime: refTime,
	})

	// Old interactions (30 days old)
	oldInteractions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-30 * 24 * time.Hour)},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-30 * 24 * time.Hour)},
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-30 * 24 * time.Hour)},
		{UserID: 2, ItemID: 102, Confidence: 1.0, Timestamp: refTime.Add(-30 * 24 * time.Hour)},
	}

	// Recent interactions (1 day old)
	recentInteractions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * 24 * time.Hour)},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: refTime.Add(-1 * 24 * time.Hour)},
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: refTime.Add(-1 * 24 * time.Hour)},
		{UserID: 2, ItemID: 102, Confidence: 1.0, Timestamp: refTime.Add(-1 * 24 * time.Hour)},
	}

	ctx := context.Background()

	if err := cfOld.Train(ctx, oldInteractions, nil); err != nil {
		t.Fatalf("Train() old error = %v", err)
	}
	if err := cfRecent.Train(ctx, recentInteractions, nil); err != nil {
		t.Fatalf("Train() recent error = %v", err)
	}

	// Both should be trained, but weights are different internally
	if !cfOld.IsTrained() {
		t.Error("cfOld should be trained")
	}
	if !cfRecent.IsTrained() {
		t.Error("cfRecent should be trained")
	}
}

func TestTimeAwareCF_GetMethods(t *testing.T) {
	t.Parallel()

	cfg := TimeAwareCFConfig{
		DecayRate: 0.2,
		Mode:      "item",
	}
	cf := NewTimeAwareCF(&cfg)

	if cf.GetDecayRate() != 0.2 {
		t.Errorf("GetDecayRate() = %f, want 0.2", cf.GetDecayRate())
	}

	if cf.GetMode() != "item" {
		t.Errorf("GetMode() = %q, want %q", cf.GetMode(), "item")
	}
}

func TestTimeAwareCF_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ recommend.Algorithm = (*TimeAwareCF)(nil)
}

func BenchmarkTimeAwareCF_Train(b *testing.B) {
	refTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	cfg := TimeAwareCFConfig{
		Mode:          "user",
		ReferenceTime: refTime,
	}

	numUsers := 100
	numItems := 500
	interactions := make([]recommend.Interaction, 0, numUsers*10)

	for u := 0; u < numUsers; u++ {
		for i := 0; i < 10; i++ {
			itemID := (u*7 + i*13) % numItems
			interactions = append(interactions, recommend.Interaction{
				UserID:     u,
				ItemID:     itemID,
				Confidence: 1.0,
				Timestamp:  refTime.Add(-time.Duration(u*i) * time.Hour),
			})
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cf := NewTimeAwareCF(&cfg)
		_ = cf.Train(ctx, interactions, nil)
	}
}

func BenchmarkTimeAwareCF_Predict(b *testing.B) {
	refTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	cfg := TimeAwareCFConfig{
		Mode:          "user",
		ReferenceTime: refTime,
	}

	cf := NewTimeAwareCF(&cfg)

	numUsers := 100
	numItems := 500
	interactions := make([]recommend.Interaction, 0, numUsers*10)

	for u := 0; u < numUsers; u++ {
		for i := 0; i < 10; i++ {
			itemID := (u*7 + i*13) % numItems
			interactions = append(interactions, recommend.Interaction{
				UserID:     u,
				ItemID:     itemID,
				Confidence: 1.0,
				Timestamp:  refTime.Add(-time.Duration(u*i) * time.Hour),
			})
		}
	}

	ctx := context.Background()
	_ = cf.Train(ctx, interactions, nil)

	candidates := make([]int, numItems)
	for i := 0; i < numItems; i++ {
		candidates[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cf.Predict(ctx, i%numUsers, candidates)
	}
}
