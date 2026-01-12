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

func TestNewEASE(t *testing.T) {
	tests := []struct {
		name   string
		cfg    EASEConfig
		verify func(t *testing.T, e *EASE)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  EASEConfig{},
			verify: func(t *testing.T, e *EASE) {
				if e.config.L2Regularization <= 0 {
					t.Errorf("L2Regularization = %f, want > 0", e.config.L2Regularization)
				}
				if e.config.MinConfidence <= 0 {
					t.Errorf("MinConfidence = %f, want > 0", e.config.MinConfidence)
				}
			},
		},
		{
			name: "uses provided config values",
			cfg: EASEConfig{
				L2Regularization: 100.0,
				MinConfidence:    0.2,
			},
			verify: func(t *testing.T, e *EASE) {
				if e.config.L2Regularization != 100.0 {
					t.Errorf("L2Regularization = %f, want 100.0", e.config.L2Regularization)
				}
				if e.config.MinConfidence != 0.2 {
					t.Errorf("MinConfidence = %f, want 0.2", e.config.MinConfidence)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEASE(tt.cfg)
			if e == nil {
				t.Fatal("NewEASE() returned nil")
			}
			if e.Name() != "ease" {
				t.Errorf("Name() = %q, want %q", e.Name(), "ease")
			}
			tt.verify(t, e)
		})
	}
}

func TestEASE_Train(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		items        []recommend.Item
		wantTrained  bool
		wantMatrix   bool
	}{
		{
			name:         "empty interactions trains successfully",
			interactions: nil,
			items:        nil,
			wantTrained:  true,
			wantMatrix:   false,
		},
		{
			name: "single user single item",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
			},
			wantTrained: true,
			wantMatrix:  true,
		},
		{
			name: "builds weight matrix from interactions",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 0.8},
				{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 0.9},
				{UserID: 3, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 3, ItemID: 102, Timestamp: baseTime, Confidence: 0.7},
			},
			wantTrained: true,
			wantMatrix:  true,
		},
		{
			name: "filters low confidence interactions",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 0.05}, // Below threshold
				{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 0.5},
			},
			wantTrained: true,
			wantMatrix:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEASE(DefaultEASEConfig())

			err := e.Train(context.Background(), tt.interactions, tt.items)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if e.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", e.IsTrained(), tt.wantTrained)
			}

			if tt.wantMatrix && e.B == nil {
				t.Error("expected weight matrix to be set")
			}

			if e.Version() < 1 {
				t.Errorf("Version() = %d, want >= 1", e.Version())
			}
		})
	}
}

func TestEASE_Predict(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create training data with clear patterns
	// Users 1, 2, 3 all like items 100 and 101
	// User 4 likes different items
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 102, Timestamp: baseTime, Confidence: 0.5},
		{UserID: 4, ItemID: 103, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 4, ItemID: 104, Timestamp: baseTime, Confidence: 1.0},
	}

	e := NewEASE(EASEConfig{L2Regularization: 100.0, MinConfidence: 0.1})
	if err := e.Train(context.Background(), interactions, nil); err != nil {
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
			candidates: []int{102, 103, 104},
			wantScores: true,
		},
		{
			name:       "user without history gets no recommendations",
			userID:     999,
			candidates: []int{100, 101, 102},
			wantScores: false,
		},
		{
			name:       "returns scores for known items only",
			userID:     3,
			candidates: []int{100, 101, 999},
			wantScores: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := e.Predict(context.Background(), tt.userID, tt.candidates)
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

func TestEASE_PredictSimilar(t *testing.T) {
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

	e := NewEASE(EASEConfig{L2Regularization: 100.0, MinConfidence: 0.1})
	if err := e.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		itemID     int
		candidates []int
		expectItem int // Item expected to score highest
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
			scores, err := e.PredictSimilar(context.Background(), tt.itemID, tt.candidates)
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

func TestEASE_ContextCancellation(t *testing.T) {
	e := NewEASE(DefaultEASEConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Large dataset to ensure cancellation is checked
	interactions := make([]recommend.Interaction, 10000)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:     i % 100,
			ItemID:     i % 500,
			Timestamp:  time.Now(),
			Confidence: 0.5,
		}
	}

	err := e.Train(ctx, interactions, nil)
	if err == nil {
		t.Error("Train() with canceled context should return error")
	}
}

func TestEASEParallel_Train(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
	}

	e := NewEASEParallel(DefaultEASEConfig(), 2)
	if err := e.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	if !e.IsTrained() {
		t.Error("expected algorithm to be trained")
	}

	// Test that prediction works
	scores, err := e.Predict(context.Background(), 1, []int{102})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	if len(scores) == 0 {
		t.Error("expected scores for known user")
	}
}

func TestCholeskyDecomposition(t *testing.T) {
	tests := []struct {
		name    string
		matrix  [][]float64
		wantErr bool
	}{
		{
			name: "valid positive definite matrix",
			matrix: [][]float64{
				{4, 2},
				{2, 5},
			},
			wantErr: false,
		},
		{
			name: "identity matrix",
			matrix: [][]float64{
				{1, 0, 0},
				{0, 1, 0},
				{0, 0, 1},
			},
			wantErr: false,
		},
		{
			name: "not positive definite",
			matrix: [][]float64{
				{1, 2},
				{2, 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			L, err := choleskyDecomposition(tt.matrix)
			if (err != nil) != tt.wantErr {
				t.Errorf("choleskyDecomposition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify L * L' = A
				n := len(tt.matrix)
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						var sum float64
						for k := 0; k < n; k++ {
							sum += L[i][k] * L[j][k]
						}
						diff := sum - tt.matrix[i][j]
						if diff < -0.001 || diff > 0.001 {
							t.Errorf("L*L'[%d][%d] = %f, want %f", i, j, sum, tt.matrix[i][j])
						}
					}
				}
			}
		})
	}
}

func TestEASE_GetWeightMatrix(t *testing.T) {
	e := NewEASE(DefaultEASEConfig())

	// Before training
	matrix := e.GetWeightMatrix()
	if matrix != nil {
		t.Error("expected nil matrix before training")
	}

	// After training
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
	}
	if err := e.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	matrix = e.GetWeightMatrix()
	if matrix == nil {
		t.Error("expected non-nil matrix after training")
	}

	// Verify diagonal is zero
	for i := range matrix {
		if matrix[i][i] != 0 {
			t.Errorf("diagonal[%d] = %f, want 0", i, matrix[i][i])
		}
	}
}

func TestEASE_GetItemIndex(t *testing.T) {
	e := NewEASE(DefaultEASEConfig())

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 200, Confidence: 1.0},
		{UserID: 2, ItemID: 300, Confidence: 1.0},
	}

	if err := e.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	index := e.GetItemIndex()
	if len(index) != 3 {
		t.Errorf("len(index) = %d, want 3", len(index))
	}

	// Verify all item IDs are in the index
	expectedItems := []int{100, 200, 300}
	for _, id := range expectedItems {
		if _, ok := index[id]; !ok {
			t.Errorf("item %d not in index", id)
		}
	}
}
