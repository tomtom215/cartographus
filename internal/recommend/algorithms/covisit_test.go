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

func TestNewCoVisitation(t *testing.T) {
	tests := []struct {
		name   string
		cfg    CoVisitConfig
		verify func(t *testing.T, cv *CoVisitation)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  CoVisitConfig{},
			verify: func(t *testing.T, cv *CoVisitation) {
				if cv.minCoOccurrence < 1 {
					t.Errorf("minCoOccurrence = %d, want >= 1", cv.minCoOccurrence)
				}
				if cv.sessionWindowHours < 1 {
					t.Errorf("sessionWindowHours = %d, want >= 1", cv.sessionWindowHours)
				}
				if cv.maxPairs < 1 {
					t.Errorf("maxPairs = %d, want >= 1", cv.maxPairs)
				}
			},
		},
		{
			name: "uses provided config values",
			cfg: CoVisitConfig{
				MinCoOccurrence:    5,
				SessionWindowHours: 12,
				MaxPairs:           50000,
			},
			verify: func(t *testing.T, cv *CoVisitation) {
				if cv.minCoOccurrence != 5 {
					t.Errorf("minCoOccurrence = %d, want 5", cv.minCoOccurrence)
				}
				if cv.sessionWindowHours != 12 {
					t.Errorf("sessionWindowHours = %d, want 12", cv.sessionWindowHours)
				}
				if cv.maxPairs != 50000 {
					t.Errorf("maxPairs = %d, want 50000", cv.maxPairs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCoVisitation(tt.cfg)
			if cv == nil {
				t.Fatal("NewCoVisitation() returned nil")
			}
			if cv.Name() != "covisit" {
				t.Errorf("Name() = %q, want %q", cv.Name(), "covisit")
			}
			tt.verify(t, cv)
		})
	}
}

func TestCoVisitation_Train(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		interactions []recommend.Interaction
		items        []recommend.Item
		wantTrained  bool
	}{
		{
			name:         "empty interactions trains successfully",
			interactions: nil,
			items:        nil,
			wantTrained:  true,
		},
		{
			name: "single user single item",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
			},
			wantTrained: true,
		},
		{
			name: "builds co-occurrence for same session",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour), Confidence: 1.0},
				{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 2, ItemID: 101, Timestamp: baseTime.Add(30 * time.Minute), Confidence: 1.0},
			},
			wantTrained: true,
		},
		{
			name: "separates different sessions",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(48 * time.Hour), Confidence: 1.0}, // Different session
			},
			wantTrained: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCoVisitation(CoVisitConfig{MinCoOccurrence: 2})

			err := cv.Train(context.Background(), tt.interactions, tt.items)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if cv.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", cv.IsTrained(), tt.wantTrained)
			}

			if cv.Version() < 1 {
				t.Errorf("Version() = %d, want >= 1", cv.Version())
			}
		})
	}
}

func TestCoVisitation_Predict(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Setup: Create interactions where items 100 and 101 are frequently watched together
	interactions := []recommend.Interaction{
		// User 1 session
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour), Confidence: 1.0},
		// User 2 session
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime.Add(30 * time.Minute), Confidence: 1.0},
		// User 3 session
		{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Timestamp: baseTime.Add(2 * time.Hour), Confidence: 1.0},
		{UserID: 3, ItemID: 102, Timestamp: baseTime.Add(3 * time.Hour), Confidence: 1.0},
	}

	cv := NewCoVisitation(CoVisitConfig{MinCoOccurrence: 2})
	if err := cv.Train(context.Background(), interactions, nil); err != nil {
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
			candidates: []int{101, 102, 103},
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
			scores, err := cv.Predict(context.Background(), tt.userID, tt.candidates)
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

func TestCoVisitation_PredictSimilar(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Setup: Create co-occurrence between items
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime.Add(1 * time.Hour), Confidence: 1.0},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 2, ItemID: 101, Timestamp: baseTime.Add(30 * time.Minute), Confidence: 1.0},
		{UserID: 3, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 3, ItemID: 101, Timestamp: baseTime.Add(45 * time.Minute), Confidence: 1.0},
	}

	cv := NewCoVisitation(CoVisitConfig{MinCoOccurrence: 2})
	if err := cv.Train(context.Background(), interactions, nil); err != nil {
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
			name:       "no similar items for unknown item",
			itemID:     999,
			candidates: []int{100, 101, 102},
			expectItem: -1, // No expectation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := cv.PredictSimilar(context.Background(), tt.itemID, tt.candidates)
			if err != nil {
				t.Fatalf("PredictSimilar() error = %v", err)
			}

			if tt.expectItem > 0 {
				if _, ok := scores[tt.expectItem]; !ok {
					t.Errorf("expected item %d to have a score", tt.expectItem)
				}
			}
		})
	}
}

func TestCoVisitation_ContextCancellation(t *testing.T) {
	cv := NewCoVisitation(CoVisitConfig{})

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Large dataset to ensure cancellation is checked
	interactions := make([]recommend.Interaction, 10000)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:    i % 100,
			ItemID:    i % 500,
			Timestamp: time.Now(),
		}
	}

	err := cv.Train(ctx, interactions, nil)
	if err == nil {
		t.Error("Train() with canceled context should return error")
	}
}
