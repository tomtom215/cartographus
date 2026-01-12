// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

// Package eventprocessor provides stub DLQ persistence for non-NATS builds.
package eventprocessor

import (
	"context"
	"database/sql"
	"time"
)

// DLQStore defines the interface for DLQ persistence backends.
type DLQStore interface {
	Save(ctx context.Context, entry *DLQEntry) error
	Get(ctx context.Context, eventID string) (*DLQEntry, error)
	Update(ctx context.Context, entry *DLQEntry) error
	Delete(ctx context.Context, eventID string) error
	List(ctx context.Context) ([]*DLQEntry, error)
	DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error)
	Count(ctx context.Context) (int64, error)
}

// DuckDBDLQStore is a stub for non-NATS builds.
type DuckDBDLQStore struct{}

// NewDuckDBDLQStore creates a stub DLQ store.
func NewDuckDBDLQStore(db *sql.DB) *DuckDBDLQStore {
	return &DuckDBDLQStore{}
}

// CreateTable is a no-op for stub.
func (s *DuckDBDLQStore) CreateTable(ctx context.Context) error {
	return nil
}

// Save is a no-op for stub.
func (s *DuckDBDLQStore) Save(ctx context.Context, entry *DLQEntry) error {
	return ErrNATSNotEnabled
}

// Get is a no-op for stub.
func (s *DuckDBDLQStore) Get(ctx context.Context, eventID string) (*DLQEntry, error) {
	return nil, ErrNATSNotEnabled
}

// Update is a no-op for stub.
func (s *DuckDBDLQStore) Update(ctx context.Context, entry *DLQEntry) error {
	return ErrNATSNotEnabled
}

// Delete is a no-op for stub.
func (s *DuckDBDLQStore) Delete(ctx context.Context, eventID string) error {
	return ErrNATSNotEnabled
}

// List is a no-op for stub.
func (s *DuckDBDLQStore) List(ctx context.Context) ([]*DLQEntry, error) {
	return nil, ErrNATSNotEnabled
}

// DeleteExpired is a no-op for stub.
func (s *DuckDBDLQStore) DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	return 0, ErrNATSNotEnabled
}

// Count is a no-op for stub.
func (s *DuckDBDLQStore) Count(ctx context.Context) (int64, error) {
	return 0, ErrNATSNotEnabled
}

// PersistentDLQHandler is a stub for non-NATS builds.
type PersistentDLQHandler struct {
	*DLQHandler
}

// NewPersistentDLQHandler creates a stub persistent handler.
func NewPersistentDLQHandler(cfg DLQConfig, store DLQStore) (*PersistentDLQHandler, error) {
	return nil, ErrNATSNotEnabled
}
