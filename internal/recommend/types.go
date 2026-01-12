// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package recommend

import (
	"context"
	"time"
)

// InteractionType classifies user-item interactions for implicit feedback.
type InteractionType int

const (
	// InteractionAbandoned indicates content was abandoned (< 10% watched).
	InteractionAbandoned InteractionType = iota
	// InteractionSampled indicates content was sampled (10-50% watched).
	InteractionSampled
	// InteractionEngaged indicates content was engaged with (50-90% watched).
	InteractionEngaged
	// InteractionCompleted indicates content was completed (>= 90% watched).
	InteractionCompleted
)

// String returns a human-readable name for the interaction type.
func (t InteractionType) String() string {
	switch t {
	case InteractionAbandoned:
		return "abandoned"
	case InteractionSampled:
		return "sampled"
	case InteractionEngaged:
		return "engaged"
	case InteractionCompleted:
		return "completed"
	default:
		return "unknown"
	}
}

// Confidence returns the confidence weight for this interaction type.
// Higher values indicate stronger positive signal.
func (t InteractionType) Confidence() float64 {
	switch t {
	case InteractionCompleted:
		return 1.0
	case InteractionEngaged:
		return 0.7
	case InteractionSampled:
		return 0.3
	case InteractionAbandoned:
		return 0.1 // Non-zero to avoid singularities
	default:
		return 0.0
	}
}

// Interaction represents a user-item interaction event.
type Interaction struct {
	// UserID is the internal user identifier.
	UserID int `json:"user_id"`

	// ItemID is the rating key of the content item.
	ItemID int `json:"item_id"`

	// Type classifies the interaction based on completion percentage.
	Type InteractionType `json:"type"`

	// Confidence is the computed confidence score for implicit feedback.
	// Higher values indicate stronger preference signals.
	Confidence float64 `json:"confidence"`

	// PercentComplete is the playback completion percentage (0-100).
	PercentComplete int `json:"percent_complete"`

	// PlayDuration is the total playback time in seconds.
	PlayDuration int `json:"play_duration"`

	// Timestamp is when the interaction occurred.
	Timestamp time.Time `json:"timestamp"`

	// SessionID groups interactions within a viewing session.
	SessionID string `json:"session_id,omitempty"`
}

// Item represents a content item with metadata for recommendations.
type Item struct {
	// ID is the rating key (unique content identifier).
	ID int `json:"id"`

	// Title is the content title.
	Title string `json:"title"`

	// MediaType is the content type (movie, episode, track).
	MediaType string `json:"media_type"`

	// Genres is a slice of genre names.
	Genres []string `json:"genres"`

	// Directors is a slice of director names.
	Directors []string `json:"directors"`

	// Actors is a slice of actor names.
	Actors []string `json:"actors"`

	// Year is the release year.
	Year int `json:"year"`

	// Studio is the production studio.
	Studio string `json:"studio,omitempty"`

	// ContentRating is the MPAA/TV rating (PG, R, TV-MA, etc.).
	ContentRating string `json:"content_rating,omitempty"`

	// Rating is the critic rating (0-10).
	Rating float64 `json:"rating,omitempty"`

	// AudienceRating is the audience rating (0-10).
	AudienceRating float64 `json:"audience_rating,omitempty"`

	// ParentID is the parent rating key (series for episodes).
	ParentID int `json:"parent_id,omitempty"`

	// GrandparentID is the grandparent rating key (series for episodes).
	GrandparentID int `json:"grandparent_id,omitempty"`

	// PopularityScore is a pre-computed popularity metric.
	PopularityScore float64 `json:"popularity_score,omitempty"`
}

// ScoredItem represents an item with a recommendation score.
type ScoredItem struct {
	// Item is the content item metadata.
	Item Item `json:"item"`

	// Score is the combined recommendation score (0-1, higher is better).
	Score float64 `json:"score"`

	// Scores is a breakdown of scores by algorithm.
	Scores map[string]float64 `json:"scores,omitempty"`

	// Reason provides an interpretable explanation for the recommendation.
	Reason string `json:"reason,omitempty"`
}

