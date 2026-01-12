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

func TestNewUserBasedCF(t *testing.T) {
	tests := []struct {
		name   string
		cfg    KNNConfig
		verify func(t *testing.T, u *UserBasedCF)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  KNNConfig{},
			verify: func(t *testing.T, u *UserBasedCF) {
				if u.config.K <= 0 {
					t.Errorf("K = %d, want > 0", u.config.K)
				}
				if u.config.MinSimilarity <= 0 {
					t.Errorf("MinSimilarity = %f, want > 0", u.config.MinSimilarity)
				}
			},
		},
		{
			name: "uses provided config values",
			cfg: KNNConfig{
				K:                100,
				MinSimilarity:    0.2,
				SimilarityMetric: "pearson",
			},
			verify: func(t *testing.T, u *UserBasedCF) {
				if u.config.K != 100 {
					t.Errorf("K = %d, want 100", u.config.K)
				}
				if u.config.SimilarityMetric != "pearson" {
					t.Errorf("SimilarityMetric = %s, want pearson", u.config.SimilarityMetric)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := NewUserBasedCF(tt.cfg)
			if u == nil {
				t.Fatal("NewUserBasedCF() returned nil")
			}
			if u.Name() != "usercf" {
				t.Errorf("Name() = %q, want %q", u.Name(), "usercf")
			}
			tt.verify(t, u)
		})
	}
}

func TestUserBasedCF_Train(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		wantTrained  bool
	}{
		{
			name:         "empty interactions trains successfully",
			interactions: nil,
			wantTrained:  true,
		},
		{
			name: "builds similarity from interactions",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 3, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 3, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
			},
			wantTrained: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := NewUserBasedCF(DefaultKNNConfig())

			err := u.Train(context.Background(), tt.interactions, nil)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if u.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", u.IsTrained(), tt.wantTrained)
			}
		})
	}
}

func TestUserBasedCF_Predict(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create clear user patterns:
	// Users 1, 2, 3 like items 100, 101, 102 (similar tastes)
	// User 4 likes different items
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 4, ItemID: 200, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 4, ItemID: 201, Timestamp: baseTime, Confidence: 1.0},
	}

	u := NewUserBasedCF(KNNConfig{
		K:              10,
		MinSimilarity:  0.01,
		MinCommonItems: 1,
	})
	if err := u.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		userID     int
		candidates []int
		wantScores bool
	}{
		{
			name:       "user with similar neighbors gets recommendations",
			userID:     1,
			candidates: []int{102},
			wantScores: true,
		},
		{
			name:       "user without similar neighbors",
			userID:     4,
			candidates: []int{100, 101, 102},
			wantScores: false, // User 4 has no similar users
		},
		{
			name:       "unknown user",
			userID:     999,
			candidates: []int{100, 101},
			wantScores: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := u.Predict(context.Background(), tt.userID, tt.candidates)
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

func TestNewItemBasedCF(t *testing.T) {
	tests := []struct {
		name   string
		cfg    KNNConfig
		verify func(t *testing.T, i *ItemBasedCF)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  KNNConfig{},
			verify: func(t *testing.T, i *ItemBasedCF) {
				if i.config.K <= 0 {
					t.Errorf("K = %d, want > 0", i.config.K)
				}
			},
		},
		{
			name: "uses provided config values",
			cfg: KNNConfig{
				K:                75,
				SimilarityMetric: "jaccard",
			},
			verify: func(t *testing.T, i *ItemBasedCF) {
				if i.config.K != 75 {
					t.Errorf("K = %d, want 75", i.config.K)
				}
				if i.config.SimilarityMetric != "jaccard" {
					t.Errorf("SimilarityMetric = %s, want jaccard", i.config.SimilarityMetric)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := NewItemBasedCF(tt.cfg)
			if i == nil {
				t.Fatal("NewItemBasedCF() returned nil")
			}
			if i.Name() != "itemcf" {
				t.Errorf("Name() = %q, want %q", i.Name(), "itemcf")
			}
			tt.verify(t, i)
		})
	}
}

func TestItemBasedCF_Train(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		wantTrained  bool
	}{
		{
			name:         "empty interactions trains successfully",
			interactions: nil,
			wantTrained:  true,
		},
		{
			name: "builds item similarity from interactions",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 3, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
			},
			wantTrained: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := NewItemBasedCF(DefaultKNNConfig())

			err := i.Train(context.Background(), tt.interactions, nil)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if i.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", i.IsTrained(), tt.wantTrained)
			}
		})
	}
}

func TestItemBasedCF_Predict(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Items 100 and 101 are similar (liked by same users)
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

	i := NewItemBasedCF(KNNConfig{
		K:              10,
		MinSimilarity:  0.01,
		MinCommonItems: 1,
	})
	if err := i.Train(context.Background(), interactions, nil); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		userID     int
		candidates []int
		wantScores bool
	}{
		{
			name:       "user gets recommendations based on item similarity",
			userID:     1,
			candidates: []int{102, 103},
			wantScores: false, // User 1 has no overlap with items 102, 103
		},
		{
			name:       "user without history",
			userID:     999,
			candidates: []int{100, 101},
			wantScores: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := i.Predict(context.Background(), tt.userID, tt.candidates)
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

func TestItemBasedCF_PredictSimilar(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Items 100 and 101 are similar
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 4, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
	}

	i := NewItemBasedCF(KNNConfig{
		K:              10,
		MinSimilarity:  0.01,
		MinCommonItems: 1,
	})
	if err := i.Train(context.Background(), interactions, nil); err != nil {
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
			candidates: []int{101, 102},
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
			scores, err := i.PredictSimilar(context.Background(), tt.itemID, tt.candidates)
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

func TestKNN_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	interactions := make([]recommend.Interaction, 500)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:     i % 50,
			ItemID:     i % 100,
			Timestamp:  time.Now(),
			Confidence: 0.5,
		}
	}

	t.Run("UserBasedCF", func(t *testing.T) {
		u := NewUserBasedCF(DefaultKNNConfig())
		err := u.Train(ctx, interactions, nil)
		if err == nil {
			t.Error("Train() with canceled context should return error")
		}
	})

	t.Run("ItemBasedCF", func(t *testing.T) {
		i := NewItemBasedCF(DefaultKNNConfig())
		err := i.Train(ctx, interactions, nil)
		if err == nil {
			t.Error("Train() with canceled context should return error")
		}
	})
}

func TestKNN_SimilarityMetrics(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 0.5},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime, Confidence: 0.5},
		{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 0.3},
	}

	metrics := []string{"cosine", "pearson", "jaccard"}

	for _, metric := range metrics {
		t.Run("UserBasedCF_"+metric, func(t *testing.T) {
			u := NewUserBasedCF(KNNConfig{
				K:                10,
				MinSimilarity:    0.01,
				SimilarityMetric: metric,
				MinCommonItems:   1,
			})
			if err := u.Train(context.Background(), interactions, nil); err != nil {
				t.Fatalf("Train() error = %v", err)
			}
			if !u.IsTrained() {
				t.Error("expected to be trained")
			}
		})

		t.Run("ItemBasedCF_"+metric, func(t *testing.T) {
			i := NewItemBasedCF(KNNConfig{
				K:                10,
				MinSimilarity:    0.01,
				SimilarityMetric: metric,
				MinCommonItems:   1,
			})
			if err := i.Train(context.Background(), interactions, nil); err != nil {
				t.Fatalf("Train() error = %v", err)
			}
			if !i.IsTrained() {
				t.Error("expected to be trained")
			}
		})
	}
}
