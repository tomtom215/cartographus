// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/backup"
	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
)

// mockBackupManager implements BackupManager interface for testing
type mockBackupManager struct {
	createBackupFunc            func(ctx context.Context, backupType backup.BackupType, notes string) (*backup.Backup, error)
	listBackupsFunc             func(opts backup.BackupListOptions) ([]*backup.Backup, error)
	getBackupFunc               func(backupID string) (*backup.Backup, error)
	deleteBackupFunc            func(backupID string) error
	validateBackupFunc          func(backupID string) (*backup.ValidationResult, error)
	restoreFromBackupFunc       func(ctx context.Context, backupID string, opts backup.RestoreOptions) (*backup.RestoreResult, error)
	downloadBackupFunc          func(backupID string) (io.ReadCloser, *backup.Backup, error)
	importBackupFunc            func(ctx context.Context, reader io.Reader, filename string) (*backup.Backup, error)
	getStatsFunc                func() (*backup.BackupStats, error)
	applyRetentionPolicyFunc    func(ctx context.Context) error
	getRetentionPreviewFunc     func() (*backup.RetentionPreview, error)
	getRetentionPolicyFunc      func() backup.RetentionPolicy
	setRetentionPolicyFunc      func(policy backup.RetentionPolicy) error
	cleanupCorruptedBackupsFunc func(ctx context.Context) (int, error)
	getScheduleConfigFunc       func() backup.ScheduleConfig
	setScheduleConfigFunc       func(ctx context.Context, schedule backup.ScheduleConfig) error
	triggerScheduledBackupFunc  func(ctx context.Context) (*backup.Backup, error)
}

func (m *mockBackupManager) CreateBackup(ctx context.Context, backupType backup.BackupType, notes string) (*backup.Backup, error) {
	if m.createBackupFunc != nil {
		return m.createBackupFunc(ctx, backupType, notes)
	}
	return nil, nil
}

func (m *mockBackupManager) ListBackups(opts backup.BackupListOptions) ([]*backup.Backup, error) {
	if m.listBackupsFunc != nil {
		return m.listBackupsFunc(opts)
	}
	return nil, nil
}

func (m *mockBackupManager) GetBackup(backupID string) (*backup.Backup, error) {
	if m.getBackupFunc != nil {
		return m.getBackupFunc(backupID)
	}
	return nil, nil
}

func (m *mockBackupManager) DeleteBackup(backupID string) error {
	if m.deleteBackupFunc != nil {
		return m.deleteBackupFunc(backupID)
	}
	return nil
}

func (m *mockBackupManager) ValidateBackup(backupID string) (*backup.ValidationResult, error) {
	if m.validateBackupFunc != nil {
		return m.validateBackupFunc(backupID)
	}
	return nil, nil
}

func (m *mockBackupManager) RestoreFromBackup(ctx context.Context, backupID string, opts backup.RestoreOptions) (*backup.RestoreResult, error) {
	if m.restoreFromBackupFunc != nil {
		return m.restoreFromBackupFunc(ctx, backupID, opts)
	}
	return nil, nil
}

func (m *mockBackupManager) DownloadBackup(backupID string) (io.ReadCloser, *backup.Backup, error) {
	if m.downloadBackupFunc != nil {
		return m.downloadBackupFunc(backupID)
	}
	return nil, nil, nil
}

func (m *mockBackupManager) ImportBackup(ctx context.Context, reader io.Reader, filename string) (*backup.Backup, error) {
	if m.importBackupFunc != nil {
		return m.importBackupFunc(ctx, reader, filename)
	}
	return nil, nil
}

func (m *mockBackupManager) GetStats() (*backup.BackupStats, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc()
	}
	return nil, nil
}

func (m *mockBackupManager) ApplyRetentionPolicy(ctx context.Context) error {
	if m.applyRetentionPolicyFunc != nil {
		return m.applyRetentionPolicyFunc(ctx)
	}
	return nil
}

func (m *mockBackupManager) GetRetentionPreview() (*backup.RetentionPreview, error) {
	if m.getRetentionPreviewFunc != nil {
		return m.getRetentionPreviewFunc()
	}
	return nil, nil
}

func (m *mockBackupManager) GetRetentionPolicy() backup.RetentionPolicy {
	if m.getRetentionPolicyFunc != nil {
		return m.getRetentionPolicyFunc()
	}
	return backup.RetentionPolicy{}
}

