// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuthenticator_Authenticate_Success(t *testing.T) {
	manager, err := NewBasicAuthManager("testuser", "securepassword123")
	if err != nil {
		t.Fatalf("Failed to create basic auth manager: %v", err)
	}

	config := &BasicAuthenticatorConfig{
		DefaultRole: "admin",
	}
	auth := NewBasicAuthenticator(manager, config)

	tests := []struct {
		name         string
		username     string
		password     string
		wantUsername string
		wantRole     string
	}{
		{
			name:         "valid credentials",
			username:     "testuser",
			password:     "securepassword123",
			wantUsername: "testuser",
			wantRole:     "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			credentials := base64.StdEncoding.EncodeToString([]byte(tt.username + ":" + tt.password))
			req.Header.Set("Authorization", "Basic "+credentials)

			subject, err := auth.Authenticate(context.Background(), req)
			if err != nil {
				t.Errorf("Authenticate() error = %v", err)
				return
			}

			if subject.Username != tt.wantUsername {
				t.Errorf("Authenticate() username = %v, want %v", subject.Username, tt.wantUsername)
			}

			if !subject.HasRole(tt.wantRole) {
				t.Errorf("Authenticate() should have role %v, has %v", tt.wantRole, subject.Roles)
			}

			if subject.AuthMethod != AuthModeBasic {
				t.Errorf("Authenticate() AuthMethod = %v, want %v", subject.AuthMethod, AuthModeBasic)
			}

			if subject.Issuer != "local" {
				t.Errorf("Authenticate() Issuer = %v, want local", subject.Issuer)
			}

			// ID should be the username for basic auth
			if subject.ID != tt.wantUsername {
				t.Errorf("Authenticate() ID = %v, want %v", subject.ID, tt.wantUsername)
			}
		})
	}
}

func TestBasicAuthenticator_Authenticate_Errors(t *testing.T) {
	manager, err := NewBasicAuthManager("testuser", "securepassword123")
	if err != nil {
		t.Fatalf("Failed to create basic auth manager: %v", err)
	}

	auth := NewBasicAuthenticator(manager, nil)

	tests := []struct {
		name         string
		setupRequest func(*http.Request)
		wantErr      error
	}{
		{
			name:         "no credentials",
			setupRequest: func(r *http.Request) {},
			wantErr:      ErrNoCredentials,
		},
		{
			name: "wrong password",
			setupRequest: func(r *http.Request) {
				credentials := base64.StdEncoding.EncodeToString([]byte("testuser:wrongpassword"))
				r.Header.Set("Authorization", "Basic "+credentials)
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "wrong username",
			setupRequest: func(r *http.Request) {
				credentials := base64.StdEncoding.EncodeToString([]byte("wronguser:securepassword123"))
				r.Header.Set("Authorization", "Basic "+credentials)
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "malformed authorization header - no Basic",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "dXNlcjpwYXNz")
			},
			wantErr: ErrNoCredentials,
		},
		{
			name: "malformed authorization header - wrong scheme",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer some-token")
			},
			wantErr: ErrNoCredentials,
		},
		{
			name: "invalid base64",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic !!invalid!!")
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "missing colon separator",
			setupRequest: func(r *http.Request) {
				credentials := base64.StdEncoding.EncodeToString([]byte("usernamepassword"))
				r.Header.Set("Authorization", "Basic "+credentials)
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "empty credentials",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic ")
			},
			wantErr: ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupRequest(req)

			_, err := auth.Authenticate(context.Background(), req)
			if err == nil {
				t.Errorf("Authenticate() expected error %v, got nil", tt.wantErr)
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Authenticate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestBasicAuthenticator_Name(t *testing.T) {
	manager, _ := NewBasicAuthManager("user", "password12345678")
	auth := NewBasicAuthenticator(manager, nil)

	if auth.Name() != string(AuthModeBasic) {
		t.Errorf("Name() = %v, want %v", auth.Name(), AuthModeBasic)
	}
}

func TestBasicAuthenticator_Priority(t *testing.T) {
	manager, _ := NewBasicAuthManager("user", "password12345678")
	auth := NewBasicAuthenticator(manager, nil)

	// Basic auth should have priority 25 (lowest, after JWT at 20)
	if auth.Priority() != 25 {
		t.Errorf("Priority() = %v, want 25", auth.Priority())
	}
}

func TestBasicAuthenticator_ImplementsInterface(t *testing.T) {
	manager, _ := NewBasicAuthManager("user", "password12345678")
	auth := NewBasicAuthenticator(manager, nil)

	// Verify it implements the Authenticator interface
	var _ Authenticator = auth
}

func TestBasicAuthenticator_DefaultRole(t *testing.T) {
	manager, _ := NewBasicAuthManager("testuser", "securepassword123")

	tests := []struct {
		name     string
		config   *BasicAuthenticatorConfig
		wantRole string
	}{
		{
			name:     "custom default role",
			config:   &BasicAuthenticatorConfig{DefaultRole: "editor"},
			wantRole: "editor",
		},
		{
			name:     "nil config defaults to viewer",
			config:   nil,
			wantRole: "viewer",
		},
		{
			name:     "empty role defaults to viewer",
			config:   &BasicAuthenticatorConfig{DefaultRole: ""},
			wantRole: "viewer",
		},
		{
			name:     "viewer role",
			config:   &BasicAuthenticatorConfig{DefaultRole: "viewer"},
			wantRole: "viewer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewBasicAuthenticator(manager, tt.config)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			credentials := base64.StdEncoding.EncodeToString([]byte("testuser:securepassword123"))
			req.Header.Set("Authorization", "Basic "+credentials)

			subject, err := auth.Authenticate(context.Background(), req)
			if err != nil {
				t.Fatalf("Authenticate() error = %v", err)
			}

			if !subject.HasRole(tt.wantRole) {
				t.Errorf("HasRole(%q) = false, want true. Roles: %v", tt.wantRole, subject.Roles)
			}
		})
	}
}

func TestBasicAuthenticator_SubjectConversion(t *testing.T) {
	manager, _ := NewBasicAuthManager("testuser", "securepassword123")
	config := &BasicAuthenticatorConfig{DefaultRole: "editor"}
	auth := NewBasicAuthenticator(manager, config)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:securepassword123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Convert back to Claims for backwards compatibility
	claims := subject.ToClaims()
	if claims.Username != "testuser" {
		t.Errorf("ToClaims() username = %v, want testuser", claims.Username)
	}
	if claims.Role != "editor" {
		t.Errorf("ToClaims() role = %v, want editor", claims.Role)
	}
}

func TestBasicAuthenticator_WWWAuthenticateHeader(t *testing.T) {
	manager, _ := NewBasicAuthManager("user", "password12345678")
	auth := NewBasicAuthenticator(manager, nil)

	header := auth.GetWWWAuthenticateHeader()
	if header == "" {
		t.Error("GetWWWAuthenticateHeader() returned empty string")
	}
	if header != `Basic realm="Cartographus", charset="UTF-8"` {
		t.Errorf("GetWWWAuthenticateHeader() = %q, unexpected value", header)
	}
}
