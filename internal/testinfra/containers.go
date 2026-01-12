// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build integration

package testinfra

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

// SkipIfNoDocker skips the test if Docker is not available.
// This allows tests to run gracefully in environments without Docker.
func SkipIfNoDocker(t *testing.T) {
	t.Helper()

	if !IsDockerAvailable() {
		t.Skip("Skipping test: Docker not available")
	}
}

// IsDockerAvailable checks if Docker daemon is running and accessible.
func IsDockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	return cmd.Run() == nil
}

// ContainerLogger adapts testcontainers logging to testing.T.
type ContainerLogger struct {
	t *testing.T
}

// NewContainerLogger creates a logger that outputs to testing.T.
func NewContainerLogger(t *testing.T) *ContainerLogger {
	return &ContainerLogger{t: t}
}

// Printf implements testcontainers.Logging interface.
func (l *ContainerLogger) Printf(format string, v ...interface{}) {
	l.t.Logf(format, v...)
}

// WaitForReady waits for a container to be ready with custom polling.
// This is useful when the default wait strategy isn't sufficient.
func WaitForReady(ctx context.Context, container testcontainers.Container, check func() bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if check() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			// Continue polling
		}
	}

	return context.DeadlineExceeded
}

// CleanupContainer is a helper for deferred container cleanup that logs errors.
func CleanupContainer(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()

	if container != nil {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Warning: failed to terminate container: %v", err)
		}
	}
}

// ContainerInfo holds information about a running container for debugging.
type ContainerInfo struct {
	ID          string
	Name        string
	Image       string
	Host        string
	Ports       map[string]string
	State       string
	StartedAt   string // ISO8601 timestamp from Docker API
	Environment map[string]string
}

// GetContainerInfo retrieves debugging information about a container.
func GetContainerInfo(ctx context.Context, container testcontainers.Container) (*ContainerInfo, error) {
	state, err := container.State(ctx)
	if err != nil {
		return nil, err
	}

	host, _ := container.Host(ctx)
	ports, _ := container.Ports(ctx)

	portMap := make(map[string]string)
	for port, bindings := range ports {
		if len(bindings) > 0 {
			portMap[string(port)] = bindings[0].HostPort
		}
	}

	return &ContainerInfo{
		ID:        container.GetContainerID()[:12],
		Host:      host,
		Ports:     portMap,
		State:     state.Status,
		StartedAt: state.StartedAt,
	}, nil
}
