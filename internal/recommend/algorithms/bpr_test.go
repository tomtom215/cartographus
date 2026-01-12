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

func TestNewBPR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         BPRConfig
		expectedConfig BPRConfig
	}{
		{
			name:   "default config",
			config: DefaultBPRConfig(),
			expectedConfig: BPRConfig{
				NumFactors:         64,
				LearningRate:       0.01,
				Regularization:     0.01,
				NumIterations:      100,
				NumNegativeSamples: 5,
				MinConfidence:      0.1,
				Seed:               42,
			},
		},
		{
			name: "custom config",
			config: BPRConfig{
				NumFactors:         32,
				LearningRate:       0.05,
				Regularization:     0.1,
				NumIterations:      50,
				NumNegativeSamples: 3,
				MinConfidence:      0.2,
				Seed:               123,
			},
			expectedConfig: BPRConfig{
				NumFactors:         32,
				LearningRate:       0.05,
				Regularization:     0.1,
				NumIterations:      50,
				NumNegativeSamples: 3,
				MinConfidence:      0.2,
				Seed:               123,
			},
		},
		{
			name: "zero values get defaults",
			config: BPRConfig{
				NumFactors:         0,
				LearningRate:       0,
				Regularization:     0,
				NumIterations:      0,
				NumNegativeSamples: 0,
				MinConfidence:      0,
				Seed:               0,
			},
			expectedConfig: BPRConfig{
				NumFactors:         64,
				LearningRate:       0.01,
				Regularization:     0.01,
				NumIterations:      100,
				NumNegativeSamples: 5,
				MinConfidence:      0.1,
				Seed:               42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bpr := NewBPR(tt.config)

			if bpr == nil {
				t.Fatal("NewBPR returned nil")
			}

			if bpr.Name() != "bpr" {
				t.Errorf("Name() = %q, want %q", bpr.Name(), "bpr")
			}

			if bpr.config.NumFactors != tt.expectedConfig.NumFactors {
				t.Errorf("NumFactors = %d, want %d", bpr.config.NumFactors, tt.expectedConfig.NumFactors)
			}
			if bpr.config.LearningRate != tt.expectedConfig.LearningRate {
				t.Errorf("LearningRate = %f, want %f", bpr.config.LearningRate, tt.expectedConfig.LearningRate)
			}
			if bpr.config.Regularization != tt.expectedConfig.Regularization {
				t.Errorf("Regularization = %f, want %f", bpr.config.Regularization, tt.expectedConfig.Regularization)
			}
			if bpr.config.NumIterations != tt.expectedConfig.NumIterations {
				t.Errorf("NumIterations = %d, want %d", bpr.config.NumIterations, tt.expectedConfig.NumIterations)
			}
			if bpr.config.NumNegativeSamples != tt.expectedConfig.NumNegativeSamples {
				t.Errorf("NumNegativeSamples = %d, want %d", bpr.config.NumNegativeSamples, tt.expectedConfig.NumNegativeSamples)
			}
		})
	}
}

func TestBPR_Train(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		items        []recommend.Item
		wantTrained  bool
	}{
		{
			name:         "empty interactions",
			interactions: nil,
			items:        nil,
			wantTrained:  true,
		},
		{
			name: "single interaction",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0},
			},
			items:       nil,
			wantTrained: true,
		},
		{
			name: "multiple users and items",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Confidence: 0.8},
				{UserID: 1, ItemID: 102, Confidence: 0.5},
				{UserID: 2, ItemID: 100, Confidence: 0.9},
				{UserID: 2, ItemID: 103, Confidence: 1.0},
				{UserID: 3, ItemID: 101, Confidence: 0.7},
				{UserID: 3, ItemID: 102, Confidence: 0.6},
				{UserID: 3, ItemID: 103, Confidence: 1.0},
			},
			items:       nil,
			wantTrained: true,
		},
		{
			name: "filtered low confidence",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 0.05}, // Below threshold
				{UserID: 1, ItemID: 101, Confidence: 0.5},
			},
			items:       nil,
			wantTrained: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultBPRConfig()
			cfg.NumIterations = 10 // Fewer iterations for testing
			cfg.Seed = 42

			bpr := NewBPR(cfg)
			ctx := context.Background()

			err := bpr.Train(ctx, tt.interactions, tt.items)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if bpr.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", bpr.IsTrained(), tt.wantTrained)
			}
		})
	}
}

