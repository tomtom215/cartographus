// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality using Zitadel OIDC.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
//
// This file provides claims mapping from Zitadel OIDC tokens to the internal
// AuthSubject structure. The mapping is deterministic and fully documented.
package auth

import (
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// ZitadelIDTokenClaims is an interface that matches what Zitadel provides.
// This allows us to work with both oidc.IDTokenClaims and oidc.TokenClaims.
type ZitadelIDTokenClaims interface {
	GetSubject() string
	GetIssuer() string
	GetAudience() []string
	GetExpiration() oidc.Time
	GetIssuedAt() oidc.Time
	GetNonce() string
	GetAuthenticationContextClassReference() string
	GetAuthTime() oidc.Time
	GetAuthorizedParty() string
	GetClaims() map[string]interface{}
}

// ZitadelUserInfoClaims provides access to profile and contact claims.
// This interface matches the common claims returned by OIDC providers.
type ZitadelUserInfoClaims interface {
	GetSubject() string
	GetName() string
	GetGivenName() string
	GetFamilyName() string
	GetMiddleName() string
	GetNickname() string
	GetPreferredUsername() string
	GetProfile() string
	GetPicture() string
	GetWebsite() string
	GetEmail() string
	IsEmailVerified() bool
	GetGender() oidc.Gender
	GetBirthdate() string
	GetZoneinfo() string
	GetLocale() oidc.Locale
	GetPhoneNumber() string
	IsPhoneNumberVerified() bool
	GetAddress() *oidc.UserInfoAddress
	GetClaims() map[string]interface{}
}

// MapZitadelClaimsToAuthSubject converts Zitadel token claims to AuthSubject.
// This function performs deterministic mapping of OIDC claims to the internal
// AuthSubject structure used throughout the application.
//
// The mapping follows this priority for username:
//  1. preferred_username claim
//  2. name claim
//  3. email claim
//  4. subject claim (fallback)
//
// Roles are extracted from the configured roles claim (default: "roles").
// Groups are extracted from the configured groups claim (default: "groups").
//
// Parameters:
//   - claims: The OIDC token claims from Zitadel
//   - config: Claims mapping configuration
//   - defaultRoles: Roles to assign if none found in token
//
// Returns:
//   - AuthSubject populated with mapped claims
func MapZitadelClaimsToAuthSubject(
	claims *oidc.IDTokenClaims,
	config *ZitadelClaimsMappingConfig,
	defaultRoles []string,
) *AuthSubject {
	if claims == nil {
		return nil
	}

	subject := &AuthSubject{
		ID:         claims.Subject,
		Email:      claims.Email,
		Issuer:     claims.Issuer,
		AuthMethod: AuthModeOIDC,
		Provider:   "oidc",
	}

	// Handle email verification
	// oidc.Bool is type Bool bool - use direct conversion
	subject.EmailVerified = bool(claims.EmailVerified)

	// Set timestamps using Zitadel's oidc.Time type
	// oidc.Time is int64 with AsTime() returning time.Time
	if !claims.IssuedAt.AsTime().IsZero() {
		subject.IssuedAt = claims.IssuedAt.AsTime().Unix()
	}
	if !claims.Expiration.AsTime().IsZero() {
		subject.ExpiresAt = claims.Expiration.AsTime().Unix()
	}

	// Extract username using configured claim priority
	subject.Username = extractZitadelUsername(claims, config)
	if subject.Username == "" {
		subject.Username = subject.ID // Fallback to subject
	}

	// Extract roles from claims
	subject.Roles = extractZitadelStringSlice(claims.Claims, config.RolesClaim)
	if len(subject.Roles) == 0 && len(defaultRoles) > 0 {
		// Copy default roles to avoid mutation
		subject.Roles = make([]string, len(defaultRoles))
		copy(subject.Roles, defaultRoles)
	}

	// Extract groups from claims
	subject.Groups = extractZitadelStringSlice(claims.Claims, config.GroupsClaim)

	// Store raw claims for extensibility
	if claims.Claims != nil {
		subject.RawClaims = claims.Claims
	}

	return subject
}

// MapZitadelUserInfoToAuthSubject creates an AuthSubject from userinfo response.
// This is used when fetching additional claims from the userinfo endpoint.
//
// Parameters:
//   - userInfo: The userinfo response from Zitadel
//   - issuer: The issuer URL for this provider
//   - config: Claims mapping configuration
//   - defaultRoles: Roles to assign if none found
//
// Returns:
//   - AuthSubject populated with userinfo claims
func MapZitadelUserInfoToAuthSubject(
	userInfo *oidc.UserInfo,
	issuer string,
	config *ZitadelClaimsMappingConfig,
	defaultRoles []string,
) *AuthSubject {
	if userInfo == nil {
		return nil
	}

	subject := &AuthSubject{
		ID:         userInfo.Subject,
		Email:      userInfo.Email,
		Issuer:     issuer,
		AuthMethod: AuthModeOIDC,
		Provider:   "oidc",
	}

	// Handle email verification using Zitadel's Bool type
	// oidc.Bool is type Bool bool - use direct conversion
	subject.EmailVerified = bool(userInfo.EmailVerified)

	// Extract username from userinfo
	subject.Username = extractZitadelUserInfoUsername(userInfo, config)
	if subject.Username == "" {
		subject.Username = subject.ID
	}

	// Extract roles and groups from claims
	if userInfo.Claims != nil {
		subject.Roles = extractZitadelStringSlice(userInfo.Claims, config.RolesClaim)
		subject.Groups = extractZitadelStringSlice(userInfo.Claims, config.GroupsClaim)
		subject.RawClaims = userInfo.Claims
	}

	// Apply default roles if needed
	if len(subject.Roles) == 0 && len(defaultRoles) > 0 {
		subject.Roles = make([]string, len(defaultRoles))
		copy(subject.Roles, defaultRoles)
	}

	return subject
}

// extractZitadelUsername extracts username from ID token claims.
// Uses the configured claim priority to find the first non-empty value.
func extractZitadelUsername(claims *oidc.IDTokenClaims, config *ZitadelClaimsMappingConfig) string {
	if claims == nil || config == nil {
		return ""
	}

	for _, claimName := range config.UsernameClaims {
		switch claimName {
		case "preferred_username":
			if claims.PreferredUsername != "" {
				return claims.PreferredUsername
			}
		case "name":
			if claims.Name != "" {
				return claims.Name
			}
		case "email":
			if claims.Email != "" {
				return claims.Email
			}
		case "nickname":
			if claims.Nickname != "" {
				return claims.Nickname
			}
		default:
			// Check raw claims for custom claim names
			if claims.Claims != nil {
				if val, ok := claims.Claims[claimName].(string); ok && val != "" {
					return val
				}
			}
		}
	}

	return ""
}

// extractZitadelUserInfoUsername extracts username from userinfo response.
func extractZitadelUserInfoUsername(userInfo *oidc.UserInfo, config *ZitadelClaimsMappingConfig) string {
	if userInfo == nil || config == nil {
		return ""
	}

	for _, claimName := range config.UsernameClaims {
		switch claimName {
		case "preferred_username":
			if userInfo.PreferredUsername != "" {
				return userInfo.PreferredUsername
			}
		case "name":
			if userInfo.Name != "" {
				return userInfo.Name
			}
		case "email":
			if userInfo.Email != "" {
				return userInfo.Email
			}
		case "nickname":
			if userInfo.Nickname != "" {
				return userInfo.Nickname
			}
		default:
			// Check raw claims for custom claim names
			if userInfo.Claims != nil {
				if val, ok := userInfo.Claims[claimName].(string); ok && val != "" {
					return val
				}
			}
		}
	}

	return ""
}

// extractZitadelStringSlice extracts a string slice from raw claims.
// Handles both []string and []interface{} types.
//
// This function is deterministic:
//   - Returns nil if claims is nil
//   - Returns nil if claimName is not found
//   - Returns the slice if it's already []string
//   - Converts []interface{} to []string, skipping non-string elements
func extractZitadelStringSlice(claims map[string]interface{}, claimName string) []string {
	if claims == nil || claimName == "" {
		return nil
	}

	val, ok := claims[claimName]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []string:
		// Already the right type - make a copy to avoid mutation
		result := make([]string, len(v))
		copy(result, v)
		return result
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case string:
		// Single value - return as slice
		return []string{v}
	default:
		return nil
	}
}

