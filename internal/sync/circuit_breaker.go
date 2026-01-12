// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	gobreaker "github.com/sony/gobreaker/v2"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/metrics"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// CircuitBreakerClient wraps TautulliClient with circuit breaker pattern
// Circuit breaker pattern prevents cascading failures when Tautulli API is unavailable or slow
//
// DETERMINISM NOTE: The circuit breaker uses real time (via sony/gobreaker) for its
// interval and timeout calculations. This is intentional for production resilience:
// - The timing determines when to recover from failures, not data integrity
// - Tests should use appropriate waits or mock the underlying client, not the breaker
// - For unit tests, consider testing the wrapped client directly
type CircuitBreakerClient struct {
	client *TautulliClient
	cb     *gobreaker.CircuitBreaker[interface{}]
	name   string
}

// NewCircuitBreakerClient creates a new Tautulli client with circuit breaker
// Circuit breaker configuration:
// - Max 3 concurrent requests in half-open state
// - 1 minute measurement window
// - 2 minute timeout before attempting recovery
// - Opens after 60% failure rate with minimum 10 requests
func NewCircuitBreakerClient(cfg *config.TautulliConfig) *CircuitBreakerClient {
	client := NewTautulliClient(cfg)
	cbName := "tautulli-api"

	// Initialize circuit breaker state metrics
	metrics.CircuitBreakerState.WithLabelValues(cbName).Set(0) // 0 = closed
	metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbName).Set(0)

	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        cbName,
		MaxRequests: 3,               // Allow 3 concurrent requests in half-open state
		Interval:    time.Minute,     // Reset counts after 1 minute in closed state
		Timeout:     2 * time.Minute, // Wait 2 minutes before transitioning from open to half-open

		// ReadyToTrip determines when to open the circuit
		// Opens when failure rate >= 60% with minimum 10 requests
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 10 {
				return false // Need at least 10 requests for statistical significance
			}

			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			shouldTrip := failureRatio >= 0.6

			if shouldTrip {
				logging.Warn().Uint32("failures", counts.TotalFailures).Float64("failure_rate", failureRatio*100).Msg("[CIRCUIT BREAKER] Opening circuit")
			}

			return shouldTrip
		},

		// OnStateChange is called whenever the circuit breaker changes state
		OnStateChange: func(name string, from, to gobreaker.State) {
			fromStr := stateToString(from)
			toStr := stateToString(to)

			logging.Info().Str("from", fromStr).Str("to", toStr).Msg("[CIRCUIT BREAKER] State transition")

			// Update metrics
			metrics.CircuitBreakerState.WithLabelValues(name).Set(stateToFloat(to))
			metrics.CircuitBreakerTransitions.WithLabelValues(name, fromStr, toStr).Inc()

			// Reset consecutive failures when transitioning to closed
			if to == gobreaker.StateClosed {
				metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(name).Set(0)
			}
		},
	})

	return &CircuitBreakerClient{
		client: client,
		cb:     cb,
		name:   cbName,
	}
}

// execute wraps a Tautulli API call with circuit breaker protection
// Returns the result or an error if circuit is open or request fails
func (cbc *CircuitBreakerClient) execute(fn func() (interface{}, error)) (interface{}, error) {
	result, err := cbc.cb.Execute(func() (interface{}, error) {
		return fn()
	})

	// Update metrics based on result
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			// Circuit is open or too many concurrent requests in half-open state
			metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "rejected").Inc()
			logging.Warn().Err(err).Msg("[CIRCUIT BREAKER] Request rejected")
		} else {
			// Request failed
			metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "failure").Inc()

			// Increment consecutive failures
			counts := cbc.cb.Counts()
			metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbc.name).Set(float64(counts.ConsecutiveFailures))
		}
		return nil, err
	}

	// Request succeeded
	metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "success").Inc()
	metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbc.name).Set(0)

	return result, nil
}

// castResult safely type-casts the circuit breaker result with error checking
// Returns typed result or error if type assertion fails
func castResult[T any](result interface{}, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	typed, ok := result.(*T)
	if !ok {
		return nil, fmt.Errorf("circuit breaker: unexpected result type %T", result)
	}
	return typed, nil
}

// stateToFloat converts circuit breaker state to numeric value for metrics
func stateToFloat(state gobreaker.State) float64 {
	switch state {
	case gobreaker.StateClosed:
		return 0
	case gobreaker.StateHalfOpen:
		return 1
	case gobreaker.StateOpen:
		return 2
	default:
		return -1
	}
}