func TestBPR_TrainContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := DefaultBPRConfig()
	cfg.NumIterations = 1000 // Many iterations
	bpr := NewBPR(cfg)

	interactions := make([]recommend.Interaction, 100)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:     i % 10,
			ItemID:     i,
			Confidence: 1.0,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := bpr.Train(ctx, interactions, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Train() with canceled context: error = %v, want context.Canceled", err)
	}
}

func TestBPR_Predict(t *testing.T) {
	t.Parallel()

	cfg := DefaultBPRConfig()
	cfg.NumIterations = 50
	cfg.NumFactors = 16
	cfg.Seed = 42

	bpr := NewBPR(cfg)

	// Create training data with clear patterns:
	// User 1 likes items 100, 101, 102
	// User 2 likes items 103, 104, 105
	// User 3 likes items 100, 103 (overlaps)
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 1, ItemID: 102, Confidence: 1.0},
		{UserID: 2, ItemID: 103, Confidence: 1.0},
		{UserID: 2, ItemID: 104, Confidence: 1.0},
		{UserID: 2, ItemID: 105, Confidence: 1.0},
		{UserID: 3, ItemID: 100, Confidence: 1.0},
		{UserID: 3, ItemID: 103, Confidence: 1.0},
	}

	ctx := context.Background()
	if err := bpr.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		userID     int
		candidates []int
		wantScores bool
	}{
		{
			name:       "user 1 predictions",
			userID:     1,
			candidates: []int{100, 101, 102, 103, 104, 105},
			wantScores: true,
		},
		{
			name:       "user 2 predictions",
			userID:     2,
			candidates: []int{100, 101, 102, 103, 104, 105},
			wantScores: true,
		},
		{
			name:       "unknown user",
			userID:     999,
			candidates: []int{100, 101},
			wantScores: false,
		},
		{
			name:       "unknown items",
			userID:     1,
			candidates: []int{999, 998},
			wantScores: false,
		},
		{
			name:       "empty candidates",
			userID:     1,
			candidates: []int{},
			wantScores: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scores, err := bpr.Predict(ctx, tt.userID, tt.candidates)
			if err != nil {
				t.Fatalf("Predict() error = %v", err)
			}

			if tt.wantScores && len(scores) == 0 {
				t.Error("Predict() returned no scores, expected some")
			}

			if !tt.wantScores && len(scores) > 0 {
				t.Errorf("Predict() returned %d scores, expected none", len(scores))
			}

			// Verify scores are normalized to [0, 1]
			for itemID, score := range scores {
				if score < 0 || score > 1 {
					t.Errorf("Score for item %d = %f, want in [0, 1]", itemID, score)
				}
			}
		})
	}
}

func TestBPR_PredictBeforeTraining(t *testing.T) {
	t.Parallel()

	bpr := NewBPR(DefaultBPRConfig())
	ctx := context.Background()

	scores, err := bpr.Predict(ctx, 1, []int{100, 101, 102})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	if scores != nil {
		t.Errorf("Predict() before training = %v, want nil", scores)
	}
}

func TestBPR_PredictSimilar(t *testing.T) {
	t.Parallel()

	cfg := DefaultBPRConfig()
	cfg.NumIterations = 50
	cfg.NumFactors = 16
	cfg.Seed = 42

	bpr := NewBPR(cfg)

	// Create interactions that make items 100-102 similar
	// and items 103-105 similar
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 1, ItemID: 102, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Confidence: 1.0},
		{UserID: 3, ItemID: 103, Confidence: 1.0},
		{UserID: 3, ItemID: 104, Confidence: 1.0},
		{UserID: 3, ItemID: 105, Confidence: 1.0},
		{UserID: 4, ItemID: 103, Confidence: 1.0},
		{UserID: 4, ItemID: 104, Confidence: 1.0},
	}

	ctx := context.Background()
	if err := bpr.Train(ctx, interactions, nil); err != nil {
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
			candidates: []int{101, 102, 103, 104, 105},
			wantScores: true,
		},
		{
			name:       "unknown item",
			itemID:     999,
			candidates: []int{100, 101},
			wantScores: false,
		},
		{
			name:       "exclude self",
			itemID:     100,
			candidates: []int{100, 101}, // 100 should be excluded
			wantScores: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scores, err := bpr.PredictSimilar(ctx, tt.itemID, tt.candidates)
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

			// Verify scores are normalized to [0, 1]
			for itemID, score := range scores {
				if score < 0 || score > 1 {
					t.Errorf("Score for item %d = %f, want in [0, 1]", itemID, score)
				}
			}
		})
	}
}

