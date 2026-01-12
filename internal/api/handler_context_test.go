// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestGetHandlerContext tests extraction of handler context from request.
func TestGetHandlerContext(t *testing.T) {
	tests := []struct {
		name         string
		subject      *auth.AuthSubject
		requestID    string
		wantUserID   string
		wantUsername string
		wantIsAdmin  bool
		wantIsEditor bool
		wantRole     string
		wantNonNil   bool
	}{
		{
			name:         "no authentication",
			subject:      nil,
			requestID:    "req-123",
			wantUserID:   "",
			wantUsername: "",
			wantIsAdmin:  false,
			wantIsEditor: false,
			wantRole:     "",
			wantNonNil:   true,
		},
		{
			name: "viewer role",
			subject: &auth.AuthSubject{
				ID:       "user-001",
				Username: "viewer_user",
				Roles:    []string{models.RoleViewer},
			},
			requestID:    "req-456",
			wantUserID:   "user-001",
			wantUsername: "viewer_user",
			wantIsAdmin:  false,
			wantIsEditor: false,
			wantRole:     models.RoleViewer,
			wantNonNil:   true,
		},
		{
			name: "editor role",
			subject: &auth.AuthSubject{
				ID:       "user-002",
				Username: "editor_user",
				Roles:    []string{models.RoleEditor},
			},
			requestID:    "req-789",
			wantUserID:   "user-002",
			wantUsername: "editor_user",
			wantIsAdmin:  false,
			wantIsEditor: true,
			wantRole:     models.RoleEditor,
			wantNonNil:   true,
		},
		{
			name: "admin role",
			subject: &auth.AuthSubject{
				ID:       "user-003",
				Username: "admin_user",
				Roles:    []string{models.RoleAdmin},
			},
			requestID:    "req-abc",
			wantUserID:   "user-003",
			wantUsername: "admin_user",
			wantIsAdmin:  true,
			wantIsEditor: true, // Admin inherits editor
			wantRole:     models.RoleAdmin,
			wantNonNil:   true,
		},
		{
			name: "multiple roles including admin",
			subject: &auth.AuthSubject{
				ID:       "user-004",
				Username: "multi_role_user",
				Roles:    []string{models.RoleViewer, models.RoleEditor, models.RoleAdmin},
			},
			requestID:    "req-def",
			wantUserID:   "user-004",
			wantUsername: "multi_role_user",
			wantIsAdmin:  true,
			wantIsEditor: true,
			wantRole:     models.RoleAdmin,
			wantNonNil:   true,
		},
		{
			name: "no roles defaults to viewer",
			subject: &auth.AuthSubject{
				ID:       "user-005",
				Username: "no_role_user",
				Roles:    []string{},
			},
			requestID:    "req-ghi",
			wantUserID:   "user-005",
			wantUsername: "no_role_user",
			wantIsAdmin:  false,
			wantIsEditor: false,
			wantRole:     models.RoleViewer,
			wantNonNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Request-ID", tt.requestID)

			if tt.subject != nil {
				ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, tt.subject)
				req = req.WithContext(ctx)
			}

			hctx := GetHandlerContext(req)

			if hctx == nil && tt.wantNonNil {
				t.Fatal("GetHandlerContext returned nil, want non-nil")
			}

			if hctx.UserID != tt.wantUserID {
				t.Errorf("UserID = %q, want %q", hctx.UserID, tt.wantUserID)
			}

			if hctx.Username != tt.wantUsername {
				t.Errorf("Username = %q, want %q", hctx.Username, tt.wantUsername)
			}

			if hctx.IsAdmin != tt.wantIsAdmin {
				t.Errorf("IsAdmin = %v, want %v", hctx.IsAdmin, tt.wantIsAdmin)
			}

			if hctx.IsEditor != tt.wantIsEditor {
				t.Errorf("IsEditor = %v, want %v", hctx.IsEditor, tt.wantIsEditor)
			}

			if hctx.EffectiveRole != tt.wantRole {
				t.Errorf("EffectiveRole = %q, want %q", hctx.EffectiveRole, tt.wantRole)
			}

			if hctx.RequestID != tt.requestID {
				t.Errorf("RequestID = %q, want %q", hctx.RequestID, tt.requestID)
			}
		})
	}
}

