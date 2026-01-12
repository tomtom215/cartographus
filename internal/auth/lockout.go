// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including account lockout.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// LockoutConfig holds configuration for the account lockout system.
type LockoutConfig struct {
	// MaxAttempts is the number of failed attempts before lockout.
	MaxAttempts int `json:"max_attempts"`

	// LockoutDuration is the base lockout period.
	LockoutDuration time.Duration `json:"lockout_duration"`

	// EnableExponentialBackoff doubles the lockout period on each subsequent lockout.
	EnableExponentialBackoff bool `json:"enable_exponential_backoff"`

	// MaxLockoutDuration caps the lockout period when using exponential backoff.
	MaxLockoutDuration time.Duration `json:"max_lockout_duration"`

	// CleanupInterval is how often to run expired lockout cleanup.
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// TrackByIP also tracks failed attempts by IP address (prevents distributed attacks).
	TrackByIP bool `json:"track_by_ip"`

	// Enabled controls whether lockout is active.
	Enabled bool `json:"enabled"`
}

// DefaultLockoutConfig returns sensible defaults.
func DefaultLockoutConfig() *LockoutConfig {
	return &LockoutConfig{
		MaxAttempts:              5,
		LockoutDuration:          15 * time.Minute,
		EnableExponentialBackoff: true,
		MaxLockoutDuration:       24 * time.Hour,
		CleanupInterval:          5 * time.Minute,
		TrackByIP:                true,
		Enabled:                  true,
	}
}

// LockoutEntry tracks failed login attempts for a subject (username or IP).
type LockoutEntry struct {
	Subject         string    `json:"subject"`
	FailedAttempts  int       `json:"failed_attempts"`
	LastAttempt     time.Time `json:"last_attempt"`
	LockoutCount    int       `json:"lockout_count"` // Number of times locked out (for exponential backoff)
	LockedUntil     time.Time `json:"locked_until"`
	LastFailedIP    string    `json:"last_failed_ip,omitempty"`
	LastFailedAgent string    `json:"last_failed_agent,omitempty"`
}

// IsLocked returns true if the entry is currently locked out.
func (e *LockoutEntry) IsLocked() bool {
	return time.Now().Before(e.LockedUntil)
}

// LockoutStore defines the interface for lockout state persistence.
type LockoutStore interface {
	// GetEntry retrieves a lockout entry by subject (username or IP).
	GetEntry(ctx context.Context, subject string) (*LockoutEntry, error)

	// SaveEntry persists a lockout entry.
	SaveEntry(ctx context.Context, entry *LockoutEntry) error

	// DeleteEntry removes a lockout entry.
	DeleteEntry(ctx context.Context, subject string) error

	// ListLockedEntries returns all currently locked entries.
	ListLockedEntries(ctx context.Context) ([]*LockoutEntry, error)

	// CleanupExpired removes expired entries.
	CleanupExpired(ctx context.Context) (int, error)
}

// LockoutManager handles account lockout logic.
type LockoutManager struct {
	config *LockoutConfig
	store  LockoutStore
	mu     sync.RWMutex

	// Callbacks for integration with audit logging
	onLockout      func(entry *LockoutEntry)
	onFailedLogin  func(subject, ip, userAgent string)
	onLockoutClear func(subject string)
}

// NewLockoutManager creates a new lockout manager.
func NewLockoutManager(store LockoutStore, config *LockoutConfig) *LockoutManager {
	if config == nil {
		config = DefaultLockoutConfig()
	}

	return &LockoutManager{
		config: config,
		store:  store,
	}
}

// SetOnLockout sets a callback for when an account is locked.
func (m *LockoutManager) SetOnLockout(fn func(entry *LockoutEntry)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onLockout = fn
}

