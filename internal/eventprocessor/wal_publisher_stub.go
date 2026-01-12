// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !wal || !nats

package eventprocessor

// WAL stubs for builds without WAL support.
// These types are placeholders that allow code to compile without
// the wal build tag. They should never be instantiated at runtime.

// WALEnabledPublisher stub for non-WAL builds.
// When WAL is disabled, use SyncEventPublisher directly instead.
type WALEnabledPublisher struct{}

// Note: NewWALEnabledPublisher is not provided in stub builds.
// Code should check for WAL availability at compile time using build tags.
