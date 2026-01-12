// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package recommend

import (
	"testing"
)

func TestInteractionType_String(t *testing.T) {
	tests := []struct {
		name     string
		itype    InteractionType
		expected string
	}{
		{"abandoned", InteractionAbandoned, "abandoned"},
		{"sampled", InteractionSampled, "sampled"},
		{"engaged", InteractionEngaged, "engaged"},
		{"completed", InteractionCompleted, "completed"},
		{"unknown value", InteractionType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.itype.String()
			if result != tt.expected {
				t.Errorf("InteractionType(%d).String() = %q, want %q", tt.itype, result, tt.expected)
			}
		})
	}
}

func TestInteractionType_Confidence(t *testing.T) {
	tests := []struct {
		name     string
		itype    InteractionType
		expected float64
	}{
		{"completed has highest confidence", InteractionCompleted, 1.0},
		{"engaged has moderate confidence", InteractionEngaged, 0.7},
		{"sampled has low confidence", InteractionSampled, 0.3},
		{"abandoned has minimal confidence", InteractionAbandoned, 0.1},
		{"unknown has zero confidence", InteractionType(99), 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.itype.Confidence()
			if result != tt.expected {
				t.Errorf("InteractionType(%d).Confidence() = %f, want %f", tt.itype, result, tt.expected)
			}
		})
	}
}

func TestClassifyInteraction(t *testing.T) {
	tests := []struct {
		name            string
		percentComplete int
		expected        InteractionType
	}{
		{"0% is abandoned", 0, InteractionAbandoned},
		{"5% is abandoned", 5, InteractionAbandoned},
		{"9% is abandoned", 9, InteractionAbandoned},
		{"10% is sampled", 10, InteractionSampled},
		{"25% is sampled", 25, InteractionSampled},
		{"49% is sampled", 49, InteractionSampled},
		{"50% is engaged", 50, InteractionEngaged},
		{"75% is engaged", 75, InteractionEngaged},
		{"89% is engaged", 89, InteractionEngaged},
		{"90% is completed", 90, InteractionCompleted},
		{"95% is completed", 95, InteractionCompleted},
		{"100% is completed", 100, InteractionCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyInteraction(tt.percentComplete)
			if result != tt.expected {
				t.Errorf("ClassifyInteraction(%d) = %v, want %v", tt.percentComplete, result, tt.expected)
			}
		})
	}
}

func TestComputeConfidence(t *testing.T) {
	tests := []struct {
		name                string
		percentComplete     int
		playDurationSeconds int
		minExpected         float64
		maxExpected         float64
	}{
		{
			name:            "0% completion has base confidence",
			percentComplete: 0,
			minExpected:     1.0,
			maxExpected:     1.1,
		},
		{
			name:            "100% completion has higher confidence",
			percentComplete: 100,
			minExpected:     10.0,
			maxExpected:     12.0,
		},
		{
			name:                "long watch adds confidence",
			percentComplete:     50,
			playDurationSeconds: 3600,
			minExpected:         6.0,
			maxExpected:         7.0,
		},
		{
			name:                "very long watch has diminishing returns",
			percentComplete:     50,
			playDurationSeconds: 36000,
			minExpected:         6.0,
			maxExpected:         7.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeConfidence(tt.percentComplete, tt.playDurationSeconds)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("ComputeConfidence(%d, %d) = %f, want in [%f, %f]",
					tt.percentComplete, tt.playDurationSeconds, result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestRecommendMode_String(t *testing.T) {
	tests := []struct {
		name     string
		mode     RecommendMode
		expected string
	}{
		{"personalized", ModePersonalized, "personalized"},
		{"continue_watching", ModeContinueWatching, "continue_watching"},
		{"similar", ModeSimilar, "similar"},
		{"explore", ModeExplore, "explore"},
		{"popular", ModePopular, "popular"},
		{"unknown value", RecommendMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mode.String()
			if result != tt.expected {
				t.Errorf("RecommendMode(%d).String() = %q, want %q", tt.mode, result, tt.expected)
			}
		})
	}
}
