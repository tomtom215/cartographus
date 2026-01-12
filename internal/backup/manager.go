// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
manager.go - Core Backup Manager

This file contains the core backup manager struct and lifecycle methods for
managing database and configuration backups of the Cartographus application.

Manager Responsibilities:
  - Backup creation orchestration
  - Metadata storage and retrieval
  - Scheduler lifecycle management
  - Callback notification for backup events

Metadata Storage:
Backup metadata is stored in metadata.json alongside backup files, containing:
  - List of all backups with their status and details
  - Scheduling information (last/next scheduled backup)
  - Retention policy configuration
  - Aggregate statistics

Thread Safety:
All metadata operations are protected by sync.RWMutex for concurrent access.
The scheduler runs in a separate goroutine and can be started/stopped safely.
*/

//nolint:staticcheck // File documentation, not package doc
package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// AppVersion is set at build time
var AppVersion = "dev"

// DatabaseInterface defines the database operations needed for backup
type DatabaseInterface interface {
	// GetDatabasePath returns the path to the database file
	GetDatabasePath() string
	// GetRecordCounts returns the count of records in main tables
	GetRecordCounts(ctx context.Context) (playbacks int64, geolocations int64, err error)
	// Close closes the database connection
	Close() error
	// Checkpoint forces a WAL checkpoint for consistent backup
	Checkpoint(ctx context.Context) error
}

// Manager handles backup and restore operations
type Manager struct {
	cfg *Config
	db  DatabaseInterface

	// Metadata storage
	metadataFile string
	metadata     *MetadataStore
	metadataMu   sync.RWMutex

	// Scheduler
	schedulerStop chan struct{}
	schedulerWg   sync.WaitGroup
	running       bool
	runningMu     sync.Mutex

	// Callbacks
	onBackupComplete func(backup *Backup)
	onRestoreStart   func(backupID string)
}

// MetadataStore holds all backup metadata
type MetadataStore struct {
	Backups       []*Backup       `json:"backups"`
	LastScheduled *time.Time      `json:"last_scheduled,omitempty"`
	NextScheduled *time.Time      `json:"next_scheduled,omitempty"`
	Stats         *BackupStats    `json:"stats,omitempty"`
	Retention     RetentionPolicy `json:"retention"`
}

// NewManager creates a new backup manager
func NewManager(cfg *Config, db DatabaseInterface) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("backup configuration is required")
	}

	// Create backup directory
	if cfg.Enabled {
		if err := cfg.EnsureBackupDir(); err != nil {
			return nil, err
		}
	}

	m := &Manager{
		cfg:           cfg,
		db:            db,
		metadataFile:  filepath.Join(cfg.BackupDir, "metadata.json"),
		schedulerStop: make(chan struct{}),
	}

	// Load existing metadata
	if err := m.loadMetadata(); err != nil {
		// Initialize empty metadata if file doesn't exist
		m.metadata = &MetadataStore{
			Backups:   make([]*Backup, 0),
			Retention: cfg.Retention,
		}
	}

	return m, nil
}

// Start begins the backup scheduler
func (m *Manager) Start(ctx context.Context) error {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()

	if m.running {
		return fmt.Errorf("backup manager is already running")
	}

	if !m.cfg.Enabled {
		return nil // Backups disabled, nothing to do
	}

	if !m.cfg.Schedule.Enabled {
		return nil // Scheduling disabled, nothing to do
	}

	m.running = true
	m.schedulerStop = make(chan struct{})

	m.schedulerWg.Add(1)
	go m.runScheduler(ctx)

	return nil
}

// Stop stops the backup scheduler
func (m *Manager) Stop() error {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()

	if !m.running {
		return nil
	}

	close(m.schedulerStop)
	m.schedulerWg.Wait()
	m.running = false

	return nil
}

// saveBackup saves a backup to the metadata store
func (m *Manager) saveBackup(backup *Backup) {
	m.metadataMu.Lock()
	defer m.metadataMu.Unlock()

	// Check if backup already exists (update)
	found := false
	for i, b := range m.metadata.Backups {
		if b.ID == backup.ID {
			m.metadata.Backups[i] = backup
			found = true
			break
		}
	}

	if !found {
		m.metadata.Backups = append(m.metadata.Backups, backup)
	}

	m.saveMetadataLocked() //nolint:errcheck // Best effort - backup file already saved
}

// loadMetadata loads backup metadata from disk
func (m *Manager) loadMetadata() error {
	m.metadataMu.Lock()
	defer m.metadataMu.Unlock()

	data, err := os.ReadFile(m.metadataFile)
	if err != nil {
		return err
	}

	var metadata MetadataStore
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	m.metadata = &metadata
	return nil
}

// saveMetadataLocked saves backup metadata to disk (must be called with lock held)
func (m *Manager) saveMetadataLocked() error {
	data, err := json.MarshalIndent(m.metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.metadataFile, data, 0o600) //nolint:gosec // Metadata file permissions are intentionally restricted
}

// SetOnBackupComplete sets the callback for backup completion
func (m *Manager) SetOnBackupComplete(fn func(backup *Backup)) {
	m.onBackupComplete = fn
}

// SetOnRestoreStart sets the callback for restore start
func (m *Manager) SetOnRestoreStart(fn func(backupID string)) {
	m.onRestoreStart = fn
}
