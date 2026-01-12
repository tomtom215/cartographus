// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ID token validation errors
var (
	// ErrInvalidIDToken indicates the ID token is malformed or invalid.
	ErrInvalidIDToken = errors.New("invalid ID token")

	// ErrIDTokenExpired indicates the ID token has expired.
	ErrIDTokenExpired = errors.New("ID token expired")

	// ErrInvalidIssuer indicates the issuer claim doesn't match.
	ErrInvalidIssuer = errors.New("invalid issuer")

	// ErrInvalidAudience indicates the audience claim doesn't match.
	ErrInvalidAudience = errors.New("invalid audience")

	// ErrMissingSubject indicates the sub claim is missing.
	ErrMissingSubject = errors.New("missing subject claim")

	// ErrInvalidNonce indicates the nonce doesn't match.
	ErrInvalidNonce = errors.New("invalid nonce")
)

// IDTokenClaims contains the parsed claims from an ID token.
type IDTokenClaims struct {
	// Standard OIDC claims
	Subject      string   `json:"sub"`
	Issuer       string   `json:"iss"`
	Audience     []string `json:"aud"`
	ExpiresAt    int64    `json:"exp"`
	IssuedAt     int64    `json:"iat"`
	AuthTime     int64    `json:"auth_time,omitempty"`
	Nonce        string   `json:"nonce,omitempty"`
	ACR          string   `json:"acr,omitempty"`
	AMR          []string `json:"amr,omitempty"`
	AZP          string   `json:"azp,omitempty"`
	AtHash       string   `json:"at_hash,omitempty"`
	CHash        string   `json:"c_hash,omitempty"`
	SessionID    string   `json:"sid,omitempty"`
	SessionState string   `json:"session_state,omitempty"`

	// Common profile claims
	Name              string `json:"name,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	MiddleName        string `json:"middle_name,omitempty"`
	Nickname          string `json:"nickname,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Profile           string `json:"profile,omitempty"`
	Picture           string `json:"picture,omitempty"`
	Website           string `json:"website,omitempty"`
	Gender            string `json:"gender,omitempty"`
	Birthdate         string `json:"birthdate,omitempty"`
	Zoneinfo          string `json:"zoneinfo,omitempty"`
	Locale            string `json:"locale,omitempty"`
	UpdatedAt         int64  `json:"updated_at,omitempty"`

	// Email claims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`

	// Phone claims
	PhoneNumber         string `json:"phone_number,omitempty"`
	PhoneNumberVerified bool   `json:"phone_number_verified,omitempty"`

	// Address claim
	Address *AddressClaim `json:"address,omitempty"`

	// Custom claims (roles, groups)
	Roles  []string `json:"roles,omitempty"`
	Groups []string `json:"groups,omitempty"`

	// Raw claims for extensibility
	RawClaims map[string]interface{} `json:"-"`
}

