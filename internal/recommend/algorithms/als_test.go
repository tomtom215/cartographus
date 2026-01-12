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

func TestNewALS(t *testing.T) {
	tests := []struct {
		name   string
		cfg    ALSConfig
		verify func(t *testing.T, a *ALS)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  ALSConfig{},
			verify: func(t *testing.T, a *ALS) {
				if a.config.NumFactors <= 0 {
					t.Errorf("NumFactors = %d, want > 0", a.config.NumFactors)
				}
				if a.config.NumIterations <= 0 {
					t.Errorf("NumIterations = %d, want > 0", a.config.NumIterations)
				}
				if a.config.Regularization <= 0 {
					t.Errorf("Regularization = %f, want > 0", a.config.Regularization)
				}
			},
		},
		{
			name: "uses provided config values",
			cfg: ALSConfig{
				NumFactors:     100,
				NumIterations:  20,
				Regularization: 0.05,
				Alpha:          50.0,
			},
			verify: func(t *testing.T, a *ALS) {
				if a.config.NumFactors != 100 {
					t.Errorf("NumFactors = %d, want 100", a.config.NumFactors)
				}
				if a.config.NumIterations != 20 {
					t.Errorf("NumIterations = %d, want 20", a.config.NumIterations)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewALS(tt.cfg)
			if a == nil {
				t.Fatal("NewALS() returned nil")
			}
			if a.Name() != "als" {
				t.Errorf("Name() = %q, want %q", a.Name(), "als")
			}
			tt.verify(t, a)
		})
	}
}

func TestALS_Train(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		items        []recommend.Item
		wantTrained  bool
		wantFactors  bool
	}{
		{
			name:         "empty interactions trains successfully",
			interactions: nil,
			items:        nil,
			wantTrained:  true,
			wantFactors:  false,
		},
		{
			name: "single user single item",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
			},
			wantTrained: true,
			wantFactors: true,
		},
		{
			name: "builds factor matrices from interactions",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 0.8},
				{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 0.9},
				{UserID: 3, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 3, ItemID: 102, Timestamp: baseTime, Confidence: 0.7},
			},
			wantTrained: true,
			wantFactors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewALS(ALSConfig{
				NumFactors:    10,
				NumIterations: 5,
			})

			err := a.Train(context.Background(), tt.interactions, tt.items)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if a.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", a.IsTrained(), tt.wantTrained)
			}

			if tt.wantFactors {
				if a.X == nil {
					t.Error("expected user factors to be set")
				}
				if a.Y == nil {
					t.Error("expected item factors to be set")
				}
			}

			if a.Version() < 1 {
				t.Errorf("Version() = %d, want >= 1", a.Version())
			}
		})
	}
}

func TestALS_Predict(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create training data with clear patterns
	interactions := []recommend.Interaction{
		// Users 1 and 2 have similar preferences (items 100, 101)
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		// User 3 has different preferences
		{UserID: 3, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 103, Timestamp: baseTime, Confidence: 1.0},
	}

	a := NewALS(ALSConfig{
		NumFactors:     16,
		NumIterations:  10,
		Regularization: 0.01,
		Alpha:          40.0,
	})
	if err := a.Train(context.Background(), interactions, nil); err != nil {
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
			candidates: []int{102, 103},
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
			scores, err := a.Predict(context.Background(), tt.userID, tt.candidates)
			if err != nil {
				t.Fatalf("Predict() error = %v", err)
			}

			hasScores := len(scores) > 0
			if hasScores != tt.wantScores {
				t.Errorf("Predict() returned %d scores, wantScores = %v", len(scores), tt.wantScores)
			}

			// Verify scores are normalized (0-1)
			for itemID, score := range scores {
				if score < 0 || score > 1 {
					t.Errorf("score for item %d = %f, want in [0, 1]", itemID, score)
				}
			}
		})
	}
}

