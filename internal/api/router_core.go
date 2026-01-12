// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/authz"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/middleware"
)

// Router sets up HTTP routes using Chi router (ADR-0016).
type Router struct {
	handler       *Handler
	middleware    *auth.Middleware
	chiMiddleware *ChiMiddleware // ADR-0016: Production-hardened Chi middleware
	indexTemplate *template.Template

	// importRouteRegistrar is called during SetupChi() to register import routes.
	// This is set externally when NATS is enabled and import is configured.
	importRouteRegistrar func(mux *http.ServeMux)

	// Zero Trust components (ADR-0015)
	// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
	// Uses certified Zitadel OIDC library for OpenID Connect authentication.
	zeroTrustConfig     *config.SecurityConfig
	sessionStore        auth.SessionStore
	sessionStoreFactory *auth.SessionStoreFactory
	sessionMiddleware   *auth.SessionMiddleware
	oidcFlow            *auth.ZitadelOIDCFlow         // Certified Zitadel OIDC implementation
	oidcStateStore      *auth.BadgerZitadelStateStore // Durable OIDC state storage
	oidcAuditLogger     *auth.OIDCAuditLogger         // Production-grade audit logging
	plexFlow            *auth.PlexFlow
	flowHandlers        *auth.FlowHandlers
	policyHandlers      *authz.PolicyHandlers
	enforcer            *authz.Enforcer // Casbin RBAC enforcer
	authzMiddleware     *authz.Middleware

	// Detection system (ADR-0020)
	detectionHandlers *DetectionHandlers

	// Audit logging system
	auditHandlers *AuditHandlers

	// DLQ management
	dlqHandlers *DLQHandlers

	// WAL stats
	walHandlers *WALHandlers

	// Replay management (CRITICAL-002: Deterministic event replay)
	replayHandlers *ReplayHandlers

	// Recommendation engine (ADR-0024)
	recommendHandler *RecommendHandler

	// Sync handlers for data sync UI
	syncHandlers *SyncHandlers
}

// ConfigureDetection sets up the detection handlers for anomaly detection endpoints.
// ADR-0020: Detection rules engine for media playback security monitoring.
func (router *Router) ConfigureDetection(handlers *DetectionHandlers) {
	router.detectionHandlers = handlers
}

// GetDetectionHandlers returns the detection handlers (for external components).
func (router *Router) GetDetectionHandlers() *DetectionHandlers {
	return router.detectionHandlers
}

// ConfigureAudit sets up the audit handlers for audit log endpoints.
func (router *Router) ConfigureAudit(handlers *AuditHandlers) {
	router.auditHandlers = handlers
}

// GetAuditHandlers returns the audit handlers (for external components).
func (router *Router) GetAuditHandlers() *AuditHandlers {
	return router.auditHandlers
}

// ConfigureDLQ sets up the DLQ handlers for dead letter queue management.
func (router *Router) ConfigureDLQ(handlers *DLQHandlers) {
	router.dlqHandlers = handlers
}

// GetDLQHandlers returns the DLQ handlers (for external components).
func (router *Router) GetDLQHandlers() *DLQHandlers {
	return router.dlqHandlers
}

// ConfigureWAL sets up the WAL handlers for write-ahead log statistics.
func (router *Router) ConfigureWAL(handlers *WALHandlers) {
	router.walHandlers = handlers
}

// GetWALHandlers returns the WAL handlers (for external components).
func (router *Router) GetWALHandlers() *WALHandlers {
	return router.walHandlers
}

// ConfigureReplay sets up the replay handlers for event replay management.
// CRITICAL-002: Deterministic event replay for disaster recovery.
func (router *Router) ConfigureReplay(handlers *ReplayHandlers) {
	router.replayHandlers = handlers
}

// GetReplayHandlers returns the replay handlers (for external components).
func (router *Router) GetReplayHandlers() *ReplayHandlers {
	return router.replayHandlers
}

// ConfigureRecommend sets up the recommendation handler for recommendation engine endpoints.
// ADR-0024: Hybrid recommendation engine for media content.
func (router *Router) ConfigureRecommend(handler *RecommendHandler) {
	router.recommendHandler = handler
}

// GetRecommendHandler returns the recommendation handler (for external components).
func (router *Router) GetRecommendHandler() *RecommendHandler {
	return router.recommendHandler
}