func (m *mockBackupManager) SetRetentionPolicy(policy backup.RetentionPolicy) error {
	if m.setRetentionPolicyFunc != nil {
		return m.setRetentionPolicyFunc(policy)
	}
	return nil
}

func (m *mockBackupManager) CleanupCorruptedBackups(ctx context.Context) (int, error) {
	if m.cleanupCorruptedBackupsFunc != nil {
		return m.cleanupCorruptedBackupsFunc(ctx)
	}
	return 0, nil
}

func (m *mockBackupManager) GetScheduleConfig() backup.ScheduleConfig {
	if m.getScheduleConfigFunc != nil {
		return m.getScheduleConfigFunc()
	}
	return backup.ScheduleConfig{}
}

func (m *mockBackupManager) SetScheduleConfig(ctx context.Context, schedule backup.ScheduleConfig) error {
	if m.setScheduleConfigFunc != nil {
		return m.setScheduleConfigFunc(ctx, schedule)
	}
	return nil
}

func (m *mockBackupManager) TriggerScheduledBackup(ctx context.Context) (*backup.Backup, error) {
	if m.triggerScheduledBackupFunc != nil {
		return m.triggerScheduledBackupFunc(ctx)
	}
	return nil, nil
}

// mockReadCloser implements io.ReadCloser for testing
type mockReadCloser struct {
	*bytes.Reader
	closed bool
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

// setupBackupTestHandler creates a handler for backup testing
func setupBackupTestHandler(t *testing.T, bm BackupManager) *Handler {
	t.Helper()
	return &Handler{
		cache:         cache.New(5 * time.Minute),
		startTime:     time.Now(),
		backupManager: bm,
	}
}

// TestBackupHandlers_MethodNotAllowed consolidates all method not allowed tests
func TestBackupHandlers_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	handler := setupBackupTestHandler(t, &mockBackupManager{})

	tests := []struct {
		name           string
		handlerFunc    http.HandlerFunc
		path           string
		invalidMethods []string
	}{
		{"CreateBackup", handler.HandleCreateBackup, "/api/v1/backup", []string{"GET", "PUT", "DELETE", "PATCH"}},
		{"QuickBackup", handler.HandleQuickBackup, "/api/v1/backup/quick", []string{"GET", "PUT", "DELETE"}},
		{"ListBackups", handler.HandleListBackups, "/api/v1/backups", []string{"POST", "PUT", "DELETE"}},
		{"GetBackup", handler.HandleGetBackup, "/api/v1/backups?id=123", []string{"POST", "PUT", "DELETE"}},
		{"DeleteBackup", handler.HandleDeleteBackup, "/api/v1/backups?id=123", []string{"GET", "POST", "PUT"}},
		{"ValidateBackup", handler.HandleValidateBackup, "/api/v1/backups/validate?id=123", []string{"POST", "PUT", "DELETE"}},
		{"RestoreBackup", handler.HandleRestoreBackup, "/api/v1/backups/restore?id=123", []string{"GET", "PUT", "DELETE"}},
		{"DownloadBackup", handler.HandleDownloadBackup, "/api/v1/backups/download?id=123", []string{"POST", "PUT", "DELETE"}},
		{"UploadBackup", handler.HandleUploadBackup, "/api/v1/backups/upload", []string{"GET", "PUT", "DELETE"}},
		{"GetBackupStats", handler.HandleGetBackupStats, "/api/v1/backup/stats", []string{"POST", "PUT", "DELETE"}},
		{"GetRetentionPolicy", handler.HandleGetRetentionPolicy, "/api/v1/backup/retention", []string{"POST", "DELETE"}},
		{"SetRetentionPolicy", handler.HandleSetRetentionPolicy, "/api/v1/backup/retention", []string{"GET", "POST", "DELETE"}},
		{"RetentionPreview", handler.HandleRetentionPreview, "/api/v1/backup/retention/preview", []string{"POST", "PUT", "DELETE"}},
		{"ApplyRetention", handler.HandleApplyRetention, "/api/v1/backup/retention/apply", []string{"GET", "PUT", "DELETE"}},
		{"CleanupCorrupted", handler.HandleCleanupCorrupted, "/api/v1/backup/cleanup", []string{"GET", "PUT", "DELETE"}},
	}

	for _, tt := range tests {
		for _, method := range tt.invalidMethods {
			t.Run(tt.name+"_"+method, func(t *testing.T) {
				req := httptest.NewRequest(method, tt.path, nil)
				w := httptest.NewRecorder()
				tt.handlerFunc(w, req)
				if w.Code != http.StatusMethodNotAllowed {
					t.Errorf("%s with %s: expected 405, got %d", tt.name, method, w.Code)
				}
			})
		}
	}
}

