// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/tomtom215/cartographus/internal/models"
)

// mockPATStore is a mock implementation of PATStore for testing.
// Uses mutex for thread-safe access since PAT manager uses goroutines.
type mockPATStore struct {
	mu        sync.RWMutex
	tokens    map[string]*models.PersonalAccessToken
	usageLogs []models.PATUsageLog
	createErr error
	getErr    error
	updateErr error
	revokeErr error
	deleteErr error
}

func newMockPATStore() *mockPATStore {
	return &mockPATStore{
		tokens:    make(map[string]*models.PersonalAccessToken),
		usageLogs: make([]models.PATUsageLog, 0),
	}
}

func (m *mockPATStore) CreatePAT(ctx context.Context, token *models.PersonalAccessToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.tokens[token.ID] = token
	return nil
}

func (m *mockPATStore) GetPATByID(ctx context.Context, id string) (*models.PersonalAccessToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	token, ok := m.tokens[id]
	if !ok {
		return nil, nil
	}
	// Return a copy to avoid race conditions
	tokenCopy := *token
	return &tokenCopy, nil
}

func (m *mockPATStore) GetPATsByUserID(ctx context.Context, userID string) ([]models.PersonalAccessToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	var tokens []models.PersonalAccessToken
	for _, t := range m.tokens {
		if t.UserID == userID {
			tokens = append(tokens, *t)
		}
	}
	return tokens, nil
}

func (m *mockPATStore) UpdatePAT(ctx context.Context, token *models.PersonalAccessToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	m.tokens[token.ID] = token
	return nil
}

func (m *mockPATStore) RevokePAT(ctx context.Context, id string, revokedBy string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.revokeErr != nil {
		return m.revokeErr
	}
	token, ok := m.tokens[id]
	if !ok {
		return errors.New("PAT not found or already revoked")
	}
	now := time.Now()
	token.RevokedAt = &now
	token.RevokedBy = revokedBy
	token.RevokeReason = reason
	return nil
}

func (m *mockPATStore) DeletePAT(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.tokens, id)
	return nil
}

func (m *mockPATStore) LogPATUsage(ctx context.Context, log *models.PATUsageLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usageLogs = append(m.usageLogs, *log)
	return nil
}

func (m *mockPATStore) GetPATUsageLogs(ctx context.Context, tokenID string, limit int) ([]models.PATUsageLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	logs := make([]models.PATUsageLog, 0)
	for _, log := range m.usageLogs {
		if log.TokenID == tokenID {
			logs = append(logs, log)
		}
	}
	return logs, nil
}

func (m *mockPATStore) GetPATStats(ctx context.Context, userID string) (*models.PATStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var stats models.PATStats
	for _, t := range m.tokens {
		if t.UserID == userID {
			stats.TotalTokens++
			if t.IsActive() {
				stats.ActiveTokens++
			} else if t.IsRevoked() {
				stats.RevokedTokens++
			} else if t.IsExpired() {
				stats.ExpiredTokens++
			}
			stats.TotalUsage += t.UseCount
		}
	}
	return &stats, nil
}

func (m *mockPATStore) GetPATByPrefix(ctx context.Context, prefix string) ([]models.PersonalAccessToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var tokens []models.PersonalAccessToken
	for _, t := range m.tokens {
		if t.TokenPrefix == prefix {
			tokens = append(tokens, *t)
		}
	}
	return tokens, nil
}

// setToken is a helper for tests to set tokens with proper locking.
func (m *mockPATStore) setToken(id string, token *models.PersonalAccessToken) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[id] = token
}