// TestHandlerContext_IsAuthenticated tests the IsAuthenticated method.
func TestHandlerContext_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name string
		hctx *HandlerContext
		want bool
	}{
		{
			name: "nil context",
			hctx: nil,
			want: false,
		},
		{
			name: "nil subject",
			hctx: &HandlerContext{Subject: nil},
			want: false,
		},
		{
			name: "valid subject",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-001"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.hctx.IsAuthenticated()
			if got != tt.want {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHandlerContext_CanAccessUser tests user data access checks.
func TestHandlerContext_CanAccessUser(t *testing.T) {
	tests := []struct {
		name         string
		hctx         *HandlerContext
		targetUserID string
		want         bool
	}{
		{
			name:         "nil context",
			hctx:         nil,
			targetUserID: "user-001",
			want:         false,
		},
		{
			name:         "nil subject",
			hctx:         &HandlerContext{Subject: nil},
			targetUserID: "user-001",
			want:         false,
		},
		{
			name: "user accessing own data",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-001"},
				UserID:  "user-001",
				IsAdmin: false,
			},
			targetUserID: "user-001",
			want:         true,
		},
		{
			name: "user accessing other user data",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-001"},
				UserID:  "user-001",
				IsAdmin: false,
			},
			targetUserID: "user-002",
			want:         false,
		},
		{
			name: "admin accessing any user data",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "admin-001"},
				UserID:  "admin-001",
				IsAdmin: true,
			},
			targetUserID: "user-002",
			want:         true,
		},
		{
			name: "admin accessing own data",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "admin-001"},
				UserID:  "admin-001",
				IsAdmin: true,
			},
			targetUserID: "admin-001",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.hctx.CanAccessUser(tt.targetUserID)
			if got != tt.want {
				t.Errorf("CanAccessUser(%q) = %v, want %v", tt.targetUserID, got, tt.want)
			}
		})
	}
}

// TestHandlerContext_RequireAdmin tests admin requirement checks.
func TestHandlerContext_RequireAdmin(t *testing.T) {
	tests := []struct {
		name    string
		hctx    *HandlerContext
		wantErr bool
		errType error
	}{
		{
			name:    "nil context",
			hctx:    nil,
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
		{
			name:    "nil subject",
			hctx:    &HandlerContext{Subject: nil},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
		{
			name: "non-admin user",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-001"},
				UserID:  "user-001",
				IsAdmin: false,
			},
			wantErr: true,
			errType: ErrNotAuthorized,
		},
		{
			name: "admin user",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "admin-001"},
				UserID:  "admin-001",
				IsAdmin: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hctx.RequireAdmin()
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireAdmin() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("RequireAdmin() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

// TestHandlerContext_RequireEditor tests editor requirement checks.
func TestHandlerContext_RequireEditor(t *testing.T) {
	tests := []struct {
		name    string
		hctx    *HandlerContext
		wantErr bool
		errType error
	}{
		{
			name:    "nil context",
			hctx:    nil,
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
		{
			name:    "nil subject",
			hctx:    &HandlerContext{Subject: nil},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
		{
			name: "viewer user",
			hctx: &HandlerContext{
				Subject:  &auth.AuthSubject{ID: "user-001"},
				UserID:   "user-001",
				IsAdmin:  false,
				IsEditor: false,
			},
			wantErr: true,
			errType: ErrNotAuthorized,
		},
		{
			name: "editor user",
			hctx: &HandlerContext{
				Subject:  &auth.AuthSubject{ID: "editor-001"},
				UserID:   "editor-001",
				IsAdmin:  false,
				IsEditor: true,
			},
			wantErr: false,
		},
		{
			name: "admin user",
			hctx: &HandlerContext{
				Subject:  &auth.AuthSubject{ID: "admin-001"},
				UserID:   "admin-001",
				IsAdmin:  true,
				IsEditor: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hctx.RequireEditor()
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireEditor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("RequireEditor() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

// TestHandlerContext_RequireAccessToUser tests user access requirement checks.
func TestHandlerContext_RequireAccessToUser(t *testing.T) {
	tests := []struct {
		name         string
		hctx         *HandlerContext
		targetUserID string
		wantErr      bool
		errType      error
	}{
		{
			name:         "nil context",
			hctx:         nil,
			targetUserID: "user-001",
			wantErr:      true,
			errType:      ErrNotAuthenticated,
		},
		{
			name:         "nil subject",
			hctx:         &HandlerContext{Subject: nil},
			targetUserID: "user-001",
			wantErr:      true,
			errType:      ErrNotAuthenticated,
		},
		{
			name: "user accessing own data",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-001"},
				UserID:  "user-001",
				IsAdmin: false,
			},
			targetUserID: "user-001",
			wantErr:      false,
		},
		{
			name: "user accessing other user data",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-001"},
				UserID:  "user-001",
				IsAdmin: false,
			},
			targetUserID: "user-002",
			wantErr:      true,
			errType:      ErrNotAuthorized,
		},
		{
			name: "admin accessing any user data",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "admin-001"},
				UserID:  "admin-001",
				IsAdmin: true,
			},
			targetUserID: "user-002",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hctx.RequireAccessToUser(tt.targetUserID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireAccessToUser(%q) error = %v, wantErr %v", tt.targetUserID, err, tt.wantErr)
			}
			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("RequireAccessToUser(%q) error = %v, want %v", tt.targetUserID, err, tt.errType)
			}
		})
	}
}

// TestHandlerContext_FilterByUser tests SQL filter generation.
func TestHandlerContext_FilterByUser(t *testing.T) {
	tests := []struct {
		name         string
		hctx         *HandlerContext
		userIDColumn string
		want         string
	}{
		{
			name:         "nil context returns impossible condition",
			hctx:         nil,
			userIDColumn: "user_id",
			want:         "1=0",
		},
		{
			name:         "nil subject returns impossible condition",
			hctx:         &HandlerContext{Subject: nil},
			userIDColumn: "user_id",
			want:         "1=0",
		},
		{
			name: "admin returns all data",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "admin-001"},
				UserID:  "admin-001",
				IsAdmin: true,
			},
			userIDColumn: "user_id",
			want:         "1=1",
		},
		{
			name: "regular user returns filtered condition",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-001"},
				UserID:  "user-001",
				IsAdmin: false,
			},
			userIDColumn: "user_id",
			want:         "user_id = 'user-001'",
		},
		{
			name: "regular user with different column",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-002"},
				UserID:  "user-002",
				IsAdmin: false,
			},
			userIDColumn: "player_user_id",
			want:         "player_user_id = 'user-002'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.hctx.FilterByUser(tt.userIDColumn)
			if got != tt.want {
				t.Errorf("FilterByUser(%q) = %q, want %q", tt.userIDColumn, got, tt.want)
			}
		})
	}
}