// TestBackupHandlers_BackupDisabled consolidates all backup disabled tests
func TestBackupHandlers_BackupDisabled(t *testing.T) {
	t.Parallel()
	handler := setupBackupTestHandler(t, nil)

	tests := []struct {
		name        string
		handlerFunc http.HandlerFunc
		method      string
		path        string
	}{
		{"CreateBackup", handler.HandleCreateBackup, "POST", "/api/v1/backup"},
		{"QuickBackup", handler.HandleQuickBackup, "POST", "/api/v1/backup/quick"},
		{"ListBackups", handler.HandleListBackups, "GET", "/api/v1/backups"},
		{"GetBackup", handler.HandleGetBackup, "GET", "/api/v1/backups?id=123"},
		{"DeleteBackup", handler.HandleDeleteBackup, "DELETE", "/api/v1/backups?id=123"},
		{"ValidateBackup", handler.HandleValidateBackup, "GET", "/api/v1/backups/validate?id=123"},
		{"RestoreBackup", handler.HandleRestoreBackup, "POST", "/api/v1/backups/restore?id=123"},
		{"DownloadBackup", handler.HandleDownloadBackup, "GET", "/api/v1/backups/download?id=123"},
		{"UploadBackup", handler.HandleUploadBackup, "POST", "/api/v1/backups/upload"},
		{"GetBackupStats", handler.HandleGetBackupStats, "GET", "/api/v1/backup/stats"},
		{"GetRetentionPolicy", handler.HandleGetRetentionPolicy, "GET", "/api/v1/backup/retention"},
		{"SetRetentionPolicy", handler.HandleSetRetentionPolicy, "PUT", "/api/v1/backup/retention"},
		{"RetentionPreview", handler.HandleRetentionPreview, "GET", "/api/v1/backup/retention/preview"},
		{"ApplyRetention", handler.HandleApplyRetention, "POST", "/api/v1/backup/retention/apply"},
		{"CleanupCorrupted", handler.HandleCleanupCorrupted, "POST", "/api/v1/backup/cleanup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			tt.handlerFunc(w, req)

			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("expected 503, got %d", w.Code)
			}

			var resp models.APIResponse
			json.NewDecoder(w.Body).Decode(&resp)
			if resp.Error == nil || resp.Error.Code != "BACKUP_DISABLED" {
				t.Error("expected BACKUP_DISABLED error")
			}
		})
	}
}