// ConfigureSync sets up the sync handlers for data sync UI endpoints.
func (router *Router) ConfigureSync(handlers *SyncHandlers) {
	router.syncHandlers = handlers
}

// GetSyncHandlers returns the sync handlers (for external components).
func (router *Router) GetSyncHandlers() *SyncHandlers {
	return router.syncHandlers
}

// NewRouter creates a new router with all routes configured
func NewRouter(handler *Handler, middleware *auth.Middleware) *Router {
	// Parse index.html template with nonce support
	tmpl, err := template.ParseFiles("./internal/templates/index.html.tmpl")
	if err != nil {
		logging.Warn().Err(err).Msg("Failed to parse index.html template")
		logging.Warn().Msg("Falling back to static index.html (CSP nonces will not work)")
		tmpl = nil
	}

	// ADR-0016: Create Chi middleware from existing config
	reqsPerWindow, rateLimitDisabled := middleware.GetRateLimitConfig()
	chiMw := NewChiMiddlewareFromAuth(
		middleware.GetCORSOrigins(),
		reqsPerWindow,
		middleware.GetRateLimitWindow(),
		rateLimitDisabled,
	)

	return &Router{
		handler:       handler,
		middleware:    middleware,
		chiMiddleware: chiMw,
		indexTemplate: tmpl,
	}
}

// SetImportRouteRegistrar sets the function that will register import routes.
// This is called from import initialization when NATS is enabled.
func (router *Router) SetImportRouteRegistrar(registrar func(*http.ServeMux)) {
	router.importRouteRegistrar = registrar
}