// Request represents a recommendation request.
type Request struct {
	// UserID is the user to generate recommendations for.
	UserID int `json:"user_id"`

	// K is the number of recommendations to return.
	// Defaults to Config.Limits.DefaultK if zero.
	K int `json:"k,omitempty"`

	// Exclude is a set of item IDs to exclude from recommendations.
	// Typically contains the user's watch history.
	Exclude map[int]struct{} `json:"-"`

	// ExcludeIDs is the JSON-serializable version of Exclude.
	ExcludeIDs []int `json:"exclude_ids,omitempty"`

	// CurrentItemID is the item the user is currently viewing.
	// Used for "similar to this" recommendations.
	CurrentItemID int `json:"current_item_id,omitempty"`

	// Mode specifies the recommendation mode.
	Mode RecommendMode `json:"mode,omitempty"`

	// Context provides additional contextual information.
	Context *RequestContext `json:"context,omitempty"`

	// RequestID is a unique identifier for tracing.
	RequestID string `json:"request_id,omitempty"`
}

// RecommendMode specifies the type of recommendations to generate.
type RecommendMode int

const (
	// ModePersonalized generates personalized recommendations.
	ModePersonalized RecommendMode = iota
	// ModeContinueWatching surfaces in-progress content.
	ModeContinueWatching
	// ModeSimilar generates "more like this" recommendations.
	ModeSimilar
	// ModeExplore emphasizes discovery over exploitation.
	ModeExplore
	// ModePopular returns popularity-ranked content.
	ModePopular
)

// String returns a human-readable mode name.
func (m RecommendMode) String() string {
	switch m {
	case ModePersonalized:
		return "personalized"
	case ModeContinueWatching:
		return "continue_watching"
	case ModeSimilar:
		return "similar"
	case ModeExplore:
		return "explore"
	case ModePopular:
		return "popular"
	default:
		return "unknown"
	}
}

// RequestContext provides contextual information for recommendations.
type RequestContext struct {
	// TimeOfDay is the hour (0-23) for time-aware recommendations.
	TimeOfDay int `json:"time_of_day,omitempty"`

	// DayOfWeek is the day (0=Sunday, 6=Saturday).
	DayOfWeek int `json:"day_of_week,omitempty"`

	// Device is the playback device type.
	Device string `json:"device,omitempty"`

	// Platform is the client platform (web, mobile, tv).
	Platform string `json:"platform,omitempty"`
}

// Response represents a recommendation response.
type Response struct {
	// Items is the ordered list of recommended items.
	Items []ScoredItem `json:"items"`

	// TotalCandidates is the number of candidate items considered.
	TotalCandidates int `json:"total_candidates"`

	// Metadata contains timing and diagnostic information.
	Metadata ResponseMetadata `json:"metadata"`
}