func TestPATManager_Create(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)

	tests := []struct {
		name    string
		userID  string
		req     models.CreatePATRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:   "valid token creation",
			userID: "user123",
			req: models.CreatePATRequest{
				Name:   "Test Token",
				Scopes: []models.TokenScope{models.ScopeReadAnalytics},
			},
			wantErr: false,
		},
		{
			name:   "with expiration",
			userID: "user123",
			req: models.CreatePATRequest{
				Name:      "Expiring Token",
				Scopes:    []models.TokenScope{models.ScopeReadAnalytics},
				ExpiresIn: intPtr(30),
			},
			wantErr: false,
		},
		{
			name:   "with IP allowlist",
			userID: "user123",
			req: models.CreatePATRequest{
				Name:        "IP Restricted Token",
				Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
				IPAllowlist: []string{"192.168.1.1", "10.0.0.1"},
			},
			wantErr: false,
		},
		{
			name:   "multiple scopes",
			userID: "user123",
			req: models.CreatePATRequest{
				Name:   "Multi-scope Token",
				Scopes: []models.TokenScope{models.ScopeReadAnalytics, models.ScopeReadUsers, models.ScopeReadPlaybacks},
			},
			wantErr: false,
		},
		{
			name:   "invalid scope",
			userID: "user123",
			req: models.CreatePATRequest{
				Name:   "Invalid Token",
				Scopes: []models.TokenScope{"invalid:scope"},
			},
			wantErr: true,
			errMsg:  "invalid scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			token, plaintext, err := manager.Create(ctx, tt.userID, "testuser", &tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify token was created
			if token == nil {
				t.Fatal("token is nil")
			}
			if token.ID == "" {
				t.Error("token ID is empty")
			}
			if token.UserID != tt.userID {
				t.Errorf("expected userID %s, got %s", tt.userID, token.UserID)
			}
			if token.Name != tt.req.Name {
				t.Errorf("expected name %s, got %s", tt.req.Name, token.Name)
			}

			// Verify plaintext token format
			if !strings.HasPrefix(plaintext, patPrefix) {
				t.Errorf("plaintext token should start with %s, got %s", patPrefix, plaintext[:20])
			}

			// Verify expiration
			if tt.req.ExpiresIn != nil {
				if token.ExpiresAt == nil {
					t.Error("expected expiration to be set")
				} else {
					expectedExpiry := time.Now().Add(time.Duration(*tt.req.ExpiresIn) * 24 * time.Hour)
					diff := token.ExpiresAt.Sub(expectedExpiry)
					if diff < -time.Minute || diff > time.Minute {
						t.Errorf("expiration time mismatch, expected around %v, got %v", expectedExpiry, *token.ExpiresAt)
					}
				}
			}
		})
	}
}

func TestPATManager_ValidateToken(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)

	// Create a valid token
	ctx := context.Background()
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	token, plaintext, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		clientIP  string
		wantErr   bool
		errMsg    string
		setupFunc func()
	}{
		{
			name:     "valid token",
			token:    plaintext,
			clientIP: "192.168.1.1",
			wantErr:  false,
		},
		{
			name:     "invalid format",
			token:    "invalid_token",
			clientIP: "192.168.1.1",
			wantErr:  true,
			errMsg:   "invalid token format",
		},
		{
			name:     "invalid prefix",
			token:    "github_pat_xxx",
			clientIP: "192.168.1.1",
			wantErr:  true,
			errMsg:   "invalid token format",
		},
		{
			name:     "revoked token",
			token:    plaintext,
			clientIP: "192.168.1.1",
			wantErr:  true,
			errMsg:   "revoked",
			setupFunc: func() {
				now := time.Now()
				revokedToken := *token
				revokedToken.RevokedAt = &now
				store.setToken(token.ID, &revokedToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset token state with a fresh copy to avoid races
			tokenCopy := *token
			tokenCopy.RevokedAt = nil
			tokenCopy.ExpiresAt = nil
			store.setToken(token.ID, &tokenCopy)

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			validated, err := manager.ValidateToken(ctx, tt.token, tt.clientIP)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if validated == nil {
				t.Fatal("validated token is nil")
			}
		})
	}
}