// TestHandlerContext_GetUserIDForQuery tests user ID extraction for queries.
func TestHandlerContext_GetUserIDForQuery(t *testing.T) {
	tests := []struct {
		name string
		hctx *HandlerContext
		want string
	}{
		{
			name: "nil context returns empty",
			hctx: nil,
			want: "",
		},
		{
			name: "nil subject returns empty",
			hctx: &HandlerContext{Subject: nil},
			want: "",
		},
		{
			name: "admin returns empty (all users)",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "admin-001"},
				UserID:  "admin-001",
				IsAdmin: true,
			},
			want: "",
		},
		{
			name: "regular user returns own ID",
			hctx: &HandlerContext{
				Subject: &auth.AuthSubject{ID: "user-001"},
				UserID:  "user-001",
				IsAdmin: false,
			},
			want: "user-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.hctx.GetUserIDForQuery()
			if got != tt.want {
				t.Errorf("GetUserIDForQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAuthError tests the AuthError type.
func TestAuthError(t *testing.T) {
	err := &AuthError{
		Code:       "TEST_ERROR",
		Message:    "Test error message",
		StatusCode: 400,
	}

	if err.Error() != "Test error message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "Test error message")
	}

	if err.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, 400)
	}

	if err.Code != "TEST_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "TEST_ERROR")
	}
}

// TestRespondAuthError tests the auth error response helper.
func TestRespondAuthError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatusCode int
	}{
		{
			name:           "AuthError with custom status",
			err:            &AuthError{Code: "CUSTOM", Message: "Custom error", StatusCode: 418},
			wantStatusCode: 418,
		},
		{
			name:           "ErrNotAuthenticated",
			err:            ErrNotAuthenticated,
			wantStatusCode: 401,
		},
		{
			name:           "ErrNotAuthorized",
			err:            ErrNotAuthorized,
			wantStatusCode: 403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			RespondAuthError(w, tt.err)

			if w.Code != tt.wantStatusCode {
				t.Errorf("RespondAuthError() status = %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}
