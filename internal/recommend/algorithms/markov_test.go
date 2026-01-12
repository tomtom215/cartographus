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

func TestNewMarkovChain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         MarkovChainConfig
		expectedConfig MarkovChainConfig
	}{
		{
			name:   "default config",
			config: DefaultMarkovChainConfig(),
			expectedConfig: MarkovChainConfig{
				OrderK:                1,
				MinTransitionCount:    2,
				MaxTransitionsPerItem: 50,
				MinConfidence:         0.1,
				SessionWindowSeconds:  21600,
				SmoothingAlpha:        0.1,
			},
		},
		{
			name: "custom config",
			config: MarkovChainConfig{
				OrderK:                2,
				MinTransitionCount:    3,
				MaxTransitionsPerItem: 20,
				MinConfidence:         0.2,
				SessionWindowSeconds:  3600,
				SmoothingAlpha:        0.5,
			},
			expectedConfig: MarkovChainConfig{
				OrderK:                2,
				MinTransitionCount:    3,
				MaxTransitionsPerItem: 20,
				MinConfidence:         0.2,
				SessionWindowSeconds:  3600,
				SmoothingAlpha:        0.5,
			},
		},
		{
			name: "zero values get defaults",
			config: MarkovChainConfig{
				OrderK:                0,
				MinTransitionCount:    0,
				MaxTransitionsPerItem: 0,
				MinConfidence:         0,
				SessionWindowSeconds:  0,
				SmoothingAlpha:        0,
			},
			expectedConfig: MarkovChainConfig{
				OrderK:                1,
				MinTransitionCount:    2,
				MaxTransitionsPerItem: 50,
				MinConfidence:         0.1,
				SessionWindowSeconds:  21600,
				SmoothingAlpha:        0.1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := NewMarkovChain(tt.config)

			if mc == nil {
				t.Fatal("NewMarkovChain returned nil")
			}

			if mc.Name() != "markov_chain" {
				t.Errorf("Name() = %q, want %q", mc.Name(), "markov_chain")
			}

			if mc.config.OrderK != tt.expectedConfig.OrderK {
				t.Errorf("OrderK = %d, want %d", mc.config.OrderK, tt.expectedConfig.OrderK)
			}
			if mc.config.MinTransitionCount != tt.expectedConfig.MinTransitionCount {
				t.Errorf("MinTransitionCount = %d, want %d", mc.config.MinTransitionCount, tt.expectedConfig.MinTransitionCount)
			}
			if mc.config.SessionWindowSeconds != tt.expectedConfig.SessionWindowSeconds {
				t.Errorf("SessionWindowSeconds = %d, want %d", mc.config.SessionWindowSeconds, tt.expectedConfig.SessionWindowSeconds)
			}
		})
	}
}

func TestMarkovChain_Train(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

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
				{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
			},
			wantTrained: true,
			wantItems:   1,
		},
		{
			name: "sequential viewing",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
				{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
				{UserID: 1, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(2 * time.Hour)},
				// Another user with same pattern
				{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
				{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
				{UserID: 2, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(2 * time.Hour)},
			},
			wantTrained: true,
			wantItems:   3,
		},
		{
			name: "session break",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
				{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
				// 24 hour gap - new session
				{UserID: 1, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(25 * time.Hour)},
			},
			wantTrained: true,
			wantItems:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultMarkovChainConfig()
			cfg.MinTransitionCount = 1 // Allow single transitions for testing
			mc := NewMarkovChain(cfg)
			ctx := context.Background()

			err := mc.Train(ctx, tt.interactions, nil)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if mc.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", mc.IsTrained(), tt.wantTrained)
			}

			if len(mc.indexToItem) != tt.wantItems {
				t.Errorf("Number of items = %d, want %d", len(mc.indexToItem), tt.wantItems)
			}
		})
	}
}

func TestMarkovChain_TrainContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := DefaultMarkovChainConfig()
	mc := NewMarkovChain(cfg)

	baseTime := time.Now()
	interactions := make([]recommend.Interaction, 1000)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:     i % 50,
			ItemID:     i % 100,
			Confidence: 1.0,
			Timestamp:  baseTime.Add(time.Duration(i) * time.Minute),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := mc.Train(ctx, interactions, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Train() with canceled context: error = %v, want context.Canceled", err)
	}
}

