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

func TestNewContentBased(t *testing.T) {
	tests := []struct {
		name   string
		cfg    ContentBasedConfig
		verify func(t *testing.T, cb *ContentBased)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  ContentBasedConfig{},
			verify: func(t *testing.T, cb *ContentBased) {
				if cb.genreWeight <= 0 {
					t.Errorf("genreWeight = %f, want > 0", cb.genreWeight)
				}
				if cb.maxYearDifference <= 0 {
					t.Errorf("maxYearDifference = %d, want > 0", cb.maxYearDifference)
				}
			},
		},
		{
			name: "uses provided config values",
			cfg: ContentBasedConfig{
				GenreWeight:       0.5,
				ActorWeight:       0.2,
				DirectorWeight:    0.2,
				YearWeight:        0.1,
				MaxYearDifference: 10,
			},
			verify: func(t *testing.T, cb *ContentBased) {
				if cb.maxYearDifference != 10 {
					t.Errorf("maxYearDifference = %d, want 10", cb.maxYearDifference)
				}
			},
		},
		{
			name: "normalizes weights to sum to 1",
			cfg: ContentBasedConfig{
				GenreWeight:    2.0,
				ActorWeight:    2.0,
				DirectorWeight: 2.0,
				YearWeight:     2.0,
			},
			verify: func(t *testing.T, cb *ContentBased) {
				sum := cb.genreWeight + cb.actorWeight + cb.directorWeight + cb.yearWeight
				if sum < 0.99 || sum > 1.01 {
					t.Errorf("weights sum = %f, want ~1.0", sum)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewContentBased(tt.cfg)
			if cb == nil {
				t.Fatal("NewContentBased() returned nil")
			}
			if cb.Name() != "content" {
				t.Errorf("Name() = %q, want %q", cb.Name(), "content")
			}
			tt.verify(t, cb)
		})
	}
}

func TestContentBased_Train(t *testing.T) {
	items := []recommend.Item{
		{ID: 100, Genres: []string{"Action", "Sci-Fi"}, Actors: []string{"Actor A"}, Year: 2020},
		{ID: 101, Genres: []string{"Comedy"}, Actors: []string{"Actor B"}, Year: 2021},
		{ID: 102, Genres: []string{"Action"}, Actors: []string{"Actor A"}, Directors: []string{"Director X"}, Year: 2022},
	}

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		items        []recommend.Item
		wantTrained  bool
		wantProfiles int
	}{
		{
			name:         "empty interactions trains successfully",
			interactions: nil,
			items:        items,
			wantTrained:  true,
			wantProfiles: 0,
		},
		{
			name: "builds user profiles from interactions",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Type: recommend.InteractionCompleted, Confidence: 1.0},
				{UserID: 1, ItemID: 102, Type: recommend.InteractionCompleted, Confidence: 1.0},
				{UserID: 2, ItemID: 101, Type: recommend.InteractionEngaged, Confidence: 0.7},
			},
			items:        items,
			wantTrained:  true,
			wantProfiles: 2,
		},
		{
			name: "ignores abandoned interactions",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Type: recommend.InteractionAbandoned, Confidence: 0.1},
			},
			items:        items,
			wantTrained:  true,
			wantProfiles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewContentBased(ContentBasedConfig{})

			err := cb.Train(context.Background(), tt.interactions, tt.items)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if cb.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", cb.IsTrained(), tt.wantTrained)
			}

			if len(cb.userProfiles) != tt.wantProfiles {
				t.Errorf("len(userProfiles) = %d, want %d", len(cb.userProfiles), tt.wantProfiles)
			}
		})
	}
}