// stateToString converts circuit breaker state to string for logging
func stateToString(state gobreaker.State) string {
	switch state {
	case gobreaker.StateClosed:
		return "closed"
	case gobreaker.StateHalfOpen:
		return "half-open"
	case gobreaker.StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// Ping verifies connectivity to Tautulli API with circuit breaker protection
func (cbc *CircuitBreakerClient) Ping(ctx context.Context) error {
	_, err := cbc.execute(func() (interface{}, error) {
		return nil, cbc.client.Ping(ctx)
	})
	return err
}

// GetHistorySince retrieves playback history with circuit breaker protection
func (cbc *CircuitBreakerClient) GetHistorySince(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
	return castResult[tautulli.TautulliHistory](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetHistorySince(ctx, since, start, length)
	}))
}

// GetGeoIPLookup retrieves geolocation data with circuit breaker protection
func (cbc *CircuitBreakerClient) GetGeoIPLookup(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
	return castResult[tautulli.TautulliGeoIP](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetGeoIPLookup(ctx, ipAddress)
	}))
}

// GetHomeStats retrieves home statistics with circuit breaker protection
func (cbc *CircuitBreakerClient) GetHomeStats(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error) {
	return castResult[tautulli.TautulliHomeStats](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetHomeStats(ctx, timeRange, statsType, statsCount)
	}))
}

// GetPlaysByDate retrieves plays by date with circuit breaker protection
func (cbc *CircuitBreakerClient) GetPlaysByDate(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDate, error) {
	return castResult[tautulli.TautulliPlaysByDate](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysByDate(ctx, timeRange, yAxis, userID, grouping)
	}))
}

// GetPlaysByDayOfWeek retrieves plays by day of week with circuit breaker protection
func (cbc *CircuitBreakerClient) GetPlaysByDayOfWeek(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDayOfWeek, error) {
	return castResult[tautulli.TautulliPlaysByDayOfWeek](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysByDayOfWeek(ctx, timeRange, yAxis, userID, grouping)
	}))
}

// GetPlaysByHourOfDay retrieves plays by hour of day with circuit breaker protection
func (cbc *CircuitBreakerClient) GetPlaysByHourOfDay(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByHourOfDay, error) {
	return castResult[tautulli.TautulliPlaysByHourOfDay](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysByHourOfDay(ctx, timeRange, yAxis, userID, grouping)
	}))
}

// GetPlaysByStreamType retrieves plays by stream type with circuit breaker protection
func (cbc *CircuitBreakerClient) GetPlaysByStreamType(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamType, error) {
	return castResult[tautulli.TautulliPlaysByStreamType](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysByStreamType(ctx, timeRange, yAxis, userID, grouping)
	}))
}

// GetConcurrentStreamsByStreamType retrieves concurrent streams with circuit breaker protection
func (cbc *CircuitBreakerClient) GetConcurrentStreamsByStreamType(ctx context.Context, timeRange int, userID int) (*tautulli.TautulliConcurrentStreamsByStreamType, error) {
	return castResult[tautulli.TautulliConcurrentStreamsByStreamType](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetConcurrentStreamsByStreamType(ctx, timeRange, userID)
	}))
}

// GetItemWatchTimeStats retrieves item watch time statistics with circuit breaker protection
func (cbc *CircuitBreakerClient) GetItemWatchTimeStats(ctx context.Context, ratingKey string, grouping int, queryDays string) (*tautulli.TautulliItemWatchTimeStats, error) {
	return castResult[tautulli.TautulliItemWatchTimeStats](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetItemWatchTimeStats(ctx, ratingKey, grouping, queryDays)
	}))
}

// GetActivity retrieves current activity with circuit breaker protection
func (cbc *CircuitBreakerClient) GetActivity(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error) {
	return castResult[tautulli.TautulliActivity](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetActivity(ctx, sessionKey)
	}))
}

// GetMetadata retrieves metadata for a rating key with circuit breaker protection
func (cbc *CircuitBreakerClient) GetMetadata(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error) {
	return castResult[tautulli.TautulliMetadata](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetMetadata(ctx, ratingKey)
	}))
}