// SetOnFailedLogin sets a callback for failed login attempts.
func (m *LockoutManager) SetOnFailedLogin(fn func(subject, ip, userAgent string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onFailedLogin = fn
}

// SetOnLockoutClear sets a callback for when a lockout is cleared.
func (m *LockoutManager) SetOnLockoutClear(fn func(subject string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onLockoutClear = fn
}

// CheckLocked returns true if the subject is currently locked out.
// Also returns the time remaining if locked.
func (m *LockoutManager) CheckLocked(ctx context.Context, subject string) (bool, time.Duration, error) {
	m.mu.RLock()
	enabled := m.config.Enabled
	m.mu.RUnlock()

	if !enabled {
		return false, 0, nil
	}

	entry, err := m.store.GetEntry(ctx, subject)
	if err != nil {
		if errors.Is(err, ErrLockoutNotFound) {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("check lockout: %w", err)
	}

	if !entry.IsLocked() {
		return false, 0, nil
	}

	remaining := time.Until(entry.LockedUntil)
	return true, remaining, nil
}

// RecordFailedAttempt records a failed login attempt and returns whether the account is now locked.
func (m *LockoutManager) RecordFailedAttempt(ctx context.Context, username, ip, userAgent string) (locked bool, remaining time.Duration, err error) {
	m.mu.RLock()
	config := *m.config
	onFailedLogin := m.onFailedLogin
	onLockout := m.onLockout
	m.mu.RUnlock()

	if !config.Enabled {
		return false, 0, nil
	}

	// Fire callback
	if onFailedLogin != nil {
		go onFailedLogin(username, ip, userAgent)
	}

	// Track by username
	locked, remaining, err = m.recordAttemptForSubject(ctx, username, ip, userAgent, &config, onLockout)
	if err != nil || locked {
		return locked, remaining, err
	}

	// Optionally track by IP as well
	if !config.TrackByIP || ip == "" {
		return false, 0, nil
	}

	return m.recordAttemptForSubject(ctx, "ip:"+ip, ip, userAgent, &config, onLockout)
}

// calculateLockoutDuration computes the lockout duration with optional exponential backoff.
func calculateLockoutDuration(config *LockoutConfig, lockoutCount int) time.Duration {
	duration := config.LockoutDuration

	if !config.EnableExponentialBackoff || lockoutCount == 0 {
		return duration
	}

	// Double duration for each previous lockout
	multiplier := 1 << lockoutCount // 2^lockoutCount
	duration = time.Duration(int64(duration) * int64(multiplier))

	if duration > config.MaxLockoutDuration {
		return config.MaxLockoutDuration
	}

	return duration
}

// updateFailedAttempt increments the failed attempt counter and updates metadata.
func updateFailedAttempt(entry *LockoutEntry, ip, userAgent string, now time.Time) {
	entry.FailedAttempts++
	entry.LastAttempt = now
	entry.LastFailedIP = ip
	entry.LastFailedAgent = userAgent
}

// applyLockout applies lockout to an entry and returns the lockout duration.
func applyLockout(entry *LockoutEntry, config *LockoutConfig, now time.Time) time.Duration {
	lockoutDuration := calculateLockoutDuration(config, entry.LockoutCount)

	entry.LockedUntil = now.Add(lockoutDuration)
	entry.LockoutCount++
	entry.FailedAttempts = 0 // Reset for next cycle

	logging.Warn().
		Str("subject", entry.Subject).
		Dur("duration", lockoutDuration).
		Int("lockout_count", entry.LockoutCount).
		Msg("Account locked")

	return lockoutDuration
}

// recordAttemptForSubject records a failed attempt for a specific subject.
func (m *LockoutManager) recordAttemptForSubject(
	ctx context.Context,
	subject, ip, userAgent string,
	config *LockoutConfig,
	onLockout func(*LockoutEntry),
) (locked bool, remaining time.Duration, err error) {
	entry, err := m.getOrCreateEntry(ctx, subject)
	if err != nil {
		return false, 0, fmt.Errorf("get entry: %w", err)
	}

	// If already locked, just return the remaining time
	if entry.IsLocked() {
		return true, time.Until(entry.LockedUntil), nil
	}

	now := time.Now()
	updateFailedAttempt(entry, ip, userAgent, now)

	// Check if we should lock
	if entry.FailedAttempts < config.MaxAttempts {
		// Save updated entry and continue
		if err := m.store.SaveEntry(ctx, entry); err != nil {
			return false, 0, fmt.Errorf("save entry: %w", err)
		}
		return false, 0, nil
	}

	// Apply lockout
	lockoutDuration := applyLockout(entry, config, now)

	// Fire callback
	if onLockout != nil {
		go onLockout(entry)
	}

	if err := m.store.SaveEntry(ctx, entry); err != nil {
		return false, 0, fmt.Errorf("save locked entry: %w", err)
	}

	return true, lockoutDuration, nil
}

// getOrCreateEntry retrieves an existing entry or creates a new one.
func (m *LockoutManager) getOrCreateEntry(ctx context.Context, subject string) (*LockoutEntry, error) {
	entry, err := m.store.GetEntry(ctx, subject)
	if err != nil && !errors.Is(err, ErrLockoutNotFound) {
		return nil, err
	}

	if entry == nil {
		entry = &LockoutEntry{
			Subject: subject,
		}
	}

	return entry, nil
}

// RecordSuccessfulLogin clears the lockout state for a subject.
func (m *LockoutManager) RecordSuccessfulLogin(ctx context.Context, username string) error {
	m.mu.RLock()
	enabled := m.config.Enabled
	onClear := m.onLockoutClear
	m.mu.RUnlock()

	if !enabled {
		return nil
	}

	if err := m.store.DeleteEntry(ctx, username); err != nil && !errors.Is(err, ErrLockoutNotFound) {
		return fmt.Errorf("clear lockout: %w", err)
	}

	if onClear != nil {
		go onClear(username)
	}

	return nil
}

// ClearLockout manually clears a lockout (admin action).
func (m *LockoutManager) ClearLockout(ctx context.Context, subject string) error {
	m.mu.RLock()
	onClear := m.onLockoutClear
	m.mu.RUnlock()

	if err := m.store.DeleteEntry(ctx, subject); err != nil && !errors.Is(err, ErrLockoutNotFound) {
		return fmt.Errorf("clear lockout: %w", err)
	}

	logging.Info().Str("subject", subject).Msg("Manually cleared lockout")

	if onClear != nil {
		go onClear(subject)
	}

	return nil
}

// filterLockedEntries returns only entries that are currently locked.
func filterLockedEntries(entries []*LockoutEntry) []*LockoutEntry {
	var locked []*LockoutEntry
	for _, entry := range entries {
		if entry.IsLocked() {
			locked = append(locked, entry)
		}
	}
	return locked
}

// GetLockedAccounts returns all currently locked accounts.
func (m *LockoutManager) GetLockedAccounts(ctx context.Context) ([]*LockoutEntry, error) {
	entries, err := m.store.ListLockedEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list locked: %w", err)
	}

	return filterLockedEntries(entries), nil
}

// performCleanup executes a cleanup operation and logs the result.
func (m *LockoutManager) performCleanup(ctx context.Context) {
	count, err := m.store.CleanupExpired(ctx)
	if err != nil {
		logging.Error().Err(err).Msg("Lockout cleanup error")
		return
	}

	if count > 0 {
		logging.Info().Int("count", count).Msg("Cleaned up expired lockout entries")
	}
}

// runCleanupLoop runs the cleanup loop until context is canceled.
func (m *LockoutManager) runCleanupLoop(ctx context.Context, ticker *time.Ticker) {
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performCleanup(ctx)
		}
	}
}

