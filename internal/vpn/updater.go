// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

const (
	// DefaultGluetunURL is the raw GitHub URL for gluetun's servers.json.
	DefaultGluetunURL = "https://raw.githubusercontent.com/qdm12/gluetun/master/internal/storage/servers.json"

	// DefaultUpdateInterval is how often to check for updates.
	DefaultUpdateInterval = 24 * time.Hour

	// DefaultHTTPTimeout is the timeout for HTTP requests.
	DefaultHTTPTimeout = 60 * time.Second
)

// UpdaterConfig configures the VPN data updater.
type UpdaterConfig struct {
	// Enabled controls whether automatic updates are active.
	Enabled bool `json:"enabled"`

	// SourceURL is the URL to fetch VPN data from.
	SourceURL string `json:"source_url"`

	// UpdateInterval is how often to check for updates.
	UpdateInterval time.Duration `json:"update_interval"`

	// HTTPTimeout is the timeout for HTTP requests.
	HTTPTimeout time.Duration `json:"http_timeout"`

	// RetryAttempts is the number of retry attempts on failure.
	RetryAttempts int `json:"retry_attempts"`

	// RetryDelay is the initial delay between retries (doubles each attempt).
	RetryDelay time.Duration `json:"retry_delay"`
}

// DefaultUpdaterConfig returns sensible defaults for the updater.
func DefaultUpdaterConfig() *UpdaterConfig {
	return &UpdaterConfig{
		Enabled:        false, // Disabled by default - user must opt-in
		SourceURL:      DefaultGluetunURL,
		UpdateInterval: DefaultUpdateInterval,
		HTTPTimeout:    DefaultHTTPTimeout,
		RetryAttempts:  3,
		RetryDelay:     5 * time.Second,
	}
}

// UpdateStatus tracks the status of VPN data updates.
type UpdateStatus struct {
	// LastUpdateAttempt is when we last tried to update.
	LastUpdateAttempt time.Time `json:"last_update_attempt"`

	// LastSuccessfulUpdate is when we last successfully updated.
	LastSuccessfulUpdate time.Time `json:"last_successful_update"`

	// LastError is the last error encountered (empty if none).
	LastError string `json:"last_error,omitempty"`

	// SourceURL is the URL data was fetched from.
	SourceURL string `json:"source_url"`

	// DataHash is the SHA256 hash of the last imported data.
	DataHash string `json:"data_hash"`

	// Version tracks provider versions from the source.
	ProviderVersions map[string]int `json:"provider_versions,omitempty"`

	// ImportResult contains details from the last import.
	LastImportResult *ImportResult `json:"last_import_result,omitempty"`

	// NextScheduledUpdate is when the next automatic update is scheduled.
	NextScheduledUpdate time.Time `json:"next_scheduled_update,omitempty"`

	// IsUpdating indicates if an update is currently in progress.
	IsUpdating bool `json:"is_updating"`
}

// Updater handles automatic updates of VPN data from remote sources.
type Updater struct {
	config  *UpdaterConfig
	service *Service
	store   *DuckDBStore
	client  *http.Client
	status  UpdateStatus

	stopChan chan struct{}
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// NewUpdater creates a new VPN data updater.
func NewUpdater(service *Service, config *UpdaterConfig) *Updater {
	if config == nil {
		config = DefaultUpdaterConfig()
	}

	return &Updater{
		config:  config,
		service: service,
		client: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		status: UpdateStatus{
			SourceURL:        config.SourceURL,
			ProviderVersions: make(map[string]int),
		},
		stopChan: make(chan struct{}),
	}
}

// NewUpdaterWithStore creates a new VPN data updater with database persistence.
func NewUpdaterWithStore(service *Service, store *DuckDBStore, config *UpdaterConfig) *Updater {
	u := NewUpdater(service, config)
	u.store = store
	return u
}

// LoadStatus loads persisted update status from the database.
func (u *Updater) LoadStatus(ctx context.Context) error {
	if u.store == nil {
		return nil
	}

	status, err := u.store.LoadUpdateStatus(ctx)
	if err != nil {
		return err
	}

	if status != nil {
		u.mu.Lock()
		// Only load non-zero values
		if !status.LastUpdateAttempt.IsZero() {
			u.status.LastUpdateAttempt = status.LastUpdateAttempt
		}
		if !status.LastSuccessfulUpdate.IsZero() {
			u.status.LastSuccessfulUpdate = status.LastSuccessfulUpdate
		}
		if status.LastError != "" {
			u.status.LastError = status.LastError
		}
		if status.SourceURL != "" {
			u.status.SourceURL = status.SourceURL
		}
		if status.DataHash != "" {
			u.status.DataHash = status.DataHash
		}
		u.mu.Unlock()
	}

	return nil
}

// persistStatus saves the current status to the database.
func (u *Updater) persistStatus(ctx context.Context) {
	if u.store == nil {
		return
	}

	u.mu.RLock()
	status := u.status
	u.mu.RUnlock()

	if err := u.store.SaveUpdateStatus(ctx, &status); err != nil {
		logging.Error().Err(err).Msg("Failed to persist VPN updater status")
	}
}

// Start begins the automatic update routine.
func (u *Updater) Start(ctx context.Context) error {
	if !u.config.Enabled {
		logging.Info().Msg("VPN automatic updates disabled")
		return nil
	}

	logging.Info().Dur("interval", u.config.UpdateInterval).Msg("Starting VPN automatic updates")

	u.wg.Add(1)
	go u.updateLoop(ctx)

	return nil
}

// Stop stops the automatic update routine.
func (u *Updater) Stop() {
	close(u.stopChan)
	u.wg.Wait()
	logging.Info().Msg("VPN updater stopped")
}

// updateLoop runs the periodic update check.
func (u *Updater) updateLoop(ctx context.Context) {
	defer u.wg.Done()

	// Run initial update
	if err := u.UpdateNow(ctx); err != nil {
		logging.Warn().Err(err).Msg("VPN initial update failed")
	}

	ticker := time.NewTicker(u.config.UpdateInterval)
	defer ticker.Stop()

	for {
		u.mu.Lock()
		u.status.NextScheduledUpdate = time.Now().Add(u.config.UpdateInterval)
		u.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-u.stopChan:
			return
		case <-ticker.C:
			if err := u.UpdateNow(ctx); err != nil {
				logging.Warn().Err(err).Msg("VPN scheduled update failed")
			}
		}
	}
}

