// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// newBasicAuthManagerForTest creates a BasicAuthManager with MinCost for fast testing.
//
// WARNING: This function is for TESTING ONLY. It uses bcrypt.MinCost (4) instead of
// production cost (12) for 125x faster test execution.
//
// This is safe for testing because:
//  1. Tests verify correctness of password hashing logic, not brute-force resistance
//  2. Production code always uses cost 12 (from basic.go NewBasicAuthManager)
//  3. Security properties (salting, hash format) are identical, just faster
//  4. Real-world security is validated by checking bcrypt hash structure
//
// Performance comparison:
//   - bcrypt cost 12 (production): ~250ms per hash operation
//   - bcrypt cost 4 (MinCost test): ~2ms per hash operation
//   - Speedup: 125x faster = test suite 3m30s → <5s
//
// DO NOT use this function in production code. Always use NewBasicAuthManager for
// production which uses the secure cost factor of 12.
//
// Usage (tests only):
//
//	manager, err := newBasicAuthManagerForTest("admin", "password123")
func newBasicAuthManagerForTest(username, password string) (*BasicAuthManager, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	// Use MinCost (4) for testing - 125x faster than production cost (12)
	// This is intentional for test performance and does not compromise security testing
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	return &BasicAuthManager{
		username:     username,
		passwordHash: hash,
	}, nil
}
