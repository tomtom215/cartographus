// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication and authorization functionality.
// This file implements Personal Access Token (PAT) management for programmatic API access.
//
// PAT Format: carto_pat_<base64-encoded-id>_<random-secret>
//
// Security:
//   - Tokens are hashed with bcrypt (cost 12) before storage
//   - Only the prefix (first 8 chars) is stored for identification
//   - Tokens support scoped permissions, expiration, and IP allowlisting
//   - Usage is logged for audit and security monitoring
//
// Example Usage:
//
//	manager := auth.NewPATManager(db, logger)
//	token, plaintext, err := manager.Create(ctx, userID, username, CreatePATRequest{...})
//	// Store plaintext securely - it's only shown once!
//
//	// Later, validate a token from request
//	token, err := manager.ValidateToken(ctx, "carto_pat_xxx_yyy")
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/tomtom215/cartographus/internal/models"
)

const (
	// patPrefix is the prefix for all Cartographus PATs.
	patPrefix = "carto_pat_"

	// patSecretLength is the length of the random secret portion (bytes).
	patSecretLength = 32

	// patPrefixDisplayLength is the length of the token prefix shown in UI.
	patPrefixDisplayLength = 8

	// bcryptCost is the bcrypt cost factor for token hashing.
	bcryptCost = 12
)

// PATStore defines the database operations required for PAT management.
// This interface allows the PAT manager to be tested independently of the database.
type PATStore interface {
	// Token CRUD
	CreatePAT(ctx context.Context, token *models.PersonalAccessToken) error
	GetPATByID(ctx context.Context, id string) (*models.PersonalAccessToken, error)
	GetPATsByUserID(ctx context.Context, userID string) ([]models.PersonalAccessToken, error)
	UpdatePAT(ctx context.Context, token *models.PersonalAccessToken) error
	RevokePAT(ctx context.Context, id string, revokedBy string, reason string) error
	DeletePAT(ctx context.Context, id string) error

	// Usage logging
	LogPATUsage(ctx context.Context, log *models.PATUsageLog) error
	GetPATUsageLogs(ctx context.Context, tokenID string, limit int) ([]models.PATUsageLog, error)

	// Stats
	GetPATStats(ctx context.Context, userID string) (*models.PATStats, error)

	// Token lookup by prefix (for validation optimization)
	GetPATByPrefix(ctx context.Context, prefix string) ([]models.PersonalAccessToken, error)
}

// PATManager handles Personal Access Token operations.
type PATManager struct {
	store  PATStore
	logger zerolog.Logger
}

// NewPATManager creates a new PAT manager.
func NewPATManager(store PATStore, logger *zerolog.Logger) *PATManager {
	return &PATManager{
		store:  store,
		logger: logger.With().Str("component", "pat_manager").Logger(),
	}
}

// Create generates a new Personal Access Token for a user.
// Returns the token record and the plaintext token (shown only once).
func (m *PATManager) Create(ctx context.Context, userID, username string, req *models.CreatePATRequest) (*models.PersonalAccessToken, string, error) {
	// Validate scopes
	for _, scope := range req.Scopes {
		if !models.IsValidScope(scope) {
			return nil, "", fmt.Errorf("invalid scope: %s", scope)
		}
	}

	// Generate token ID
	tokenID := uuid.New().String()

	// Generate random secret
	secretBytes := make([]byte, patSecretLength)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate token secret: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)

	// Construct plaintext token
	idEncoded := base64.RawURLEncoding.EncodeToString([]byte(tokenID))
	plaintextToken := fmt.Sprintf("%s%s_%s", patPrefix, idEncoded, secret)

	// Extract prefix for display/identification
	tokenPrefixStr := plaintextToken[:patPrefixDisplayLength+len(patPrefix)]

	// Hash the token for storage
	// Since bcrypt has a 72-byte limit, we first SHA-256 the token to get a fixed-length hash.
	// This is a common pattern used by GitHub and other services.
	tokenHash, err := hashToken(plaintextToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash token: %w", err)
	}

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
		expiresAt = &exp
	}

	// Create token record
	token := &models.PersonalAccessToken{
		ID:          tokenID,
		UserID:      userID,
		Username:    username,
		Name:        req.Name,
		Description: req.Description,
		TokenPrefix: tokenPrefixStr,
		TokenHash:   string(tokenHash),
		Scopes:      req.Scopes,
		ExpiresAt:   expiresAt,
		IPAllowlist: req.IPAllowlist,
		CreatedAt:   time.Now(),
		UseCount:    0,
	}

	// Store token
	if err := m.store.CreatePAT(ctx, token); err != nil {
		return nil, "", fmt.Errorf("failed to store token: %w", err)
	}

	m.logger.Info().
		Str("token_id", tokenID).
		Str("user_id", userID).
		Str("name", req.Name).
		Int("scopes_count", len(req.Scopes)).
		Msg("PAT created")

	return token, plaintextToken, nil
}

