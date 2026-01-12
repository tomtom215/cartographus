// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build integration

package testinfra

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// DefaultTautulliImage is the official Tautulli Docker image
	DefaultTautulliImage = "lscr.io/linuxserver/tautulli:latest"

	// DefaultTautulliPort is the default Tautulli web interface port
	DefaultTautulliPort = "8181"

	// DefaultAPIKey is the test API key used in seeded databases
	// Must be a valid 32-character hexadecimal string (Tautulli validates format)
	DefaultAPIKey = "0123456789abcdef0123456789abcdef"
)

// TautulliContainer represents a running Tautulli container for testing.
type TautulliContainer struct {
	testcontainers.Container
	URL    string
	APIKey string
}

// TautulliOption configures the Tautulli container.
type TautulliOption func(*tautulliConfig)

type tautulliConfig struct {
	image        string
	seedDBPath   string
	apiKey       string
	startTimeout time.Duration
}

// WithTautulliImage sets a custom Tautulli Docker image.
func WithTautulliImage(image string) TautulliOption {
	return func(c *tautulliConfig) {
		c.image = image
	}
}

// WithSeedDatabase configures the container to use a pre-seeded SQLite database.
// The database file will be copied into the container at startup.
func WithSeedDatabase(dbPath string) TautulliOption {
	return func(c *tautulliConfig) {
		c.seedDBPath = dbPath
	}
}

// WithAPIKey sets a custom API key for the Tautulli instance.
func WithAPIKey(apiKey string) TautulliOption {
	return func(c *tautulliConfig) {
		c.apiKey = apiKey
	}
}

// WithStartTimeout sets the timeout for waiting for Tautulli to start.
func WithStartTimeout(timeout time.Duration) TautulliOption {
	return func(c *tautulliConfig) {
		c.startTimeout = timeout
	}
}

