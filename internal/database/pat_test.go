// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/models"
)

func TestDB_CreatePAT(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		token   *models.PersonalAccessToken
		wantErr bool
	}{
		{
			name: "valid token",
			token: &models.PersonalAccessToken{
				ID:          uuid.New().String(),
				UserID:      "user123",
				Username:    "testuser",
				Name:        "Test Token",
				Description: "A test token",
				TokenPrefix: "carto_pat_abc12345",
				TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
				Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
				CreatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "token with expiration",
			token: &models.PersonalAccessToken{
				ID:          uuid.New().String(),
				UserID:      "user123",
				Username:    "testuser",
				Name:        "Expiring Token",
				TokenPrefix: "carto_pat_def12345",
				TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
				Scopes:      []models.TokenScope{models.ScopeReadAnalytics, models.ScopeReadUsers},
				ExpiresAt:   timePtr(time.Now().Add(30 * 24 * time.Hour)),
				CreatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "token with IP allowlist",
			token: &models.PersonalAccessToken{
				ID:          uuid.New().String(),
				UserID:      "user456",
				Username:    "testuser2",
				Name:        "IP Restricted Token",
				TokenPrefix: "carto_pat_ghi12345",
				TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
				Scopes:      []models.TokenScope{models.ScopeAdmin},
				IPAllowlist: []string{"192.168.1.1", "10.0.0.1"},
				CreatedAt:   time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.CreatePAT(ctx, tt.token)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify token was stored
			stored, err := db.GetPATByID(ctx, tt.token.ID)
			if err != nil {
				t.Fatalf("failed to retrieve token: %v", err)
			}
			if stored == nil {
				t.Fatal("stored token is nil")
			}
			if stored.Name != tt.token.Name {
				t.Errorf("name mismatch: got %s, want %s", stored.Name, tt.token.Name)
			}
			if stored.UserID != tt.token.UserID {
				t.Errorf("user_id mismatch: got %s, want %s", stored.UserID, tt.token.UserID)
			}
			if len(stored.Scopes) != len(tt.token.Scopes) {
				t.Errorf("scopes count mismatch: got %d, want %d", len(stored.Scopes), len(tt.token.Scopes))
			}
		})
	}
}

func TestDB_GetPATByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a token first
	token := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Test Token",
		TokenPrefix: "carto_pat_test1234",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "existing token",
			id:      token.ID,
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "non-existent token",
			id:      uuid.New().String(),
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "empty id",
			id:      "",
			wantNil: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.GetPATByID(ctx, tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got token with ID %s", result.ID)
				}
				return
			}

			if result == nil {
				t.Fatal("expected token, got nil")
			}
			if result.ID != tt.id {
				t.Errorf("id mismatch: got %s, want %s", result.ID, tt.id)
			}
		})
	}
}

func TestDB_GetPATsByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create tokens for two users
	user1Tokens := []*models.PersonalAccessToken{
		{
			ID:          uuid.New().String(),
			UserID:      "user1",
			Username:    "user1",
			Name:        "User1 Token 1",
			TokenPrefix: "carto_pat_u1t11234",
			TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
			Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New().String(),
			UserID:      "user1",
			Username:    "user1",
			Name:        "User1 Token 2",
			TokenPrefix: "carto_pat_u1t21234",
			TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
			Scopes:      []models.TokenScope{models.ScopeReadUsers},
			CreatedAt:   time.Now(),
		},
	}

	user2Token := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user2",
		Username:    "user2",
		Name:        "User2 Token",
		TokenPrefix: "carto_pat_u2t11234",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeAdmin},
		CreatedAt:   time.Now(),
	}

	for _, token := range user1Tokens {
		if err := db.CreatePAT(ctx, token); err != nil {
			t.Fatalf("failed to create user1 token: %v", err)
		}
	}
	if err := db.CreatePAT(ctx, user2Token); err != nil {
		t.Fatalf("failed to create user2 token: %v", err)
	}

	tests := []struct {
		name      string
		userID    string
		wantCount int
	}{
		{
			name:      "user1 with 2 tokens",
			userID:    "user1",
			wantCount: 2,
		},
		{
			name:      "user2 with 1 token",
			userID:    "user2",
			wantCount: 1,
		},
		{
			name:      "user3 with no tokens",
			userID:    "user3",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := db.GetPATsByUserID(ctx, tt.userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != tt.wantCount {
				t.Errorf("count mismatch: got %d, want %d", len(tokens), tt.wantCount)
			}

			// Verify all returned tokens belong to the user
			for _, token := range tokens {
				if token.UserID != tt.userID {
					t.Errorf("token %s belongs to %s, not %s", token.ID, token.UserID, tt.userID)
				}
			}
		})
	}
}