// ConfigureZeroTrust initializes Zero Trust authentication components based on config.
// ADR-0015: Zero Trust Authentication & Authorization
// This should be called before SetupChi() if Zero Trust auth is needed.
//
//nolint:gocyclo // Complexity is inherent to multi-provider auth configuration
func (router *Router) ConfigureZeroTrust(ctx context.Context, securityCfg *config.SecurityConfig) error {
	if securityCfg == nil {
		return nil
	}

	router.zeroTrustConfig = securityCfg

	// Create session store based on configuration (Phase 4A.1)
	storeType := auth.SessionStoreType(securityCfg.SessionStore)
	if storeType == "" {
		storeType = auth.SessionStoreMemory
	}

	// Validate session store type
	if storeType != auth.SessionStoreMemory && storeType != auth.SessionStoreBadger {
		return fmt.Errorf("invalid session store type: %s (must be 'memory' or 'badger')", storeType)
	}

	// Create session store factory
	factory, err := auth.NewSessionStoreFactory(storeType, securityCfg.SessionStorePath)
	if err != nil {
		return fmt.Errorf("create session store: %w", err)
	}
	router.sessionStoreFactory = factory
	router.sessionStore = factory.CreateStore()

	logging.Info().Str("type", string(storeType)).Msg("Session store initialized")
	if storeType == auth.SessionStoreBadger {
		logging.Info().Str("path", securityCfg.SessionStorePath).Msg("Session store path")
	}

	// Create session middleware config
	sessionMWConfig := &auth.SessionMiddlewareConfig{
		CookieName:     securityCfg.OIDC.CookieName,
		SessionTTL:     securityCfg.OIDC.SessionMaxAge,
		SlidingSession: true,
		CookiePath:     "/",
		CookieSecure:   securityCfg.OIDC.CookieSecure,
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
	}
	if sessionMWConfig.CookieName == "" {
		sessionMWConfig.CookieName = "tautulli_session"
	}
	if sessionMWConfig.SessionTTL == 0 {
		sessionMWConfig.SessionTTL = 24 * time.Hour
	}

	router.sessionMiddleware = auth.NewSessionMiddleware(router.sessionStore, sessionMWConfig)

	// Check which auth modes are enabled
	authMode := securityCfg.AuthMode
	shouldConfigureOIDC := authMode == "oidc" || authMode == "multi"
	shouldConfigurePlex := authMode == "plex" || authMode == "multi"

	// Configure OIDC flow if enabled
	// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
	// Uses certified Zitadel OIDC library for OpenID Connect authentication.
	// Features:
	//   - OpenID Foundation certified OIDC implementation
	//   - PKCE support (RFC 7636)
	//   - at_hash and c_hash validation
	//   - Nonce validation for ID tokens
	if shouldConfigureOIDC && securityCfg.OIDC.IssuerURL != "" {
		oidcConfig := &auth.OIDCFlowConfig{
			IssuerURL:            securityCfg.OIDC.IssuerURL,
			ClientID:             securityCfg.OIDC.ClientID,
			ClientSecret:         securityCfg.OIDC.ClientSecret,
			RedirectURL:          securityCfg.OIDC.RedirectURL,
			Scopes:               securityCfg.OIDC.Scopes,
			PKCEEnabled:          securityCfg.OIDC.PKCEEnabled,
			StateTTL:             10 * time.Minute, // Default 10 minutes for state
			SessionDuration:      securityCfg.OIDC.SessionMaxAge,
			PostLoginRedirectURL: "/",
			JWKSCacheTTL:         securityCfg.OIDC.JWKSCacheTTL,
			RolesClaim:           securityCfg.OIDC.RolesClaim,
			UsernameClaims:       securityCfg.OIDC.UsernameClaims,
			DefaultRoles:         securityCfg.OIDC.DefaultRoles,
			NonceEnabled:         true, // Enable nonce for security
		}

		if len(oidcConfig.Scopes) == 0 {
			oidcConfig.Scopes = []string{"openid", "profile", "email"}
		}
		if oidcConfig.SessionDuration == 0 {
			oidcConfig.SessionDuration = 24 * time.Hour
		}
		if oidcConfig.JWKSCacheTTL == 0 {
			oidcConfig.JWKSCacheTTL = time.Hour
		}

		// Create Zitadel OIDC flow with durable state store
		// ADR-0015: Production-grade durable state storage for OIDC authorization flows.
		// Uses BadgerDB for ACID-compliant persistence that survives server restarts.
		var stateStore auth.ZitadelStateStore
		stateStorePath := securityCfg.SessionStorePath + "/oidc_state"
		if storeType == auth.SessionStoreBadger && securityCfg.SessionStorePath != "" {
			badgerStateStore, stateErr := auth.NewBadgerZitadelStateStore(stateStorePath)
			if stateErr != nil {
				logging.Warn().Err(stateErr).Msg("Failed to create durable OIDC state store, falling back to memory")
				stateStore = auth.NewZitadelMemoryStateStore()
			} else {
				stateStore = badgerStateStore
				router.oidcStateStore = badgerStateStore
				// Start cleanup routine for expired states
				badgerStateStore.StartCleanupRoutine(ctx, 5*time.Minute)
				logging.Info().Str("path", stateStorePath).Msg("Durable OIDC state store initialized (BadgerDB)")
			}
		} else {
			stateStore = auth.NewZitadelMemoryStateStore()
			logging.Info().Msg("OIDC state store initialized (in-memory)")
		}

		// Zitadel performs OIDC discovery during initialization
		zitadelFlow, err := auth.NewZitadelOIDCFlowFromConfig(ctx, oidcConfig, stateStore)
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to initialize Zitadel OIDC flow")
			logging.Warn().Msg("OIDC authentication will be unavailable until configuration is corrected")
			router.oidcFlow = nil
		} else {
			router.oidcFlow = zitadelFlow
			logging.Info().
				Str("issuer", securityCfg.OIDC.IssuerURL).
				Bool("pkce", securityCfg.OIDC.PKCEEnabled).
				Bool("nonce", true).
				Bool("durable_state", router.oidcStateStore != nil).
				Msg("Zitadel OIDC authentication configured (OpenID Foundation certified)")
		}
	}

	// Configure Plex flow if enabled
	if shouldConfigurePlex && securityCfg.PlexAuth.ClientID != "" {
		plexConfig := &auth.PlexFlowConfig{
			ClientID:     securityCfg.PlexAuth.ClientID,
			Product:      "Cartographus",
			Device:       "Cartographus Server",
			RedirectURI:  securityCfg.PlexAuth.RedirectURI,
			DefaultRoles: securityCfg.PlexAuth.DefaultRoles,
			PlexPassRole: securityCfg.PlexAuth.PlexPassRole,
			PINTimeout:   securityCfg.PlexAuth.Timeout,
		}

		if len(plexConfig.DefaultRoles) == 0 {
			plexConfig.DefaultRoles = []string{"viewer"}
		}
		if plexConfig.PINTimeout == 0 {
			plexConfig.PINTimeout = 30 * time.Second
		}

		pinStore := auth.NewMemoryPlexPINStore()
		router.plexFlow = auth.NewPlexFlow(plexConfig, pinStore)
		logging.Info().Str("clientID", securityCfg.PlexAuth.ClientID).Msg("Plex authentication configured")
	}

	// Create flow handlers
	flowConfig := &auth.FlowHandlersConfig{
		SessionDuration:          securityCfg.OIDC.SessionMaxAge,
		DefaultPostLoginRedirect: "/",
		ErrorRedirectURL:         "/login?error=",
		AllowInsecureCookies:     !securityCfg.OIDC.CookieSecure,
	}
	if flowConfig.SessionDuration == 0 {
		flowConfig.SessionDuration = 24 * time.Hour
	}

	router.flowHandlers = auth.NewFlowHandlers(
		router.oidcFlow,
		router.plexFlow,
		router.sessionStore,
		router.sessionMiddleware,
		flowConfig,
	)

	// Note: OIDC audit logger is set up externally via SetOIDCAuditLogger
	// when the audit store is available (after database initialization).
	// This allows for flexible integration with the audit subsystem.

	// Initialize Casbin enforcer and policy handlers (ADR-0015)
	// The enforcer provides RBAC authorization with sub-millisecond policy evaluation
	enforcerConfig := &authz.EnforcerConfig{
		ModelPath:      securityCfg.Casbin.ModelPath,
		PolicyPath:     securityCfg.Casbin.PolicyPath,
		DefaultRole:    securityCfg.Casbin.DefaultRole,
		AutoReload:     securityCfg.Casbin.AutoReload,
		ReloadInterval: securityCfg.Casbin.ReloadInterval,
		CacheEnabled:   securityCfg.Casbin.CacheEnabled,
		CacheTTL:       securityCfg.Casbin.CacheTTL,
	}

	// Set defaults if not specified
	if enforcerConfig.DefaultRole == "" {
		enforcerConfig.DefaultRole = "viewer"
	}
	if enforcerConfig.ReloadInterval == 0 {
		enforcerConfig.ReloadInterval = 30 * time.Second
	}
	if enforcerConfig.CacheTTL == 0 {
		enforcerConfig.CacheTTL = 5 * time.Minute
	}

	enforcer, err := authz.NewEnforcer(ctx, enforcerConfig)
	if err != nil {
		return fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}
	router.enforcer = enforcer
	router.authzMiddleware = authz.NewMiddleware(enforcer)
	router.policyHandlers = authz.NewPolicyHandlers(enforcer)

	logging.Info().
		Str("model", enforcerConfig.ModelPath).
		Str("policy", enforcerConfig.PolicyPath).
		Bool("cache", enforcerConfig.CacheEnabled).
		Msg("Casbin RBAC authorization initialized")

	logging.Info().Str("mode", authMode).Msg("Zero Trust authentication configured")
	return nil
}

