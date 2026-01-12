// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/recommend"
)

func TestNewFPMC(t *testing.T) {
	tests := []struct {
		name   string
		cfg    FPMCConfig
		verify func(t *testing.T, f *FPMC)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  FPMCConfig{},
			verify: func(t *testing.T, f *FPMC) {
				if f.config.NumFactors <= 0 {
					t.Errorf("NumFactors = %d, want > 0", f.config.NumFactors)
				}
				if f.config.LearningRate <= 0 {
					t.Errorf("LearningRate = %f, want > 0", f.config.LearningRate)
				}
				if f.config.MaxHistory <= 0 {
					t.Errorf("MaxHistory = %d, want > 0", f.config.MaxHistory)
				}
			},
		},
		{
			name: "uses provided config values",
			cfg: FPMCConfig{
				NumFactors:      32,
				LearningRate:    0.05,
				NumIterations:   10,
				NegativeSamples: 5,
			},
			verify: func(t *testing.T, f *FPMC) {
				if f.config.NumFactors != 32 {
					t.Errorf("NumFactors = %d, want 32", f.config.NumFactors)
				}
				if f.config.LearningRate != 0.05 {
					t.Errorf("LearningRate = %f, want 0.05", f.config.LearningRate)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFPMC(tt.cfg)
			if f == nil {
				t.Fatal("NewFPMC() returned nil")
			}
			if f.Name() != "fpmc" {
				t.Errorf("Name() = %q, want %q", f.Name(), "fpmc")
			}
			tt.verify(t, f)
		})
	}
}

func TestFPMC_Train(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		wantTrained  bool
		wantFactors  bool
	}{
		{
			name:         "empty interactions trains successfully",
			interactions: nil,
			wantTrained:  true,
			wantFactors:  false,
		},
		{
			name: "single user sequence",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime},
				{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
				{UserID: 1, ItemID: 102, Timestamp: baseTime.Add(2 * time.Hour)},
			},
			wantTrained: true,
			wantFactors: true,
		},
		{
			name: "multiple user sequences",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime},
				{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
				{UserID: 2, ItemID: 101, Timestamp: baseTime},
				{UserID: 2, ItemID: 102, Timestamp: baseTime.Add(1 * time.Hour)},
				{UserID: 2, ItemID: 103, Timestamp: baseTime.Add(2 * time.Hour)},
			},
			wantTrained: true,
			wantFactors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFPMC(FPMCConfig{
				NumFactors:    8,
				NumIterations: 5,
			})

			err := f.Train(context.Background(), tt.interactions, nil)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if f.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", f.IsTrained(), tt.wantTrained)
			}

			if tt.wantFactors {
				if f.VU == nil || f.VIU == nil || f.VIL == nil || f.VII == nil {
					t.Error("expected all factor matrices to be set")
				}
			}

			if f.Version() < 1 {
				t.Errorf("Version() = %d, want >= 1", f.Version())
			}
		})
	}
}

func TestFPMC_Predict(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create sequential patterns:
	// User 1: 100 -> 101 -> 102
	// User 2: 100 -> 101 -> 103
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 1, ItemID: 102, Timestamp: baseTime.Add(2 * time.Hour)},
		{UserID: 2, ItemID: 100, Timestamp: baseTime},
		{UserID: 2, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 2, ItemID: 103, Timestamp: baseTime.Add(2 * time.Hour)},
	}

	f := NewFPMC(FPMCConfig{
		NumFactors:    16,
		NumIterations: 10,
	})
	if err := f.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		userID     int
		candidates []int
		wantScores bool
	}{
		{
			name:       "user with history gets recommendations",
			userID:     1,
			candidates: []int{103},
			wantScores: true,
		},
		{
			name:       "user without history gets no recommendations",
			userID:     999,
			candidates: []int{100, 101, 102},
			wantScores: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := f.Predict(context.Background(), tt.userID, tt.candidates)
			if err != nil {
				t.Fatalf("Predict() error = %v", err)
			}

			hasScores := len(scores) > 0
			if hasScores != tt.wantScores {
				t.Errorf("Predict() returned %d scores, wantScores = %v", len(scores), tt.wantScores)
			}
		})
	}
}

