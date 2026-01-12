// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Plex OAuth 2.0 PKCE Authentication Flow
 *
 * Implements client-side OAuth flow for secure Plex Media Server authentication.
 * This module handles the complete OAuth lifecycle:
 * - Initiating OAuth flow (redirecting to Plex)
 * - Handling OAuth callback (exchanging code for token)
 * - Token storage and refresh
 * - Token revocation (logout)
 *
 * Security:
 * - PKCE code_verifier stored in HTTP-only cookie (backend-managed)
 * - CSRF protection via state parameter validation
 * - Automatic token refresh before expiration
 * - Secure token storage with expiration tracking
 *
 * @module plex-oauth
 */

import { queuedFetch } from '../api/client';
import { createLogger } from '../logger';
import { SafeStorage } from '../utils/SafeStorage';

const logger = createLogger('OAuth');

/**
 * OAuth token response from backend
 */
export interface PlexOAuthToken {
    access_token: string;
    token_type: string;
    expires_in: number;
    expires_at: number;
    refresh_token?: string;
}

/**
 * OAuth start response from /api/v1/auth/plex/start
 */
interface OAuthStartResponse {
    authorization_url: string;
    state: string;
}

/**
 * Token storage keys
 */
const STORAGE_KEYS = {
    ACCESS_TOKEN: 'plex_access_token',
    REFRESH_TOKEN: 'plex_refresh_token',
    EXPIRES_AT: 'plex_token_expires_at',
    STATE: 'plex_oauth_state',
} as const;

/**
 * Initiates Plex OAuth 2.0 PKCE flow.
 *
 * Workflow:
 * 1. Calls GET /api/v1/auth/plex/start
 * 2. Backend generates PKCE challenge and stores code_verifier in cookie
 * 3. Receives authorization URL with embedded code_challenge
 * 4. Stores state in sessionStorage for CSRF validation
 * 5. Redirects user to Plex authorization page
 *
 * @throws {Error} If OAuth start request fails
 *
 * Example:
 *   button.addEventListener('click', () => {
 *       initiatePlexOAuth().catch(err => console.error('OAuth failed:', err));
 *   });
 */
export async function initiatePlexOAuth(): Promise<void> {
    try {
        // Call backend to generate OAuth URL with PKCE
        const response = await queuedFetch('/api/v1/auth/plex/start', {
            method: 'GET',
            credentials: 'include', // Send cookies (backend will set OAuth state cookie)
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`OAuth start failed: ${response.status} ${errorText}`);
        }

        const data: { status: string; data: OAuthStartResponse } = await response.json();

        if (data.status !== 'success') {
            throw new Error('OAuth start returned non-success status');
        }

        const { authorization_url, state } = data.data;

        // Store state in sessionStorage for CSRF validation on callback
        sessionStorage.setItem(STORAGE_KEYS.STATE, state);

        // Redirect to Plex authorization page
        logger.debug('[OAuth] Redirecting to Plex authorization:', authorization_url);
        window.location.href = authorization_url;
    } catch (error) {
        logger.error('[OAuth] Failed to initiate OAuth flow:', error);
        throw error;
    }
}

/**
 * Handles OAuth callback after user authorizes on Plex.
 *
 * Workflow:
 * 1. Parses ?code= and ?state= from callback URL
 * 2. Validates state parameter matches stored value (CSRF protection)
 * 3. Calls GET /api/v1/auth/plex/callback with code and state
 * 4. Backend validates state, retrieves code_verifier from cookie, exchanges for token
 * 5. Stores access_token and refresh_token in localStorage
 * 6. Redirects to application home page
 *
 * @throws {Error} If state validation fails or token exchange fails
 *
 * Example:
 *   // In your router or main app initialization
 *   if (window.location.pathname === '/api/v1/auth/plex/callback') {
 *       handleOAuthCallback().catch(err => showError(err.message));
 *   }
 */
