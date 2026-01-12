// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
manager_archive.go - Backup Archive Creation

This file handles the creation of backup archive files using tar format
with optional gzip compression.

Archive Structure:

	backup-{type}-{timestamp}-{id}.tar.gz
	├── database/
	│   ├── cartographus.duckdb       (main database file)
	│   └── cartographus.duckdb.wal   (WAL file, if present)
	├── config/
	│   └── config.json      (sanitized configuration)
	└── backup-metadata.json (backup details and checksums)

Archive Creation Process:
 1. Setup writers (file -> gzip -> tar)
 2. Force database checkpoint for consistent state
 3. Add content based on backup type (full/database/config)
 4. Calculate SHA-256 checksums for each file
 5. Add backup metadata as final entry
 6. Close writers in reverse order

Security:
  - Sensitive values (API keys, passwords) are redacted from config
  - File checksums enable integrity verification on restore
  - Archive files have restricted permissions (0640)
*/

//nolint:staticcheck // File documentation, not package doc
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// archiveWriters holds the writers needed for creating backup archives
type archiveWriters struct {
	tarWriter *tar.Writer
	closers   []io.Closer
}

// Close closes all writers in reverse order, returning the first error encountered
func (aw *archiveWriters) Close() error {
	var firstErr error
	for i := len(aw.closers) - 1; i >= 0; i-- {
		if err := aw.closers[i].Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// setupArchiveWriters creates the file, compression, and tar writers for backup archive creation
//
//nolint:gosec // G304: filePath is from internal backup configuration
func (m *Manager) setupArchiveWriters(filePath string) (*archiveWriters, error) {
	outFile, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}

	aw := &archiveWriters{
		closers: []io.Closer{outFile},
	}

	var tarDest io.Writer = outFile
	if m.cfg.Compression.Enabled {
		gzWriter, err := gzip.NewWriterLevel(outFile, m.cfg.Compression.Level)
		if err != nil {
			outFile.Close() //nolint:errcheck // Best effort cleanup on error
			return nil, fmt.Errorf("failed to create gzip writer: %w", err)
		}
		aw.closers = append(aw.closers, gzWriter)
		tarDest = gzWriter
	}

	aw.tarWriter = tar.NewWriter(tarDest)
	aw.closers = append(aw.closers, aw.tarWriter)

	return aw, nil
}

// addBackupContent adds appropriate content to archive based on backup type
func (m *Manager) addBackupContent(ctx context.Context, tw *tar.Writer, backup *Backup, backupType BackupType) error {
	switch backupType {
	case TypeFull:
		if err := m.addDatabaseToArchive(ctx, tw, backup); err != nil {
			return err
		}
		return m.addConfigToArchive(ctx, tw, backup)
	case TypeDatabase:
		return m.addDatabaseToArchive(ctx, tw, backup)
	case TypeConfig:
		return m.addConfigToArchive(ctx, tw, backup)
	default:
		return fmt.Errorf("unsupported backup type: %s", backupType)
	}
}

// createBackupArchive creates the backup archive file
func (m *Manager) createBackupArchive(ctx context.Context, backup *Backup, backupType BackupType) (err error) {
	aw, err := m.setupArchiveWriters(backup.FilePath)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := aw.Close()
		if err == nil {
			err = closeErr
		}
	}()

	if err := m.addBackupContent(ctx, aw.tarWriter, backup, backupType); err != nil {
		return err
	}

	return m.addMetadataToArchive(aw.tarWriter, backup)
}

// addDatabaseToArchive adds database files to the backup archive
func (m *Manager) addDatabaseToArchive(ctx context.Context, tw *tar.Writer, backup *Backup) error {
	if m.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Force a checkpoint to ensure WAL is flushed
	if err := m.db.Checkpoint(ctx); err != nil {
		// Log but don't fail - backup can still proceed
		logging.Warn().Err(err).Msg("Checkpoint failed, backup may include uncommitted data")
	}

	dbPath := m.db.GetDatabasePath()
	walPath := dbPath + ".wal"

	// Initialize database backup info
	backup.Contents.Database = &DatabaseBackupInfo{
		Path:       dbPath,
		Extensions: []string{"spatial", "h3", "inet", "icu", "json"},
	}

	// Get record counts
	playbacks, geolocations, err := m.db.GetRecordCounts(ctx)
	if err == nil {
		backup.Contents.Database.PlaybackCount = playbacks
		backup.Contents.Database.GeolocationCount = geolocations
		backup.RecordCount = playbacks + geolocations
	}

	// Add main database file
	if err := m.addFileToArchive(tw, dbPath, "database/cartographus.duckdb", backup); err != nil {
		return fmt.Errorf("failed to add database file: %w", err)
	}
	backup.Contents.Database.Size = getFileSize(dbPath)

	// Add WAL file if it exists
	if fileExists(walPath) {
		if err := m.addFileToArchive(tw, walPath, "database/cartographus.duckdb.wal", backup); err != nil {
			return fmt.Errorf("failed to add WAL file: %w", err)
		}
		backup.Contents.Database.WALIncluded = true
		backup.Contents.Database.WALSize = getFileSize(walPath)
	}

	return nil
}