func TestMarkovChain_PredictNext(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	cfg := DefaultMarkovChainConfig()
	cfg.MinTransitionCount = 2 // Require 2 occurrences

	mc := NewMarkovChain(cfg)

	// Create clear sequential pattern:
	// 100 -> 101 (2 times)
	// 101 -> 102 (2 times)
	// 102 -> 103 (2 times)
	interactions := []recommend.Interaction{
		// User 1 sequence
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 1, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(2 * time.Hour)},
		{UserID: 1, ItemID: 103, Confidence: 1.0, Timestamp: baseTime.Add(3 * time.Hour)},
		// User 2 same sequence
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 2, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(2 * time.Hour)},
		{UserID: 2, ItemID: 103, Confidence: 1.0, Timestamp: baseTime.Add(3 * time.Hour)},
	}

	ctx := context.Background()
	if err := mc.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		lastItemID int
		candidates []int
		wantScores bool
		wantItem   int // Expected top item
	}{
		{
			name:       "after item 100, expect 101",
			lastItemID: 100,
			candidates: []int{101, 102, 103},
			wantScores: true,
			wantItem:   101,
		},
		{
			name:       "after item 101, expect 102",
			lastItemID: 101,
			candidates: []int{100, 102, 103},
			wantScores: true,
			wantItem:   102,
		},
		{
			name:       "unknown last item",
			lastItemID: 999,
			candidates: []int{100, 101, 102},
			wantScores: false,
		},
		{
			name:       "unknown candidates",
			lastItemID: 100,
			candidates: []int{999, 998},
			wantScores: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scores, err := mc.PredictNext(ctx, tt.lastItemID, tt.candidates)
			if err != nil {
				t.Fatalf("PredictNext() error = %v", err)
			}

			if tt.wantScores && len(scores) == 0 {
				t.Error("PredictNext() returned no scores, expected some")
			}

			if !tt.wantScores && len(scores) > 0 {
				t.Errorf("PredictNext() returned %d scores, expected none", len(scores))
			}

			// Verify expected item has highest score
			if tt.wantScores && tt.wantItem > 0 {
				maxItem := 0
				maxScore := 0.0
				for item, score := range scores {
					if score > maxScore {
						maxScore = score
						maxItem = item
					}
				}
				if maxItem != tt.wantItem {
					t.Errorf("Top item = %d, want %d", maxItem, tt.wantItem)
				}
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

func TestMarkovChain_PredictBeforeTraining(t *testing.T) {
	t.Parallel()

	mc := NewMarkovChain(DefaultMarkovChainConfig())
	ctx := context.Background()

	scores, err := mc.PredictNext(ctx, 100, []int{101, 102, 103})
	if err != nil {
		t.Fatalf("PredictNext() error = %v", err)
	}

	if scores != nil {
		t.Errorf("PredictNext() before training = %v, want nil", scores)
	}
}

func TestMarkovChain_SessionBreaks(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	cfg := DefaultMarkovChainConfig()
	cfg.SessionWindowSeconds = 3600 // 1 hour window
	cfg.MinTransitionCount = 2

	mc := NewMarkovChain(cfg)

	// User watches 100 -> 101 within session, but 101 -> 102 across session
	interactions := []recommend.Interaction{
		// User 1
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(30 * time.Minute)}, // Within session
		{UserID: 1, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(3 * time.Hour)},    // Session break
		// User 2 same pattern
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(30 * time.Minute)},
		{UserID: 2, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(3 * time.Hour)},
	}

	ctx := context.Background()
	if err := mc.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// 100 -> 101 should exist (within session for both users)
	scores, _ := mc.PredictNext(ctx, 100, []int{101, 102})
	if _, ok := scores[101]; !ok {
		t.Error("Expected transition 100 -> 101 to be learned")
	}

	// 101 -> 102 should NOT exist (session break for both users)
	scoresAfter101, _ := mc.PredictNext(ctx, 101, []int{102})
	if _, ok := scoresAfter101[102]; ok {
		t.Error("Transition 101 -> 102 should not be learned (session break)")
	}
}

func TestMarkovChain_MinTransitionCount(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	cfg := DefaultMarkovChainConfig()
	cfg.MinTransitionCount = 3 // Require 3 occurrences

	mc := NewMarkovChain(cfg)

	// 100 -> 101 appears 3 times (above threshold)
	// 100 -> 102 appears 2 times (below threshold)
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 3, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 3, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 4, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 4, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)}, // Only 2 times
		{UserID: 5, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 5, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
	}

	ctx := context.Background()
	if err := mc.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	scores, _ := mc.PredictNext(ctx, 100, []int{101, 102})

	// 101 should be in results (3 occurrences >= threshold)
	if _, ok := scores[101]; !ok {
		t.Error("Expected 101 to be in results (above threshold)")
	}

	// 102 should NOT be in results (2 occurrences < threshold)
	if _, ok := scores[102]; ok {
		t.Error("Expected 102 to NOT be in results (below threshold)")
	}
}