// GetSessionStore returns the session store (for external components that need it).
func (router *Router) GetSessionStore() auth.SessionStore {
	return router.sessionStore
}

// GetSessionMiddleware returns the session middleware (for external components).
func (router *Router) GetSessionMiddleware() *auth.SessionMiddleware {
	return router.sessionMiddleware
}

// GetAuthzMiddleware returns the authorization middleware for protected routes.
// This middleware enforces RBAC policies using Casbin.
func (router *Router) GetAuthzMiddleware() *authz.Middleware {
	return router.authzMiddleware
}

// GetEnforcer returns the Casbin enforcer for direct policy operations.
func (router *Router) GetEnforcer() *authz.Enforcer {
	return router.enforcer
}

// SetOIDCAuditLogger configures the OIDC audit logger for authentication events.
// This should be called after the audit store is initialized (typically in main.go).
// ADR-0015: Zero Trust Authentication (Zitadel Amendment)
// Production-grade audit logging for OIDC operations.
func (router *Router) SetOIDCAuditLogger(logger *auth.OIDCAuditLogger) {
	router.oidcAuditLogger = logger
	if router.flowHandlers != nil {
		router.flowHandlers.SetAuditLogger(logger)
	}
	logging.Info().Msg("OIDC audit logger configured")
}

// GetOIDCAuditLogger returns the OIDC audit logger.
func (router *Router) GetOIDCAuditLogger() *auth.OIDCAuditLogger {
	return router.oidcAuditLogger
}

