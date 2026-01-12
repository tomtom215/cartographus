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

func TestNewLinUCB(t *testing.T) {
	tests := []struct {
		name   string
		cfg    LinUCBConfig
		verify func(t *testing.T, l *LinUCB)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  LinUCBConfig{},
			verify: func(t *testing.T, l *LinUCB) {
				if l.config.Alpha <= 0 {
					t.Errorf("Alpha = %f, want > 0", l.config.Alpha)
				}
				if l.config.NumFeatures <= 0 {
					t.Errorf("NumFeatures = %d, want > 0", l.config.NumFeatures)
				}
			},
		},
		{
			name: "uses provided config values",
			cfg: LinUCBConfig{
				Alpha:       2.0,
				NumFeatures: 64,
				DecayRate:   0.01,
			},
			verify: func(t *testing.T, l *LinUCB) {
				if l.config.Alpha != 2.0 {
					t.Errorf("Alpha = %f, want 2.0", l.config.Alpha)
				}
				if l.config.NumFeatures != 64 {
					t.Errorf("NumFeatures = %d, want 64", l.config.NumFeatures)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLinUCB(tt.cfg)
			if l == nil {
				t.Fatal("NewLinUCB() returned nil")
			}
			if l.Name() != "linucb" {
				t.Errorf("Name() = %q, want %q", l.Name(), "linucb")
			}
			tt.verify(t, l)
		})
	}
}

func TestLinUCB_Train(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	items := []recommend.Item{
		{ID: 100, Genres: []string{"Action"}, Year: 2020, Rating: 8.0},
		{ID: 101, Genres: []string{"Comedy"}, Year: 2021, Rating: 7.5},
		{ID: 102, Genres: []string{"Drama"}, Year: 2019, Rating: 9.0},
	}

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
			name: "trains with interactions and items",
			interactions: []recommend.Interaction{
				{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
				{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 0.8},
				{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 0.9},
			},
			items:       items,
			wantTrained: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLinUCB(LinUCBConfig{NumFeatures: 16})

			err := l.Train(context.Background(), tt.interactions, tt.items)
			if err != nil {
				t.Fatalf("Train() error = %v", err)
			}

			if l.IsTrained() != tt.wantTrained {
				t.Errorf("IsTrained() = %v, want %v", l.IsTrained(), tt.wantTrained)
			}

			if l.Version() < 1 {
				t.Errorf("Version() = %d, want >= 1", l.Version())
			}
		})
	}
}

