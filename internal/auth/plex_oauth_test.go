// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

func TestNewPlexOAuthClient(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		redirectURI  string
	}{
		{"basic initialization", "test-client-id", "test-client-secret", "http://localhost:3857/api/v1/auth/plex/callback"},
		{"public client without secret", "public-client", "", "http://localhost:3857/callback"},
		{"production-like config", "cartographus-12345", "super-secret-key", "https://maps.example.com/api/v1/auth/plex/callback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewPlexOAuthClient(tt.clientID, tt.clientSecret, tt.redirectURI)

			if client == nil {
				t.Fatal("NewPlexOAuthClient() returned nil")
			}
			if client.ClientID != tt.clientID {
				t.Errorf("ClientID = %q, want %q", client.ClientID, tt.clientID)
			}
			if client.ClientSecret != tt.clientSecret {
				t.Errorf("ClientSecret = %q, want %q", client.ClientSecret, tt.clientSecret)
			}
			if client.RedirectURI != tt.redirectURI {
				t.Errorf("RedirectURI = %q, want %q", client.RedirectURI, tt.redirectURI)
			}
			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}
		})
	}
}

func TestGeneratePKCE(t *testing.T) {
	client := NewPlexOAuthClient("test-client", "", "http://localhost/callback")

	t.Run("generates valid PKCE challenge", func(t *testing.T) {
		pkce, err := client.GeneratePKCE()
		if err != nil {
			t.Fatalf("GeneratePKCE() error = %v", err)
		}
		if pkce == nil {
			t.Fatal("GeneratePKCE() returned nil")
		}
		if len(pkce.CodeVerifier) != 43 {
			t.Errorf("CodeVerifier length = %d, want 43", len(pkce.CodeVerifier))
		}
		if len(pkce.CodeChallenge) != 43 {
			t.Errorf("CodeChallenge length = %d, want 43", len(pkce.CodeChallenge))
		}
		if pkce.CodeVerifier == pkce.CodeChallenge {
			t.Error("CodeVerifier and CodeChallenge should be different")
		}
	})

	t.Run("generates unique challenges", func(t *testing.T) {
		pkce1, _ := client.GeneratePKCE()
		pkce2, _ := client.GeneratePKCE()
		if pkce1.CodeVerifier == pkce2.CodeVerifier || pkce1.CodeChallenge == pkce2.CodeChallenge {
			t.Error("Multiple calls should generate unique values")
		}
	})

	t.Run("valid base64url characters", func(t *testing.T) {
		pkce, _ := client.GeneratePKCE()
		for _, code := range []string{pkce.CodeVerifier, pkce.CodeChallenge} {
			for _, c := range code {
				if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
					t.Errorf("Contains invalid character: %c", c)
				}
			}
		}
	})
}

func TestBuildAuthorizationURL(t *testing.T) {
	tests := []struct {
		name          string
		clientID      string
		redirectURI   string
		codeChallenge string
		state         string
		wantContains  []string
	}{
		{
			name:          "basic URL construction",
			clientID:      "my-client-id",
			redirectURI:   "http://localhost:3857/callback",
			codeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			state:         "random-state-123",
			wantContains:  []string{"https://app.plex.tv/auth#?", "clientID=my-client-id", "code_challenge_method=S256", "state=random-state-123"},
		},
		{
			name:          "URL with special characters",
			clientID:      "client-123",
			redirectURI:   "https://example.com/callback",
			codeChallenge: "test_challenge_abc123",
			state:         "state-with-special+chars",
			wantContains:  []string{"clientID=client-123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewPlexOAuthClient(tt.clientID, "", tt.redirectURI)
			url := client.BuildAuthorizationURL(tt.codeChallenge, tt.state)

			for _, want := range tt.wantContains {
				if !strings.Contains(url, want) {
					t.Errorf("BuildAuthorizationURL() missing %q", want)
				}
			}
		})
	}
}

func TestExchangeCodeForToken(t *testing.T) {
	tests := []struct {
		name         string
		clientSecret string
		handler      http.HandlerFunc
		wantErr      bool
		errContains  string
		wantToken    string
	}{
		{
			name: "successful exchange",
			handler: func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "test-auth-code" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "mock-access-token", "token_type": "Bearer", "expires_in": 7776000,
				})
			},
			wantErr:   false,
			wantToken: "mock-access-token",
		},
		{
			name:         "with client secret",
			clientSecret: "my-secret",
			handler: func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				if r.Form.Get("client_secret") != "my-secret" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "token-with-secret", "expires_in": 3600})
			},
			wantErr:   false,
			wantToken: "token-with-secret",
		},
		{
			name: "invalid code error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "invalid_grant"}`))
			},
			wantErr:     true,
			errContains: "400",
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "invalid JSON response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{invalid json`))
			},
			wantErr:     true,
			errContains: "parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := NewPlexOAuthClient("test-client", tt.clientSecret, "http://localhost/callback")
			client.tokenURL = server.URL

			token, err := client.ExchangeCodeForToken(context.Background(), "test-auth-code", "test-verifier")

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if token.AccessToken != tt.wantToken {
				t.Errorf("AccessToken = %q, want %q", token.AccessToken, tt.wantToken)
			}
		})
	}

	t.Run("context canceled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer server.Close()

		client := NewPlexOAuthClient("test-client", "", "http://localhost/callback")
		client.tokenURL = server.URL

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.ExchangeCodeForToken(ctx, "code", "verifier")
		if err == nil {
			t.Error("Expected error for canceled context")
		}
	})
}

func TestRefreshToken(t *testing.T) {
	tests := []struct {
		name         string
		clientSecret string
		handler      http.HandlerFunc
		wantErr      bool
		wantToken    string
	}{
		{
			name: "successful refresh",
			handler: func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "old-refresh-token" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "new-access-token", "refresh_token": "new-refresh-token", "expires_in": 7776000,
				})
			},
			wantErr:   false,
			wantToken: "new-access-token",
		},
		{
			name:         "with client secret",
			clientSecret: "secret-123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				if r.Form.Get("client_secret") != "secret-123" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "refreshed-token", "expires_in": 3600})
			},
			wantErr:   false,
			wantToken: "refreshed-token",
		},
		{
			name: "invalid refresh token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "invalid_grant"}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := NewPlexOAuthClient("test-client", tt.clientSecret, "http://localhost/callback")
			client.tokenURL = server.URL

			token, err := client.RefreshToken(context.Background(), "old-refresh-token")

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if token.AccessToken != tt.wantToken {
				t.Errorf("AccessToken = %q, want %q", token.AccessToken, tt.wantToken)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	client := NewPlexOAuthClient("test-client", "", "http://localhost/callback")

	tests := []struct {
		name      string
		token     *PlexOAuthToken
		wantValid bool
	}{
		{"nil token", nil, false},
		{"empty access token", &PlexOAuthToken{AccessToken: "", ExpiresAt: time.Now().Unix() + 3600}, false},
		{"valid token", &PlexOAuthToken{AccessToken: "valid-token", ExpiresAt: time.Now().Unix() + 3600}, true},
		{"expired token", &PlexOAuthToken{AccessToken: "expired-token", ExpiresAt: time.Now().Unix() - 3600}, false},
		{"expiring within buffer", &PlexOAuthToken{AccessToken: "soon-expiring", ExpiresAt: time.Now().Unix() + 60}, false},
		{"outside buffer", &PlexOAuthToken{AccessToken: "valid-outside-buffer", ExpiresAt: time.Now().Unix() + 400}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := client.ValidateToken(tt.token); got != tt.wantValid {
				t.Errorf("ValidateToken() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}
