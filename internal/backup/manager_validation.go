// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
manager_validation.go - Backup Validation and Integrity Checking

This file provides validation functions to verify backup integrity and
completeness before restore operations.

Validation Steps:
 1. File Existence: Verify backup file exists on disk
 2. Checksum Verification: Compare stored SHA-256 with calculated checksum
 3. Archive Readability: Attempt to read tar/gzip archive structure
 4. Required Files Check: Verify expected files based on backup type

Required Files by Type:
  - TypeFull: database/cartographus.duckdb AND config/config.json
  - TypeDatabase: database/cartographus.duckdb
  - TypeConfig: config/config.json

Validation Result:
The ValidationResult struct provides detailed status:
  - Valid: Overall validity (all checks passed)
  - ChecksumValid: SHA-256 checksum matches
  - ArchiveReadable: Archive can be read without errors
  - FilesComplete: All required files present
  - DatabaseValid: Database component valid (for full/database backups)
  - ConfigValid: Config component valid (for full/config backups)
  - Errors: List of validation failures
  - Warnings: Non-fatal issues detected

Error Handling:
Validation errors are collected in the result struct rather than returned
as function errors. This allows partial validation results to be returned
even when some checks fail.
*/

//nolint:staticcheck // File documentation, not package doc
package backup

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ValidateBackup validates a backup's integrity
func (m *Manager) ValidateBackup(backupID string) (*ValidationResult, error) {
	backup, err := m.GetBackup(backupID)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{
		Valid:    true,
		Backup:   backup,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Check if file exists
	if !fileExists(backup.FilePath) {
		result.Valid = false
		result.Errors = append(result.Errors, "backup file does not exist")
		return result, nil
	}

	// Verify checksum
	if err := m.validateChecksum(backup, result); err != nil {
		//nolint:nilerr // Validation errors are recorded in result, not returned as error
		return result, nil
	}

	// Try to read the archive
	if err := m.validateArchiveReadable(backup.FilePath, result); err != nil {
		//nolint:nilerr // Validation errors are recorded in result, not returned as error
		return result, nil
	}

	// Check for required files based on backup type
	m.validateRequiredFiles(backup, result)

	return result, nil
}

// validateChecksum verifies the backup's checksum
func (m *Manager) validateChecksum(backup *Backup, result *ValidationResult) error {
	actualChecksum, err := m.calculateFileChecksum(backup.FilePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to calculate checksum: %v", err))
		return err
	}

	result.ExpectedChecksum = backup.Checksum
	result.ActualChecksum = actualChecksum
	result.ChecksumValid = actualChecksum == backup.Checksum

	if !result.ChecksumValid {
		result.Valid = false
		result.Errors = append(result.Errors, "checksum mismatch - backup may be corrupted")
	}

	return nil
}

// validateArchiveReadable checks if the archive can be read
func (m *Manager) validateArchiveReadable(archivePath string, result *ValidationResult) error {
	result.ArchiveReadable = true
	if err := m.validateArchiveContents(archivePath, result); err != nil {
		result.Valid = false
		result.ArchiveReadable = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to read archive: %v", err))
		return err
	}
	return nil
}

// validateRequiredFiles checks for required files based on backup type
func (m *Manager) validateRequiredFiles(backup *Backup, result *ValidationResult) {
	result.FilesComplete = true

	switch backup.Type {
	case TypeFull:
		m.checkRequiredFile(result, "database/cartographus.duckdb")
		m.checkRequiredFile(result, "config/config.json")
	case TypeDatabase:
		m.checkRequiredFile(result, "database/cartographus.duckdb")
	case TypeConfig:
		m.checkRequiredFile(result, "config/config.json")
	}

	if !result.FilesComplete {
		result.Valid = false
	}

	// Set component validity
	result.DatabaseValid = backup.Type == TypeConfig || containsFile(result, "database/cartographus.duckdb")
	result.ConfigValid = backup.Type == TypeDatabase || containsFile(result, "config/config.json")
}

// checkRequiredFile checks if a required file is present in the backup
func (m *Manager) checkRequiredFile(result *ValidationResult, filename string) {
	if !containsFile(result, filename) {
		result.FilesComplete = false
		result.MissingFiles = append(result.MissingFiles, filename)
	}
}

// validateArchiveContents validates the contents of a backup archive
//
//nolint:gosec // G304: archivePath is from internal backup storage
func (m *Manager) validateArchiveContents(archivePath string, result *ValidationResult) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck // Best effort cleanup

	var reader io.Reader = file

	// Handle gzip compression
	if strings.HasSuffix(archivePath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close() //nolint:errcheck // Best effort cleanup
		reader = gzReader
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Track files found
		result.Backup.Contents.Files = append(result.Backup.Contents.Files, BackupFile{
			Path: header.Name,
			Size: header.Size,
		})
	}

	return nil
}