export async function handleOAuthCallback(): Promise<void> {
    try {
        // Parse callback URL parameters
        const params = new URLSearchParams(window.location.search);
        const code = params.get('code');
        const state = params.get('state');

        if (!code) {
            throw new Error('Missing authorization code in callback');
        }

        if (!state) {
            throw new Error('Missing state parameter in callback');
        }

        // Retrieve stored state for CSRF validation
        const storedState = sessionStorage.getItem(STORAGE_KEYS.STATE);
        if (!storedState) {
            throw new Error('No stored state found (possible session timeout or CSRF attack)');
        }

        // Validate state parameter (CSRF protection)
        if (state !== storedState) {
            throw new Error('State parameter mismatch (possible CSRF attack)');
        }

        // Clear state from storage (one-time use)
        sessionStorage.removeItem(STORAGE_KEYS.STATE);

        logger.debug('[OAuth] Exchanging authorization code for access token...');

        // Exchange code for token (backend handles code_verifier from cookie)
        const response = await queuedFetch(`/api/v1/auth/plex/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`, {
            method: 'GET',
            credentials: 'include', // Send cookies (backend needs OAuth state cookie)
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Token exchange failed: ${response.status} ${errorText}`);
        }

        const data: { status: string; data: PlexOAuthToken } = await response.json();

        if (data.status !== 'success') {
            throw new Error('Token exchange returned non-success status');
        }

        const token = data.data;

        // Store tokens in localStorage
        storeToken(token);

        logger.debug('[OAuth] Successfully authenticated with Plex');
        logger.debug('[OAuth] Token expires at:', new Date(token.expires_at * 1000).toISOString());

        // Redirect to home page (remove callback URL from history)
        window.location.href = '/';
    } catch (error) {
        logger.error('[OAuth] Callback handling failed:', error);
        throw error;
    }
}

/**
 * Stores OAuth token using SafeStorage.
 *
 * Stores:
 * - access_token: Bearer token for API calls
 * - refresh_token: Token for obtaining new access tokens (if provided)
 * - expires_at: Unix timestamp when token expires
 *
 * Uses SafeStorage for robust persistence with private browsing fallback.
 *
 * @param token - OAuth token response from backend
 *
 * @internal
 */
function storeToken(token: PlexOAuthToken): void {
    SafeStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, token.access_token);
    SafeStorage.setItem(STORAGE_KEYS.EXPIRES_AT, token.expires_at.toString());

    if (token.refresh_token) {
        SafeStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, token.refresh_token);
    }

    logger.debug('[OAuth] Token stored in SafeStorage');
}

/**
 * Gets the stored Plex access token.
 *
 * Returns null if:
 * - Token is not stored
 * - Token has expired (automatic refresh not implemented in getter)
 *
 * @returns Access token or null if not available
 *
 * Example:
 *   const token = getAccessToken();
 *   if (token) {
 *       fetch('/api/v1/protected', {
 *           headers: { 'Authorization': `Bearer ${token}` }
 *       });
 *   }
 */
export function getAccessToken(): string | null {
    const token = SafeStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
    const expiresAt = SafeStorage.getItem(STORAGE_KEYS.EXPIRES_AT);

    if (!token || !expiresAt) {
        return null;
    }

    // Check if token is expired
    const expirationTime = parseInt(expiresAt, 10);
    const currentTime = Math.floor(Date.now() / 1000);

    if (currentTime >= expirationTime) {
        logger.debug('[OAuth] Access token expired');
        return null;
    }

    return token;
}

/**
 * Checks if user has a valid OAuth token.
 *
 * Returns true if:
 * - Access token exists in localStorage
 * - Token has not expired
 *
 * @returns True if authenticated, false otherwise
 *
 * Example:
 *   if (!isAuthenticated()) {
 *       showLoginButton();
 *   }
 */
export function isAuthenticated(): boolean {
    return getAccessToken() !== null;
}

/**
 * Refreshes the Plex access token using the refresh token.
 *
 * Workflow:
 * 1. Retrieves refresh_token from localStorage
 * 2. Calls POST /api/v1/auth/plex/refresh with refresh_token
 * 3. Stores new access_token and refresh_token
 * 4. Returns new access token
 *
 * Use Case:
 * - Called automatically when access token is close to expiration
 * - Called when API request returns 401 Unauthorized
 *
 * @returns New access token
 * @throws {Error} If refresh fails or refresh token is missing
 *
 * Example:
 *   try {
 *       const newToken = await refreshAccessToken();
 *       // Retry failed request with new token
 *   } catch (error) {
 *       // Token refresh failed, redirect to login
 *       await revokeToken();
 *       window.location.href = '/login';
 *   }
 */
export async function refreshAccessToken(): Promise<string> {
    const refreshToken = SafeStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);

    if (!refreshToken) {
        throw new Error('No refresh token available');
    }

    logger.debug('[OAuth] Refreshing access token...');

    try {
        const response = await queuedFetch('/api/v1/auth/plex/refresh', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ refresh_token: refreshToken }),
            credentials: 'include',
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Token refresh failed: ${response.status} ${errorText}`);
        }

        const data: { status: string; data: PlexOAuthToken } = await response.json();

        if (data.status !== 'success') {
            throw new Error('Token refresh returned non-success status');
        }

        const token = data.data;

        // Store new tokens
        storeToken(token);

        logger.debug('[OAuth] Token refreshed successfully');
        logger.debug('[OAuth] New token expires at:', new Date(token.expires_at * 1000).toISOString());

        return token.access_token;
    } catch (error) {
        logger.error('[OAuth] Token refresh failed:', error);
        // Clear invalid tokens
        clearTokens();
        throw error;
    }
}

/**
 * Revokes the Plex access token and logs out user.
 *
 * Workflow:
 * 1. Retrieves access_token from localStorage
 * 2. Calls POST /api/v1/auth/plex/revoke with access_token
 * 3. Backend revokes token with Plex (if endpoint available)
 * 4. Clears all tokens from localStorage
 * 5. Clears session cookies
 *
 * @throws {Error} If revocation request fails (tokens still cleared)
 *
 * Example:
 *   logoutButton.addEventListener('click', async () => {
 *       await revokeToken();
 *       window.location.href = '/login';
 *   });
 */