func TestContentBased_Predict(t *testing.T) {
	items := []recommend.Item{
		{ID: 100, Genres: []string{"Action", "Sci-Fi"}, Actors: []string{"Actor A"}, Year: 2020},
		{ID: 101, Genres: []string{"Comedy"}, Actors: []string{"Actor B"}, Year: 2021},
		{ID: 102, Genres: []string{"Action"}, Actors: []string{"Actor A"}, Year: 2022},
		{ID: 103, Genres: []string{"Drama"}, Actors: []string{"Actor C"}, Year: 2019},
	}

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Type: recommend.InteractionCompleted, Confidence: 1.0},
	}

	cb := NewContentBased(ContentBasedConfig{})
	if err := cb.Train(context.Background(), interactions, items); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name        string
		userID      int
		candidates  []int
		expectHigh  int  // Item expected to score highest
		expectLow   int  // Item expected to score lowest
		expectEmpty bool // Expect no scores
	}{
		{
			name:       "user who likes Action prefers Action items",
			userID:     1,
			candidates: []int{101, 102, 103},
			expectHigh: 102, // Action with same actor
			expectLow:  103, // Drama, different actor
		},
		{
			name:        "unknown user gets no recommendations",
			userID:      999,
			candidates:  []int{100, 101, 102},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := cb.Predict(context.Background(), tt.userID, tt.candidates)
			if err != nil {
				t.Fatalf("Predict() error = %v", err)
			}

			if tt.expectEmpty {
				if len(scores) > 0 {
					t.Errorf("expected empty scores, got %d", len(scores))
				}
				return
			}

			if len(scores) == 0 {
				t.Fatal("expected scores, got empty")
			}

			// Verify high/low expectations
			if tt.expectHigh > 0 && tt.expectLow > 0 {
				highScore, hOk := scores[tt.expectHigh]
				lowScore, lOk := scores[tt.expectLow]
				if hOk && lOk && highScore <= lowScore {
					t.Errorf("item %d score (%f) should be > item %d score (%f)",
						tt.expectHigh, highScore, tt.expectLow, lowScore)
				}
			}
		})
	}
}

func TestContentBased_PredictSimilar(t *testing.T) {
	items := []recommend.Item{
		{ID: 100, Genres: []string{"Action", "Sci-Fi"}, Actors: []string{"Actor A"}, Directors: []string{"Dir X"}, Year: 2020},
		{ID: 101, Genres: []string{"Action"}, Actors: []string{"Actor A"}, Directors: []string{"Dir X"}, Year: 2021},
		{ID: 102, Genres: []string{"Comedy"}, Actors: []string{"Actor B"}, Directors: []string{"Dir Y"}, Year: 2020},
		{ID: 103, Genres: []string{"Action"}, Actors: []string{"Actor C"}, Directors: []string{"Dir Z"}, Year: 2000},
	}

	cb := NewContentBased(ContentBasedConfig{})
	if err := cb.Train(context.Background(), nil, items); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		itemID     int
		candidates []int
		expectHigh int // Most similar item
	}{
		{
			name:       "finds similar Action items",
			itemID:     100,
			candidates: []int{101, 102, 103},
			expectHigh: 101, // Same genre, actor, director, close year
		},
		{
			name:       "Comedy is dissimilar to Action",
			itemID:     102,
			candidates: []int{100, 101, 103},
			expectHigh: -1, // All should score low
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := cb.PredictSimilar(context.Background(), tt.itemID, tt.candidates)
			if err != nil {
				t.Fatalf("PredictSimilar() error = %v", err)
			}

			if tt.expectHigh > 0 {
				var maxID int
				var maxScore float64
				for id, score := range scores {
					if score > maxScore {
						maxScore = score
						maxID = id
					}
				}
				if maxID != tt.expectHigh {
					t.Errorf("highest scoring item = %d, want %d", maxID, tt.expectHigh)
				}
			}
		})
	}
}

func TestContentBased_ContextCancellation(t *testing.T) {
	cb := NewContentBased(ContentBasedConfig{})

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create dataset
	items := make([]recommend.Item, 1000)
	interactions := make([]recommend.Interaction, 10000)
	for i := range items {
		items[i] = recommend.Item{ID: i, Genres: []string{"Genre"}}
	}
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:    i % 100,
			ItemID:    i % 1000,
			Type:      recommend.InteractionCompleted,
			Timestamp: time.Now(),
		}
	}

	err := cb.Train(ctx, interactions, items)
	if err == nil {
		t.Error("Train() with canceled context should return error")
	}
}