func TestBPR_GetFactors(t *testing.T) {
	t.Parallel()

	cfg := DefaultBPRConfig()
	cfg.NumIterations = 10
	cfg.NumFactors = 8
	cfg.Seed = 42

	bpr := NewBPR(cfg)

	// Before training
	if factors := bpr.GetUserFactors(); factors != nil {
		t.Error("GetUserFactors() before training should return nil")
	}
	if factors := bpr.GetItemFactors(); factors != nil {
		t.Error("GetItemFactors() before training should return nil")
	}

	// Train
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Confidence: 1.0},
		{UserID: 2, ItemID: 102, Confidence: 1.0},
	}

	ctx := context.Background()
	if err := bpr.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// After training
	userFactors := bpr.GetUserFactors()
	if userFactors == nil {
		t.Fatal("GetUserFactors() after training should not return nil")
	}
	if len(userFactors) != 2 {
		t.Errorf("GetUserFactors() len = %d, want 2", len(userFactors))
	}
	if len(userFactors[0]) != cfg.NumFactors {
		t.Errorf("User factor dimension = %d, want %d", len(userFactors[0]), cfg.NumFactors)
	}

	itemFactors := bpr.GetItemFactors()
	if itemFactors == nil {
		t.Fatal("GetItemFactors() after training should not return nil")
	}
	if len(itemFactors) != 3 {
		t.Errorf("GetItemFactors() len = %d, want 3", len(itemFactors))
	}
	if len(itemFactors[0]) != cfg.NumFactors {
		t.Errorf("Item factor dimension = %d, want %d", len(itemFactors[0]), cfg.NumFactors)
	}
}

func TestBPR_Determinism(t *testing.T) {
	t.Parallel()

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 0.8},
		{UserID: 2, ItemID: 100, Confidence: 0.9},
		{UserID: 2, ItemID: 102, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Confidence: 0.7},
		{UserID: 3, ItemID: 102, Confidence: 0.6},
	}

	ctx := context.Background()

	// Train same model twice with same seed (reset between runs)
	cfg := DefaultBPRConfig()
	cfg.NumIterations = 20
	cfg.Seed = 12345

	bpr1 := NewBPR(cfg)
	if err := bpr1.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Get predictions from first model
	candidates := []int{100, 101, 102}
	scores1 := make(map[int]map[int]float64)
	for userID := 1; userID <= 3; userID++ {
		s, _ := bpr1.Predict(ctx, userID, candidates)
		scores1[userID] = s
	}

	// Train second model with same config
	bpr2 := NewBPR(cfg)
	if err := bpr2.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Predictions should be very close (within tolerance for SGD variance)
	// BPR uses stochastic gradient descent, so minor variance is acceptable
	const tolerance = 0.05 // 5% tolerance for SGD-based algorithms
	for userID := 1; userID <= 3; userID++ {
		scores2, _ := bpr2.Predict(ctx, userID, candidates)

		for _, itemID := range candidates {
			s1, s2 := scores1[userID][itemID], scores2[itemID]
			diff := s1 - s2
			if diff < 0 {
				diff = -diff
			}
			// Check relative difference (or absolute if scores are near zero)
			maxScore := s1
			if s2 > maxScore {
				maxScore = s2
			}
			if maxScore > 0 && diff/maxScore > tolerance {
				t.Errorf("User %d, Item %d: score1 = %f, score2 = %f (diff %.2f%% > %.0f%% tolerance)",
					userID, itemID, s1, s2, diff/maxScore*100, tolerance*100)
			}
		}
	}
}