func TestFPMC_PredictSimilar(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Items 100 -> 101 is a common transition
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 2, ItemID: 100, Timestamp: baseTime},
		{UserID: 2, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 3, ItemID: 100, Timestamp: baseTime},
		{UserID: 3, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
	}

	f := NewFPMC(FPMCConfig{
		NumFactors:    16,
		NumIterations: 10,
	})
	if err := f.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		itemID     int
		candidates []int
		expectItem int
	}{
		{
			name:       "finds transition-similar items",
			itemID:     100,
			candidates: []int{101},
			expectItem: 101,
		},
		{
			name:       "unknown item returns empty",
			itemID:     999,
			candidates: []int{100, 101},
			expectItem: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := f.PredictSimilar(context.Background(), tt.itemID, tt.candidates)
			if err != nil {
				t.Fatalf("PredictSimilar() error = %v", err)
			}

			if tt.expectItem > 0 && len(scores) > 0 {
				if _, ok := scores[tt.expectItem]; !ok {
					t.Errorf("expected item %d to have a score", tt.expectItem)
				}
			}
		})
	}
}

func TestFPMC_PredictNext(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 1, ItemID: 102, Timestamp: baseTime.Add(2 * time.Hour)},
		{UserID: 2, ItemID: 100, Timestamp: baseTime},
		{UserID: 2, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
	}

	f := NewFPMC(FPMCConfig{
		NumFactors:    16,
		NumIterations: 10,
	})
	if err := f.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	t.Run("known user with recent items", func(t *testing.T) {
		scores, err := f.PredictNext(context.Background(), 1, []int{100, 101}, []int{102})
		if err != nil {
			t.Fatalf("PredictNext() error = %v", err)
		}
		if len(scores) == 0 {
			t.Error("expected scores for known user")
		}
	})

	t.Run("new user with recent items", func(t *testing.T) {
		scores, err := f.PredictNext(context.Background(), 999, []int{100}, []int{101, 102})
		if err != nil {
			t.Fatalf("PredictNext() error = %v", err)
		}
		// Should work with just Markov component
		if len(scores) == 0 {
			t.Error("expected scores even for new user")
		}
	})

	t.Run("empty recent items", func(t *testing.T) {
		scores, err := f.PredictNext(context.Background(), 1, []int{}, []int{100, 101})
		if err != nil {
			t.Fatalf("PredictNext() error = %v", err)
		}
		if len(scores) != 0 {
			t.Error("expected empty scores for empty history")
		}
	})
}

func TestFPMC_GetUserHistory(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 1, ItemID: 102, Timestamp: baseTime.Add(2 * time.Hour)},
	}

	f := NewFPMC(FPMCConfig{NumFactors: 8, NumIterations: 3})
	if err := f.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	history := f.GetUserHistory(1)
	if len(history) != 3 {
		t.Errorf("len(history) = %d, want 3", len(history))
	}

	// Verify order is preserved
	expected := []int{100, 101, 102}
	for i, itemID := range history {
		if itemID != expected[i] {
			t.Errorf("history[%d] = %d, want %d", i, itemID, expected[i])
		}
	}

	// Unknown user
	history = f.GetUserHistory(999)
	if history != nil {
		t.Error("expected nil history for unknown user")
	}
}

func TestFPMC_ContextCancellation(t *testing.T) {
	f := NewFPMC(FPMCConfig{NumFactors: 8, NumIterations: 50})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	interactions := make([]recommend.Interaction, 500)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:    i % 50,
			ItemID:    i % 100,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
	}

	err := f.Train(ctx, interactions, nil)
	if err == nil {
		t.Error("Train() with canceled context should return error")
	}
}

func TestFPMC_MaxHistory(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a long sequence
	interactions := make([]recommend.Interaction, 50)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:    1,
			ItemID:    100 + i,
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
		}
	}

	f := NewFPMC(FPMCConfig{
		NumFactors:    8,
		NumIterations: 3,
		MaxHistory:    10,
	})
	if err := f.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	history := f.GetUserHistory(1)
	if len(history) != 10 {
		t.Errorf("len(history) = %d, want 10 (MaxHistory)", len(history))
	}

	// Should be the most recent items
	if history[len(history)-1] != 149 { // Last item
		t.Errorf("last history item = %d, want 149", history[len(history)-1])
	}
}