// AddressClaim represents the address claim structure.
type AddressClaim struct {
	Formatted     string `json:"formatted,omitempty"`
	StreetAddress string `json:"street_address,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Country       string `json:"country,omitempty"`
}

// IDTokenValidationConfig holds configuration for ID token validation.
type IDTokenValidationConfig struct {
	// Issuer is the expected issuer (iss claim).
	Issuer string

	// ClientID is the expected audience (aud claim).
	ClientID string

	// ClockSkew allows for clock differences between IdP and this server.
	ClockSkew time.Duration

	// RolesClaim is the claim name for roles (default: "roles").
	RolesClaim string

	// GroupsClaim is the claim name for groups (default: "groups").
	GroupsClaim string

	// UsernameClaims is the list of claims to try for username.
	UsernameClaims []string

	// DefaultRoles are roles assigned if no roles in token.
	DefaultRoles []string
}

// IDTokenValidator validates and parses OIDC ID tokens.
type IDTokenValidator struct {
	config    *IDTokenValidationConfig
	jwksCache *JWKSCache
}

// NewIDTokenValidator creates a new ID token validator.
func NewIDTokenValidator(config *IDTokenValidationConfig, jwksCache *JWKSCache) *IDTokenValidator {
	if config.ClockSkew == 0 {
		config.ClockSkew = 1 * time.Minute
	}
	if config.RolesClaim == "" {
		config.RolesClaim = "roles"
	}
	if config.GroupsClaim == "" {
		config.GroupsClaim = "groups"
	}
	if len(config.UsernameClaims) == 0 {
		config.UsernameClaims = []string{"preferred_username", "name", "email"}
	}
	return &IDTokenValidator{
		config:    config,
		jwksCache: jwksCache,
	}
}

// GetConfig returns the validation configuration.
// Used by back-channel logout to verify issuer and client ID.
// ADR-0015 Phase 4B.3: Back-Channel Logout
func (v *IDTokenValidator) GetConfig() *IDTokenValidationConfig {
	return v.config
}

// ValidateAndParse validates an ID token and returns the parsed claims.
func (v *IDTokenValidator) ValidateAndParse(ctx context.Context, idToken string, expectedNonce string) (*IDTokenClaims, error) {
	if idToken == "" {
		return nil, ErrInvalidIDToken
	}

	// Parse and validate the token
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		// Verify signing algorithm is RSA
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from header
		kidVal, ok := token.Header["kid"]
		if !ok {
			return nil, errors.New("token missing kid header")
		}
		kid, ok := kidVal.(string)
		if !ok {
			return nil, errors.New("token kid header is not a string")
		}

		// Get the public key from JWKS cache
		key, err := v.jwksCache.GetKey(ctx, kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get key for kid %s: %w", kid, err)
		}

		return key, nil
	}, jwt.WithLeeway(v.config.ClockSkew))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrIDTokenExpired
		}
		return nil, fmt.Errorf("%w: %s", ErrInvalidIDToken, err.Error())
	}

	if !token.Valid {
		return nil, ErrInvalidIDToken
	}

	// Extract claims
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidIDToken
	}

	// Parse into IDTokenClaims
	claims := v.parseMapClaims(mapClaims)

	// Validate issuer
	if claims.Issuer != v.config.Issuer {
		return nil, fmt.Errorf("%w: got %s, want %s", ErrInvalidIssuer, claims.Issuer, v.config.Issuer)
	}

	// Validate audience
	if !v.containsAudience(claims.Audience, v.config.ClientID) {
		return nil, fmt.Errorf("%w: client ID %s not in audience %v", ErrInvalidAudience, v.config.ClientID, claims.Audience)
	}

	// Validate subject
	if claims.Subject == "" {
		return nil, ErrMissingSubject
	}

	// Validate nonce if expected
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		return nil, fmt.Errorf("%w: got %s, want %s", ErrInvalidNonce, claims.Nonce, expectedNonce)
	}

	return claims, nil
}

// parseMapClaims parses jwt.MapClaims into IDTokenClaims.
func (v *IDTokenValidator) parseMapClaims(claims jwt.MapClaims) *IDTokenClaims {
	result := &IDTokenClaims{
		RawClaims: claims,
	}

	// Parse standard OIDC claims
	v.parseStandardClaims(claims, result)

	// Parse profile claims
	v.parseProfileClaims(claims, result)

	// Parse contact claims (email, phone)
	v.parseContactClaims(claims, result)

	// Parse custom claims (roles, groups)
	result.Roles = v.parseStringSlice(claims, v.config.RolesClaim)
	result.Groups = v.parseStringSlice(claims, v.config.GroupsClaim)

	// Apply default roles if none found
	if len(result.Roles) == 0 && len(v.config.DefaultRoles) > 0 {
		result.Roles = append([]string{}, v.config.DefaultRoles...)
	}

	return result
}

// parseStandardClaims parses standard OIDC claims (sub, iss, aud, exp, etc.).
func (v *IDTokenValidator) parseStandardClaims(claims jwt.MapClaims, result *IDTokenClaims) {
	result.Subject = getStringClaim(claims, "sub")
	result.Issuer = getStringClaim(claims, "iss")
	result.Audience = v.parseAudienceClaim(claims["aud"])
	result.ExpiresAt = getTimestampClaim(claims, "exp")
	result.IssuedAt = getTimestampClaim(claims, "iat")
	result.AuthTime = getTimestampClaim(claims, "auth_time")
	result.Nonce = getStringClaim(claims, "nonce")
	result.ACR = getStringClaim(claims, "acr")
	result.AMR = v.parseStringSlice(claims, "amr")
	result.AZP = getStringClaim(claims, "azp")
	result.AtHash = getStringClaim(claims, "at_hash")
	result.CHash = getStringClaim(claims, "c_hash")
	result.SessionID = getStringClaim(claims, "sid")
	result.SessionState = getStringClaim(claims, "session_state")
}

// parseProfileClaims parses profile-related claims (name, picture, etc.).
func (v *IDTokenValidator) parseProfileClaims(claims jwt.MapClaims, result *IDTokenClaims) {
	result.Name = getStringClaim(claims, "name")
	result.GivenName = getStringClaim(claims, "given_name")
	result.FamilyName = getStringClaim(claims, "family_name")
	result.MiddleName = getStringClaim(claims, "middle_name")
	result.Nickname = getStringClaim(claims, "nickname")
	result.PreferredUsername = getStringClaim(claims, "preferred_username")
	result.Profile = getStringClaim(claims, "profile")
	result.Picture = getStringClaim(claims, "picture")
	result.Website = getStringClaim(claims, "website")
	result.Gender = getStringClaim(claims, "gender")
	result.Birthdate = getStringClaim(claims, "birthdate")
	result.Zoneinfo = getStringClaim(claims, "zoneinfo")
	result.Locale = getStringClaim(claims, "locale")
	result.UpdatedAt = getTimestampClaim(claims, "updated_at")
}

// parseContactClaims parses email and phone claims.
func (v *IDTokenValidator) parseContactClaims(claims jwt.MapClaims, result *IDTokenClaims) {
	result.Email = getStringClaim(claims, "email")
	result.EmailVerified = getBoolClaim(claims, "email_verified")
	result.PhoneNumber = getStringClaim(claims, "phone_number")
	result.PhoneNumberVerified = getBoolClaim(claims, "phone_number_verified")
}

// getStringClaim extracts a string claim value.
func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

// getTimestampClaim extracts a numeric timestamp claim as int64.
func getTimestampClaim(claims jwt.MapClaims, key string) int64 {
	if val, ok := claims[key].(float64); ok {
		return int64(val)
	}
	return 0
}

// getBoolClaim extracts a boolean claim value.
func getBoolClaim(claims jwt.MapClaims, key string) bool {
	if val, ok := claims[key].(bool); ok {
		return val
	}
	return false
}

// parseAudienceClaim parses the aud claim which can be a string or array.
func (v *IDTokenValidator) parseAudienceClaim(aud interface{}) []string {
	if aud == nil {
		return nil
	}
	switch a := aud.(type) {
	case string:
		return []string{a}
	case []interface{}:
		result := make([]string, 0, len(a))
		for _, item := range a {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return a
	default:
		return nil
	}
}

// parseStringSlice extracts a string slice from claims.
func (v *IDTokenValidator) parseStringSlice(claims jwt.MapClaims, key string) []string {
	val, ok := claims[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// containsAudience checks if the audience list contains the expected value.
func (v *IDTokenValidator) containsAudience(audience []string, expected string) bool {
	for _, aud := range audience {
		if aud == expected {
			return true
		}
	}
	return false
}

// ToAuthSubject converts IDTokenClaims to an AuthSubject.
func (c *IDTokenClaims) ToAuthSubject(usernameClaims []string) *AuthSubject {
	subject := &AuthSubject{
		ID:            c.Subject,
		Email:         c.Email,
		EmailVerified: c.EmailVerified,
		Roles:         c.Roles,
		Groups:        c.Groups,
		Issuer:        c.Issuer,
		AuthMethod:    AuthModeOIDC,
		IssuedAt:      c.IssuedAt,
		ExpiresAt:     c.ExpiresAt,
		RawClaims:     c.RawClaims,
		Provider:      "oidc",
	}

	// Find username from configured claims
	for _, claim := range usernameClaims {
		switch claim {
		case "preferred_username":
			if c.PreferredUsername != "" {
				subject.Username = c.PreferredUsername
				return subject
			}
		case "name":
			if c.Name != "" {
				subject.Username = c.Name
				return subject
			}
		case "email":
			if c.Email != "" {
				subject.Username = c.Email
				return subject
			}
		case "nickname":
			if c.Nickname != "" {
				subject.Username = c.Nickname
				return subject
			}
		default:
			// Try raw claims for custom claim names
			if val, ok := c.RawClaims[claim].(string); ok && val != "" {
				subject.Username = val
				return subject
			}
		}
	}

	// Fall back to subject if no username found
	if subject.Username == "" {
		subject.Username = c.Subject
	}

	return subject
}
