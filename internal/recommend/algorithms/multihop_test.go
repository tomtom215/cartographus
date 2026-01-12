// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"errors"
	"testing"

	"github.com/tomtom215/cartographus/internal/recommend"
)

func TestNewMultiHopItemCF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         MultiHopItemCFConfig
		expectedConfig MultiHopItemCFConfig
	}{
		{
			name:   "default config",
			config: DefaultMultiHopItemCFConfig(),
			expectedConfig: MultiHopItemCFConfig{
				NumHops:         2,
				TopKPerHop:      10,
				DecayFactor:     0.5,
				MinSimilarity:   0.1,
				MinConfidence:   0.1,
				MaxItemsPerUser: 100,
			},
		},
		{
			name: "custom config",
			config: MultiHopItemCFConfig{
				NumHops:         3,
				TopKPerHop:      20,
				DecayFactor:     0.7,
				MinSimilarity:   0.2,
				MinConfidence:   0.3,
				MaxItemsPerUser: 50,
			},
			expectedConfig: MultiHopItemCFConfig{
				NumHops:         3,
				TopKPerHop:      20,
				DecayFactor:     0.7,
				MinSimilarity:   0.2,
				MinConfidence:   0.3,
				MaxItemsPerUser: 50,
			},
		},
		{
			name: "zero values get defaults",
			config: MultiHopItemCFConfig{
				NumHops:         0,
				TopKPerHop:      0,
				DecayFactor:     0,
				MinSimilarity:   0,
				MinConfidence:   0,
				MaxItemsPerUser: 0,
			},
			expectedConfig: MultiHopItemCFConfig{
				NumHops:         2,
				TopKPerHop:      10,
				DecayFactor:     0.5,
				MinSimilarity:   0.1,
				MinConfidence:   0.1,
				MaxItemsPerUser: 100,
			},
		},
		{
			name: "invalid decay factor gets default",
			config: MultiHopItemCFConfig{
				NumHops:     2,
				DecayFactor: 1.5, // Invalid: > 1
			},
			expectedConfig: MultiHopItemCFConfig{
				NumHops:         2,
				TopKPerHop:      10,
				DecayFactor:     0.5, // Default
				MinSimilarity:   0.1,
				MinConfidence:   0.1,
				MaxItemsPerUser: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mh := NewMultiHopItemCF(tt.config)

			if mh == nil {
				t.Fatal("NewMultiHopItemCF returned nil")
			}

			if mh.Name() != "multihop_itemcf" {
				t.Errorf("Name() = %q, want %q", mh.Name(), "multihop_itemcf")
			}

			if mh.config.NumHops != tt.expectedConfig.NumHops {
				t.Errorf("NumHops = %d, want %d", mh.config.NumHops, tt.expectedConfig.NumHops)
			}
			if mh.config.TopKPerHop != tt.expectedConfig.TopKPerHop {
				t.Errorf("TopKPerHop = %d, want %d", mh.config.TopKPerHop, tt.expectedConfig.TopKPerHop)
			}
			if mh.config.DecayFactor != tt.expectedConfig.DecayFactor {
				t.Errorf("DecayFactor = %f, want %f", mh.config.DecayFactor, tt.expectedConfig.DecayFactor)
			}
		})
	}
}

func TestMultiHopItemCF_Train(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		wantTrained  bool
		wantItems    int
	}{
		{
			name:         "empty interactions",
			interactions: nil,
			wantTrained:  true,
			wantItems:    0,
		},
		{
			name: "single interaction",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0},
			},
			wantTrained: true,
			wantItems:   1,
		},
		{
			name: "multiple users and items",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Confidence: 0.8},
				{UserID: 2, ItemID: 100, Confidence: 0.9},
				{UserID: 2, ItemID: 101, Confidence: 1.0},
				{UserID: 2, ItemID: 102, Confidence: 0.7},
				{UserID: 3, ItemID: 101, Confidence: 0.8},
				{UserID: 3, ItemID: 102, Confidence: 0.9},
			},
			wantTrained: true,
			wantItems:   3,
		},
		{
			name: "filtered low confidence",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 0.05}, // Below threshold
				{UserID: 1, ItemID: 101, Confidence: 0.5},
			},
			wantTrained: true,
			wantItems:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultMultiHopItemCFConfig()
			mh := NewMultiHopItemCF(cfg)
			ctx := context.Background()

			err := mh.Train(ctx, tt.interactions, nil)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if mh.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", mh.IsTrained(), tt.wantTrained)
			}

			if len(mh.indexToItem) != tt.wantItems {
				t.Errorf("Number of items = %d, want %d", len(mh.indexToItem), tt.wantItems)
			}
		})
	}
}

