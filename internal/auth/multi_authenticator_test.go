// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMultiAuthenticator_Interface verifies the Authenticator interface
func TestMultiAuthenticator_Interface(t *testing.T) {
	multi := NewMultiAuthenticator()

	// Verify interface implementation
	var _ Authenticator = multi

	// Test Name()
	if multi.Name() != "multi" {
		t.Errorf("Name() = %v, want multi", multi.Name())
	}

	// Test Priority()
	if multi.Priority() != 0 {
		t.Errorf("Priority() = %v, want 0", multi.Priority())
	}
}

// TestMultiAuthenticator_NoAuthenticators tests empty authenticator list
func TestMultiAuthenticator_NoAuthenticators(t *testing.T) {
	multi := NewMultiAuthenticator()

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	_, err := multi.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrNoCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrNoCredentials", err)
	}
}

// TestMultiAuthenticator_SingleAuthenticator tests with one authenticator
func TestMultiAuthenticator_SingleAuthenticator(t *testing.T) {
	mock := &mockAuthenticator{
		name:     "mock",
		priority: 10,
		returnSubj: &AuthSubject{
			ID:       "user-123",
			Username: "testuser",
		},
	}

	multi := NewMultiAuthenticator(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	subject, err := multi.Authenticate(context.Background(), req)

	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if subject.ID != "user-123" {
		t.Errorf("subject.ID = %v, want user-123", subject.ID)
	}
}

// TestMultiAuthenticator_PriorityOrder tests authenticators are tried in priority order
func TestMultiAuthenticator_PriorityOrder(t *testing.T) {
	// Create authenticators with different priorities
	// Higher priority (lower number) should be tried first
	highPriority := &mockAuthenticator{
		name:       "high",
		priority:   10,
		shouldFail: true,
		returnErr:  ErrNoCredentials, // Should try next
	}
	medPriority := &mockAuthenticator{
		name:     "med",
		priority: 20,
		returnSubj: &AuthSubject{
			ID:       "med-user",
			Username: "meduser",
		},
	}
	lowPriority := &mockAuthenticator{
		name:     "low",
		priority: 30,
		returnSubj: &AuthSubject{
			ID:       "low-user",
			Username: "lowuser",
		},
	}

	// Add in wrong order to verify sorting
	multi := NewMultiAuthenticator(lowPriority, highPriority, medPriority)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	subject, err := multi.Authenticate(context.Background(), req)

	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should get med-user because high priority returned ErrNoCredentials
	if subject.ID != "med-user" {
		t.Errorf("subject.ID = %v, want med-user", subject.ID)
	}

	// Verify high priority was tried first
	if highPriority.callCount != 1 {
		t.Errorf("highPriority.callCount = %d, want 1", highPriority.callCount)
	}

	// Verify med priority was tried second
	if medPriority.callCount != 1 {
		t.Errorf("medPriority.callCount = %d, want 1", medPriority.callCount)
	}

	// Verify low priority was NOT tried (med succeeded)
	if lowPriority.callCount != 0 {
		t.Errorf("lowPriority.callCount = %d, want 0", lowPriority.callCount)
	}
}

// TestMultiAuthenticator_StopsOnInvalidCredentials tests that invalid credentials stops the chain
func TestMultiAuthenticator_StopsOnInvalidCredentials(t *testing.T) {
	first := &mockAuthenticator{
		name:       "first",
		priority:   10,
		shouldFail: true,
		returnErr:  ErrInvalidCredentials, // Should stop, not try next
	}
	second := &mockAuthenticator{
		name:     "second",
		priority: 20,
		returnSubj: &AuthSubject{
			ID: "second-user",
		},
	}

	multi := NewMultiAuthenticator(first, second)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	_, err := multi.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrInvalidCredentials", err)
	}

	// Verify second was NOT tried
	if second.callCount != 0 {
		t.Errorf("second.callCount = %d, want 0", second.callCount)
	}
}