func TestBPR_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ recommend.Algorithm = (*BPR)(nil)
}

func TestBPR_Version(t *testing.T) {
	t.Parallel()

	cfg := DefaultBPRConfig()
	cfg.NumIterations = 5
	bpr := NewBPR(cfg)

	if bpr.Version() != 0 {
		t.Errorf("Version() before training = %d, want 0", bpr.Version())
	}

	ctx := context.Background()
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
	}

	// First training
	if err := bpr.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}
	if bpr.Version() != 1 {
		t.Errorf("Version() after first training = %d, want 1", bpr.Version())
	}

	// Second training
	if err := bpr.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}
	if bpr.Version() != 2 {
		t.Errorf("Version() after second training = %d, want 2", bpr.Version())
	}
}

func TestBPR_LastTrainedAt(t *testing.T) {
	t.Parallel()

	cfg := DefaultBPRConfig()
	cfg.NumIterations = 5
	bpr := NewBPR(cfg)

	before := bpr.LastTrainedAt()
	if !before.IsZero() {
		t.Error("LastTrainedAt() before training should be zero")
	}

	ctx := context.Background()
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
	}

	startTime := time.Now()
	if err := bpr.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	after := bpr.LastTrainedAt()
	if after.Before(startTime) {
		t.Error("LastTrainedAt() should be after training start time")
	}
}

func TestBPR_LargeScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large-scale test in short mode")
	}
	t.Parallel()

	cfg := DefaultBPRConfig()
	cfg.NumIterations = 10
	cfg.NumFactors = 32
	cfg.Seed = 42

	bpr := NewBPR(cfg)

	// Create larger dataset
	numUsers := 100
	numItems := 500
	interactions := make([]recommend.Interaction, 0, numUsers*10)

	for u := 0; u < numUsers; u++ {
		// Each user interacts with ~10 random items
		for i := 0; i < 10; i++ {
			itemID := (u*7 + i*13) % numItems
			interactions = append(interactions, recommend.Interaction{
				UserID:     u,
				ItemID:     itemID,
				Confidence: float64(i+1) / 10.0,
			})
		}
	}

	ctx := context.Background()
	if err := bpr.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Verify predictions work
	candidates := make([]int, numItems)
	for i := 0; i < numItems; i++ {
		candidates[i] = i
	}

	scores, err := bpr.Predict(ctx, 0, candidates)
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	if len(scores) == 0 {
		t.Error("Predict() returned no scores for large dataset")
	}
}

func BenchmarkBPR_Train(b *testing.B) {
	cfg := DefaultBPRConfig()
	cfg.NumIterations = 20
	cfg.NumFactors = 32
	cfg.Seed = 42

	// Create benchmark dataset
	numUsers := 50
	numItems := 200
	interactions := make([]recommend.Interaction, 0, numUsers*10)

	for u := 0; u < numUsers; u++ {
		for i := 0; i < 10; i++ {
			itemID := (u*7 + i*13) % numItems
			interactions = append(interactions, recommend.Interaction{
				UserID:     u,
				ItemID:     itemID,
				Confidence: 1.0,
			})
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bpr := NewBPR(cfg)
		_ = bpr.Train(ctx, interactions, nil)
	}
}

func BenchmarkBPR_Predict(b *testing.B) {
	cfg := DefaultBPRConfig()
	cfg.NumIterations = 20
	cfg.NumFactors = 32
	cfg.Seed = 42

	bpr := NewBPR(cfg)

	numUsers := 50
	numItems := 200
	interactions := make([]recommend.Interaction, 0, numUsers*10)

	for u := 0; u < numUsers; u++ {
		for i := 0; i < 10; i++ {
			itemID := (u*7 + i*13) % numItems
			interactions = append(interactions, recommend.Interaction{
				UserID:     u,
				ItemID:     itemID,
				Confidence: 1.0,
			})
		}
	}

	ctx := context.Background()
	_ = bpr.Train(ctx, interactions, nil)

	candidates := make([]int, numItems)
	for i := 0; i < numItems; i++ {
		candidates[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bpr.Predict(ctx, i%numUsers, candidates)
	}
}