func TestPATManager_List(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create tokens for two different users
	req := models.CreatePATRequest{
		Name:   "User1 Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	_, _, err := manager.Create(ctx, "user1", "testuser1", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	req.Name = "User1 Token 2"
	_, _, err = manager.Create(ctx, "user1", "testuser1", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	req.Name = "User2 Token"
	_, _, err = manager.Create(ctx, "user2", "testuser2", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// List tokens for user1
	tokens, err := manager.List(ctx, "user1")
	if err != nil {
		t.Fatalf("failed to list tokens: %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens for user1, got %d", len(tokens))
	}

	// List tokens for user2
	tokens, err = manager.List(ctx, "user2")
	if err != nil {
		t.Fatalf("failed to list tokens: %v", err)
	}

	if len(tokens) != 1 {
		t.Errorf("expected 1 token for user2, got %d", len(tokens))
	}
}

func TestPATManager_Revoke(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create a token
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	token, _, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Revoke the token
	err = manager.Revoke(ctx, token.ID, "user123", "No longer needed")
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Verify token is revoked
	storedToken := store.tokens[token.ID]
	if storedToken.RevokedAt == nil {
		t.Error("token should be revoked")
	}
	if storedToken.RevokedBy != "user123" {
		t.Errorf("expected revoked_by 'user123', got %s", storedToken.RevokedBy)
	}
	if storedToken.RevokeReason != "No longer needed" {
		t.Errorf("expected reason 'No longer needed', got %s", storedToken.RevokeReason)
	}
}

func TestPATManager_Regenerate(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create a token
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	token, originalPlaintext, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	originalHash := token.TokenHash

	// Regenerate the token
	regenerated, newPlaintext, err := manager.Regenerate(ctx, token.ID, "user123")
	if err != nil {
		t.Fatalf("failed to regenerate token: %v", err)
	}

	// Verify new plaintext is different
	if newPlaintext == originalPlaintext {
		t.Error("regenerated plaintext should be different from original")
	}

	// Verify hash changed
	if regenerated.TokenHash == originalHash {
		t.Error("token hash should have changed")
	}

	// Verify token metadata preserved
	if regenerated.Name != token.Name {
		t.Errorf("expected name %s, got %s", token.Name, regenerated.Name)
	}
	if regenerated.UserID != token.UserID {
		t.Errorf("expected userID %s, got %s", token.UserID, regenerated.UserID)
	}

	// Verify use count reset
	if regenerated.UseCount != 0 {
		t.Errorf("expected use_count 0, got %d", regenerated.UseCount)
	}
}

func TestPersonalAccessToken_Methods(t *testing.T) {
	t.Run("IsExpired", func(t *testing.T) {
		token := &models.PersonalAccessToken{}

		// No expiration
		if token.IsExpired() {
			t.Error("token without expiration should not be expired")
		}

		// Future expiration
		future := time.Now().Add(24 * time.Hour)
		token.ExpiresAt = &future
		if token.IsExpired() {
			t.Error("token with future expiration should not be expired")
		}

		// Past expiration
		past := time.Now().Add(-24 * time.Hour)
		token.ExpiresAt = &past
		if !token.IsExpired() {
			t.Error("token with past expiration should be expired")
		}
	})

	t.Run("IsRevoked", func(t *testing.T) {
		token := &models.PersonalAccessToken{}

		if token.IsRevoked() {
			t.Error("token without RevokedAt should not be revoked")
		}

		now := time.Now()
		token.RevokedAt = &now
		if !token.IsRevoked() {
			t.Error("token with RevokedAt should be revoked")
		}
	})

	t.Run("IsActive", func(t *testing.T) {
		token := &models.PersonalAccessToken{}

		// Active by default
		if !token.IsActive() {
			t.Error("new token should be active")
		}

		// Revoked = not active
		now := time.Now()
		token.RevokedAt = &now
		if token.IsActive() {
			t.Error("revoked token should not be active")
		}

		// Expired = not active
		token.RevokedAt = nil
		past := time.Now().Add(-24 * time.Hour)
		token.ExpiresAt = &past
		if token.IsActive() {
			t.Error("expired token should not be active")
		}
	})

	t.Run("HasScope", func(t *testing.T) {
		token := &models.PersonalAccessToken{
			Scopes: []models.TokenScope{models.ScopeReadAnalytics, models.ScopeReadUsers},
		}

		if !token.HasScope(models.ScopeReadAnalytics) {
			t.Error("token should have read:analytics scope")
		}
		if !token.HasScope(models.ScopeReadUsers) {
			t.Error("token should have read:users scope")
		}
		if token.HasScope(models.ScopeAdmin) {
			t.Error("token should not have admin scope")
		}

		// Admin scope grants all
		adminToken := &models.PersonalAccessToken{
			Scopes: []models.TokenScope{models.ScopeAdmin},
		}
		if !adminToken.HasScope(models.ScopeReadAnalytics) {
			t.Error("admin token should grant read:analytics")
		}
		if !adminToken.HasScope(models.ScopeWritePlaybacks) {
			t.Error("admin token should grant write:playbacks")
		}
	})

	t.Run("IsIPAllowed", func(t *testing.T) {
		// No allowlist = all allowed
		token := &models.PersonalAccessToken{}
		if !token.IsIPAllowed("192.168.1.1") {
			t.Error("no allowlist should allow all IPs")
		}

		// With allowlist
		token.IPAllowlist = []string{"192.168.1.1", "10.0.0.1"}
		if !token.IsIPAllowed("192.168.1.1") {
			t.Error("IP in allowlist should be allowed")
		}
		if token.IsIPAllowed("192.168.1.2") {
			t.Error("IP not in allowlist should not be allowed")
		}
	})
}

func TestIsPATToken(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"carto_pat_xxx_yyy", true},
		{"carto_pat_", true},
		{"github_pat_xxx", false},
		{"Bearer xxx", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			if got := IsPATToken(tt.token); got != tt.want {
				t.Errorf("IsPATToken(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"Bearer carto_pat_xxx_yyy", "carto_pat_xxx_yyy"},
		{"bearer carto_pat_xxx_yyy", "carto_pat_xxx_yyy"},
		{"BEARER carto_pat_xxx_yyy", "carto_pat_xxx_yyy"},
		{"carto_pat_xxx_yyy", "carto_pat_xxx_yyy"},
		{"Bearer invalid_token", "invalid_token"},
		{"Basic dXNlcjpwYXNz", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := ExtractTokenFromHeader(tt.header); got != tt.want {
				t.Errorf("ExtractTokenFromHeader(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestPATManager_Get(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create a token first
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	token, _, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	t.Run("get existing token", func(t *testing.T) {
		got, err := manager.Get(ctx, token.ID, "user123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected token, got nil")
		}
		if got.ID != token.ID {
			t.Errorf("expected ID %s, got %s", token.ID, got.ID)
		}
	})

	t.Run("get non-existent token", func(t *testing.T) {
		got, err := manager.Get(ctx, "nonexistent", "user123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil for non-existent token")
		}
	})

	t.Run("access denied for other user", func(t *testing.T) {
		_, err := manager.Get(ctx, token.ID, "otheruser")
		if err == nil {
			t.Fatal("expected access denied error")
		}
		if !strings.Contains(err.Error(), "access denied") {
			t.Errorf("expected access denied error, got: %v", err)
		}
	})
}

func TestPATManager_Delete(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create a token first
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	token, _, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	t.Run("delete existing token", func(t *testing.T) {
		err := manager.Delete(ctx, token.ID, "user123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify token is deleted
		got, _ := manager.Get(ctx, token.ID, "user123")
		if got != nil {
			t.Error("token should be deleted")
		}
	})

	t.Run("delete non-existent token", func(t *testing.T) {
		err := manager.Delete(ctx, "nonexistent", "user123")
		if err == nil {
			t.Fatal("expected error for non-existent token")
		}
	})

	t.Run("access denied for other user", func(t *testing.T) {
		// Create another token
		token2, _, _ := manager.Create(ctx, "user456", "testuser2", &req)

		err := manager.Delete(ctx, token2.ID, "otheruser")
		if err == nil {
			t.Fatal("expected access denied error")
		}
		if !strings.Contains(err.Error(), "access denied") {
			t.Errorf("expected access denied error, got: %v", err)
		}
	})
}

func TestPATManager_GetStats(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create some tokens
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	_, _, _ = manager.Create(ctx, "user123", "testuser", &req)
	_, _, _ = manager.Create(ctx, "user123", "testuser", &req)

	stats, err := manager.GetStats(ctx, "user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.TotalTokens != 2 {
		t.Errorf("expected 2 total tokens, got %d", stats.TotalTokens)
	}
}

func TestPATManager_GetUsageLogs(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	logs, err := manager.GetUsageLogs(ctx, "token123", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty slice, not error
	if logs == nil {
		t.Error("expected empty slice, got nil")
	}
}

func TestCheckScope(t *testing.T) {
	t.Run("has required scope", func(t *testing.T) {
		token := &models.PersonalAccessToken{
			Scopes: []models.TokenScope{models.ScopeReadAnalytics, models.ScopeReadUsers},
		}
		err := CheckScope(token, models.ScopeReadAnalytics)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing required scope", func(t *testing.T) {
		token := &models.PersonalAccessToken{
			Scopes: []models.TokenScope{models.ScopeReadAnalytics},
		}
		err := CheckScope(token, models.ScopeWritePlaybacks)
		if err == nil {
			t.Error("expected error for missing scope")
		}
		if !strings.Contains(err.Error(), "insufficient scope") {
			t.Errorf("expected insufficient scope error, got: %v", err)
		}
	})

	t.Run("admin scope grants all", func(t *testing.T) {
		token := &models.PersonalAccessToken{
			Scopes: []models.TokenScope{models.ScopeAdmin},
		}
		err := CheckScope(token, models.ScopeWritePlaybacks)
		if err != nil {
			t.Errorf("admin should have all scopes: %v", err)
		}
	})
}

func TestCheckAnyScope(t *testing.T) {
	t.Run("has one of required scopes", func(t *testing.T) {
		token := &models.PersonalAccessToken{
			Scopes: []models.TokenScope{models.ScopeReadAnalytics},
		}
		err := CheckAnyScope(token, models.ScopeReadAnalytics, models.ScopeReadUsers)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("has none of required scopes", func(t *testing.T) {
		token := &models.PersonalAccessToken{
			Scopes: []models.TokenScope{models.ScopeReadAnalytics},
		}
		err := CheckAnyScope(token, models.ScopeWritePlaybacks, models.ScopeWriteDetection)
		if err == nil {
			t.Error("expected error when missing all scopes")
		}
	})

	t.Run("empty required scopes", func(t *testing.T) {
		token := &models.PersonalAccessToken{
			Scopes: []models.TokenScope{models.ScopeReadAnalytics},
		}
		err := CheckAnyScope(token)
		if err == nil {
			t.Error("expected error for empty required scopes")
		}
	})
}

func TestPATManager_ValidateToken_ExpiredToken(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create a token with expiration
	req := models.CreatePATRequest{
		Name:      "Expiring Token",
		Scopes:    []models.TokenScope{models.ScopeReadAnalytics},
		ExpiresIn: intPtr(1), // 1 day
	}
	token, plaintext, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Manually expire the token
	past := time.Now().Add(-48 * time.Hour)
	token.ExpiresAt = &past
	store.setToken(token.ID, token)

	_, err = manager.ValidateToken(ctx, plaintext, "192.168.1.1")
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expired error, got: %v", err)
	}
}

func TestPATManager_ValidateToken_IPNotAllowed(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create a token with IP allowlist
	req := models.CreatePATRequest{
		Name:        "IP Restricted Token",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		IPAllowlist: []string{"10.0.0.1", "10.0.0.2"},
	}
	_, plaintext, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Try to validate from non-allowed IP
	_, err = manager.ValidateToken(ctx, plaintext, "192.168.1.1")
	if err == nil {
		t.Fatal("expected error for non-allowed IP")
	}
	if !strings.Contains(err.Error(), "IP address not allowed") {
		t.Errorf("expected IP not allowed error, got: %v", err)
	}

	// Validate from allowed IP should work
	_, err = manager.ValidateToken(ctx, plaintext, "10.0.0.1")
	if err != nil {
		t.Errorf("validation should succeed from allowed IP: %v", err)
	}
}

func TestPATManager_LogAPIRequest(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// This should not panic or error
	manager.LogAPIRequest(ctx, "token123", "user123", "/api/v1/test", "GET", "192.168.1.1", "TestAgent", true, "", 50)

	// Give the goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	// Verify log was recorded
	logs, _ := store.GetPATUsageLogs(ctx, "token123", 10)
	if len(logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logs))
	}
}

func TestPATManager_List_StoreError(t *testing.T) {
	store := &errorMockPATStore{
		mockPATStore: newMockPATStore(),
		listError:    fmt.Errorf("database error"),
	}
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	_, err := manager.List(ctx, "user123")
	if err == nil {
		t.Fatal("expected error from store")
	}
	if !strings.Contains(err.Error(), "failed to list tokens") {
		t.Errorf("expected 'failed to list tokens' error, got: %v", err)
	}
}

func TestPATManager_Get_StoreError(t *testing.T) {
	store := &errorMockPATStore{
		mockPATStore: newMockPATStore(),
		getError:     fmt.Errorf("database error"),
	}
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	_, err := manager.Get(ctx, "token123", "user123")
	if err == nil {
		t.Fatal("expected error from store")
	}
	if !strings.Contains(err.Error(), "failed to get token") {
		t.Errorf("expected 'failed to get token' error, got: %v", err)
	}
}

func TestPATManager_Revoke_StoreErrors(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		store := &errorMockPATStore{
			mockPATStore: newMockPATStore(),
			getError:     fmt.Errorf("database error"),
		}
		logger := zerolog.Nop()
		manager := NewPATManager(store, &logger)
		ctx := context.Background()

		err := manager.Revoke(ctx, "token123", "user123", "test")
		if err == nil {
			t.Fatal("expected error from store")
		}
		if !strings.Contains(err.Error(), "failed to get token") {
			t.Errorf("expected 'failed to get token' error, got: %v", err)
		}
	})

	t.Run("revoke error", func(t *testing.T) {
		store := &errorMockPATStore{
			mockPATStore: newMockPATStore(),
			revokeError:  fmt.Errorf("database error"),
		}
		logger := zerolog.Nop()
		manager := NewPATManager(store, &logger)
		ctx := context.Background()

		// First create a token
		req := models.CreatePATRequest{
			Name:   "Test Token",
			Scopes: []models.TokenScope{models.ScopeReadAnalytics},
		}
		token, _, err := manager.Create(ctx, "user123", "testuser", &req)
		if err != nil {
			t.Fatalf("failed to create token: %v", err)
		}

		err = manager.Revoke(ctx, token.ID, "user123", "test")
		if err == nil {
			t.Fatal("expected error from store")
		}
		if !strings.Contains(err.Error(), "failed to revoke token") {
			t.Errorf("expected 'failed to revoke token' error, got: %v", err)
		}
	})
}

func TestPATManager_Delete_StoreError(t *testing.T) {
	store := &errorMockPATStore{
		mockPATStore: newMockPATStore(),
		deleteError:  fmt.Errorf("database error"),
	}
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// First create a token
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	token, _, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	err = manager.Delete(ctx, token.ID, "user123")
	if err == nil {
		t.Fatal("expected error from store")
	}
	if !strings.Contains(err.Error(), "failed to delete token") {
		t.Errorf("expected 'failed to delete token' error, got: %v", err)
	}
}

func TestPATManager_Regenerate_StoreErrors(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		store := &errorMockPATStore{
			mockPATStore: newMockPATStore(),
			getError:     fmt.Errorf("database error"),
		}
		logger := zerolog.Nop()
		manager := NewPATManager(store, &logger)
		ctx := context.Background()

		_, _, err := manager.Regenerate(ctx, "token123", "user123")
		if err == nil {
			t.Fatal("expected error from store")
		}
		if !strings.Contains(err.Error(), "failed to get token") {
			t.Errorf("expected 'failed to get token' error, got: %v", err)
		}
	})

	t.Run("token not found", func(t *testing.T) {
		store := newMockPATStore()
		logger := zerolog.Nop()
		manager := NewPATManager(store, &logger)
		ctx := context.Background()

		_, _, err := manager.Regenerate(ctx, "nonexistent", "user123")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "token not found") {
			t.Errorf("expected 'token not found' error, got: %v", err)
		}
	})

	t.Run("access denied", func(t *testing.T) {
		store := newMockPATStore()
		logger := zerolog.Nop()
		manager := NewPATManager(store, &logger)
		ctx := context.Background()

		// Create a token for user1
		req := models.CreatePATRequest{
			Name:   "Test Token",
			Scopes: []models.TokenScope{models.ScopeReadAnalytics},
		}
		token, _, err := manager.Create(ctx, "user1", "testuser", &req)
		if err != nil {
			t.Fatalf("failed to create token: %v", err)
		}

		// Try to regenerate as user2
		_, _, err = manager.Regenerate(ctx, token.ID, "user2")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "access denied") {
			t.Errorf("expected 'access denied' error, got: %v", err)
		}
	})

	t.Run("update error", func(t *testing.T) {
		store := &errorMockPATStore{
			mockPATStore: newMockPATStore(),
			updateError:  fmt.Errorf("database error"),
		}
		logger := zerolog.Nop()
		manager := NewPATManager(store, &logger)
		ctx := context.Background()

		// First create a token
		req := models.CreatePATRequest{
			Name:   "Test Token",
			Scopes: []models.TokenScope{models.ScopeReadAnalytics},
		}
		token, _, err := manager.Create(ctx, "user123", "testuser", &req)
		if err != nil {
			t.Fatalf("failed to create token: %v", err)
		}

		_, _, err = manager.Regenerate(ctx, token.ID, "user123")
		if err == nil {
			t.Fatal("expected error from store")
		}
		if !strings.Contains(err.Error(), "failed to update token") {
			t.Errorf("expected 'failed to update token' error, got: %v", err)
		}
	})
}

func TestPATManager_Create_StoreError(t *testing.T) {
	store := &errorMockPATStore{
		mockPATStore: newMockPATStore(),
		createError:  fmt.Errorf("database error"),
	}
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	_, _, err := manager.Create(ctx, "user123", "testuser", &req)
	if err == nil {
		t.Fatal("expected error from store")
	}
	if !strings.Contains(err.Error(), "failed to store token") {
		t.Errorf("expected 'failed to store token' error, got: %v", err)
	}
}

func TestPATManager_ValidateToken_StoreError(t *testing.T) {
	store := &errorMockPATStore{
		mockPATStore: newMockPATStore(),
		getError:     fmt.Errorf("database error"),
	}
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create a valid-looking token format
	token := "carto_pat_dGVzdC1pZA_dGVzdC1zZWNyZXQ"
	_, err := manager.ValidateToken(ctx, token, "192.168.1.1")
	if err == nil {
		t.Fatal("expected error from store")
	}
	if !strings.Contains(err.Error(), "token lookup failed") {
		t.Errorf("expected 'token lookup failed' error, got: %v", err)
	}
}

func TestPATManager_ValidateToken_TokenNotFound(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Use a valid format but non-existent token ID
	token := "carto_pat_dGVzdC1pZA_dGVzdC1zZWNyZXQ"
	_, err := manager.ValidateToken(ctx, token, "192.168.1.1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "token not found") {
		t.Errorf("expected 'token not found' error, got: %v", err)
	}
}

func TestPATManager_ValidateToken_InvalidHash(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create a token but then try to validate with wrong secret
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	token, plaintext, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Tamper with the token secret
	parts := strings.Split(plaintext, "_")
	tamperedToken := strings.Join(parts[:len(parts)-1], "_") + "_tamperedsecret"

	_, err = manager.ValidateToken(ctx, tamperedToken, "192.168.1.1")
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Errorf("expected 'invalid token' error, got: %v", err)
	}

	// Let goroutines complete
	time.Sleep(50 * time.Millisecond)
	_ = token
}