// GetUser retrieves user information with circuit breaker protection
func (cbc *CircuitBreakerClient) GetUser(ctx context.Context, userID int) (*tautulli.TautulliUser, error) {
	return castResult[tautulli.TautulliUser](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetUser(ctx, userID)
	}))
}

// GetUsers retrieves all users with circuit breaker protection
func (cbc *CircuitBreakerClient) GetUsers(ctx context.Context) (*tautulli.TautulliUsers, error) {
	return castResult[tautulli.TautulliUsers](cbc.execute(func() (interface{}, error) { return cbc.client.GetUsers(ctx) }))
}

// GetLibraryUserStats retrieves library user statistics with circuit breaker protection
func (cbc *CircuitBreakerClient) GetLibraryUserStats(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error) {
	return castResult[tautulli.TautulliLibraryUserStats](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetLibraryUserStats(ctx, sectionID, grouping)
	}))
}

// GetRecentlyAdded retrieves recently added content with circuit breaker protection
func (cbc *CircuitBreakerClient) GetRecentlyAdded(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error) {
	return castResult[tautulli.TautulliRecentlyAdded](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetRecentlyAdded(ctx, count, start, mediaType, sectionID)
	}))
}

// GetLibraries retrieves all libraries with circuit breaker protection
func (cbc *CircuitBreakerClient) GetLibraries(ctx context.Context) (*tautulli.TautulliLibraries, error) {
	return castResult[tautulli.TautulliLibraries](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetLibraries(ctx)
	}))
}

// GetLibrary retrieves a specific library with circuit breaker protection
func (cbc *CircuitBreakerClient) GetLibrary(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error) {
	return castResult[tautulli.TautulliLibrary](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetLibrary(ctx, sectionID)
	}))
}

// GetServerInfo retrieves server information with circuit breaker protection
func (cbc *CircuitBreakerClient) GetServerInfo(ctx context.Context) (*tautulli.TautulliServerInfo, error) {
	return castResult[tautulli.TautulliServerInfo](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetServerInfo(ctx)
	}))
}

// GetSyncedItems retrieves synced items with circuit breaker protection
func (cbc *CircuitBreakerClient) GetSyncedItems(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error) {
	return castResult[tautulli.TautulliSyncedItems](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetSyncedItems(ctx, machineID, userID)
	}))
}

// TerminateSession terminates a session with circuit breaker protection
func (cbc *CircuitBreakerClient) TerminateSession(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error) {
	return castResult[tautulli.TautulliTerminateSession](cbc.execute(func() (interface{}, error) {
		return cbc.client.TerminateSession(ctx, sessionID, message)
	}))
}

// NOTE: All remaining interface methods would follow the same pattern
// For brevity, I've shown representative examples. In production, all 60+ methods
// would be implemented following this exact pattern with circuit breaker protection.

// Placeholder implementations for remaining interface methods
// (In production, implement all 60+ methods following the pattern above)

func (cbc *CircuitBreakerClient) GetPlaysBySourceResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error) {
	return castResult[tautulli.TautulliPlaysBySourceResolution](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysBySourceResolution(ctx, timeRange, yAxis, userID, grouping)
	}))
}

func (cbc *CircuitBreakerClient) GetPlaysByStreamResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error) {
	return castResult[tautulli.TautulliPlaysByStreamResolution](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysByStreamResolution(ctx, timeRange, yAxis, userID, grouping)
	}))
}

func (cbc *CircuitBreakerClient) GetPlaysByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error) {
	return castResult[tautulli.TautulliPlaysByTop10Platforms](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysByTop10Platforms(ctx, timeRange, yAxis, userID, grouping)
	}))
}

func (cbc *CircuitBreakerClient) GetPlaysByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error) {
	return castResult[tautulli.TautulliPlaysByTop10Users](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysByTop10Users(ctx, timeRange, yAxis, userID, grouping)
	}))
}

func (cbc *CircuitBreakerClient) GetPlaysPerMonth(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error) {
	return castResult[tautulli.TautulliPlaysPerMonth](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaysPerMonth(ctx, timeRange, yAxis, userID, grouping)
	}))
}

func (cbc *CircuitBreakerClient) GetUserPlayerStats(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error) {
	return castResult[tautulli.TautulliUserPlayerStats](cbc.execute(func() (interface{}, error) { return cbc.client.GetUserPlayerStats(ctx, userID) }))
}