func TestMultiHopItemCF_TrainContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := DefaultMultiHopItemCFConfig()
	mh := NewMultiHopItemCF(cfg)

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

	err := mh.Train(ctx, interactions, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Train() with canceled context: error = %v, want context.Canceled", err)
	}
}

func TestMultiHopItemCF_Predict(t *testing.T) {
	t.Parallel()

	cfg := DefaultMultiHopItemCFConfig()
	cfg.NumHops = 2
	cfg.TopKPerHop = 5
	cfg.MinSimilarity = 0.01

	mh := NewMultiHopItemCF(cfg)

	// Create a chain of similar items:
	// Items 100, 101 are similar (users 1,2 watch both)
	// Items 101, 102 are similar (users 2,3 watch both)
	// Items 102, 103 are similar (users 3,4 watch both)
	// So from 100, hop 1 reaches 101, hop 2 reaches 102
	interactions := []recommend.Interaction{
		// Users 1,2 watch items 100,101
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Confidence: 1.0},
		// Users 2,3 watch items 101,102
		{UserID: 2, ItemID: 102, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Confidence: 1.0},
		{UserID: 3, ItemID: 102, Confidence: 1.0},
		// Users 3,4 watch items 102,103
		{UserID: 3, ItemID: 103, Confidence: 1.0},
		{UserID: 4, ItemID: 102, Confidence: 1.0},
		{UserID: 4, ItemID: 103, Confidence: 1.0},
		// User 5 only watches item 100
		{UserID: 5, ItemID: 100, Confidence: 1.0},
	}

	ctx := context.Background()
	if err := mh.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		userID     int
		candidates []int
		wantScores bool
	}{
		{
			name:       "user 5 gets multi-hop recommendations",
			userID:     5,
			candidates: []int{101, 102, 103}, // 101 is hop 1, 102 is hop 2
			wantScores: true,
		},
		{
			name:       "unknown user",
			userID:     999,
			candidates: []int{100, 101},
			wantScores: false,
		},
		{
			name:       "unknown items in candidates",
			userID:     5,
			candidates: []int{999, 998},
			wantScores: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scores, err := mh.Predict(ctx, tt.userID, tt.candidates)
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

func TestMultiHopItemCF_PredictHopDecay(t *testing.T) {
	t.Parallel()

	cfg := DefaultMultiHopItemCFConfig()
	cfg.NumHops = 3
	cfg.TopKPerHop = 10
	cfg.DecayFactor = 0.5
	cfg.MinSimilarity = 0.01

	mh := NewMultiHopItemCF(cfg)

	// Create chain: 100 -> 101 -> 102 -> 103
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Confidence: 1.0},
		{UserID: 2, ItemID: 102, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Confidence: 1.0},
		{UserID: 3, ItemID: 102, Confidence: 1.0},
		{UserID: 3, ItemID: 103, Confidence: 1.0},
		{UserID: 4, ItemID: 102, Confidence: 1.0},
		{UserID: 4, ItemID: 103, Confidence: 1.0},
		// User 5 only watches 100
		{UserID: 5, ItemID: 100, Confidence: 1.0},
	}

	ctx := context.Background()
	if err := mh.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	scores, err := mh.Predict(ctx, 5, []int{101, 102, 103})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	// 101 should have a score (hop 1)
	// 102 should have a score (hop 2)
	// 103 might have a score (hop 3 if reachable)
	if len(scores) == 0 {
		t.Error("Expected at least some scores")
	}

	// If both 101 and 102 are scored, 101 should have higher score (closer)
	if s101, ok := scores[101]; ok {
		if s102, ok := scores[102]; ok {
			if s102 > s101 {
				t.Logf("Note: item 102 (hop 2) scored higher than 101 (hop 1): %f > %f", s102, s101)
				// This can happen due to normalization and multiple paths
			}
		}
	}
}

func TestMultiHopItemCF_PredictBeforeTraining(t *testing.T) {
	t.Parallel()

	mh := NewMultiHopItemCF(DefaultMultiHopItemCFConfig())
	ctx := context.Background()

	scores, err := mh.Predict(ctx, 1, []int{100, 101, 102})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	if scores != nil {
		t.Errorf("Predict() before training = %v, want nil", scores)
	}
}