// ResponseMetadata contains timing and diagnostic information.
type ResponseMetadata struct {
	// RequestID is the unique request identifier.
	RequestID string `json:"request_id"`

	// UserID is the user the recommendations are for.
	UserID int `json:"user_id"`

	// Mode is the recommendation mode used.
	Mode string `json:"mode"`

	// AlgorithmsUsed lists the algorithms that contributed scores.
	AlgorithmsUsed []string `json:"algorithms_used"`

	// LatencyMS is the total recommendation latency in milliseconds.
	LatencyMS int64 `json:"latency_ms"`

	// CacheHit indicates whether the result was served from cache.
	CacheHit bool `json:"cache_hit"`

	// ModelVersion is the version of the trained model used.
	ModelVersion int `json:"model_version"`

	// TrainedAt is when the model was last trained.
	TrainedAt time.Time `json:"trained_at"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
}

// Algorithm defines the interface all recommendation algorithms must implement.
type Algorithm interface {
	// Name returns the algorithm identifier (e.g., "ease", "content", "covisit").
	Name() string

	// Train fits the model on interaction data.
	// Returns an error if training fails.
	Train(ctx context.Context, interactions []Interaction, items []Item) error

	// Predict returns scores for candidate items for a user.
	// The returned map contains item IDs as keys and scores as values.
	// Scores should be normalized to [0, 1] range.
	Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error)

	// PredictSimilar returns items similar to the given item.
	// Used for "more like this" recommendations.
	PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error)

	// IsTrained returns whether the model has been trained.
	IsTrained() bool

	// Version returns the model version (incremented on each train).
	Version() int

	// LastTrainedAt returns when the model was last trained.
	LastTrainedAt() time.Time
}

// Reranker modifies a ranked list for diversity or other objectives.
type Reranker interface {
	// Name returns the reranker identifier (e.g., "mmr", "calibration").
	Name() string

	// Rerank modifies the order of scored items to optimize a secondary objective.
	// The input items are already scored and sorted by relevance.
	// Returns up to k items with potentially reordered rankings.
	Rerank(ctx context.Context, items []ScoredItem, k int) []ScoredItem
}

// TrainingStatus represents the current training state.
type TrainingStatus struct {
	// IsTraining indicates whether training is currently in progress.
	IsTraining bool `json:"is_training"`

	// Progress is the training progress (0-100).
	Progress int `json:"progress"`

	// CurrentAlgorithm is the algorithm currently being trained.
	CurrentAlgorithm string `json:"current_algorithm,omitempty"`

	// LastTrainedAt is when training last completed.
	LastTrainedAt time.Time `json:"last_trained_at"`

	// LastTrainingDurationMS is how long the last training took.
	LastTrainingDurationMS int64 `json:"last_training_duration_ms"`

	// LastError contains the last training error, if any.
	LastError string `json:"last_error,omitempty"`

	// InteractionCount is the number of interactions in the training set.
	InteractionCount int `json:"interaction_count"`

	// ItemCount is the number of unique items.
	ItemCount int `json:"item_count"`

	// UserCount is the number of unique users.
	UserCount int `json:"user_count"`

	// ModelVersion is the current model version.
	ModelVersion int `json:"model_version"`

	// NextScheduledTraining is when the next training is scheduled.
	NextScheduledTraining time.Time `json:"next_scheduled_training,omitempty"`
}

// Metrics contains recommendation system metrics for observability.
type Metrics struct {
	// RequestCount is the total number of recommendation requests.
	RequestCount int64 `json:"request_count"`

	// CacheHits is the number of cache hits.
	CacheHits int64 `json:"cache_hits"`

	// CacheMisses is the number of cache misses.
	CacheMisses int64 `json:"cache_misses"`

	// AverageLatencyMS is the average recommendation latency.
	AverageLatencyMS float64 `json:"average_latency_ms"`

	// P99LatencyMS is the 99th percentile latency.
	P99LatencyMS float64 `json:"p99_latency_ms"`

	// TrainingCount is the number of training runs completed.
	TrainingCount int64 `json:"training_count"`

	// LastTrainingDurationMS is the duration of the last training.
	LastTrainingDurationMS int64 `json:"last_training_duration_ms"`

	// ErrorCount is the total number of errors.
	ErrorCount int64 `json:"error_count"`

	// AlgorithmMetrics contains per-algorithm metrics.
	AlgorithmMetrics map[string]AlgorithmMetrics `json:"algorithm_metrics"`
}

// AlgorithmMetrics contains metrics for a single algorithm.
type AlgorithmMetrics struct {
	// Name is the algorithm name.
	Name string `json:"name"`

	// PredictionCount is the number of predictions made.
	PredictionCount int64 `json:"prediction_count"`

	// AveragePredictionTimeMS is the average prediction time.
	AveragePredictionTimeMS float64 `json:"average_prediction_time_ms"`

	// CoveragePercent is the percentage of items the algorithm can score.
	CoveragePercent float64 `json:"coverage_percent"`

	// LastTrainedAt is when this algorithm was last trained.
	LastTrainedAt time.Time `json:"last_trained_at"`

	// ModelSizeBytes is the serialized model size.
	ModelSizeBytes int64 `json:"model_size_bytes"`
}

// ClassifyInteraction classifies an interaction based on completion percentage.
func ClassifyInteraction(percentComplete int) InteractionType {
	switch {
	case percentComplete >= 90:
		return InteractionCompleted
	case percentComplete >= 50:
		return InteractionEngaged
	case percentComplete >= 10:
		return InteractionSampled
	default:
		return InteractionAbandoned
	}
}

// ComputeConfidence computes the confidence score for an interaction.
// Higher confidence indicates stronger preference signal.
func ComputeConfidence(percentComplete, playDurationSeconds int) float64 {
	// Base confidence from completion
	c := 1.0 + 0.1*float64(percentComplete)

	// Boost for longer engagement (diminishing returns)
	if playDurationSeconds > 0 {
		// Log scale to prevent very long sessions from dominating
		c += 0.5 * (1.0 - 1.0/(1.0+float64(playDurationSeconds)/3600.0))
	}

	return c
}
