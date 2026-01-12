// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !wal || !nats

package eventprocessor

// WALStore is a stub when WAL is not enabled.
// Use DuckDBStore directly for non-WAL builds.
type WALStore struct{}

// WALStoreStats is a stub.
type WALStoreStats struct{}