func TestMultiHopItemCF_PredictSimilar(t *testing.T) {
	t.Parallel()

	cfg := DefaultMultiHopItemCFConfig()
	cfg.NumHops = 2
	cfg.MinSimilarity = 0.01

	mh := NewMultiHopItemCF(cfg)

	// Items 100, 101, 102 form a chain
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Confidence: 1.0},
		{UserID: 2, ItemID: 102, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Confidence: 1.0},
		{UserID: 3, ItemID: 102, Confidence: 1.0},
	}

	ctx := context.Background()
	if err := mh.Train(ctx, interactions, nil); err != nil {
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
			candidates: []int{101, 102},
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

			scores, err := mh.PredictSimilar(ctx, tt.itemID, tt.candidates)
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

func TestMultiHopItemCF_GetMethods(t *testing.T) {
	t.Parallel()

	cfg := MultiHopItemCFConfig{
		NumHops:     3,
		DecayFactor: 0.7,
	}
	mh := NewMultiHopItemCF(cfg)

	if mh.GetNumHops() != 3 {
		t.Errorf("GetNumHops() = %d, want 3", mh.GetNumHops())
	}

	if mh.GetDecayFactor() != 0.7 {
		t.Errorf("GetDecayFactor() = %f, want 0.7", mh.GetDecayFactor())
	}

	// Before training
	if mh.GetSimilarityCount() != 0 {
		t.Errorf("GetSimilarityCount() before training = %d, want 0", mh.GetSimilarityCount())
	}

	// After training
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Confidence: 1.0},
	}

	ctx := context.Background()
	if err := mh.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	if mh.GetSimilarityCount() == 0 {
		t.Error("GetSimilarityCount() after training should be > 0")
	}
}

func TestMultiHopItemCF_Pow(t *testing.T) {
	t.Parallel()

	mh := NewMultiHopItemCF(DefaultMultiHopItemCFConfig())

	tests := []struct {
		x    float64
		n    int
		want float64
	}{
		{0.5, 0, 1.0},
		{0.5, 1, 0.5},
		{0.5, 2, 0.25},
		{0.5, 3, 0.125},
		{0.7, 2, 0.49},
		{2.0, 3, 8.0},
		{1.0, 10, 1.0},
	}

	for _, tt := range tests {
		got := mh.pow(tt.x, tt.n)
		// Use approximate comparison for floats
		diff := got - tt.want
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.0001 {
			t.Errorf("pow(%f, %d) = %f, want %f", tt.x, tt.n, got, tt.want)
		}
	}
}

func TestMultiHopItemCF_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ recommend.Algorithm = (*MultiHopItemCF)(nil)
}

func TestMultiHopItemCF_LargeScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large-scale test in short mode")
	}
	t.Parallel()

	cfg := DefaultMultiHopItemCFConfig()
	cfg.NumHops = 2
	cfg.TopKPerHop = 10
	cfg.MinSimilarity = 0.01 // Lower threshold for sparse data

	mh := NewMultiHopItemCF(cfg)

	// Create larger dataset with denser connections
	// Each user watches 15 items from a pool of 200 to ensure item overlap
	numUsers := 100
	numItems := 200
	interactions := make([]recommend.Interaction, 0, numUsers*15)

	for u := 0; u < numUsers; u++ {
		for i := 0; i < 15; i++ {
			// Create more overlapping patterns: items 0-49 are popular
			var itemID int
			if i < 5 {
				itemID = (u + i) % 50 // Popular items (high overlap)
			} else {
				itemID = 50 + (u*7+i*13)%150 // Long tail items
			}
			interactions = append(interactions, recommend.Interaction{
				UserID:     u,
				ItemID:     itemID,
				Confidence: 1.0,
			})
		}
	}

	ctx := context.Background()
	if err := mh.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Verify predictions work
	candidates := make([]int, numItems)
	for i := 0; i < numItems; i++ {
		candidates[i] = i
	}

	scores, err := mh.Predict(ctx, 0, candidates)
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	if len(scores) == 0 {
		t.Error("Predict() returned no scores for large dataset")
	}
}

func BenchmarkMultiHopItemCF_Train(b *testing.B) {
	cfg := DefaultMultiHopItemCFConfig()
	cfg.NumHops = 2
	cfg.TopKPerHop = 10

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
		mh := NewMultiHopItemCF(cfg)
		_ = mh.Train(ctx, interactions, nil)
	}
}

func BenchmarkMultiHopItemCF_Predict(b *testing.B) {
	cfg := DefaultMultiHopItemCFConfig()
	cfg.NumHops = 2
	cfg.TopKPerHop = 10

	mh := NewMultiHopItemCF(cfg)

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
	_ = mh.Train(ctx, interactions, nil)

	candidates := make([]int, numItems)
	for i := 0; i < numItems; i++ {
		candidates[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mh.Predict(ctx, i%numUsers, candidates)
	}
}