func TestDB_UpdatePAT(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a token
	token := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Original Name",
		Description: "Original Description",
		TokenPrefix: "carto_pat_updt1234",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Update the token
	token.Name = "Updated Name"
	token.Description = "Updated Description"
	token.UseCount = 5
	now := time.Now()
	token.LastUsedAt = &now
	token.LastUsedIP = "192.168.1.100"

	err := db.UpdatePAT(ctx, token)
	if err != nil {
		t.Fatalf("failed to update token: %v", err)
	}

	// Verify updates
	updated, err := db.GetPATByID(ctx, token.ID)
	if err != nil {
		t.Fatalf("failed to get updated token: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("name not updated: got %s, want %s", updated.Name, "Updated Name")
	}
	if updated.Description != "Updated Description" {
		t.Errorf("description not updated: got %s, want %s", updated.Description, "Updated Description")
	}
	if updated.UseCount != 5 {
		t.Errorf("use_count not updated: got %d, want %d", updated.UseCount, 5)
	}
	if updated.LastUsedIP != "192.168.1.100" {
		t.Errorf("last_used_ip not updated: got %s, want %s", updated.LastUsedIP, "192.168.1.100")
	}
}

func TestDB_RevokePAT(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a token
	token := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "To Be Revoked",
		TokenPrefix: "carto_pat_revk1234",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Revoke the token
	err := db.RevokePAT(ctx, token.ID, "admin", "Security concern")
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Verify revocation
	revoked, err := db.GetPATByID(ctx, token.ID)
	if err != nil {
		t.Fatalf("failed to get revoked token: %v", err)
	}

	if revoked.RevokedAt == nil {
		t.Error("revoked_at should not be nil")
	}
	if revoked.RevokedBy != "admin" {
		t.Errorf("revoked_by mismatch: got %s, want %s", revoked.RevokedBy, "admin")
	}
	if revoked.RevokeReason != "Security concern" {
		t.Errorf("revoke_reason mismatch: got %s, want %s", revoked.RevokeReason, "Security concern")
	}

	// Try to revoke again - should fail
	err = db.RevokePAT(ctx, token.ID, "admin2", "Double revoke")
	if err == nil {
		t.Error("revoking already revoked token should fail")
	}
}

func TestDB_DeletePAT(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a token
	token := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "To Be Deleted",
		TokenPrefix: "carto_pat_delt1234",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Delete the token
	err := db.DeletePAT(ctx, token.ID)
	if err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	// Verify deletion
	deleted, err := db.GetPATByID(ctx, token.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != nil {
		t.Error("token should be deleted")
	}

	// Try to delete again - should fail
	err = db.DeletePAT(ctx, token.ID)
	if err == nil {
		t.Error("deleting non-existent token should fail")
	}
}

func TestDB_LogPATUsage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a token first
	token := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Test Token",
		TokenPrefix: "carto_pat_logg1234",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Log usage
	logEntry := &models.PATUsageLog{
		ID:             uuid.New().String(),
		Timestamp:      time.Now(),
		TokenID:        token.ID,
		UserID:         token.UserID,
		Action:         "authenticate",
		Endpoint:       "/api/v1/analytics",
		Method:         "GET",
		IPAddress:      "192.168.1.1",
		UserAgent:      "curl/7.68.0",
		Success:        true,
		ResponseTimeMS: 42,
	}

	err := db.LogPATUsage(ctx, logEntry)
	if err != nil {
		t.Fatalf("failed to log PAT usage: %v", err)
	}

	// Retrieve logs
	logs, err := db.GetPATUsageLogs(ctx, token.ID, 10)
	if err != nil {
		t.Fatalf("failed to get PAT usage logs: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logs))
	}

	if len(logs) > 0 {
		if logs[0].Action != "authenticate" {
			t.Errorf("action mismatch: got %s, want %s", logs[0].Action, "authenticate")
		}
		if logs[0].Endpoint != "/api/v1/analytics" {
			t.Errorf("endpoint mismatch: got %s, want %s", logs[0].Endpoint, "/api/v1/analytics")
		}
		if !logs[0].Success {
			t.Error("success should be true")
		}
	}
}

func TestDB_GetPATStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create tokens with different states
	activeToken := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Active Token",
		TokenPrefix: "carto_pat_stat1234",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		UseCount:    10,
		CreatedAt:   time.Now(),
	}

	expiredToken := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Expired Token",
		TokenPrefix: "carto_pat_stat2345",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		ExpiresAt:   timePtr(time.Now().Add(-24 * time.Hour)), // Expired
		UseCount:    5,
		CreatedAt:   time.Now(),
	}

	for _, token := range []*models.PersonalAccessToken{activeToken, expiredToken} {
		if err := db.CreatePAT(ctx, token); err != nil {
			t.Fatalf("failed to create token: %v", err)
		}
	}

	// Revoke one more token
	revokedToken := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Revoked Token",
		TokenPrefix: "carto_pat_stat3456",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		UseCount:    3,
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, revokedToken); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if err := db.RevokePAT(ctx, revokedToken.ID, "admin", "test"); err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Get stats
	stats, err := db.GetPATStats(ctx, "user123")
	if err != nil {
		t.Fatalf("failed to get PAT stats: %v", err)
	}

	if stats.TotalTokens != 3 {
		t.Errorf("total_tokens mismatch: got %d, want %d", stats.TotalTokens, 3)
	}
	if stats.ActiveTokens != 1 {
		t.Errorf("active_tokens mismatch: got %d, want %d", stats.ActiveTokens, 1)
	}
	if stats.ExpiredTokens != 1 {
		t.Errorf("expired_tokens mismatch: got %d, want %d", stats.ExpiredTokens, 1)
	}
	if stats.RevokedTokens != 1 {
		t.Errorf("revoked_tokens mismatch: got %d, want %d", stats.RevokedTokens, 1)
	}
	if stats.TotalUsage != 18 { // 10 + 5 + 3
		t.Errorf("total_usage mismatch: got %d, want %d", stats.TotalUsage, 18)
	}
}

func TestDB_GetPATByPrefix(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create tokens with same prefix (unlikely in production but tests the query)
	token := &models.PersonalAccessToken{
		ID:          uuid.New().String(),
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Test Token",
		TokenPrefix: "carto_pat_pfx12345",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Search by prefix
	tokens, err := db.GetPATByPrefix(ctx, "carto_pat_pfx12345")
	if err != nil {
		t.Fatalf("failed to get tokens by prefix: %v", err)
	}

	if len(tokens) != 1 {
		t.Errorf("expected 1 token, got %d", len(tokens))
	}

	// Search for non-existent prefix
	tokens, err = db.GetPATByPrefix(ctx, "carto_pat_nonexist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}

// Note: timePtr is defined in database_test.go