// Close gracefully shuts down router resources, including the session store and Casbin enforcer.
// This should be called on server shutdown to ensure BadgerDB is closed properly.
func (router *Router) Close() error {
	// Close Casbin enforcer (stops auto-reload and cleans up cache)
	if router.enforcer != nil {
		router.enforcer.Close()
	}

	// Close OIDC state store
	if router.oidcStateStore != nil {
		if err := router.oidcStateStore.Close(); err != nil {
			logging.Warn().Err(err).Msg("Failed to close OIDC state store")
		}
	}

	// Close session store
	if router.sessionStoreFactory != nil {
		if err := router.sessionStoreFactory.Close(); err != nil {
			return fmt.Errorf("close session store: %w", err)
		}
	}
	return nil
}

// wrap applies common middlewares (RequestID, Compression, Prometheus, CORS, RateLimit) to a handler.
// This is used by tests and provides the standard middleware stack for HTTP handlers.
func (router *Router) wrap(handler http.HandlerFunc) http.HandlerFunc {
	return router.middleware.CORS(
		router.middleware.RateLimit(
			middleware.RequestID(
				middleware.Compression(
					middleware.PrometheusMetrics(handler),
				),
			),
		),
	)
}

// serveStaticOrIndex serves static files or index.html for SPA routing
//
//nolint:gocyclo // Complexity is inherent to file type detection and caching logic
func (router *Router) serveStaticOrIndex(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Set Cache-Control headers based on file type
	if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
		// Long cache for versioned assets (1 year)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".svg") || strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".webp") || strings.HasSuffix(path, ".avif") {
		// Cache images for 7 days
		w.Header().Set("Cache-Control", "public, max-age=604800")
	} else if strings.HasSuffix(path, ".json") && path != "/manifest.json" {
		// Cache JSON files for 1 hour (except manifest)
		w.Header().Set("Cache-Control", "public, max-age=3600")
	} else if path == "/" || path == "/index.html" || path == "/manifest.json" {
		// Short cache for HTML and manifest to allow quick updates
		w.Header().Set("Cache-Control", "public, max-age=300")
	}

	fs := http.FileServer(http.Dir("./web/dist"))

	if path == "/" || path == "/index.html" {
		// P1: HTTP/2 Server Push for critical assets
		// Push index.js, styles.css, and manifest.json before serving index.html
		if pusher, ok := w.(http.Pusher); ok {
			// Push main JavaScript bundle (ES module)
			if err := pusher.Push("/index.js", nil); err != nil {
				// Log but don't fail - push is an optimization
				logging.Debug().Err(err).Str("resource", "/index.js").Msg("HTTP/2 push failed")
			}
			// Push main stylesheet
			if err := pusher.Push("/styles.css", nil); err != nil {
				logging.Debug().Err(err).Str("resource", "/styles.css").Msg("HTTP/2 push failed")
			}
			// Push PWA manifest
			if err := pusher.Push("/manifest.json", nil); err != nil {
				logging.Debug().Err(err).Str("resource", "/manifest.json").Msg("HTTP/2 push failed")
			}
		}

		// Render template with CSP nonce if available
		router.renderIndexTemplate(w, r)
		return
	}

	if fileExists(path) {
		fs.ServeHTTP(w, r)
		return
	}

	// SPA fallback - serve index.html for unknown routes
	// Only set default cache if not already set for specific file types
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
	router.renderIndexTemplate(w, r)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := http.Dir("./web/dist").Open(path)
	if err != nil {
		return false
	}
	defer info.Close()

	stat, err := info.Stat()
	if err != nil {
		return false
	}

	return !stat.IsDir()
}

// renderIndexTemplate renders the index.html template with CSP nonce
func (router *Router) renderIndexTemplate(w http.ResponseWriter, r *http.Request) {
	// Get CSP nonce from request context (generated by SecurityHeaders middleware)
	nonce, ok := r.Context().Value(auth.CSPNonceContextKey).(string)
	if !ok {
		logging.Warn().Msg("CSP nonce not found in context, using empty string")
		nonce = ""
	}

	// If template is available, render it with nonce
	if router.indexTemplate != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		data := struct {
			Nonce string
		}{
			Nonce: nonce,
		}

		err := router.indexTemplate.Execute(w, data)
		if err != nil {
			logging.Error().Err(err).Msg("Failed to execute index template")
			// Fallback to static file on error
			http.ServeFile(w, r, "./web/dist/index.html")
		}
		return
	}

	// Fallback to static file if template not available
	http.ServeFile(w, r, "./web/dist/index.html")
}
