// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package authz provides authorization functionality using Casbin.
// It implements RBAC (Role-Based Access Control) with support for
// hierarchical roles, path-based permissions, and caching.
//
// ADR-0015: Zero Trust Authentication & Authorization
package authz

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
)

//go:embed model.conf
var embeddedModel string

//go:embed policy.csv
var embeddedPolicy string

// EnforcerConfig holds configuration for the Casbin enforcer.
type EnforcerConfig struct {
	// ModelPath is the path to the Casbin model file.
	// If empty, uses embedded model.
	ModelPath string

	// PolicyPath is the path to the Casbin policy file.
	// If empty, uses embedded policy.
	PolicyPath string

	// AutoReload enables automatic policy reload.
	AutoReload bool

	// ReloadInterval is how often to check for policy changes.
	ReloadInterval time.Duration

	// DefaultRole is assigned to users without explicit roles.
	DefaultRole string

	// CacheEnabled enables enforcement decision caching.
	CacheEnabled bool

	// CacheTTL is how long to cache decisions.
	CacheTTL time.Duration
}

// DefaultEnforcerConfig returns default configuration.
func DefaultEnforcerConfig() *EnforcerConfig {
	return &EnforcerConfig{
		AutoReload:     true,
		ReloadInterval: 30 * time.Second,
		DefaultRole:    "viewer",
		CacheEnabled:   true,
		CacheTTL:       5 * time.Minute,
	}
}

// Enforcer wraps the Casbin enforcer with additional functionality.
type Enforcer struct {
	config   *EnforcerConfig
	enforcer *casbin.SyncedEnforcer
	cache    *enforcementCache
}