// StartCleanupRoutine starts a background routine to clean up expired entries.
func (m *LockoutManager) StartCleanupRoutine(ctx context.Context) {
	m.mu.RLock()
	interval := m.config.CleanupInterval
	m.mu.RUnlock()

	ticker := time.NewTicker(interval)
	go m.runCleanupLoop(ctx, ticker)
}

// Config returns the current configuration.
func (m *LockoutManager) Config() LockoutConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.config
}

// SetEnabled enables or disables the lockout system.
func (m *LockoutManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Enabled = enabled
}

// ErrLockoutNotFound is returned when a lockout entry doesn't exist.
var ErrLockoutNotFound = errors.New("lockout entry not found")

// ErrAccountLocked is returned when authentication is blocked due to lockout.
var ErrAccountLocked = errors.New("account temporarily locked due to too many failed attempts")

// MemoryLockoutStore implements LockoutStore using in-memory storage.
// Suitable for development and single-instance deployments.
type MemoryLockoutStore struct {
	entries map[string]*LockoutEntry
	mu      sync.RWMutex
}

// NewMemoryLockoutStore creates a new in-memory lockout store.
func NewMemoryLockoutStore() *MemoryLockoutStore {
	return &MemoryLockoutStore{
		entries: make(map[string]*LockoutEntry),
	}
}