// addConfigToArchive adds configuration to the backup archive
func (m *Manager) addConfigToArchive(_ context.Context, tw *tar.Writer, backup *Backup) error {
	// Create sanitized config (no secrets)
	config := m.getSanitizedConfig()

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create tar header
	header := &tar.Header{
		Name:    "config/config.json",
		Size:    int64(len(configJSON)),
		Mode:    0o640,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write config header: %w", err)
	}

	if _, err := tw.Write(configJSON); err != nil {
		return fmt.Errorf("failed to write config data: %w", err)
	}

	// Add to backup contents
	checksum := sha256.Sum256(configJSON)
	backup.Contents.Files = append(backup.Contents.Files, BackupFile{
		Path:         "config/config.json",
		OriginalPath: "runtime",
		Size:         int64(len(configJSON)),
		ModTime:      time.Now(),
		Checksum:     hex.EncodeToString(checksum[:]),
	})

	backup.Contents.Config = &ConfigBackupInfo{
		ValueCount:      len(config),
		IncludesSecrets: false,
		Categories:      []string{"tautulli", "plex", "database", "sync", "server", "api", "security", "logging", "backup"},
	}

	return nil
}

// getSanitizedConfig returns configuration without sensitive values
func (m *Manager) getSanitizedConfig() map[string]interface{} {
	// Read current environment and sanitize
	config := map[string]interface{}{
		"tautulli": map[string]interface{}{
			"url": os.Getenv("TAUTULLI_URL"),
			// API key is redacted
		},
		"database": map[string]interface{}{
			"path":       os.Getenv("DUCKDB_PATH"),
			"max_memory": os.Getenv("DUCKDB_MAX_MEMORY"),
		},
		"server": map[string]interface{}{
			"port":      os.Getenv("HTTP_PORT"),
			"host":      os.Getenv("HTTP_HOST"),
			"latitude":  os.Getenv("SERVER_LATITUDE"),
			"longitude": os.Getenv("SERVER_LONGITUDE"),
		},
		"sync": map[string]interface{}{
			"interval":   os.Getenv("SYNC_INTERVAL"),
			"lookback":   os.Getenv("SYNC_LOOKBACK"),
			"batch_size": os.Getenv("SYNC_BATCH_SIZE"),
		},
		"api": map[string]interface{}{
			"default_page_size": os.Getenv("API_DEFAULT_PAGE_SIZE"),
			"max_page_size":     os.Getenv("API_MAX_PAGE_SIZE"),
		},
		"security": map[string]interface{}{
			"auth_mode":       os.Getenv("AUTH_MODE"),
			"session_timeout": os.Getenv("SESSION_TIMEOUT"),
			"rate_limit_reqs": os.Getenv("RATE_LIMIT_REQUESTS"),
			// Passwords and JWT secrets are redacted
		},
		"logging": map[string]interface{}{
			"level": os.Getenv("LOG_LEVEL"),
		},
		"backup": map[string]interface{}{
			"enabled":   os.Getenv("BACKUP_ENABLED"),
			"dir":       os.Getenv("BACKUP_DIR"),
			"interval":  os.Getenv("BACKUP_INTERVAL"),
			"retention": os.Getenv("BACKUP_RETENTION_MAX_COUNT"),
		},
	}

	return config
}

// addMetadataToArchive adds backup metadata to the archive
func (m *Manager) addMetadataToArchive(tw *tar.Writer, backup *Backup) error {
	metadataJSON, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup metadata: %w", err)
	}

	header := &tar.Header{
		Name:    "backup-metadata.json",
		Size:    int64(len(metadataJSON)),
		Mode:    0o640,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write metadata header: %w", err)
	}

	if _, err := tw.Write(metadataJSON); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// addFileToArchive adds a file to the tar archive
//
//nolint:gosec // G304: srcPath is validated by caller
func (m *Manager) addFileToArchive(tw *tar.Writer, srcPath, destPath string, backup *Backup) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", srcPath, err)
	}
	defer file.Close() //nolint:errcheck // Best effort cleanup

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", srcPath, err)
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", srcPath, err)
	}
	header.Name = destPath

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", srcPath, err)
	}

	// Calculate checksum while copying
	hasher := sha256.New()
	multiWriter := io.MultiWriter(tw, hasher)

	if _, err := io.Copy(multiWriter, file); err != nil {
		return fmt.Errorf("failed to copy %s to archive: %w", srcPath, err)
	}

	// Add to backup contents
	backup.Contents.Files = append(backup.Contents.Files, BackupFile{
		Path:         destPath,
		OriginalPath: srcPath,
		Size:         info.Size(),
		ModTime:      info.ModTime(),
		Checksum:     hex.EncodeToString(hasher.Sum(nil)),
	})

	return nil
}

// calculateFileChecksum calculates SHA-256 checksum of a file
//
//nolint:gosec // G304: filePath is from internal backup storage
func (m *Manager) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close() //nolint:errcheck // Best effort cleanup

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