// TestMultiAuthenticator_StopsOnExpiredCredentials tests that expired credentials stops the chain
func TestMultiAuthenticator_StopsOnExpiredCredentials(t *testing.T) {
	first := &mockAuthenticator{
		name:       "first",
		priority:   10,
		shouldFail: true,
		returnErr:  ErrExpiredCredentials, // Should stop, not try next
	}
	second := &mockAuthenticator{
		name:     "second",
		priority: 20,
		returnSubj: &AuthSubject{
			ID: "second-user",
		},
	}

	multi := NewMultiAuthenticator(first, second)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	_, err := multi.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrExpiredCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrExpiredCredentials", err)
	}

	// Verify second was NOT tried
	if second.callCount != 0 {
		t.Errorf("second.callCount = %d, want 0", second.callCount)
	}
}

// TestMultiAuthenticator_ContinuesOnNoCredentials tests fallback on no credentials
func TestMultiAuthenticator_ContinuesOnNoCredentials(t *testing.T) {
	first := &mockAuthenticator{
		name:       "first",
		priority:   10,
		shouldFail: true,
		returnErr:  ErrNoCredentials, // Should try next
	}
	second := &mockAuthenticator{
		name:       "second",
		priority:   20,
		shouldFail: true,
		returnErr:  ErrNoCredentials, // Should try next
	}
	third := &mockAuthenticator{
		name:     "third",
		priority: 30,
		returnSubj: &AuthSubject{
			ID: "third-user",
		},
	}

	multi := NewMultiAuthenticator(first, second, third)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	subject, err := multi.Authenticate(context.Background(), req)

	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if subject.ID != "third-user" {
		t.Errorf("subject.ID = %v, want third-user", subject.ID)
	}

	// Verify all three were tried
	if first.callCount != 1 {
		t.Errorf("first.callCount = %d, want 1", first.callCount)
	}
	if second.callCount != 1 {
		t.Errorf("second.callCount = %d, want 1", second.callCount)
	}
	if third.callCount != 1 {
		t.Errorf("third.callCount = %d, want 1", third.callCount)
	}
}

// TestMultiAuthenticator_ContinuesOnUnavailable tests fallback on unavailable authenticator
func TestMultiAuthenticator_ContinuesOnUnavailable(t *testing.T) {
	first := &mockAuthenticator{
		name:       "first",
		priority:   10,
		shouldFail: true,
		returnErr:  ErrAuthenticatorUnavailable, // Should try next
	}
	second := &mockAuthenticator{
		name:     "second",
		priority: 20,
		returnSubj: &AuthSubject{
			ID: "second-user",
		},
	}

	multi := NewMultiAuthenticator(first, second)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	subject, err := multi.Authenticate(context.Background(), req)

	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if subject.ID != "second-user" {
		t.Errorf("subject.ID = %v, want second-user", subject.ID)
	}
}

// TestMultiAuthenticator_AllFail tests when all authenticators fail
func TestMultiAuthenticator_AllFail(t *testing.T) {
	first := &mockAuthenticator{
		name:       "first",
		priority:   10,
		shouldFail: true,
		returnErr:  ErrNoCredentials,
	}
	second := &mockAuthenticator{
		name:       "second",
		priority:   20,
		shouldFail: true,
		returnErr:  ErrNoCredentials,
	}

	multi := NewMultiAuthenticator(first, second)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	_, err := multi.Authenticate(context.Background(), req)

	// Should return the last error (ErrNoCredentials)
	if !errors.Is(err, ErrNoCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrNoCredentials", err)
	}
}

// TestMultiAuthenticator_AddAuthenticator tests adding authenticators
func TestMultiAuthenticator_AddAuthenticator(t *testing.T) {
	multi := NewMultiAuthenticator()

	mock := &mockAuthenticator{
		name:     "mock",
		priority: 10,
		returnSubj: &AuthSubject{
			ID: "user-123",
		},
	}

	multi.AddAuthenticator(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	subject, err := multi.Authenticate(context.Background(), req)

	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if subject.ID != "user-123" {
		t.Errorf("subject.ID = %v, want user-123", subject.ID)
	}
}

// TestMultiAuthenticator_AuthenticatorsList tests getting list of authenticators
func TestMultiAuthenticator_AuthenticatorsList(t *testing.T) {
	mock1 := &mockAuthenticator{name: "mock1", priority: 10}
	mock2 := &mockAuthenticator{name: "mock2", priority: 20}

	multi := NewMultiAuthenticator(mock1, mock2)

	list := multi.Authenticators()
	if len(list) != 2 {
		t.Errorf("len(Authenticators()) = %d, want 2", len(list))
	}
}
