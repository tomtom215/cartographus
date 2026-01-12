// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkCreateBackup benchmarks backup creation
func BenchmarkCreateBackup(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "backup-bench-*")
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.duckdb")
	// Create a larger test database
	content := make([]byte, 1024*1024) // 1MB
	for i := range content {
		content[i] = byte(i % 256)
	}
	os.WriteFile(dbPath, content, 0644)

	mockDB := &MockDatabase{path: dbPath}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Retention: DefaultRetentionPolicy(),
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     1, // Fast compression for benchmark
			Algorithm: "gzip",
		},
	}

	manager, _ := NewManager(cfg, mockDB)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backup, err := manager.CreateBackup(ctx, TypeDatabase, "benchmark")
		if err != nil {
			b.Fatalf("backup failed: %v", err)
		}
		// Clean up
		os.Remove(backup.FilePath)
	}
}

// BenchmarkCalculateChecksum benchmarks checksum calculation
func BenchmarkCalculateChecksum(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "backup-bench-*")
	defer os.RemoveAll(tempDir)

	// Create test file
	filePath := filepath.Join(tempDir, "test.bin")
	content := make([]byte, 10*1024*1024) // 10MB
	for i := range content {
		content[i] = byte(i % 256)
	}
	os.WriteFile(filePath, content, 0644)

	manager := &Manager{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.calculateFileChecksum(filePath)
		if err != nil {
			b.Fatalf("checksum failed: %v", err)
		}
	}
}

// BenchmarkListBackups benchmarks backup listing
func BenchmarkListBackups(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "backup-bench-*")
	defer os.RemoveAll(tempDir)

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Retention: DefaultRetentionPolicy(),
	}

	manager, _ := NewManager(cfg, nil)

	// Add many backups to metadata for realistic benchmark
	for i := 0; i < 100; i++ {
		manager.metadata.Backups = append(manager.metadata.Backups, &Backup{
			ID:     filepath.Join("backup", string(rune(i))),
			Type:   TypeFull,
			Status: StatusCompleted,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ListBackups(BackupListOptions{
			Limit:    10,
			SortDesc: true,
		})
	}
}
