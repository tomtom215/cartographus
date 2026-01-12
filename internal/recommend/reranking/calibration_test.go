// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package reranking

import (
	"context"
	"testing"

	"github.com/tomtom215/cartographus/internal/recommend"
)

func TestNewCalibration(t *testing.T) {
	tests := []struct {
		name   string
		cfg    CalibrationConfig
		verify func(t *testing.T, c *Calibration)
	}{
		{
			name: "applies defaults for zero config",
			cfg:  CalibrationConfig{},
			verify: func(t *testing.T, c *Calibration) {
				if len(c.config.AttributeWeights) == 0 {
					t.Error("expected default attribute weights")
				}
			},
		},
		{
			name: "clamps lambda to valid range",
			cfg:  CalibrationConfig{Lambda: 1.5},
			verify: func(t *testing.T, c *Calibration) {
				if c.config.Lambda != 1.0 {
					t.Errorf("Lambda = %f, want 1.0", c.config.Lambda)
				}
			},
		},
		{
			name: "uses provided config",
			cfg: CalibrationConfig{
				Lambda: 0.8,
				AttributeWeights: map[string]float64{
					"genre": 1.0,
					"year":  0.5,
				},
			},
			verify: func(t *testing.T, c *Calibration) {
				if c.config.Lambda != 0.8 {
					t.Errorf("Lambda = %f, want 0.8", c.config.Lambda)
				}
				if len(c.config.AttributeWeights) != 2 {
					t.Errorf("len(AttributeWeights) = %d, want 2", len(c.config.AttributeWeights))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCalibration(tt.cfg)
			if c == nil {
				t.Fatal("NewCalibration() returned nil")
			}
			if c.Name() != "calibration" {
				t.Errorf("Name() = %q, want %q", c.Name(), "calibration")
			}
			tt.verify(t, c)
		})
	}
}

func TestCalibration_Rerank(t *testing.T) {
	// Create items with varying genres
	items := []recommend.ScoredItem{
		{Item: recommend.Item{ID: 1, Genres: []string{"Action"}}, Score: 1.0},
		{Item: recommend.Item{ID: 2, Genres: []string{"Action"}}, Score: 0.9},
		{Item: recommend.Item{ID: 3, Genres: []string{"Comedy"}}, Score: 0.85},
		{Item: recommend.Item{ID: 4, Genres: []string{"Action"}}, Score: 0.8},
		{Item: recommend.Item{ID: 5, Genres: []string{"Drama"}}, Score: 0.75},
		{Item: recommend.Item{ID: 6, Genres: []string{"Comedy"}}, Score: 0.7},
	}

	tests := []struct {
		name        string
		lambda      float64
		k           int
		wantLen     int
		description string
	}{
		{
			name:        "pure relevance (lambda=1)",
			lambda:      1.0,
			k:           3,
			wantLen:     3,
			description: "returns top 3 by score",
		},
		{
			name:        "balanced (lambda=0.7)",
			lambda:      0.7,
			k:           3,
			wantLen:     3,
			description: "balances relevance and calibration",
		},
		{
			name:        "k larger than items",
			lambda:      0.7,
			k:           10,
			wantLen:     6,
			description: "returns all items when k > len",
		},
		{
			name:        "k zero returns input",
			lambda:      0.7,
			k:           0,
			wantLen:     6,
			description: "returns original list for k=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCalibration(CalibrationConfig{Lambda: tt.lambda})
			result := c.Rerank(context.Background(), items, tt.k)

			if len(result) != tt.wantLen {
				t.Errorf("len(result) = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestCalibration_LearnFromHistory(t *testing.T) {
	interactions := []recommend.Interaction{
		{UserID: 1, ItemID: 100, Confidence: 1.0},
		{UserID: 1, ItemID: 101, Confidence: 1.0},
		{UserID: 1, ItemID: 102, Confidence: 0.5},
		{UserID: 2, ItemID: 100, Confidence: 1.0},
	}

	items := map[int]recommend.Item{
		100: {ID: 100, Genres: []string{"Action"}, Year: 2020, ContentRating: "PG-13"},
		101: {ID: 101, Genres: []string{"Action", "Sci-Fi"}, Year: 2021, ContentRating: "R"},
		102: {ID: 102, Genres: []string{"Comedy"}, Year: 2015, ContentRating: "PG"},
	}

	c := NewCalibration(DefaultCalibrationConfig())
	c.LearnFromHistory(interactions, items)

	// Verify user 1 has a profile
	profile, ok := c.userProfiles[1]
	if !ok {
		t.Fatal("expected user 1 to have a profile")
	}

	// Verify genre distribution
	genreDist, ok := profile["genre"]
	if !ok {
		t.Fatal("expected genre distribution")
	}

	if genreDist["Action"] <= 0 {
		t.Error("expected positive Action count")
	}

	// Verify distribution is normalized
	var total float64
	for _, v := range genreDist {
		total += v
	}
	if total < 0.99 || total > 1.01 {
		t.Errorf("genre distribution sum = %f, want ~1.0", total)
	}
}

func TestCalibration_WithTargetDistribution(t *testing.T) {
	items := []recommend.ScoredItem{
		{Item: recommend.Item{ID: 1, Genres: []string{"Action"}}, Score: 1.0},
		{Item: recommend.Item{ID: 2, Genres: []string{"Action"}}, Score: 0.9},
		{Item: recommend.Item{ID: 3, Genres: []string{"Comedy"}}, Score: 0.85},
		{Item: recommend.Item{ID: 4, Genres: []string{"Drama"}}, Score: 0.8},
	}

	// Target: 50% Action, 25% Comedy, 25% Drama
	targetDist := map[string]map[string]float64{
		"genre": {
			"Action": 0.5,
			"Comedy": 0.25,
			"Drama":  0.25,
		},
	}

	c := NewCalibration(CalibrationConfig{
		Lambda: 0.5, // Strong calibration influence
		AttributeWeights: map[string]float64{
			"genre": 1.0,
		},
		TargetDistribution: targetDist,
	})

	result := c.Rerank(context.Background(), items, 4)

	if len(result) != 4 {
		t.Errorf("len(result) = %d, want 4", len(result))
	}

	// With calibration, should include diverse genres
	genresSeen := make(map[string]bool)
	for _, item := range result {
		for _, g := range item.Item.Genres {
			genresSeen[g] = true
		}
	}

	// Should see at least 2 different genres
	if len(genresSeen) < 2 {
		t.Errorf("expected genre diversity, only saw %v", genresSeen)
	}
}

func TestCalibration_EmptyInput(t *testing.T) {
	c := NewCalibration(DefaultCalibrationConfig())

	t.Run("nil items", func(t *testing.T) {
		result := c.Rerank(context.Background(), nil, 5)
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		result := c.Rerank(context.Background(), []recommend.ScoredItem{}, 5)
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})

	t.Run("single item", func(t *testing.T) {
		items := []recommend.ScoredItem{
			{Item: recommend.Item{ID: 1, Genres: []string{"Action"}}, Score: 1.0},
		}
		result := c.Rerank(context.Background(), items, 5)
		if len(result) != 1 {
			t.Errorf("expected 1 item, got %d", len(result))
		}
	})
}

func TestDecadeBucket(t *testing.T) {
	tests := []struct {
		year     int
		expected string
	}{
		{2024, "2020s"},
		{2020, "2020s"},
		{2015, "2010s"},
		{2005, "2000s"},
		{1995, "1990s"},
		{1985, "1980s"},
		{1975, "pre-1980"},
		{1950, "pre-1980"},
	}

	for _, tt := range tests {
		result := decadeBucket(tt.year)
		if result != tt.expected {
			t.Errorf("decadeBucket(%d) = %q, want %q", tt.year, result, tt.expected)
		}
	}
}

func TestKLDivergence(t *testing.T) {
	tests := []struct {
		name     string
		p        map[string]float64
		q        map[string]float64
		wantZero bool
	}{
		{
			name:     "identical distributions",
			p:        map[string]float64{"A": 0.5, "B": 0.5},
			q:        map[string]float64{"A": 0.5, "B": 0.5},
			wantZero: true,
		},
		{
			name:     "different distributions",
			p:        map[string]float64{"A": 0.9, "B": 0.1},
			q:        map[string]float64{"A": 0.1, "B": 0.9},
			wantZero: false,
		},
		{
			name:     "empty distributions",
			p:        map[string]float64{},
			q:        map[string]float64{},
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kl := klDivergence(tt.p, tt.q)

			if tt.wantZero && kl > 0.001 {
				t.Errorf("KL = %f, want ~0", kl)
			}
			if !tt.wantZero && kl <= 0.001 {
				t.Errorf("KL = %f, want > 0", kl)
			}
			if kl < 0 {
				t.Errorf("KL = %f, should be non-negative", kl)
			}
		})
	}
}

func TestCalibration_SetUserProfile(t *testing.T) {
	c := NewCalibration(DefaultCalibrationConfig())

	profile := map[string]map[string]float64{
		"genre": {
			"Action": 0.6,
			"Comedy": 0.4,
		},
	}

	c.SetUserProfile(123, profile)

	stored, ok := c.userProfiles[123]
	if !ok {
		t.Fatal("expected profile to be stored")
	}

	if stored["genre"]["Action"] != 0.6 {
		t.Errorf("Action = %f, want 0.6", stored["genre"]["Action"])
	}
}

func TestNormalizeDistribution(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]float64
	}{
		{
			name:  "normal values",
			input: map[string]float64{"A": 3, "B": 2, "C": 5},
		},
		{
			name:  "already normalized",
			input: map[string]float64{"A": 0.5, "B": 0.5},
		},
		{
			name:  "empty",
			input: map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := make(map[string]float64)
			for k, v := range tt.input {
				dist[k] = v
			}

			normalizeDistribution(dist)

			var sum float64
			for _, v := range dist {
				sum += v
			}

			if len(dist) > 0 && (sum < 0.99 || sum > 1.01) {
				t.Errorf("sum = %f, want ~1.0", sum)
			}
		})
	}
}