// ValidateToken validates a plaintext token and returns the token record if valid.
// This method:
//   - Checks token format
//   - Verifies the hash
//   - Checks expiration
//   - Checks revocation status
//   - Logs usage (on success or failure)
func (m *PATManager) ValidateToken(ctx context.Context, plaintextToken string, clientIP string) (*models.PersonalAccessToken, error) {
	// Validate format
	if !strings.HasPrefix(plaintextToken, patPrefix) {
		return nil, fmt.Errorf("invalid token format")
	}

	// Extract token ID from the token
	tokenParts := strings.TrimPrefix(plaintextToken, patPrefix)
	parts := strings.SplitN(tokenParts, "_", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	idEncoded := parts[0]
	idBytes, err := base64.RawURLEncoding.DecodeString(idEncoded)
	if err != nil {
		return nil, fmt.Errorf("invalid token format")
	}
	tokenID := string(idBytes)

	// Retrieve token from database
	token, err := m.store.GetPATByID(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("token lookup failed: %w", err)
	}
	if token == nil {
		m.logUsage(ctx, tokenID, "", "authenticate", "", "", clientIP, "", false, "TOKEN_NOT_FOUND", 0)
		return nil, fmt.Errorf("token not found")
	}

	// Verify hash
	if !verifyToken(plaintextToken, token.TokenHash) {
		m.logUsage(ctx, tokenID, token.UserID, "authenticate", "", "", clientIP, "", false, "INVALID_TOKEN", 0)
		return nil, fmt.Errorf("invalid token")
	}

	// Check revocation
	if token.IsRevoked() {
		m.logUsage(ctx, tokenID, token.UserID, "authenticate", "", "", clientIP, "", false, "TOKEN_REVOKED", 0)
		return nil, fmt.Errorf("token has been revoked")
	}

	// Check expiration
	if token.IsExpired() {
		m.logUsage(ctx, tokenID, token.UserID, "authenticate", "", "", clientIP, "", false, "TOKEN_EXPIRED", 0)
		return nil, fmt.Errorf("token has expired")
	}

	// Check IP allowlist
	if !token.IsIPAllowed(clientIP) {
		m.logUsage(ctx, tokenID, token.UserID, "authenticate", "", "", clientIP, "", false, "IP_NOT_ALLOWED", 0)
		return nil, fmt.Errorf("IP address not allowed for this token")
	}

	// Update last used
	now := time.Now()
	token.LastUsedAt = &now
	token.LastUsedIP = clientIP
	token.UseCount++

	// Update in database (fire and forget for performance)
	// Make a copy to avoid race conditions with the returned token
	tokenCopy := *token
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.store.UpdatePAT(updateCtx, &tokenCopy); err != nil {
			m.logger.Warn().Err(err).Str("token_id", tokenID).Msg("Failed to update token last used")
		}
	}()

	// Log successful authentication
	m.logUsage(ctx, tokenID, token.UserID, "authenticate", "", "", clientIP, "", true, "", 0)

	return token, nil
}

// List returns all PATs for a user.
func (m *PATManager) List(ctx context.Context, userID string) ([]models.PersonalAccessToken, error) {
	tokens, err := m.store.GetPATsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	return tokens, nil
}

// Get retrieves a specific PAT by ID.
func (m *PATManager) Get(ctx context.Context, tokenID string, userID string) (*models.PersonalAccessToken, error) {
	token, err := m.store.GetPATByID(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return nil, nil
	}

	// Verify ownership (unless admin check is needed elsewhere)
	if token.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return token, nil
}

// Revoke revokes a PAT.
func (m *PATManager) Revoke(ctx context.Context, tokenID string, revokedBy string, reason string) error {
	token, err := m.store.GetPATByID(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return fmt.Errorf("token not found")
	}

	if err := m.store.RevokePAT(ctx, tokenID, revokedBy, reason); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	m.logger.Info().
		Str("token_id", tokenID).
		Str("revoked_by", revokedBy).
		Str("reason", reason).
		Msg("PAT revoked")

	// Log revocation
	m.logUsage(ctx, tokenID, token.UserID, "revoke", "", "", "", "", true, "", 0)

	return nil
}

// Delete permanently deletes a PAT.
func (m *PATManager) Delete(ctx context.Context, tokenID string, userID string) error {
	token, err := m.store.GetPATByID(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return fmt.Errorf("token not found")
	}

	// Verify ownership
	if token.UserID != userID {
		return fmt.Errorf("access denied")
	}

	if err := m.store.DeletePAT(ctx, tokenID); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	m.logger.Info().
		Str("token_id", tokenID).
		Str("user_id", userID).
		Msg("PAT deleted")

	return nil
}