// NewTautulliContainer creates and starts a new Tautulli container for testing.
//
// Example:
//
//	ctx := context.Background()
//	tautulli, err := NewTautulliContainer(ctx)
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer tautulli.Terminate(ctx)
//
//	// Use tautulli.URL for API calls
//	resp, err := http.Get(tautulli.URL + "/api/v2?apikey=" + tautulli.APIKey + "&cmd=get_activity")
func NewTautulliContainer(ctx context.Context, opts ...TautulliOption) (*TautulliContainer, error) {
	cfg := &tautulliConfig{
		image:        DefaultTautulliImage,
		apiKey:       DefaultAPIKey,
		startTimeout: 60 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Build container request
	req := testcontainers.ContainerRequest{
		Image:        cfg.image,
		ExposedPorts: []string{DefaultTautulliPort + "/tcp"},
		Env: map[string]string{
			"PUID": "1000",
			"PGID": "1000",
			"TZ":   "UTC",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(DefaultTautulliPort+"/tcp"),
			wait.ForHTTP("/").WithPort(DefaultTautulliPort+"/tcp"),
		).WithStartupTimeout(cfg.startTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("create tautulli container: %w", err)
	}

	// Copy seed database if provided
	if cfg.seedDBPath != "" {
		if err := copySeedDatabase(ctx, container, cfg.seedDBPath); err != nil {
			container.Terminate(ctx) //nolint:errcheck
			return nil, fmt.Errorf("copy seed database: %w", err)
		}
	}

	// Configure API key in Tautulli's config.ini
	if err := configureAPIKey(ctx, container, cfg.apiKey); err != nil {
		container.Terminate(ctx) //nolint:errcheck
		return nil, fmt.Errorf("configure API key: %w", err)
	}

	// Get container host and port
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx) //nolint:errcheck
		return nil, fmt.Errorf("get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, DefaultTautulliPort)
	if err != nil {
		container.Terminate(ctx) //nolint:errcheck
		return nil, fmt.Errorf("get mapped port: %w", err)
	}

	return &TautulliContainer{
		Container: container,
		URL:       fmt.Sprintf("http://%s:%s", host, port.Port()),
		APIKey:    cfg.apiKey,
	}, nil
}

// copySeedDatabase copies a seed database file into the Tautulli container.
func copySeedDatabase(ctx context.Context, container testcontainers.Container, dbPath string) error {
	// Tautulli stores its database in /config/tautulli.db
	targetPath := "/config/tautulli.db"

	err := container.CopyFileToContainer(ctx, dbPath, targetPath, 0644)
	if err != nil {
		return fmt.Errorf("copy file to container: %w", err)
	}

	return nil
}

// configureAPIKey sets the API key in Tautulli's config.ini file.
// Tautulli generates a random API key on first startup, so we need to
// replace it with our known test API key and restart Tautulli.
func configureAPIKey(ctx context.Context, container testcontainers.Container, apiKey string) error {
	// Tautulli stores config in /config/config.ini
	// We need to update the api_key setting in the [General] section

	// First, check if config.ini exists (Tautulli should have created it on startup)
	code, _, err := container.Exec(ctx, []string{"test", "-f", "/config/config.ini"})
	if err != nil || code != 0 {
		// Config doesn't exist yet - create a minimal one with our API key
		configContent := fmt.Sprintf("[General]\napi_key = %s\nfirst_run_complete = 1\n", apiKey)
		err = container.CopyToContainer(ctx, []byte(configContent), "/config/config.ini", 0644)
		if err != nil {
			return fmt.Errorf("create config.ini: %w", err)
		}
		return nil
	}

	// Config exists - replace the api_key line using sed
	sedCmd := fmt.Sprintf("sed -i 's/^api_key = .*/api_key = %s/' /config/config.ini", apiKey)
	code, outputReader, err := container.Exec(ctx, []string{"sh", "-c", sedCmd})
	if err != nil {
		return fmt.Errorf("exec sed: %w", err)
	}
	if code != 0 {
		output, _ := io.ReadAll(outputReader)
		return fmt.Errorf("sed failed with code %d: %s", code, string(output))
	}

	// Tautulli has already started with its own API key - we need to restart it
	// to pick up the new config. Use pkill to stop Tautulli, s6 will restart it.
	code, _, err = container.Exec(ctx, []string{"pkill", "-f", "Tautulli.py"})
	if err != nil {
		return fmt.Errorf("exec pkill: %w", err)
	}
	// pkill returns 0 if processes matched, 1 if no match - both are OK

	// Wait for Tautulli to restart and become ready
	// Poll the HTTP endpoint to verify Tautulli is back up
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		// Check if Tautulli's web interface is responding
		code, _, err = container.Exec(ctx, []string{"curl", "-sf", "http://localhost:8181/"})
		if err == nil && code == 0 {
			return nil // Tautulli is back up
		}
	}

	return fmt.Errorf("tautulli did not restart within 30 seconds")
}

// GetDefaultSeedDBPath returns the path to the default seed database.
// This looks for seed.db in the testdata/tautulli directory.
func GetDefaultSeedDBPath() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get caller info")
	}

	// Navigate from internal/testinfra to testdata/tautulli
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	seedPath := filepath.Join(projectRoot, "testdata", "tautulli", "seed.db")

	return seedPath, nil
}

// Terminate stops and removes the Tautulli container.
func (c *TautulliContainer) Terminate(ctx context.Context) error {
	return c.Container.Terminate(ctx)
}

// GetAPIEndpoint returns the full URL for a Tautulli API endpoint.
func (c *TautulliContainer) GetAPIEndpoint(cmd string) string {
	return fmt.Sprintf("%s/api/v2?apikey=%s&cmd=%s", c.URL, c.APIKey, cmd)
}

// Logs returns the container logs for debugging.
func (c *TautulliContainer) Logs(ctx context.Context) (string, error) {
	reader, err := c.Container.Logs(ctx)
	if err != nil {
		return "", fmt.Errorf("get logs: %w", err)
	}
	defer reader.Close()

	var logs []byte
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			logs = append(logs, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return string(logs), nil
}
