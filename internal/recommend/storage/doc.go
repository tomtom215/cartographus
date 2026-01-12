// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package storage provides model persistence for recommendation algorithms.
//
// This package handles the serialization, compression, and storage of trained
// recommendation models. It enables model persistence across application
// restarts, version management for A/B testing, and model rollback capabilities.
//
// # Overview
//
// The storage system provides:
//   - Gob serialization for efficient Go type encoding
//   - Gzip compression to reduce storage footprint
//   - SHA-256 checksums for data integrity verification
//   - Version tracking for model lineage
//   - Automatic cleanup of old model versions
//
// # Storage Format
//
// Models are stored with metadata in a gob-encoded, gzip-compressed format:
//
//	filename: {algorithm_name}_v{version}.gob.gz
//
//	structure:
//	  - Metadata (ModelMetadata)
//	  - CompressedData (gzip-compressed gob-encoded model state)
//
// # Usage Example
//
// Saving a trained model:
//
//	store, err := storage.NewStore("/data/models")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save EASE model state
//	state := storage.EASEModelState{
//	    B:           easeModel.B,
//	    ItemIndex:   easeModel.ItemIndex,
//	    IndexToItem: easeModel.IndexToItem,
//	    UserVectors: easeModel.UserVectors,
//	}
//
//	meta := storage.ModelMetadata{
//	    Name:             "ease",
//	    InteractionCount: len(trainingData.Interactions),
//	    ItemCount:        len(trainingData.Items),
//	    TrainedAt:        time.Now(),
//	}
//
//	err = store.Save(ctx, "ease", 1, state, meta)
//
// Loading a model:
//
//	var state storage.EASEModelState
//	meta, err := store.Load(ctx, "ease", 0, &state)  // 0 = latest version
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	log.Printf("Loaded model v%d trained at %v", meta.Version, meta.TrainedAt)
//
// # Model State Types
//
// Pre-defined state types for each algorithm:
//
// EASEModelState:
//   - B: Item-item weight matrix
//   - ItemIndex: Item ID to matrix index mapping
//   - IndexToItem: Matrix index to item ID mapping
//   - UserVectors: User interaction vectors
//
// CoVisitModelState:
//   - CoOccurrence: Item pair co-occurrence counts
//   - ItemCounts: Per-item occurrence counts
//   - UserHistory: Recent items per user
//
// ContentModelState:
//   - UserProfiles: User genre/actor/director preferences
//   - Items: Pre-computed item feature vectors
//
// # Metadata
//
// Each saved model includes rich metadata:
//
//	type ModelMetadata struct {
//	    Name               string    // Algorithm name
//	    Version            int       // Model version (monotonically increasing)
//	    TrainedAt          time.Time // Training timestamp
//	    SavedAt            time.Time // Save timestamp
//	    InteractionCount   int       // Training data size
//	    ItemCount          int       // Unique items
//	    UserCount          int       // Unique users
//	    Checksum           string    // SHA-256 of uncompressed data
//	    SizeBytes          int64     // Compressed size
//	    TrainingDurationMS int64     // Training time
//	}
//
// # Version Management
//
// The store tracks the latest version of each algorithm:
//
//	// Get latest version
//	version, ok := store.GetLatestVersion("ease")
//
//	// Load specific version
//	meta, err := store.Load(ctx, "ease", 5, &state)
//
//	// Load latest version (version=0)
//	meta, err := store.Load(ctx, "ease", 0, &state)
//
// # Cleanup and Pruning
//
// Remove old model versions to manage disk space:
//
//	// Delete specific version
//	err := store.Delete(ctx, "ease", 2)
//
//	// Keep only latest 3 versions
//	err := store.Prune(ctx, "ease", 3)
//
// # Data Integrity
//
// Models are validated on load using SHA-256 checksums:
//
//  1. Decompress gzip data
//  2. Compute SHA-256 of decompressed data
//  3. Compare with stored checksum
//  4. Return error if mismatch
//
// This prevents loading corrupted models that could produce incorrect
// recommendations.
//
// # Directory Structure
//
//	/data/models/
//	  ease_v1.gob.gz
//	  ease_v2.gob.gz
//	  ease_v3.gob.gz     <- latest
//	  covisit_v1.gob.gz
//	  content_v1.gob.gz
//
// # Thread Safety
//
// All store operations are thread-safe:
//   - Save: Acquires write lock
//   - Load: Acquires read lock
//   - Multiple loads can run concurrently
//   - Saves block other operations on the same algorithm
//
// # Performance
//
// Typical performance characteristics:
//   - Save: 100-500ms for medium models (10K items)
//   - Load: 50-200ms for medium models
//   - Compression ratio: 60-80% size reduction
//   - Checksum validation: <10ms
//
// # See Also
//
//   - internal/recommend/algorithms: Model implementations
//   - internal/recommend: Engine using stored models
//   - encoding/gob: Go serialization format
package storage
