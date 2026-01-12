// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestFileExists tests file existence helper
func TestFileExists(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	existingFile := filepath.Join(env.tempDir, "exists.txt")
	_ = os.WriteFile(existingFile, []byte("test"), 0644)

	if !fileExists(existingFile) {
		t.Error("expected file to exist")
	}

	if fileExists(filepath.Join(env.tempDir, "nonexistent.txt")) {
		t.Error("expected file to not exist")
	}
}

// TestGetFileSize tests file size helper
func TestGetFileSize(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	content := []byte("test content here")
	filePath := filepath.Join(env.tempDir, "test.txt")
	_ = os.WriteFile(filePath, content, 0644)

	size := getFileSize(filePath)
	if size != int64(len(content)) {
		t.Errorf("expected size=%d, got %d", len(content), size)
	}

	// Non-existent file
	size = getFileSize("/nonexistent/file.txt")
	if size != 0 {
		t.Error("expected 0 for non-existent file")
	}
}

// TestContainsFile tests file containment helper
func TestContainsFile(t *testing.T) {
	result := &ValidationResult{
		Backup: &Backup{
			Contents: BackupContents{
				Files: []BackupFile{
					{Path: "database/cartographus.duckdb"},
					{Path: "config/config.json"},
				},
			},
		},
	}

	if !containsFile(result, "database/cartographus.duckdb") {
		t.Error("expected to find database file")
	}

	if containsFile(result, "nonexistent/file.txt") {
		t.Error("expected not to find nonexistent file")
	}
}

// TestArchiveWriters tests archive writer management
func TestArchiveWriters(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := &Config{
		BackupDir: env.tempDir,
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}

	manager := &Manager{cfg: cfg}

	t.Run("with compression", func(t *testing.T) {
		filePath := filepath.Join(env.tempDir, "test.tar.gz")
		aw, err := manager.setupArchiveWriters(filePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if aw.tarWriter == nil {
			t.Error("tar writer should not be nil")
		}
		if len(aw.closers) != 3 { // file, gzip, tar
			t.Errorf("expected 3 closers, got %d", len(aw.closers))
		}

		err = aw.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("without compression", func(t *testing.T) {
		cfg.Compression.Enabled = false
		filePath := filepath.Join(env.tempDir, "test.tar")
		aw, err := manager.setupArchiveWriters(filePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(aw.closers) != 2 { // file, tar
			t.Errorf("expected 2 closers, got %d", len(aw.closers))
		}

		aw.Close()
	})
}

// TestOpenArchiveReader tests archive reader opening
func TestOpenArchiveReader(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	t.Run("non-existent file", func(t *testing.T) {
		_, _, err := openArchiveReader("/nonexistent/file.tar.gz")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("uncompressed tar", func(t *testing.T) {
		filePath := filepath.Join(env.tempDir, "test.tar")
		_ = os.WriteFile(filePath, []byte("fake tar content"), 0644)

		reader, closers, err := openArchiveReader(filePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer closeAll(closers)

		if reader == nil {
			t.Error("reader should not be nil")
		}
	})
}

// mockCloser is a mock io.Closer for testing
type mockCloser struct {
	onClose func() error
}

func (m *mockCloser) Close() error {
	if m.onClose != nil {
		return m.onClose()
	}
	return nil
}

// TestCloseAll tests closer cleanup
func TestCloseAll(t *testing.T) {
	closeCalled := make([]bool, 3)
	closers := make([]io.Closer, 3)
	for i := range closers {
		idx := i
		closers[i] = &mockCloser{onClose: func() error {
			closeCalled[idx] = true
			return nil
		}}
	}

	closeAll(closers)

	for i, called := range closeCalled {
		if !called {
			t.Errorf("closer %d was not called", i)
		}
	}
}