func TestLinUCB_Predict(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	items := []recommend.Item{
		{ID: 100, Genres: []string{"Action"}, Year: 2020, Rating: 8.0},
		{ID: 101, Genres: []string{"Comedy"}, Year: 2021, Rating: 7.5},
		{ID: 102, Genres: []string{"Drama"}, Year: 2019, Rating: 9.0},
	}

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Timestamp: baseTime, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Timestamp: baseTime, Confidence: 0.8},
		{UserID: 2, ItemID: 100, Timestamp: baseTime, Confidence: 0.9},
		{UserID: 2, ItemID: 102, Timestamp: baseTime, Confidence: 1.0},
	}

	l := NewLinUCB(LinUCBConfig{
		Alpha:       1.0,
		NumFeatures: 16,
	})
	if err := l.Train(context.Background(), interactions, items); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	tests := []struct {
		name       string
		userID     int
		candidates []int
		wantScores bool
	}{
		{
			name:       "known user gets recommendations",
			userID:     1,
			candidates: []int{102},
			wantScores: true,
		},
		{
			name:       "new user gets exploration-based recommendations",
			userID:     999,
			candidates: []int{100, 101, 102},
			wantScores: true, // LinUCB provides exploration bonus for new users
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := l.Predict(context.Background(), tt.userID, tt.candidates)
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

func TestLinUCB_PredictSimilar(t *testing.T) {
	items := []recommend.Item{
		{ID: 100, Genres: []string{"Action"}, Year: 2020},
		{ID: 101, Genres: []string{"Action"}, Year: 2021}, // Similar to 100
		{ID: 102, Genres: []string{"Comedy"}, Year: 2020}, // Different genre
	}

	l := NewLinUCB(LinUCBConfig{NumFeatures: 16})
	if err := l.Train(context.Background(), nil, items); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	scores, err := l.PredictSimilar(context.Background(), 100, []int{101, 102})
	if err != nil {
		t.Fatalf("PredictSimilar() error = %v", err)
	}

	if len(scores) == 0 {
		t.Error("expected some similar item scores")
	}

	// Item 101 (same genre) should be more similar than 102
	if scores[101] < scores[102] {
		t.Errorf("item 101 should be more similar to 100 than item 102")
	}
}

func TestLinUCB_RecordFeedback(t *testing.T) {
	items := []recommend.Item{
		{ID: 100, Genres: []string{"Action"}, Year: 2020},
		{ID: 101, Genres: []string{"Comedy"}, Year: 2021},
	}

	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
	}

	l := NewLinUCB(LinUCBConfig{NumFeatures: 16})
	if err := l.Train(context.Background(), interactions, items); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	initialObs := l.totalObservations

	// Record positive feedback
	l.RecordFeedback(1, 101, 1.0)

	if l.totalObservations != initialObs+1 {
		t.Errorf("totalObservations = %d, want %d", l.totalObservations, initialObs+1)
	}

	// The arm should now have observations
	if l.observations[101] != 1 {
		t.Errorf("observations[101] = %d, want 1", l.observations[101])
	}
}

func TestLinUCB_GetExplorationRate(t *testing.T) {
	l := NewLinUCB(LinUCBConfig{Alpha: 1.0, NumFeatures: 8})

	// Before training
	rate := l.GetExplorationRate()
	if rate != 1.0 {
		t.Errorf("initial exploration rate = %f, want 1.0", rate)
	}

	// Train with some interactions
	interactions := make([]recommend.Interaction, 100)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:     i % 10,
			ItemID:     100 + i%5,
			Confidence: 0.5,
		}
	}

	items := []recommend.Item{
		{ID: 100}, {ID: 101}, {ID: 102}, {ID: 103}, {ID: 104},
	}

	if err := l.Train(context.Background(), interactions, items); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// After training, exploration rate should decrease
	rate = l.GetExplorationRate()
	if rate >= 1.0 {
		t.Errorf("exploration rate after training = %f, want < 1.0", rate)
	}
}

func TestLinUCB_ContextCancellation(t *testing.T) {
	l := NewLinUCB(LinUCBConfig{NumFeatures: 16})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	interactions := make([]recommend.Interaction, 500)
	items := make([]recommend.Item, 100)
	for i := range interactions {
		interactions[i] = recommend.Interaction{
			UserID:    i % 50,
			ItemID:    i % 100,
			Timestamp: time.Now(),
		}
	}
	for i := range items {
		items[i] = recommend.Item{ID: i, Genres: []string{"Genre"}}
	}

	err := l.Train(ctx, interactions, items)
	if err == nil {
		t.Error("Train() with canceled context should return error")
	}
}

func TestIdentityMatrix(t *testing.T) {
	m := identityMatrix(3)

	if len(m) != 3 {
		t.Fatalf("len(m) = %d, want 3", len(m))
	}

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			if m[i][j] != expected {
				t.Errorf("m[%d][%d] = %f, want %f", i, j, m[i][j], expected)
			}
		}
	}
}

func TestInvertMatrix(t *testing.T) {
	// Test with a simple 2x2 matrix
	A := [][]float64{
		{4, 7},
		{2, 6},
	}

	inv := invertMatrix(A)
	if inv == nil {
		t.Fatal("invertMatrix returned nil")
	}

	// Verify A * A^(-1) = I
	n := len(A)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			var sum float64
			for k := 0; k < n; k++ {
				sum += A[i][k] * inv[k][j]
			}

			expected := 0.0
			if i == j {
				expected = 1.0
			}

			diff := sum - expected
			if diff < -0.01 || diff > 0.01 {
				t.Errorf("(A*A^-1)[%d][%d] = %f, want %f", i, j, sum, expected)
			}
		}
	}
}

func TestLinUCB_DecayRate(t *testing.T) {
	l := NewLinUCB(LinUCBConfig{
		NumFeatures: 8,
		DecayRate:   0.1, // High decay for testing
	})

	items := []recommend.Item{{ID: 100}}
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
	}

	if err := l.Train(context.Background(), interactions, items); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	// Record many feedbacks
	for i := 0; i < 10; i++ {
		l.RecordFeedback(1, 100, 1.0)
	}

	// With decay, old observations should have less weight
	// Just verify it doesn't crash and model is still functional
	scores, err := l.Predict(context.Background(), 1, []int{100})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}

	if len(scores) == 0 {
		t.Error("expected scores after feedback recording")
	}
}
