// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

// Package eventprocessor provides stub implementations for non-NATS builds.
package eventprocessor

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Checkpoint represents a saved replay position.
type Checkpoint struct {
	ID             int64
	ConsumerName   string
	StreamName     string
	LastSequence   uint64
	LastTimestamp  time.Time
	ProcessedCount int64
	ErrorCount     int64
	Status         string
	ReplayMode     string
	StartSequence  uint64
	StartTime      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CheckpointStore is a stub for non-NATS builds.
type CheckpointStore struct{}

// NewCheckpointStore returns a stub store.
func NewCheckpointStore(_ *sql.DB) *CheckpointStore {
	return &CheckpointStore{}
}

// CreateTable is a stub.
func (s *CheckpointStore) CreateTable(_ context.Context) error {
	return errors.New("checkpoint store requires NATS build tag")
}

// Save is a stub.
func (s *CheckpointStore) Save(_ context.Context, _ *Checkpoint) error {
	return errors.New("checkpoint store requires NATS build tag")
}

// Get is a stub.
func (s *CheckpointStore) Get(_ context.Context, _, _ string) (*Checkpoint, error) {
	return nil, errors.New("checkpoint store requires NATS build tag")
}

// GetByID is a stub.
func (s *CheckpointStore) GetByID(_ context.Context, _ int64) (*Checkpoint, error) {
	return nil, errors.New("checkpoint store requires NATS build tag")
}

// List is a stub.
func (s *CheckpointStore) List(_ context.Context, _ string) ([]*Checkpoint, error) {
	return nil, errors.New("checkpoint store requires NATS build tag")
}

// Delete is a stub.
func (s *CheckpointStore) Delete(_ context.Context, _ int64) error {
	return errors.New("checkpoint store requires NATS build tag")
}

// DeleteOld is a stub.
func (s *CheckpointStore) DeleteOld(_ context.Context, _ time.Duration) (int64, error) {
	return 0, errors.New("checkpoint store requires NATS build tag")
}

// GetLastForStream is a stub.
func (s *CheckpointStore) GetLastForStream(_ context.Context, _ string) (*Checkpoint, error) {
	return nil, errors.New("checkpoint store requires NATS build tag")
}
