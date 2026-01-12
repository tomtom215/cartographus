// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including multi-mode support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"
)

// MultiAuthenticator tries multiple authenticators in priority order.
// It implements a chain of responsibility pattern where each authenticator
// is tried until one succeeds or returns a fatal error.
//
// Error handling:
//   - ErrNoCredentials: Try next authenticator (no credentials for this method)
//   - ErrAuthenticatorUnavailable: Try next authenticator (service unavailable)
//   - ErrInvalidCredentials: Stop and return error (credentials were provided but invalid)
//   - ErrExpiredCredentials: Stop and return error (credentials expired)
//   - Other errors: Stop and return error
type MultiAuthenticator struct {
	mu             sync.RWMutex
	authenticators []Authenticator
}

// NewMultiAuthenticator creates a new multi-authenticator with the given authenticators.
// Authenticators are sorted by priority (lower priority number = higher priority).
func NewMultiAuthenticator(authenticators ...Authenticator) *MultiAuthenticator {
	m := &MultiAuthenticator{
		authenticators: make([]Authenticator, 0, len(authenticators)),
	}

	m.authenticators = append(m.authenticators, authenticators...)

	// Sort by priority
	m.sortByPriority()

	return m
}

// AddAuthenticator adds an authenticator to the chain.
func (m *MultiAuthenticator) AddAuthenticator(auth Authenticator) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.authenticators = append(m.authenticators, auth)
	m.sortByPriority()
}

// Authenticators returns the list of authenticators in priority order.
func (m *MultiAuthenticator) Authenticators() []Authenticator {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Authenticator, len(m.authenticators))
	copy(result, m.authenticators)
	return result
}

// Authenticate tries each authenticator in priority order.
func (m *MultiAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthSubject, error) {
	m.mu.RLock()
	authenticators := make([]Authenticator, len(m.authenticators))
	copy(authenticators, m.authenticators)
	m.mu.RUnlock()

	if len(authenticators) == 0 {
		return nil, ErrNoCredentials
	}

	lastErr := ErrNoCredentials

	for _, auth := range authenticators {
		subject, err := auth.Authenticate(ctx, r)
		if err == nil {
			return subject, nil
		}

		lastErr = err

		// Check if we should continue to next authenticator
		if shouldTryNext(err) {
			continue
		}

		// Fatal error - stop trying
		return nil, err
	}

	return nil, lastErr
}

// Name returns the authenticator name.
func (m *MultiAuthenticator) Name() string {
	return string(AuthModeMulti)
}

// Priority returns the authenticator priority.
// Multi-auth always has highest priority (0) since it wraps other authenticators.
func (m *MultiAuthenticator) Priority() int {
	return 0
}

// shouldTryNext returns true if the error indicates we should try the next authenticator.
func shouldTryNext(err error) bool {
	// No credentials provided by this authenticator - try next
	if errors.Is(err, ErrNoCredentials) {
		return true
	}

	// Authenticator unavailable (network error, etc.) - try next
	if errors.Is(err, ErrAuthenticatorUnavailable) {
		return true
	}

	// Invalid or expired credentials are fatal - don't try next
	return false
}

// sortByPriority sorts authenticators by priority (lower number = higher priority).
// This method assumes the caller holds the write lock.
func (m *MultiAuthenticator) sortByPriority() {
	sort.Slice(m.authenticators, func(i, j int) bool {
		return m.authenticators[i].Priority() < m.authenticators[j].Priority()
	})
}