// MapZitadelTokensToMetadata creates session metadata from OIDC tokens.
// This stores tokens for refresh and logout operations.
//
// The metadata map contains:
//   - "access_token": OAuth 2.0 access token
//   - "refresh_token": OAuth 2.0 refresh token (if provided)
//   - "id_token": OIDC ID token (if provided)
//
// Parameters:
//   - accessToken: The OAuth 2.0 access token
//   - refreshToken: The OAuth 2.0 refresh token (may be empty)
//   - idToken: The OIDC ID token (may be empty)
//
// Returns:
//   - Map containing non-empty token values
func MapZitadelTokensToMetadata(accessToken, refreshToken, idToken string) map[string]string {
	metadata := make(map[string]string)
	if accessToken != "" {
		metadata["access_token"] = accessToken
	}
	if refreshToken != "" {
		metadata["refresh_token"] = refreshToken
	}
	if idToken != "" {
		metadata["id_token"] = idToken
	}
	return metadata
}

// ExtractIDTokenFromMetadata retrieves the ID token from session metadata.
// Returns empty string if not present.
func ExtractIDTokenFromMetadata(metadata map[string]string) string {
	if metadata == nil {
		return ""
	}
	return metadata["id_token"]
}

// ExtractAccessTokenFromMetadata retrieves the access token from session metadata.
// Returns empty string if not present.
func ExtractAccessTokenFromMetadata(metadata map[string]string) string {
	if metadata == nil {
		return ""
	}
	return metadata["access_token"]
}

// ExtractRefreshTokenFromMetadata retrieves the refresh token from session metadata.
// Returns empty string if not present.
func ExtractRefreshTokenFromMetadata(metadata map[string]string) string {
	if metadata == nil {
		return ""
	}
	return metadata["refresh_token"]
}