// TestHandleCreateBackup tests backup creation scenarios
func TestHandleCreateBackup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		body         string
		mockFunc     func(ctx context.Context, bt backup.BackupType, notes string) (*backup.Backup, error)
		expectedCode int
		checkType    backup.BackupType
	}{
		{
			name: "default type (full)",
			body: "",
			mockFunc: func(ctx context.Context, bt backup.BackupType, notes string) (*backup.Backup, error) {
				return &backup.Backup{ID: "b1", Type: bt}, nil
			},
			expectedCode: http.StatusCreated,
			checkType:    backup.TypeFull,
		},
		{
			name: "database type",
			body: `{"type": "database", "notes": "test"}`,
			mockFunc: func(ctx context.Context, bt backup.BackupType, notes string) (*backup.Backup, error) {
				return &backup.Backup{ID: "b2", Type: bt}, nil
			},
			expectedCode: http.StatusCreated,
			checkType:    backup.TypeDatabase,
		},
		{
			name: "config type",
			body: `{"type": "config"}`,
			mockFunc: func(ctx context.Context, bt backup.BackupType, notes string) (*backup.Backup, error) {
				return &backup.Backup{ID: "b3", Type: bt}, nil
			},
			expectedCode: http.StatusCreated,
			checkType:    backup.TypeConfig,
		},
		{
			name:         "invalid type",
			body:         `{"type": "invalid"}`,
			mockFunc:     nil,
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "create error",
			body: `{"type": "full"}`,
			mockFunc: func(ctx context.Context, bt backup.BackupType, notes string) (*backup.Backup, error) {
				return nil, errors.New("disk full")
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{createBackupFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/backup", body)
			w := httptest.NewRecorder()

			handler.HandleCreateBackup(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d: %s", tt.expectedCode, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleListBackups tests backup listing scenarios
func TestHandleListBackups(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		mock := &mockBackupManager{
			listBackupsFunc: func(opts backup.BackupListOptions) ([]*backup.Backup, error) {
				return []*backup.Backup{{ID: "b1"}, {ID: "b2"}}, nil
			},
		}
		handler := setupBackupTestHandler(t, mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/backups", nil)
		w := httptest.NewRecorder()
		handler.HandleListBackups(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("with query params", func(t *testing.T) {
		var captured backup.BackupListOptions
		mock := &mockBackupManager{
			listBackupsFunc: func(opts backup.BackupListOptions) ([]*backup.Backup, error) {
				captured = opts
				return []*backup.Backup{}, nil
			},
		}
		handler := setupBackupTestHandler(t, mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/backups?limit=50&offset=10&sort=asc&type=full&status=completed", nil)
		w := httptest.NewRecorder()
		handler.HandleListBackups(w, req)

		if captured.Limit != 50 || captured.Offset != 10 {
			t.Errorf("params not parsed: limit=%d, offset=%d", captured.Limit, captured.Offset)
		}
		if captured.SortDesc {
			t.Error("expected SortDesc=false for asc")
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockBackupManager{
			listBackupsFunc: func(opts backup.BackupListOptions) ([]*backup.Backup, error) {
				return nil, errors.New("db error")
			},
		}
		handler := setupBackupTestHandler(t, mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/backups", nil)
		w := httptest.NewRecorder()
		handler.HandleListBackups(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

// TestHandleGetBackup tests get backup scenarios
func TestHandleGetBackup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		mockFunc     func(id string) (*backup.Backup, error)
		expectedCode int
	}{
		{"missing id", "/api/v1/backups", nil, http.StatusBadRequest},
		{"not found", "/api/v1/backups?id=x", func(id string) (*backup.Backup, error) {
			return nil, errors.New("not found")
		}, http.StatusNotFound},
		{"success", "/api/v1/backups?id=123", func(id string) (*backup.Backup, error) {
			return &backup.Backup{ID: id}, nil
		}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{getBackupFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			handler.HandleGetBackup(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestHandleDeleteBackup tests delete backup scenarios
func TestHandleDeleteBackup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		mockFunc     func(id string) error
		expectedCode int
	}{
		{"missing id", "/api/v1/backups", nil, http.StatusBadRequest},
		{"success", "/api/v1/backups?id=123", func(id string) error { return nil }, http.StatusOK},
		{"error", "/api/v1/backups?id=123", func(id string) error { return errors.New("fail") }, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{deleteBackupFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)
			w := httptest.NewRecorder()
			handler.HandleDeleteBackup(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestHandleValidateBackup tests validate backup scenarios
func TestHandleValidateBackup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		mockFunc     func(id string) (*backup.ValidationResult, error)
		expectedCode int
	}{
		{"missing id", "/api/v1/backups/validate", nil, http.StatusBadRequest},
		{"success", "/api/v1/backups/validate?id=123", func(id string) (*backup.ValidationResult, error) {
			return &backup.ValidationResult{Valid: true}, nil
		}, http.StatusOK},
		{"error", "/api/v1/backups/validate?id=123", func(id string) (*backup.ValidationResult, error) {
			return nil, errors.New("fail")
		}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{validateBackupFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			handler.HandleValidateBackup(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestHandleRestoreBackup tests restore backup scenarios
func TestHandleRestoreBackup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		body         string
		mockFunc     func(ctx context.Context, id string, opts backup.RestoreOptions) (*backup.RestoreResult, error)
		expectedCode int
	}{
		{"missing id", "/api/v1/backups/restore", "", nil, http.StatusBadRequest},
		{"success", "/api/v1/backups/restore?id=123", `{"restore_database": true}`, func(ctx context.Context, id string, opts backup.RestoreOptions) (*backup.RestoreResult, error) {
			return &backup.RestoreResult{Success: true}, nil
		}, http.StatusOK},
		{"failure result", "/api/v1/backups/restore?id=123", "", func(ctx context.Context, id string, opts backup.RestoreOptions) (*backup.RestoreResult, error) {
			return &backup.RestoreResult{Success: false, Error: "failed"}, nil
		}, http.StatusInternalServerError},
		{"error", "/api/v1/backups/restore?id=123", "", func(ctx context.Context, id string, opts backup.RestoreOptions) (*backup.RestoreResult, error) {
			return nil, errors.New("fail")
		}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{restoreFromBackupFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(http.MethodPost, tt.path, body)
			w := httptest.NewRecorder()
			handler.HandleRestoreBackup(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestHandleDownloadBackup tests download backup scenarios
func TestHandleDownloadBackup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		mockFunc     func(id string) (io.ReadCloser, *backup.Backup, error)
		expectedCode int
		checkHeaders bool
	}{
		{"missing id", "/api/v1/backups/download", nil, http.StatusBadRequest, false},
		{"not found", "/api/v1/backups/download?id=x", func(id string) (io.ReadCloser, *backup.Backup, error) {
			return nil, nil, errors.New("not found")
		}, http.StatusNotFound, false},
		{"success", "/api/v1/backups/download?id=123", func(id string) (io.ReadCloser, *backup.Backup, error) {
			return &mockReadCloser{Reader: bytes.NewReader([]byte("data"))}, &backup.Backup{ID: id, CreatedAt: time.Now()}, nil
		}, http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{downloadBackupFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			handler.HandleDownloadBackup(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.checkHeaders {
				if w.Header().Get("Content-Type") != "application/gzip" {
					t.Error("expected Content-Type: application/gzip")
				}
				if !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
					t.Error("expected attachment disposition")
				}
			}
		})
	}
}

// TestHandleUploadBackup tests upload backup scenarios
func TestHandleUploadBackup(t *testing.T) {
	t.Parallel()

	t.Run("missing file", func(t *testing.T) {
		handler := setupBackupTestHandler(t, &mockBackupManager{})

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/backups/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleUploadBackup(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		mock := &mockBackupManager{
			importBackupFunc: func(ctx context.Context, r io.Reader, fn string) (*backup.Backup, error) {
				return &backup.Backup{ID: "imported"}, nil
			},
		}
		handler := setupBackupTestHandler(t, mock)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("backup", "backup.tar.gz")
		part.Write([]byte("content"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/backups/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleUploadBackup(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", w.Code)
		}
	})

	t.Run("import error", func(t *testing.T) {
		mock := &mockBackupManager{
			importBackupFunc: func(ctx context.Context, r io.Reader, fn string) (*backup.Backup, error) {
				return nil, errors.New("invalid format")
			},
		}
		handler := setupBackupTestHandler(t, mock)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("backup", "backup.tar.gz")
		part.Write([]byte("bad"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/backups/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleUploadBackup(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

// TestHandleBackupStats tests backup stats scenarios
func TestHandleBackupStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mockFunc     func() (*backup.BackupStats, error)
		expectedCode int
	}{
		{"success", func() (*backup.BackupStats, error) {
			return &backup.BackupStats{TotalCount: 10}, nil
		}, http.StatusOK},
		{"error", func() (*backup.BackupStats, error) {
			return nil, errors.New("fail")
		}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{getStatsFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/stats", nil)
			w := httptest.NewRecorder()
			handler.HandleGetBackupStats(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestHandleRetentionPolicy tests retention policy endpoints
func TestHandleRetentionPolicy(t *testing.T) {
	t.Parallel()

	t.Run("get success", func(t *testing.T) {
		mock := &mockBackupManager{
			getRetentionPolicyFunc: func() backup.RetentionPolicy {
				return backup.RetentionPolicy{MinCount: 5, MaxCount: 50}
			},
		}
		handler := setupBackupTestHandler(t, mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/retention", nil)
		w := httptest.NewRecorder()
		handler.HandleGetRetentionPolicy(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("set success", func(t *testing.T) {
		var captured backup.RetentionPolicy
		mock := &mockBackupManager{
			setRetentionPolicyFunc: func(p backup.RetentionPolicy) error {
				captured = p
				return nil
			},
		}
		handler := setupBackupTestHandler(t, mock)

		body := `{"min_count": 5, "max_count": 100}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/backup/retention", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler.HandleSetRetentionPolicy(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if captured.MinCount != 5 || captured.MaxCount != 100 {
			t.Error("policy not captured correctly")
		}
	})

	t.Run("set invalid json", func(t *testing.T) {
		handler := setupBackupTestHandler(t, &mockBackupManager{})

		req := httptest.NewRequest(http.MethodPut, "/api/v1/backup/retention", strings.NewReader("invalid"))
		w := httptest.NewRecorder()
		handler.HandleSetRetentionPolicy(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("set invalid policy", func(t *testing.T) {
		mock := &mockBackupManager{
			setRetentionPolicyFunc: func(p backup.RetentionPolicy) error {
				return errors.New("invalid")
			},
		}
		handler := setupBackupTestHandler(t, mock)

		body := `{"min_count": -1}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/backup/retention", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler.HandleSetRetentionPolicy(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

// TestHandleRetentionPreview tests retention preview endpoint
func TestHandleRetentionPreview(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mockFunc     func() (*backup.RetentionPreview, error)
		expectedCode int
	}{
		{"success", func() (*backup.RetentionPreview, error) {
			return &backup.RetentionPreview{DeletedCount: 3}, nil
		}, http.StatusOK},
		{"error", func() (*backup.RetentionPreview, error) {
			return nil, errors.New("fail")
		}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{getRetentionPreviewFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/retention/preview", nil)
			w := httptest.NewRecorder()
			handler.HandleRetentionPreview(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestHandleApplyRetention tests apply retention endpoint
func TestHandleApplyRetention(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		previewFunc  func() (*backup.RetentionPreview, error)
		applyFunc    func(ctx context.Context) error
		expectedCode int
	}{
		{"success", func() (*backup.RetentionPreview, error) {
			return &backup.RetentionPreview{DeletedCount: 5}, nil
		}, func(ctx context.Context) error { return nil }, http.StatusOK},
		{"preview error but apply ok", func() (*backup.RetentionPreview, error) {
			return nil, errors.New("preview fail")
		}, func(ctx context.Context) error { return nil }, http.StatusOK},
		{"apply error", nil, func(ctx context.Context) error {
			return errors.New("apply fail")
		}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{
				getRetentionPreviewFunc:  tt.previewFunc,
				applyRetentionPolicyFunc: tt.applyFunc,
			}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/retention/apply", nil)
			w := httptest.NewRecorder()
			handler.HandleApplyRetention(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestHandleCleanupCorrupted tests cleanup corrupted endpoint
func TestHandleCleanupCorrupted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mockFunc     func(ctx context.Context) (int, error)
		expectedCode int
		checkCount   int
	}{
		{"success", func(ctx context.Context) (int, error) { return 3, nil }, http.StatusOK, 3},
		{"error", func(ctx context.Context) (int, error) { return 0, errors.New("fail") }, http.StatusInternalServerError, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{cleanupCorruptedBackupsFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/cleanup", nil)
			w := httptest.NewRecorder()
			handler.HandleCleanupCorrupted(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectedCode == http.StatusOK {
				var resp models.APIResponse
				json.NewDecoder(w.Body).Decode(&resp)
				if data, ok := resp.Data.(map[string]interface{}); ok {
					if data["cleaned_count"] != float64(tt.checkCount) {
						t.Errorf("expected count %d, got %v", tt.checkCount, data["cleaned_count"])
					}
				}
			}
		})
	}
}

// TestHandleQuickBackup tests quick backup endpoint
func TestHandleQuickBackup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mockFunc     func(ctx context.Context, bt backup.BackupType, notes string) (*backup.Backup, error)
		expectedCode int
	}{
		{"success", func(ctx context.Context, bt backup.BackupType, notes string) (*backup.Backup, error) {
			if bt != backup.TypeFull {
				t.Error("expected full backup type")
			}
			if !strings.Contains(notes, "Quick backup") {
				t.Error("expected Quick backup in notes")
			}
			return &backup.Backup{ID: "quick"}, nil
		}, http.StatusCreated},
		{"error", func(ctx context.Context, bt backup.BackupType, notes string) (*backup.Backup, error) {
			return nil, errors.New("fail")
		}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBackupManager{createBackupFunc: tt.mockFunc}
			handler := setupBackupTestHandler(t, mock)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/quick", nil)
			w := httptest.NewRecorder()
			handler.HandleQuickBackup(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestRestoreOptions tests that all restore options are captured
func TestRestoreOptions(t *testing.T) {
	t.Parallel()

	var captured backup.RestoreOptions
	mock := &mockBackupManager{
		restoreFromBackupFunc: func(ctx context.Context, id string, opts backup.RestoreOptions) (*backup.RestoreResult, error) {
			captured = opts
			return &backup.RestoreResult{Success: true}, nil
		},
	}
	handler := setupBackupTestHandler(t, mock)

	body := `{
		"validate_only": true,
		"create_pre_restore_backup": true,
		"restore_database": true,
		"restore_config": true,
		"force_restore": true,
		"verify_after_restore": true
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backups/restore?id=123", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.HandleRestoreBackup(w, req)

	if !captured.ValidateOnly || !captured.CreatePreRestoreBackup ||
		!captured.RestoreDatabase || !captured.RestoreConfig ||
		!captured.ForceRestore || !captured.VerifyAfterRestore {
		t.Error("not all restore options captured")
	}
}

// TestRetentionPolicyAllFields tests all retention policy fields
func TestRetentionPolicyAllFields(t *testing.T) {
	t.Parallel()

	var captured backup.RetentionPolicy
	mock := &mockBackupManager{
		setRetentionPolicyFunc: func(p backup.RetentionPolicy) error {
			captured = p
			return nil
		},
	}
	handler := setupBackupTestHandler(t, mock)

	body := `{
		"min_count": 3,
		"max_count": 50,
		"max_age_days": 90,
		"keep_recent_hours": 24,
		"keep_daily_for_days": 7,
		"keep_weekly_for_weeks": 4,
		"keep_monthly_for_months": 6
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/backup/retention", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.HandleSetRetentionPolicy(w, req)

	if captured.MinCount != 3 || captured.MaxCount != 50 || captured.MaxAgeDays != 90 ||
		captured.KeepRecentHours != 24 || captured.KeepDailyForDays != 7 ||
		captured.KeepWeeklyForWeeks != 4 || captured.KeepMonthlyForMonths != 6 {
		t.Error("not all policy fields captured")
	}
}

// ========================================
// Schedule Config Tests (0% coverage)
// ========================================

func TestHandleGetScheduleConfig_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	mock := &mockBackupManager{
		getScheduleConfigFunc: func() backup.ScheduleConfig {
			return backup.ScheduleConfig{}
		},
	}
	handler := setupBackupTestHandler(t, mock)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/backup/schedule", nil)
			w := httptest.NewRecorder()
			handler.HandleGetScheduleConfig(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

func TestHandleGetScheduleConfig_NoBackupManager(t *testing.T) {
	t.Parallel()

	handler := setupBackupTestHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/schedule", nil)
	w := httptest.NewRecorder()
	handler.HandleGetScheduleConfig(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleGetScheduleConfig_Success(t *testing.T) {
	t.Parallel()

	mock := &mockBackupManager{
		getScheduleConfigFunc: func() backup.ScheduleConfig {
			return backup.ScheduleConfig{
				Enabled:       true,
				Interval:      24 * time.Hour,
				PreferredHour: 3,
				BackupType:    backup.TypeFull,
				PreSyncBackup: true,
			}
		},
	}
	handler := setupBackupTestHandler(t, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/schedule", nil)
	w := httptest.NewRecorder()
	handler.HandleGetScheduleConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got %v", response.Status)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if data["enabled"] != true {
		t.Errorf("Expected enabled=true, got %v", data["enabled"])
	}
	if data["interval_hours"].(float64) != 24 {
		t.Errorf("Expected interval_hours=24, got %v", data["interval_hours"])
	}
	if data["preferred_hour"].(float64) != 3 {
		t.Errorf("Expected preferred_hour=3, got %v", data["preferred_hour"])
	}
}

func TestHandleSetScheduleConfig_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	mock := &mockBackupManager{
		setScheduleConfigFunc: func(_ context.Context, _ backup.ScheduleConfig) error {
			return nil
		},
	}
	handler := setupBackupTestHandler(t, mock)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/backup/schedule", nil)
			w := httptest.NewRecorder()
			handler.HandleSetScheduleConfig(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

func TestHandleSetScheduleConfig_NoBackupManager(t *testing.T) {
	t.Parallel()

	handler := setupBackupTestHandler(t, nil)

	body := `{"enabled": true, "interval_hours": 24}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/backup/schedule", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.HandleSetScheduleConfig(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleSetScheduleConfig_InvalidJSON(t *testing.T) {
	t.Parallel()

	mock := &mockBackupManager{}
	handler := setupBackupTestHandler(t, mock)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/backup/schedule", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	handler.HandleSetScheduleConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleSetScheduleConfig_Success(t *testing.T) {
	t.Parallel()

	var captured backup.ScheduleConfig
	mock := &mockBackupManager{
		setScheduleConfigFunc: func(_ context.Context, schedule backup.ScheduleConfig) error {
			captured = schedule
			return nil
		},
	}
	handler := setupBackupTestHandler(t, mock)

	body := `{
		"enabled": true,
		"interval_hours": 12,
		"preferred_hour": 2,
		"backup_type": "full",
		"pre_sync_backup": true
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/backup/schedule", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.HandleSetScheduleConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !captured.Enabled {
		t.Error("Expected enabled=true")
	}
	if captured.Interval != 12*time.Hour {
		t.Errorf("Expected interval 12h, got %v", captured.Interval)
	}
	if captured.PreferredHour != 2 {
		t.Errorf("Expected preferred_hour=2, got %d", captured.PreferredHour)
	}
	if captured.BackupType != backup.TypeFull {
		t.Errorf("Expected backup_type=full, got %v", captured.BackupType)
	}
	if !captured.PreSyncBackup {
		t.Error("Expected pre_sync_backup=true")
	}
}

func TestHandleSetScheduleConfig_Error(t *testing.T) {
	t.Parallel()

	mock := &mockBackupManager{
		setScheduleConfigFunc: func(_ context.Context, _ backup.ScheduleConfig) error {
			return errors.New("invalid schedule configuration")
		},
	}
	handler := setupBackupTestHandler(t, mock)

	body := `{"enabled": true, "interval_hours": 24}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/backup/schedule", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.HandleSetScheduleConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleTriggerScheduledBackup_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	mock := &mockBackupManager{
		triggerScheduledBackupFunc: func(_ context.Context) (*backup.Backup, error) {
			return &backup.Backup{ID: "test-123"}, nil
		},
	}
	handler := setupBackupTestHandler(t, mock)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/backup/schedule/trigger", nil)
			w := httptest.NewRecorder()
			handler.HandleTriggerScheduledBackup(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

func TestHandleTriggerScheduledBackup_NoBackupManager(t *testing.T) {
	t.Parallel()

	handler := setupBackupTestHandler(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/schedule/trigger", nil)
	w := httptest.NewRecorder()
	handler.HandleTriggerScheduledBackup(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleTriggerScheduledBackup_Success(t *testing.T) {
	t.Parallel()

	mock := &mockBackupManager{
		triggerScheduledBackupFunc: func(_ context.Context) (*backup.Backup, error) {
			return &backup.Backup{
				ID:        "backup-triggered-123",
				Type:      backup.TypeFull,
				Notes:     "Scheduled backup triggered manually",
				CreatedAt: time.Now(),
			}, nil
		},
	}
	handler := setupBackupTestHandler(t, mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/schedule/trigger", nil)
	w := httptest.NewRecorder()
	handler.HandleTriggerScheduledBackup(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got %v", response.Status)
	}
}

func TestHandleTriggerScheduledBackup_Error(t *testing.T) {
	t.Parallel()

	mock := &mockBackupManager{
		triggerScheduledBackupFunc: func(_ context.Context) (*backup.Backup, error) {
			return nil, errors.New("backup failed: disk full")
		},
	}
	handler := setupBackupTestHandler(t, mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/schedule/trigger", nil)
	w := httptest.NewRecorder()
	handler.HandleTriggerScheduledBackup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestBuildScheduleResponseData(t *testing.T) {
	t.Parallel()

	schedule := backup.ScheduleConfig{
		Enabled:       true,
		Interval:      48 * time.Hour,
		PreferredHour: 4,
		BackupType:    backup.TypeIncremental,
		PreSyncBackup: false,
	}

	result := buildScheduleResponseData(schedule)

	if result["enabled"] != true {
		t.Errorf("Expected enabled=true, got %v", result["enabled"])
	}
	if result["interval_hours"] != 48 {
		t.Errorf("Expected interval_hours=48, got %v", result["interval_hours"])
	}
	if result["preferred_hour"] != 4 {
		t.Errorf("Expected preferred_hour=4, got %v", result["preferred_hour"])
	}
	if result["backup_type"] != "incremental" {
		t.Errorf("Expected backup_type=incremental, got %v", result["backup_type"])
	}
	if result["pre_sync_backup"] != false {
		t.Errorf("Expected pre_sync_backup=false, got %v", result["pre_sync_backup"])
	}
}