// UpdateNow triggers an immediate update from the configured source.
func (u *Updater) UpdateNow(ctx context.Context) error {
	return u.UpdateFromURL(ctx, u.config.SourceURL)
}

// UpdateFromURL fetches and imports VPN data from a specific URL.
func (u *Updater) UpdateFromURL(ctx context.Context, url string) error {
	u.mu.Lock()
	if u.status.IsUpdating {
		u.mu.Unlock()
		return fmt.Errorf("update already in progress")
	}
	u.status.IsUpdating = true
	u.status.LastUpdateAttempt = time.Now()
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		u.status.IsUpdating = false
		u.mu.Unlock()
	}()

	logging.Info().Str("url", url).Msg("Fetching VPN data")

	// Fetch with retries
	data, err := u.fetchWithRetry(ctx, url)
	if err != nil {
		u.mu.Lock()
		u.status.LastError = err.Error()
		u.mu.Unlock()
		return fmt.Errorf("failed to fetch VPN data: %w", err)
	}

	// Calculate hash to detect changes
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	u.mu.RLock()
	previousHash := u.status.DataHash
	u.mu.RUnlock()

	if hashStr == previousHash {
		logging.Info().Str("hash", hashStr[:16]).Msg("VPN data unchanged")
		u.mu.Lock()
		u.status.LastSuccessfulUpdate = time.Now()
		u.status.LastError = ""
		u.mu.Unlock()
		u.persistStatus(ctx)
		return nil
	}

	// Import the data
	result, err := u.service.ImportFromBytes(ctx, data)
	if err != nil {
		u.mu.Lock()
		u.status.LastError = err.Error()
		u.mu.Unlock()
		return fmt.Errorf("failed to import VPN data: %w", err)
	}

	// Extract provider versions
	providerVersions := u.extractProviderVersions(data)

	// Update status
	u.mu.Lock()
	u.status.LastSuccessfulUpdate = time.Now()
	u.status.LastError = ""
	u.status.SourceURL = url
	u.status.DataHash = hashStr
	u.status.ProviderVersions = providerVersions
	u.status.LastImportResult = result
	u.mu.Unlock()

	// Persist status to database
	u.persistStatus(ctx)

	logging.Info().
		Int("providers", result.ProvidersImported).
		Int("servers", result.ServersImported).
		Int("ips", result.IPsImported).
		Str("hash", hashStr[:16]).
		Msg("VPN data successfully updated")

	return nil
}

// fetchWithRetry fetches data from URL with exponential backoff retries.
func (u *Updater) fetchWithRetry(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	delay := u.config.RetryDelay

	for attempt := 0; attempt <= u.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			logging.Info().
				Int("attempt", attempt).
				Int("max_attempts", u.config.RetryAttempts).
				Dur("delay", delay).
				Msg("Retrying VPN data fetch")
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2 // Exponential backoff
		}

		data, err := u.fetch(ctx, url)
		if err == nil {
			return data, nil
		}
		lastErr = err
		logging.Warn().Err(err).Int("attempt", attempt+1).Msg("VPN fetch attempt failed")
	}

	return nil, fmt.Errorf("all %d attempts failed: %w", u.config.RetryAttempts+1, lastErr)
}

// fetch performs a single HTTP GET request.
func (u *Updater) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Cartographus-VPN-Updater/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Limit response size to 50MB to prevent memory issues
	limitedReader := io.LimitReader(resp.Body, 50*1024*1024)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// extractProviderVersions parses the JSON to get provider versions.
func (u *Updater) extractProviderVersions(data []byte) map[string]int {
	var gluetunData map[string]json.RawMessage
	if err := json.Unmarshal(data, &gluetunData); err != nil {
		return nil
	}

	versions := make(map[string]int)
	for provider, raw := range gluetunData {
		if provider == "version" {
			continue
		}
		var providerData struct {
			Version int `json:"version"`
		}
		if err := json.Unmarshal(raw, &providerData); err == nil {
			versions[provider] = providerData.Version
		}
	}
	return versions
}

// GetStatus returns the current update status.
func (u *Updater) GetStatus() UpdateStatus {
	u.mu.RLock()
	defer u.mu.RUnlock()

	// Return a copy
	status := u.status
	status.ProviderVersions = make(map[string]int)
	for k, v := range u.status.ProviderVersions {
		status.ProviderVersions[k] = v
	}
	return status
}

// SetEnabled enables or disables automatic updates.
func (u *Updater) SetEnabled(enabled bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.config.Enabled = enabled
}

// IsEnabled returns whether automatic updates are enabled.
func (u *Updater) IsEnabled() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.config.Enabled
}

// SetUpdateInterval changes the update interval.
func (u *Updater) SetUpdateInterval(interval time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.config.UpdateInterval = interval
}

// GetConfig returns the current updater configuration.
func (u *Updater) GetConfig() UpdaterConfig {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return *u.config
}