// GetEntry retrieves a lockout entry.
func (s *MemoryLockoutStore) GetEntry(ctx context.Context, subject string) (*LockoutEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[subject]
	if !ok {
		return nil, ErrLockoutNotFound
	}

	return copyEntry(entry), nil
}

// SaveEntry persists a lockout entry.
func (s *MemoryLockoutStore) SaveEntry(ctx context.Context, entry *LockoutEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[entry.Subject] = copyEntry(entry)
	return nil
}

// DeleteEntry removes a lockout entry.
func (s *MemoryLockoutStore) DeleteEntry(ctx context.Context, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[subject]; !ok {
		return ErrLockoutNotFound
	}

	delete(s.entries, subject)
	return nil
}

// copyEntry creates a copy of a lockout entry.
func copyEntry(entry *LockoutEntry) *LockoutEntry {
	copied := *entry
	return &copied
}

// ListLockedEntries returns all entries that are currently locked.
func (s *MemoryLockoutStore) ListLockedEntries(ctx context.Context) ([]*LockoutEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var locked []*LockoutEntry
	now := time.Now()

	for _, entry := range s.entries {
		if now.Before(entry.LockedUntil) {
			locked = append(locked, copyEntry(entry))
		}
	}

	return locked, nil
}

// shouldCleanupEntry determines if an entry is eligible for cleanup.
func shouldCleanupEntry(entry *LockoutEntry, expireThreshold time.Time) bool {
	return !entry.IsLocked() && entry.LastAttempt.Before(expireThreshold)
}

// CleanupExpired removes expired entries.
func (s *MemoryLockoutStore) CleanupExpired(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	expireThreshold := now.Add(-24 * time.Hour) // Keep entries for 24h after unlock

	count := 0
	for subject, entry := range s.entries {
		if shouldCleanupEntry(entry, expireThreshold) {
			delete(s.entries, subject)
			count++
		}
	}

	return count, nil
}

// writeLockoutResponse writes a standardized lockout response to the client.
func writeLockoutResponse(w http.ResponseWriter, remaining time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", fmt.Sprintf("%d", int(remaining.Seconds())))
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"error":            "Account temporarily locked",
		"retry_after_secs": int(remaining.Seconds()),
		"message":          fmt.Sprintf("Too many failed attempts. Try again in %v", remaining.Round(time.Second)),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error().Err(err).Msg("Error encoding lockout response")
	}
}

// LockoutMiddleware creates middleware that checks for account lockout before authentication.
func LockoutMiddleware(manager *LockoutManager, getSubject func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subject := getSubject(r)
			if subject == "" {
				next.ServeHTTP(w, r)
				return
			}

			locked, remaining, err := manager.CheckLocked(r.Context(), subject)
			if err != nil {
				logging.Error().Err(err).Msg("Error checking lockout")
				next.ServeHTTP(w, r)
				return
			}

			if locked {
				writeLockoutResponse(w, remaining)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
