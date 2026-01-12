// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package authz provides authorization middleware using Casbin.
// ADR-0015: Zero Trust Authentication & Authorization
package authz

import (
	"net/http"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/logging"
)

// Middleware provides authorization middleware using Casbin.
type Middleware struct {
	enforcer *Enforcer
}

// NewMiddleware creates a new authorization middleware.
func NewMiddleware(enforcer *Enforcer) *Middleware {
	return &Middleware{
		enforcer: enforcer,
	}
}

// Authorize is middleware that enforces authorization for a specific object and action.
func (m *Middleware) Authorize(object, action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject := auth.GetAuthSubject(r.Context())
		if subject == nil {
			http.Error(w, "Forbidden: no authentication context", http.StatusForbidden)
			return
		}

		allowed, err := m.enforcer.EnforceWithRoles(subject.ID, subject.Roles, object, action)
		if err != nil {
			logging.Error().Err(err).Msg("Authorization error")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// AuthorizeRequest is middleware that determines the action from the HTTP method
// and authorizes based on the request path.
func (m *Middleware) AuthorizeRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject := auth.GetAuthSubject(r.Context())
		if subject == nil {
			http.Error(w, "Forbidden: no authentication context", http.StatusForbidden)
			return
		}

		// Map HTTP method to action
		action := methodToAction(r.Method)
		object := r.URL.Path

		allowed, err := m.enforcer.EnforceWithRoles(subject.ID, subject.Roles, object, action)
		if err != nil {
			logging.Error().Err(err).Msg("Authorization error")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// methodToAction maps HTTP methods to Casbin actions.
func methodToAction(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "read"
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return "write"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}

// AuthorizeWithGroups is middleware that also considers group memberships.
func (m *Middleware) AuthorizeWithGroups(object, action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject := auth.GetAuthSubject(r.Context())
		if subject == nil {
			http.Error(w, "Forbidden: no authentication context", http.StatusForbidden)
			return
		}

		// Combine roles and groups for authorization
		allRoles := make([]string, 0, len(subject.Roles)+len(subject.Groups))
		allRoles = append(allRoles, subject.Roles...)
		allRoles = append(allRoles, subject.Groups...)

		allowed, err := m.enforcer.EnforceWithRoles(subject.ID, allRoles, object, action)
		if err != nil {
			logging.Error().Err(err).Msg("Authorization error")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