// Regenerate creates a new secret for an existing PAT.
// Returns the new plaintext token (shown only once).
func (m *PATManager) Regenerate(ctx context.Context, tokenID string, userID string) (*models.PersonalAccessToken, string, error) {
	token, err := m.store.GetPATByID(ctx, tokenID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return nil, "", fmt.Errorf("token not found")
	}

	// Verify ownership
	if token.UserID != userID {
		return nil, "", fmt.Errorf("access denied")
	}

	// Generate new secret
	secretBytes := make([]byte, patSecretLength)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate token secret: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)

	// Construct new plaintext token
	idEncoded := base64.RawURLEncoding.EncodeToString([]byte(tokenID))
	plaintextToken := fmt.Sprintf("%s%s_%s", patPrefix, idEncoded, secret)

	// Hash the new token
	tokenHash, err := hashToken(plaintextToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash token: %w", err)
	}

	// Update token
	token.TokenHash = string(tokenHash)
	token.TokenPrefix = plaintextToken[:patPrefixDisplayLength+len(patPrefix)]
	token.UseCount = 0
	token.LastUsedAt = nil
	token.LastUsedIP = ""

	if err := m.store.UpdatePAT(ctx, token); err != nil {
		return nil, "", fmt.Errorf("failed to update token: %w", err)
	}

	m.logger.Info().
		Str("token_id", tokenID).
		Str("user_id", userID).
		Msg("PAT regenerated")

	return token, plaintextToken, nil
}

// GetStats returns aggregated PAT statistics for a user.
func (m *PATManager) GetStats(ctx context.Context, userID string) (*models.PATStats, error) {
	return m.store.GetPATStats(ctx, userID)
}

// GetUsageLogs returns usage logs for a token.
func (m *PATManager) GetUsageLogs(ctx context.Context, tokenID string, limit int) ([]models.PATUsageLog, error) {
	return m.store.GetPATUsageLogs(ctx, tokenID, limit)
}

// logUsage logs a PAT usage event.
func (m *PATManager) logUsage(_ context.Context, tokenID, userID, action, endpoint, method, ip, userAgent string, success bool, errorCode string, responseTimeMS int) {
	log := &models.PATUsageLog{
		ID:             uuid.New().String(),
		Timestamp:      time.Now(),
		TokenID:        tokenID,
		UserID:         userID,
		Action:         action,
		Endpoint:       endpoint,
		Method:         method,
		IPAddress:      ip,
		UserAgent:      userAgent,
		Success:        success,
		ErrorCode:      errorCode,
		ResponseTimeMS: responseTimeMS,
	}

	// Fire and forget for performance
	go func() {
		logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.store.LogPATUsage(logCtx, log); err != nil {
			m.logger.Warn().Err(err).Str("token_id", tokenID).Msg("Failed to log PAT usage")
		}
	}()
}

// LogAPIRequest logs an API request made with a PAT.
// This should be called by the API middleware after request processing.
func (m *PATManager) LogAPIRequest(ctx context.Context, tokenID, userID, endpoint, method, ip, userAgent string, success bool, errorCode string, responseTimeMS int) {
	m.logUsage(ctx, tokenID, userID, "request", endpoint, method, ip, userAgent, success, errorCode, responseTimeMS)
}

// CheckScope verifies that a token has the required scope for an operation.
func CheckScope(token *models.PersonalAccessToken, required models.TokenScope) error {
	if !token.HasScope(required) {
		return fmt.Errorf("insufficient scope: requires %s", required)
	}
	return nil
}

// CheckAnyScope verifies that a token has any of the required scopes.
func CheckAnyScope(token *models.PersonalAccessToken, required ...models.TokenScope) error {
	if !token.HasAnyScope(required...) {
		return fmt.Errorf("insufficient scope: requires one of %v", required)
	}
	return nil
}

// ExtractTokenFromHeader extracts a PAT from the Authorization header.
// Supports both "Bearer <token>" and raw token formats.
func ExtractTokenFromHeader(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)

	// Check for Bearer prefix
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	// Check if it's a raw PAT
	if strings.HasPrefix(authHeader, patPrefix) {
		return authHeader
	}

	return ""
}

// IsPATToken checks if a token string looks like a PAT.
func IsPATToken(token string) bool {
	return strings.HasPrefix(token, patPrefix)
}

// hashToken creates a bcrypt hash of a token.
// Since bcrypt has a 72-byte limit, we first SHA-256 the token to get a fixed-length hash.
// This is a common pattern used by GitHub, GitLab, and other services.
func hashToken(plaintext string) (string, error) {
	// SHA-256 hash to get fixed 32 bytes
	sha := sha256.Sum256([]byte(plaintext))

	// bcrypt the SHA-256 hash
	hash, err := bcrypt.GenerateFromPassword(sha[:], bcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt failed: %w", err)
	}

	return string(hash), nil
}

// verifyToken checks if a plaintext token matches a stored hash.
func verifyToken(plaintext, storedHash string) bool {
	// SHA-256 hash the plaintext
	sha := sha256.Sum256([]byte(plaintext))

	// Compare with stored bcrypt hash
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), sha[:])
	return err == nil
}