export async function revokeToken(): Promise<void> {
    const accessToken = SafeStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);

    // Clear tokens from SafeStorage regardless of revocation success
    clearTokens();

    if (!accessToken) {
        logger.debug('[OAuth] No token to revoke');
        return;
    }

    logger.debug('[OAuth] Revoking access token...');

    try {
        const response = await queuedFetch('/api/v1/auth/plex/revoke', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ access_token: accessToken }),
            credentials: 'include',
        });

        if (!response.ok) {
            const errorText = await response.text();
            logger.warn('[OAuth] Token revocation failed:', response.status, errorText);
            // Don't throw - tokens are already cleared
        } else {
            logger.debug('[OAuth] Token revoked successfully');
        }
    } catch (error) {
        logger.error('[OAuth] Token revocation request failed:', error);
        // Don't throw - tokens are already cleared
    }
}

/**
 * Clears all OAuth tokens from SafeStorage.
 *
 * Use Case:
 * - Called after successful revocation
 * - Called when refresh fails (invalid tokens)
 * - Called on logout
 *
 * @internal
 */
function clearTokens(): void {
    SafeStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
    SafeStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    SafeStorage.removeItem(STORAGE_KEYS.EXPIRES_AT);
    logger.debug('[OAuth] Tokens cleared from SafeStorage');
}

/**
 * Schedules automatic token refresh before expiration.
 *
 * Workflow:
 * 1. Checks if token exists and has expiration time
 * 2. Calculates time until expiration
 * 3. Schedules refresh for 5 minutes before expiration
 * 4. Automatically refreshes token when timer fires
 *
 * Use Case:
 * - Called on app initialization if user is authenticated
 * - Ensures seamless user experience without re-login
 *
 * @returns Timeout ID for cancellation (if needed)
 *
 * Example:
 *   // In app initialization
 *   if (isAuthenticated()) {
 *       scheduleTokenRefresh();
 *   }
 */
export function scheduleTokenRefresh(): number | null {
    const expiresAt = SafeStorage.getItem(STORAGE_KEYS.EXPIRES_AT);

    if (!expiresAt) {
        logger.debug('[OAuth] No expiration time found, skipping refresh schedule');
        return null;
    }

    const expirationTime = parseInt(expiresAt, 10) * 1000; // Convert to milliseconds
    const currentTime = Date.now();
    const timeUntilExpiration = expirationTime - currentTime;

    // Refresh 5 minutes before expiration
    const refreshBuffer = 5 * 60 * 1000; // 5 minutes in milliseconds
    const timeUntilRefresh = timeUntilExpiration - refreshBuffer;

    if (timeUntilRefresh <= 0) {
        logger.debug('[OAuth] Token already expired or expiring soon, refreshing immediately');
        refreshAccessToken().catch(error => {
            logger.error('[OAuth] Auto-refresh failed:', error);
            // Token is invalid, redirect to login
            window.location.href = '/login';
        });
        return null;
    }

    logger.debug(`[OAuth] Scheduling token refresh in ${Math.round(timeUntilRefresh / 1000 / 60)} minutes`);

    const timeoutId = window.setTimeout(() => {
        logger.debug('[OAuth] Auto-refreshing token...');
        refreshAccessToken().catch(error => {
            logger.error('[OAuth] Auto-refresh failed:', error);
            // Token refresh failed, redirect to login
            window.location.href = '/login';
        });
    }, timeUntilRefresh);

    return timeoutId;
}

/**
 * Initializes OAuth system on app startup.
 *
 * Performs:
 * 1. Check if current page is OAuth callback
 * 2. Handle callback if present
 * 3. Schedule automatic token refresh if authenticated
 *
 * @returns Promise that resolves when initialization is complete
 *
 * Example:
 *   // In main app initialization (index.ts)
 *   document.addEventListener('DOMContentLoaded', async () => {
 *       await initializeOAuth();
 *       // Continue with app initialization
 *   });
 */
export async function initializeOAuth(): Promise<void> {
    // Check if this is an OAuth callback
    if (window.location.pathname === '/api/v1/auth/plex/callback' || window.location.search.includes('code=')) {
        logger.debug('[OAuth] Detected OAuth callback, handling...');
        try {
            await handleOAuthCallback();
        } catch (error) {
            logger.error('[OAuth] Callback handling failed:', error);
            // Show error to user
            alert(`OAuth authentication failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
            // Redirect to home/login
            window.location.href = '/';
        }
        return;
    }

    // Schedule token refresh if user is authenticated
    if (isAuthenticated()) {
        logger.debug('[OAuth] User authenticated, scheduling token refresh');
        scheduleTokenRefresh();
    } else {
        logger.debug('[OAuth] User not authenticated');
    }
}
