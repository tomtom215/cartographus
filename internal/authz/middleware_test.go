// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package authz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tomtom215/cartographus/internal/auth"
)

// mockAuthSubjectContext creates a context with an AuthSubject for testing
func mockAuthSubjectContext(subject *auth.AuthSubject) context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, auth.AuthSubjectContextKey, subject)
}

func TestMiddleware_Authorize_AdminRole(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	tests := []struct {
		name       string
		object     string
		action     string
		subject    *auth.AuthSubject
		wantStatus int
		wantCalled bool
	}{
		{
			name:   "admin can read any resource",
			object: "/api/sessions",
			action: "read",
			subject: &auth.AuthSubject{
				ID:       "admin-user",
				Username: "admin",
				Roles:    []string{"admin"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:   "admin can write any resource",
			object: "/api/settings",
			action: "write",
			subject: &auth.AuthSubject{
				ID:       "admin-user",
				Username: "admin",
				Roles:    []string{"admin"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:   "admin can delete any resource",
			object: "/api/admin/users",
			action: "delete",
			subject: &auth.AuthSubject{
				ID:       "admin-user",
				Username: "admin",
				Roles:    []string{"admin"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.Authorize(tt.object, tt.action, func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.object, nil)
			req = req.WithContext(mockAuthSubjectContext(tt.subject))
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestMiddleware_Authorize_ViewerRole(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	tests := []struct {
		name       string
		object     string
		action     string
		subject    *auth.AuthSubject
		wantStatus int
		wantCalled bool
	}{
		{
			name:   "viewer can read public resources",
			object: "/api/sessions",
			action: "read",
			subject: &auth.AuthSubject{
				ID:       "viewer-user",
				Username: "viewer",
				Roles:    []string{"viewer"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:   "viewer cannot write resources",
			object: "/api/settings",
			action: "write",
			subject: &auth.AuthSubject{
				ID:       "viewer-user",
				Username: "viewer",
				Roles:    []string{"viewer"},
			},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name:   "viewer cannot delete resources",
			object: "/api/sessions",
			action: "delete",
			subject: &auth.AuthSubject{
				ID:       "viewer-user",
				Username: "viewer",
				Roles:    []string{"viewer"},
			},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.Authorize(tt.object, tt.action, func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.object, nil)
			req = req.WithContext(mockAuthSubjectContext(tt.subject))
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestMiddleware_Authorize_EditorRole(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	tests := []struct {
		name       string
		object     string
		action     string
		subject    *auth.AuthSubject
		wantStatus int
		wantCalled bool
	}{
		{
			name:   "editor can read resources",
			object: "/api/sessions",
			action: "read",
			subject: &auth.AuthSubject{
				ID:       "editor-user",
				Username: "editor",
				Roles:    []string{"editor"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:   "editor can write resources",
			object: "/api/settings",
			action: "write",
			subject: &auth.AuthSubject{
				ID:       "editor-user",
				Username: "editor",
				Roles:    []string{"editor"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:   "editor cannot delete resources",
			object: "/api/admin/users",
			action: "delete",
			subject: &auth.AuthSubject{
				ID:       "editor-user",
				Username: "editor",
				Roles:    []string{"editor"},
			},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.Authorize(tt.object, tt.action, func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.object, nil)
			req = req.WithContext(mockAuthSubjectContext(tt.subject))
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestMiddleware_Authorize_NoSubject(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	handlerCalled := false
	handler := m.Authorize("/api/sessions", "read", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	// No AuthSubject in context
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if handlerCalled {
		t.Error("Handler should not be called when no subject in context")
	}
}

func TestMiddleware_Authorize_EmptyRoles(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	// User with no roles should get default role (viewer)
	subject := &auth.AuthSubject{
		ID:       "no-role-user",
		Username: "noroles",
		Roles:    []string{}, // Empty roles
	}

	tests := []struct {
		name       string
		object     string
		action     string
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "user with no roles gets default viewer - can read",
			object:     "/api/sessions",
			action:     "read",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "user with no roles gets default viewer - cannot write",
			object:     "/api/settings",
			action:     "write",
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.Authorize(tt.object, tt.action, func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.object, nil)
			req = req.WithContext(mockAuthSubjectContext(subject))
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestMiddleware_AuthorizeRequest(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	tests := []struct {
		name       string
		method     string
		path       string
		subject    *auth.AuthSubject
		wantStatus int
		wantCalled bool
	}{
		{
			name:   "GET request - read action",
			method: http.MethodGet,
			path:   "/api/sessions",
			subject: &auth.AuthSubject{
				ID:    "viewer-user",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:   "POST request - write action",
			method: http.MethodPost,
			path:   "/api/settings",
			subject: &auth.AuthSubject{
				ID:    "editor-user",
				Roles: []string{"editor"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:   "PUT request - write action",
			method: http.MethodPut,
			path:   "/api/settings",
			subject: &auth.AuthSubject{
				ID:    "viewer-user",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name:   "DELETE request - delete action",
			method: http.MethodDelete,
			path:   "/api/sessions/123",
			subject: &auth.AuthSubject{
				ID:    "admin-user",
				Roles: []string{"admin"},
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.AuthorizeRequest(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req = req.WithContext(mockAuthSubjectContext(tt.subject))
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestMiddleware_MultipleRoles(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	// User with both viewer and editor roles
	subject := &auth.AuthSubject{
		ID:       "multi-role-user",
		Username: "multirole",
		Roles:    []string{"viewer", "editor"},
	}

	tests := []struct {
		name       string
		object     string
		action     string
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "can read (viewer role)",
			object:     "/api/sessions",
			action:     "read",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "can write (editor role)",
			object:     "/api/settings",
			action:     "write",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "cannot delete (no admin role)",
			object:     "/api/admin/users",
			action:     "delete",
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.Authorize(tt.object, tt.action, func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.object, nil)
			req = req.WithContext(mockAuthSubjectContext(subject))
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestNewMiddleware(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)
	if m == nil {
		t.Fatal("NewMiddleware returned nil")
	}
}

// =====================================================
// AuthorizeWithGroups Tests
// ADR-0015: Zero Trust - Group-based authorization
// =====================================================

func TestMiddleware_AuthorizeWithGroups_NoSubject(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	handlerCalled := false
	handler := m.AuthorizeWithGroups("/api/sessions", "read", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	// No AuthSubject in context
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if handlerCalled {
		t.Error("Handler should not be called when no subject in context")
	}
}

func TestMiddleware_AuthorizeWithGroups_WithGroups(t *testing.T) {
	ctx := context.Background()
	enforcer, err := NewEnforcer(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	// Add a policy that allows a group to read
	_, err = enforcer.AddPolicy("homelab-admins", "/api/admin/users", "GET")
	if err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	m := NewMiddleware(enforcer)

	tests := []struct {
		name       string
		subject    *auth.AuthSubject
		object     string
		action     string
		wantStatus int
		wantCalled bool
	}{
		{
			name: "user with group permission allowed",
			subject: &auth.AuthSubject{
				ID:       "user-1",
				Username: "user1",
				Roles:    []string{},
				Groups:   []string{"homelab-admins"},
			},
			object:     "/api/admin/users",
			action:     "GET",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name: "user with role and group permission allowed via role",
			subject: &auth.AuthSubject{
				ID:       "user-2",
				Username: "user2",
				Roles:    []string{"admin"},
				Groups:   []string{"some-group"},
			},
			object:     "/api/admin/users",
			action:     "GET",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name: "user with role and group permission allowed via group",
			subject: &auth.AuthSubject{
				ID:       "user-3",
				Username: "user3",
				Roles:    []string{"viewer"},
				Groups:   []string{"homelab-admins"},
			},
			object:     "/api/admin/users",
			action:     "GET",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name: "user without matching role or group denied",
			subject: &auth.AuthSubject{
				ID:       "user-4",
				Username: "user4",
				Roles:    []string{"viewer"},
				Groups:   []string{"regular-users"},
			},
			object:     "/api/admin/users",
			action:     "GET",
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name: "user with empty roles and groups uses default",
			subject: &auth.AuthSubject{
				ID:       "user-5",
				Username: "user5",
				Roles:    []string{},
				Groups:   []string{},
			},
			object:     "/api/maps",
			action:     "GET",
			wantStatus: http.StatusOK,
			wantCalled: true, // default role (viewer) allows GET /api/maps
		},
		{
			name: "user with nil roles and groups",
			subject: &auth.AuthSubject{
				ID:       "user-6",
				Username: "user6",
				Roles:    nil,
				Groups:   nil,
			},
			object:     "/api/maps",
			action:     "GET",
			wantStatus: http.StatusOK,
			wantCalled: true, // default role applied
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.AuthorizeWithGroups(tt.object, tt.action, func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.object, nil)
			req = req.WithContext(mockAuthSubjectContext(tt.subject))
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestMiddleware_AuthorizeWithGroups_MultipleGroups(t *testing.T) {
	ctx := context.Background()
	enforcer, err := NewEnforcer(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	// Add policies for different groups
	_, err = enforcer.AddPolicy("ops-team", "/api/metrics", "read")
	if err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}
	_, err = enforcer.AddPolicy("dev-team", "/api/debug", "read")
	if err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	m := NewMiddleware(enforcer)

	// User in multiple groups should have combined permissions
	subject := &auth.AuthSubject{
		ID:       "multi-group-user",
		Username: "multigroup",
		Roles:    []string{},
		Groups:   []string{"ops-team", "dev-team"},
	}

	// Should be able to access ops resource
	handlerCalled := false
	handler := m.AuthorizeWithGroups("/api/metrics", "read", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	req = req.WithContext(mockAuthSubjectContext(subject))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d for ops resource", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called for ops resource")
	}

	// Should also be able to access dev resource
	handlerCalled = false
	handler = m.AuthorizeWithGroups("/api/debug", "read", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req = httptest.NewRequest(http.MethodGet, "/api/debug", nil)
	req = req.WithContext(mockAuthSubjectContext(subject))
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d for dev resource", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called for dev resource")
	}
}

// =====================================================
// AuthorizeRequest Additional Tests
// =====================================================

func TestMiddleware_AuthorizeRequest_NoSubject(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	handlerCalled := false
	handler := m.AuthorizeRequest(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	// No AuthSubject in context
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if handlerCalled {
		t.Error("Handler should not be called when no subject in context")
	}
}

func TestMiddleware_AuthorizeRequest_AllMethods(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	m := NewMiddleware(enforcer)

	// Admin subject for testing all methods
	subject := &auth.AuthSubject{
		ID:    "admin-user",
		Roles: []string{"admin"},
	}

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{"HEAD request maps to read", http.MethodHead, http.StatusOK},
		{"OPTIONS request maps to read", http.MethodOptions, http.StatusOK},
		{"PATCH request maps to write", http.MethodPatch, http.StatusOK},
		{"CONNECT request maps to read (default)", "CONNECT", http.StatusOK},
		{"TRACE request maps to read (default)", "TRACE", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.AuthorizeRequest(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, "/api/sessions", nil)
			req = req.WithContext(mockAuthSubjectContext(subject))
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if !handlerCalled {
				t.Errorf("handler should be called for %s", tt.method)
			}
		})
	}
}

// =====================================================
// methodToAction Tests
// =====================================================

func TestMethodToAction(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{http.MethodGet, "read"},
		{http.MethodHead, "read"},
		{http.MethodOptions, "read"},
		{http.MethodPost, "write"},
		{http.MethodPut, "write"},
		{http.MethodPatch, "write"},
		{http.MethodDelete, "delete"},
		{"CONNECT", "read"}, // default case
		{"TRACE", "read"},   // default case
		{"CUSTOM", "read"},  // unknown method defaults to read
		{"", "read"},        // empty method defaults to read
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := methodToAction(tt.method)
			if got != tt.want {
				t.Errorf("methodToAction(%q) = %q, want %q", tt.method, got, tt.want)
			}
		})
	}
}