func (cbc *CircuitBreakerClient) GetUserWatchTimeStats(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error) {
	return castResult[tautulli.TautulliUserWatchTimeStats](cbc.execute(func() (interface{}, error) { return cbc.client.GetUserWatchTimeStats(ctx, userID, queryDays) }))
}

func (cbc *CircuitBreakerClient) GetItemUserStats(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error) {
	return castResult[tautulli.TautulliItemUserStats](cbc.execute(func() (interface{}, error) { return cbc.client.GetItemUserStats(ctx, ratingKey, grouping) }))
}

func (cbc *CircuitBreakerClient) GetLibrariesTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error) {
	return castResult[tautulli.TautulliLibrariesTable](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetLibrariesTable(ctx, grouping, orderColumn, orderDir, start, length, search)
	}))
}

func (cbc *CircuitBreakerClient) GetLibraryMediaInfo(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error) {
	return castResult[tautulli.TautulliLibraryMediaInfo](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetLibraryMediaInfo(ctx, sectionID, orderColumn, orderDir, start, length)
	}))
}

func (cbc *CircuitBreakerClient) GetLibraryWatchTimeStats(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error) {
	return castResult[tautulli.TautulliLibraryWatchTimeStats](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetLibraryWatchTimeStats(ctx, sectionID, grouping, queryDays)
	}))
}

func (cbc *CircuitBreakerClient) GetChildrenMetadata(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error) {
	return castResult[tautulli.TautulliChildrenMetadata](cbc.execute(func() (interface{}, error) { return cbc.client.GetChildrenMetadata(ctx, ratingKey, mediaType) }))
}

func (cbc *CircuitBreakerClient) GetUserIPs(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error) {
	return castResult[tautulli.TautulliUserIPs](cbc.execute(func() (interface{}, error) { return cbc.client.GetUserIPs(ctx, userID) }))
}

func (cbc *CircuitBreakerClient) GetUsersTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error) {
	return castResult[tautulli.TautulliUsersTable](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetUsersTable(ctx, grouping, orderColumn, orderDir, start, length, search)
	}))
}

func (cbc *CircuitBreakerClient) GetUserLogins(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error) {
	return castResult[tautulli.TautulliUserLogins](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetUserLogins(ctx, userID, orderColumn, orderDir, start, length, search)
	}))
}

func (cbc *CircuitBreakerClient) GetStreamData(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error) {
	return castResult[tautulli.TautulliStreamData](cbc.execute(func() (interface{}, error) { return cbc.client.GetStreamData(ctx, rowID, sessionKey) }))
}

func (cbc *CircuitBreakerClient) GetLibraryNames(ctx context.Context) (*tautulli.TautulliLibraryNames, error) {
	return castResult[tautulli.TautulliLibraryNames](cbc.execute(func() (interface{}, error) { return cbc.client.GetLibraryNames(ctx) }))
}

func (cbc *CircuitBreakerClient) ExportMetadata(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error) {
	return castResult[tautulli.TautulliExportMetadata](cbc.execute(func() (interface{}, error) {
		return cbc.client.ExportMetadata(ctx, sectionID, exportType, userID, ratingKey, fileFormat)
	}))
}

func (cbc *CircuitBreakerClient) GetExportFields(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error) {
	return castResult[tautulli.TautulliExportFields](cbc.execute(func() (interface{}, error) { return cbc.client.GetExportFields(ctx, mediaType) }))
}

func (cbc *CircuitBreakerClient) GetStreamTypeByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Users, error) {
	return castResult[tautulli.TautulliStreamTypeByTop10Users](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetStreamTypeByTop10Users(ctx, timeRange, yAxis, userID, grouping)
	}))
}

func (cbc *CircuitBreakerClient) GetStreamTypeByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Platforms, error) {
	return castResult[tautulli.TautulliStreamTypeByTop10Platforms](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetStreamTypeByTop10Platforms(ctx, timeRange, yAxis, userID, grouping)
	}))
}

func (cbc *CircuitBreakerClient) Search(ctx context.Context, query string, limit int) (*tautulli.TautulliSearch, error) {
	return castResult[tautulli.TautulliSearch](cbc.execute(func() (interface{}, error) { return cbc.client.Search(ctx, query, limit) }))
}