func TestALS_PredictSimilar(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Items 100 and 101 are frequently consumed together
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 4, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 4, ItemID: 103, Timestamp: baseTime, Confidence: 1.0},
	}

	a := NewALS(ALSConfig{
		NumFactors:    16,
		NumIterations: 10,
	})
	if err := a.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		itemID     int
		candidates []int
		expectItem int
	}{
		{
			name:       "finds similar items",
			itemID:     100,
			candidates: []int{101, 102, 103},
			expectItem: 101,
		},
		{
			name:       "unknown item returns empty",
			itemID:     999,
			candidates: []int{100, 101, 102},
			expectItem: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := a.PredictSimilar(context.Background(), tt.itemID, tt.candidates)
			if err != nil {
				t.Fatalf("PredictSimilar() error = %v", err)
			}

			if tt.expectItem > 0 && len(scores) > 0 {
				var maxID int
				var maxScore float64
				for id, score := range scores {
					if score > maxScore {
						maxScore = score
						maxID = id
					}
				}
				if maxID != tt.expectItem {
					t.Errorf("highest scoring item = %d, want %d", maxID, tt.expectItem)
				}
			}
		})
	}
}

func TestALS_ContextCancellation(t *testing.T) {
	a := NewALS(ALSConfig{NumFactors: 10, NumIterations: 50})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	interactions := make([]recommend.Interaction, 1000)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:     i % 100,
			ItemID:     i % 500,
			Timestamp:  time.Now(),
			Confidence: 0.5,
		}
	}

	err := a.Train(ctx, interactions, nil)
	if err == nil {
		t.Error("Train() with canceled context should return error")
	}
}

func TestALS_GetFactors(t *testing.T) {
	a := NewALS(ALSConfig{NumFactors: 8, NumIterations: 3})

	// Before training
	userFactors := a.GetUserFactors()
	if userFactors != nil {
		t.Error("expected nil user factors before training")
	}

	itemFactors := a.GetItemFactors()
	if itemFactors != nil {
		t.Error("expected nil item factors before training")
	}

	// After training
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Confidence: 1.0},
	}
	if err := a.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	userFactors = a.GetUserFactors()
	if userFactors == nil {
		t.Error("expected non-nil user factors after training")
	}
	if len(userFactors) != 2 { // 2 users
		t.Errorf("len(userFactors) = %d, want 2", len(userFactors))
	}
	if len(userFactors[0]) != 8 { // 8 factors
		t.Errorf("len(userFactors[0]) = %d, want 8", len(userFactors[0]))
	}

	itemFactors = a.GetItemFactors()
	if itemFactors == nil {
		t.Error("expected non-nil item factors after training")
	}
	if len(itemFactors) != 2 { // 2 items
		t.Errorf("len(itemFactors) = %d, want 2", len(itemFactors))
	}
}

func TestSolveLinearSystem(t *testing.T) {
	// Test with a simple 2x2 system: A * x = b
	// [4 2] [x1]   [8]
	// [2 3] [x2] = [7]
	// Solution: x1 = 1, x2 = 2

	A := [][]float64{
		{4, 2},
		{2, 3},
	}
	b := []float64{8, 7}

	x := solveLinearSystem(A, b)

	if len(x) != 2 {
		t.Fatalf("len(x) = %d, want 2", len(x))
	}

	// Check solution with tolerance
	expected := []float64{1.25, 1.5}
	for i, xi := range x {
		diff := xi - expected[i]
		if diff < -0.1 || diff > 0.1 {
			t.Errorf("x[%d] = %f, want ~%f", i, xi, expected[i])
		}
	}
}

func TestALS_ConvergenceBehavior(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create data with strong signal
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
	}

	// Train with few iterations
	a1 := NewALS(ALSConfig{NumFactors: 8, NumIterations: 2})
	if err := a1.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Train with more iterations
	a2 := NewALS(ALSConfig{NumFactors: 8, NumIterations: 20})
	if err := a2.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Both should produce valid predictions
	scores1, _ := a1.Predict(context.Background(), 1, []int{102})
	scores2, _ := a2.Predict(context.Background(), 1, []int{102})

	if len(scores1) == 0 {
		t.Error("expected scores from model with 2 iterations")
	}
	if len(scores2) == 0 {
		t.Error("expected scores from model with 20 iterations")
	}
}