func TestMarkovChain_GetTopTransitions(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	cfg := DefaultMarkovChainConfig()
	cfg.MinTransitionCount = 1

	mc := NewMarkovChain(cfg)

	// Create varied transition counts from item 100
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 3, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 3, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 4, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 4, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 5, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 5, ItemID: 102, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
	}

	ctx := context.Background()
	if err := mc.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Get top transitions from item 100
	trans := mc.GetTopTransitions(100, 5)
	if len(trans) == 0 {
		t.Fatal("GetTopTransitions() returned no transitions")
	}

	// First should be 101 (3 occurrences) or 102 (2 occurrences)
	// The order depends on counts
	if trans[0].NextItem != 101 {
		t.Errorf("Top transition from 100: got %d, want 101", trans[0].NextItem)
	}

	// Probabilities should be > 0
	for _, tr := range trans {
		if tr.Probability <= 0 {
			t.Errorf("Probability for %d = %f, want > 0", tr.NextItem, tr.Probability)
		}
	}
}

func TestMarkovChain_GetTransitionCount(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	cfg := DefaultMarkovChainConfig()
	cfg.MinTransitionCount = 1

	mc := NewMarkovChain(cfg)

	// Before training
	if mc.GetTransitionCount() != 0 {
		t.Errorf("GetTransitionCount() before training = %d, want 0", mc.GetTransitionCount())
	}

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
		{UserID: 2, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 2, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
	}

	ctx := context.Background()
	if err := mc.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	count := mc.GetTransitionCount()
	if count == 0 {
		t.Error("GetTransitionCount() after training should be > 0")
	}
}

func TestMarkovChain_GetOrderK(t *testing.T) {
	t.Parallel()

	cfg := MarkovChainConfig{OrderK: 3}
	mc := NewMarkovChain(cfg)

	if mc.GetOrderK() != 3 {
		t.Errorf("GetOrderK() = %d, want 3", mc.GetOrderK())
	}
}

func TestMarkovChain_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ recommend.Algorithm = (*MarkovChain)(nil)
}

func TestMarkovChain_Predict(t *testing.T) {
	t.Parallel()

	// Note: Predict() for MarkovChain returns nil because it needs
	// the user's last item, not just a user ID. Use PredictNext instead.
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	cfg := DefaultMarkovChainConfig()
	cfg.MinTransitionCount = 1

	mc := NewMarkovChain(cfg)

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0, Timestamp: baseTime},
		{UserID: 1, ItemID: 101, Confidence: 1.0, Timestamp: baseTime.Add(1 * time.Hour)},
	}

	ctx := context.Background()
	if err := mc.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Predict returns nil for this algorithm (use PredictNext instead)
	scores, err := mc.Predict(ctx, 1, []int{101, 102})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	if scores != nil {
		t.Error("Predict() should return nil for MarkovChain (use PredictNext)")
	}
}

func TestMarkovChain_LargeScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large-scale test in short mode")
	}
	t.Parallel()

	cfg := DefaultMarkovChainConfig()
	cfg.MinTransitionCount = 1

	mc := NewMarkovChain(cfg)

	baseTime := time.Now()
	numUsers := 100
	numItems := 500
	interactions := make([]recommend.Interaction, 0, numUsers*10)

	for u := 0; u < numUsers; u++ {
		// Each user watches a sequence of 10 items
		for i := 0; i < 10; i++ {
			itemID := (u*7 + i*13) % numItems
			interactions = append(interactions, recommend.Interaction{
				UserID:     u,
				ItemID:     itemID,
				Confidence: 1.0,
				Timestamp:  baseTime.Add(time.Duration(u*100+i) * time.Minute),
			})
		}
	}

	ctx := context.Background()
	if err := mc.Train(ctx, interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Verify predictions work
	candidates := make([]int, numItems)
	for i := 0; i < numItems; i++ {
		candidates[i] = i
	}

	scores, err := mc.PredictNext(ctx, 0, candidates)
	if err != nil {
		t.Fatalf("PredictNext() error = %v", err)
	}

	if len(scores) == 0 {
		t.Log("Note: PredictNext returned no scores (may need more transition occurrences)")
	}
}

func BenchmarkMarkovChain_Train(b *testing.B) {
	cfg := DefaultMarkovChainConfig()
	cfg.MinTransitionCount = 1

	baseTime := time.Now()
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
				Timestamp:  baseTime.Add(time.Duration(u*100+i) * time.Minute),
			})
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc := NewMarkovChain(cfg)
		_ = mc.Train(ctx, interactions, nil)
	}
}

func BenchmarkMarkovChain_PredictNext(b *testing.B) {
	cfg := DefaultMarkovChainConfig()
	cfg.MinTransitionCount = 1

	mc := NewMarkovChain(cfg)

	baseTime := time.Now()
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
				Timestamp:  baseTime.Add(time.Duration(u*100+i) * time.Minute),
			})
		}
	}

	ctx := context.Background()
	_ = mc.Train(ctx, interactions, nil)

	candidates := make([]int, numItems)
	for i := 0; i < numItems; i++ {
		candidates[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mc.PredictNext(ctx, i%numItems, candidates)
	}
}