func TestPATManager_ValidateToken_RevokedToken(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	// Create and then revoke a token
	req := models.CreatePATRequest{
		Name:   "Test Token",
		Scopes: []models.TokenScope{models.ScopeReadAnalytics},
	}
	token, plaintext, err := manager.Create(ctx, "user123", "testuser", &req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Revoke the token
	err = manager.Revoke(ctx, token.ID, "admin", "test revocation")
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Try to validate revoked token
	_, err = manager.ValidateToken(ctx, plaintext, "192.168.1.1")
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
	if !strings.Contains(err.Error(), "token has been revoked") {
		t.Errorf("expected 'token has been revoked' error, got: %v", err)
	}

	// Let goroutines complete
	time.Sleep(50 * time.Millisecond)
}

func TestPATManager_Revoke_TokenNotFound(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	err := manager.Revoke(ctx, "nonexistent", "user123", "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "token not found") {
		t.Errorf("expected 'token not found' error, got: %v", err)
	}
}

func TestPATManager_Delete_TokenNotFound(t *testing.T) {
	store := newMockPATStore()
	logger := zerolog.Nop()
	manager := NewPATManager(store, &logger)
	ctx := context.Background()

	err := manager.Delete(ctx, "nonexistent", "user123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "token not found") {
		t.Errorf("expected 'token not found' error, got: %v", err)
	}
}

// errorMockPATStore is a mock store that can simulate errors for specific operations
type errorMockPATStore struct {
	*mockPATStore
	createError error
	getError    error
	listError   error
	updateError error
	revokeError error
	deleteError error
}

func (e *errorMockPATStore) CreatePAT(ctx context.Context, token *models.PersonalAccessToken) error {
	if e.createError != nil {
		return e.createError
	}
	return e.mockPATStore.CreatePAT(ctx, token)
}

func (e *errorMockPATStore) GetPATByID(ctx context.Context, id string) (*models.PersonalAccessToken, error) {
	if e.getError != nil {
		return nil, e.getError
	}
	return e.mockPATStore.GetPATByID(ctx, id)
}

func (e *errorMockPATStore) GetPATsByUserID(ctx context.Context, userID string) ([]models.PersonalAccessToken, error) {
	if e.listError != nil {
		return nil, e.listError
	}
	return e.mockPATStore.GetPATsByUserID(ctx, userID)
}

func (e *errorMockPATStore) UpdatePAT(ctx context.Context, token *models.PersonalAccessToken) error {
	if e.updateError != nil {
		return e.updateError
	}
	return e.mockPATStore.UpdatePAT(ctx, token)
}

func (e *errorMockPATStore) RevokePAT(ctx context.Context, id, revokedBy, reason string) error {
	if e.revokeError != nil {
		return e.revokeError
	}
	return e.mockPATStore.RevokePAT(ctx, id, revokedBy, reason)
}

func (e *errorMockPATStore) DeletePAT(ctx context.Context, id string) error {
	if e.deleteError != nil {
		return e.deleteError
	}
	return e.mockPATStore.DeletePAT(ctx, id)
}