func (cbc *CircuitBreakerClient) GetNewRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliNewRatingKeys, error) {
	return castResult[tautulli.TautulliNewRatingKeys](cbc.execute(func() (interface{}, error) { return cbc.client.GetNewRatingKeys(ctx, ratingKey) }))
}

func (cbc *CircuitBreakerClient) GetOldRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliOldRatingKeys, error) {
	return castResult[tautulli.TautulliOldRatingKeys](cbc.execute(func() (interface{}, error) { return cbc.client.GetOldRatingKeys(ctx, ratingKey) }))
}

func (cbc *CircuitBreakerClient) GetCollectionsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliCollectionsTable, error) {
	return castResult[tautulli.TautulliCollectionsTable](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetCollectionsTable(ctx, sectionID, orderColumn, orderDir, start, length, search)
	}))
}

func (cbc *CircuitBreakerClient) GetPlaylistsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliPlaylistsTable, error) {
	return castResult[tautulli.TautulliPlaylistsTable](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetPlaylistsTable(ctx, sectionID, orderColumn, orderDir, start, length, search)
	}))
}

func (cbc *CircuitBreakerClient) GetServerFriendlyName(ctx context.Context) (*tautulli.TautulliServerFriendlyName, error) {
	return castResult[tautulli.TautulliServerFriendlyName](cbc.execute(func() (interface{}, error) { return cbc.client.GetServerFriendlyName(ctx) }))
}

func (cbc *CircuitBreakerClient) GetServerID(ctx context.Context) (*tautulli.TautulliServerID, error) {
	return castResult[tautulli.TautulliServerID](cbc.execute(func() (interface{}, error) { return cbc.client.GetServerID(ctx) }))
}

func (cbc *CircuitBreakerClient) GetServerIdentity(ctx context.Context) (*tautulli.TautulliServerIdentity, error) {
	return castResult[tautulli.TautulliServerIdentity](cbc.execute(func() (interface{}, error) { return cbc.client.GetServerIdentity(ctx) }))
}

func (cbc *CircuitBreakerClient) GetExportsTable(ctx context.Context, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliExportsTable, error) {
	return castResult[tautulli.TautulliExportsTable](cbc.execute(func() (interface{}, error) {
		return cbc.client.GetExportsTable(ctx, orderColumn, orderDir, start, length, search)
	}))
}

func (cbc *CircuitBreakerClient) DownloadExport(ctx context.Context, exportID int) (*tautulli.TautulliDownloadExport, error) {
	return castResult[tautulli.TautulliDownloadExport](cbc.execute(func() (interface{}, error) { return cbc.client.DownloadExport(ctx, exportID) }))
}

func (cbc *CircuitBreakerClient) DeleteExport(ctx context.Context, exportID int) (*tautulli.TautulliDeleteExport, error) {
	return castResult[tautulli.TautulliDeleteExport](cbc.execute(func() (interface{}, error) { return cbc.client.DeleteExport(ctx, exportID) }))
}

func (cbc *CircuitBreakerClient) GetTautulliInfo(ctx context.Context) (*tautulli.TautulliTautulliInfo, error) {
	return castResult[tautulli.TautulliTautulliInfo](cbc.execute(func() (interface{}, error) { return cbc.client.GetTautulliInfo(ctx) }))
}

func (cbc *CircuitBreakerClient) GetServerPref(ctx context.Context, pref string) (*tautulli.TautulliServerPref, error) {
	return castResult[tautulli.TautulliServerPref](cbc.execute(func() (interface{}, error) { return cbc.client.GetServerPref(ctx, pref) }))
}

func (cbc *CircuitBreakerClient) GetServerList(ctx context.Context) (*tautulli.TautulliServerList, error) {
	return castResult[tautulli.TautulliServerList](cbc.execute(func() (interface{}, error) { return cbc.client.GetServerList(ctx) }))
}

func (cbc *CircuitBreakerClient) GetServersInfo(ctx context.Context) (*tautulli.TautulliServersInfo, error) {
	return castResult[tautulli.TautulliServersInfo](cbc.execute(func() (interface{}, error) { return cbc.client.GetServersInfo(ctx) }))
}

func (cbc *CircuitBreakerClient) GetPMSUpdate(ctx context.Context) (*tautulli.TautulliPMSUpdate, error) {
	return castResult[tautulli.TautulliPMSUpdate](cbc.execute(func() (interface{}, error) { return cbc.client.GetPMSUpdate(ctx) }))
}