// NewEnforcer creates a new authorization enforcer.
func NewEnforcer(ctx context.Context, config *EnforcerConfig) (*Enforcer, error) {
	if config == nil {
		config = DefaultEnforcerConfig()
	}

	// Load model
	var m model.Model
	var err error

	if config.ModelPath != "" && fileExists(config.ModelPath) {
		m, err = model.NewModelFromFile(config.ModelPath)
	} else {
		m, err = model.NewModelFromString(embeddedModel)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load casbin model: %w", err)
	}

	// Create enforcer with model first
	var enforcer *casbin.SyncedEnforcer

	// Load policy
	if config.PolicyPath != "" && fileExists(config.PolicyPath) {
		adapter := fileadapter.NewAdapter(config.PolicyPath)
		enforcer, err = casbin.NewSyncedEnforcer(m, adapter)
	} else {
		// Use string adapter for embedded policy
		enforcer, err = casbin.NewSyncedEnforcer(m)
		if err == nil {
			// Load embedded policy manually
			err = loadEmbeddedPolicy(enforcer, embeddedPolicy)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	// Enable auto-reload if configured
	if config.AutoReload && config.PolicyPath != "" {
		enforcer.StartAutoLoadPolicy(config.ReloadInterval)
	}

	e := &Enforcer{
		config:   config,
		enforcer: enforcer,
	}

	// Initialize cache if enabled
	if config.CacheEnabled {
		e.cache = newEnforcementCache(config.CacheTTL)
	}

	return e, nil
}

// loadEmbeddedPolicy parses and loads the embedded policy CSV.
func loadEmbeddedPolicy(enforcer *casbin.SyncedEnforcer, policy string) error {
	lines := strings.Split(policy, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		// Trim whitespace from each part
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		ptype := parts[0]
		rule := parts[1:]

		switch ptype {
		case "p":
			if len(rule) >= 3 {
				_, err := enforcer.AddPolicy(rule[0], rule[1], rule[2])
				if err != nil {
					return fmt.Errorf("failed to add policy %v: %w", rule, err)
				}
			}
		case "g":
			if len(rule) >= 2 {
				_, err := enforcer.AddGroupingPolicy(rule[0], rule[1])
				if err != nil {
					return fmt.Errorf("failed to add grouping policy %v: %w", rule, err)
				}
			}
		}
	}
	return nil
}

// Enforce checks if the subject can perform the action on the object.
func (e *Enforcer) Enforce(subject, object, action string) (bool, error) {
	// Check cache first
	if e.cache != nil {
		if allowed, ok := e.cache.get(subject, object, action); ok {
			return allowed, nil
		}
	}

	// Perform enforcement
	allowed, err := e.enforcer.Enforce(subject, object, action)
	if err != nil {
		return false, fmt.Errorf("enforcement failed: %w", err)
	}

	// Cache result
	if e.cache != nil {
		e.cache.set(subject, object, action, allowed)
	}

	return allowed, nil
}

// EnforceWithRoles checks if any of the subject's roles allow the action.
func (e *Enforcer) EnforceWithRoles(subject string, roles []string, object, action string) (bool, error) {
	// Check if subject directly has permission
	if allowed, err := e.Enforce(subject, object, action); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}

	// Check each role
	for _, role := range roles {
		if allowed, err := e.Enforce(role, object, action); err != nil {
			return false, err
		} else if allowed {
			return true, nil
		}
	}

	// Check default role
	if e.config.DefaultRole != "" && len(roles) == 0 {
		return e.Enforce(e.config.DefaultRole, object, action)
	}

	return false, nil
}

// AddRoleForUser assigns a role to a user.
func (e *Enforcer) AddRoleForUser(user, role string) (bool, error) {
	added, err := e.enforcer.AddGroupingPolicy(user, role)
	if err != nil {
		return false, fmt.Errorf("failed to add role: %w", err)
	}
	if e.cache != nil {
		e.cache.invalidateUser(user)
	}
	return added, nil
}

// DeleteRoleForUser removes a role from a user.
func (e *Enforcer) DeleteRoleForUser(user, role string) (bool, error) {
	removed, err := e.enforcer.RemoveGroupingPolicy(user, role)
	if err != nil {
		return false, fmt.Errorf("failed to remove role: %w", err)
	}
	if e.cache != nil {
		e.cache.invalidateUser(user)
	}
	return removed, nil
}

// GetRolesForUser returns all roles for a user.
func (e *Enforcer) GetRolesForUser(user string) ([]string, error) {
	return e.enforcer.GetRolesForUser(user)
}

// GetUsersForRole returns all users with a specific role.
func (e *Enforcer) GetUsersForRole(role string) ([]string, error) {
	return e.enforcer.GetUsersForRole(role)
}

// AddPolicy adds a new policy rule.
func (e *Enforcer) AddPolicy(subject, object, action string) (bool, error) {
	added, err := e.enforcer.AddPolicy(subject, object, action)
	if err != nil {
		return false, fmt.Errorf("failed to add policy: %w", err)
	}
	if e.cache != nil {
		e.cache.clear()
	}
	return added, nil
}

// RemovePolicy removes a policy rule.
func (e *Enforcer) RemovePolicy(subject, object, action string) (bool, error) {
	removed, err := e.enforcer.RemovePolicy(subject, object, action)
	if err != nil {
		return false, fmt.Errorf("failed to remove policy: %w", err)
	}
	if e.cache != nil {
		e.cache.clear()
	}
	return removed, nil
}

// ErrNoAdapter is returned when SavePolicy or LoadPolicy is called
// but no file adapter is configured.
var ErrNoAdapter = errors.New("no policy adapter configured; using embedded policy")

// SavePolicy persists the policy to storage.
// Returns ErrNoAdapter if no file adapter is configured (using embedded policy).
func (e *Enforcer) SavePolicy() error {
	if e.config.PolicyPath == "" {
		return ErrNoAdapter
	}
	return e.enforcer.SavePolicy()
}

// LoadPolicy reloads the policy from storage.
// Returns ErrNoAdapter if no file adapter is configured (using embedded policy).
func (e *Enforcer) LoadPolicy() error {
	if e.config.PolicyPath == "" {
		return ErrNoAdapter
	}
	if err := e.enforcer.LoadPolicy(); err != nil {
		return err
	}
	if e.cache != nil {
		e.cache.clear()
	}
	return nil
}

// Close stops the enforcer and cleans up resources.
func (e *Enforcer) Close() {
	e.enforcer.StopAutoLoadPolicy()
	if e.cache != nil {
		e.cache.stop()
	}
}

// GetPolicy returns all policy rules.
func (e *Enforcer) GetPolicy() [][]string {
	//nolint:errcheck // GetPolicy only fails if enforcer is nil, which is a programming error
	policies, _ := e.enforcer.GetPolicy()
	return policies
}

// GetFilteredPolicy returns filtered policy rules.
// fieldIndex: the field index to filter by (0=subject, 1=object, 2=action)
// fieldValues: the values to match
func (e *Enforcer) GetFilteredPolicy(fieldIndex int, fieldValues ...string) [][]string {
	//nolint:errcheck // GetFilteredPolicy only fails if enforcer is nil, which is a programming error
	policies, _ := e.enforcer.GetFilteredPolicy(fieldIndex, fieldValues...)
	return policies
}

// GetGroupingPolicy returns all role inheritance rules.
func (e *Enforcer) GetGroupingPolicy() [][]string {
	//nolint:errcheck // GetGroupingPolicy only fails if enforcer is nil, which is a programming error
	policies, _ := e.enforcer.GetGroupingPolicy()
	return policies
}

// AddGroupingPolicy adds a role assignment (g, user, role).
func (e *Enforcer) AddGroupingPolicy(user, role string) error {
	_, err := e.enforcer.AddGroupingPolicy(user, role)
	if err != nil {
		return fmt.Errorf("failed to add grouping policy: %w", err)
	}
	if e.cache != nil {
		e.cache.invalidateUser(user)
	}
	return nil
}

// RemoveGroupingPolicy removes a role assignment.
func (e *Enforcer) RemoveGroupingPolicy(user, role string) error {
	_, err := e.enforcer.RemoveGroupingPolicy(user, role)
	if err != nil {
		return fmt.Errorf("failed to remove grouping policy: %w", err)
	}
	if e.cache != nil {
		e.cache.invalidateUser(user)
	}
	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
