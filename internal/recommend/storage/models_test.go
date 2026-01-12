// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "creates directory if not exists",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "new_dir")
			},
			wantErr: false,
		},
		{
			name: "uses existing directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			store, err := NewStore(dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && store == nil {
				t.Error("NewStore() returned nil store without error")
			}
		})
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// Create test data
	testData := EASEModelState{
		B: [][]float64{
			{0, 0.5, 0.3},
			{0.5, 0, 0.7},
			{0.3, 0.7, 0},
		},
		ItemIndex:        map[int]int{100: 0, 200: 1, 300: 2},
		IndexToItem:      []int{100, 200, 300},
		L2Regularization: 500.0,
		MinConfidence:    0.1,
	}

	meta := ModelMetadata{
		Name:             "ease",
		Version:          1,
		TrainedAt:        time.Now(),
		InteractionCount: 1000,
		ItemCount:        100,
		UserCount:        50,
	}

	// Save model
	if err := store.Save(ctx, "ease", 1, testData, meta); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load model
	var loaded EASEModelState
	loadedMeta, err := store.Load(ctx, "ease", 1, &loaded)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify metadata
	if loadedMeta.Name != "ease" {
		t.Errorf("Name = %s, want ease", loadedMeta.Name)
	}
	if loadedMeta.Version != 1 {
		t.Errorf("Version = %d, want 1", loadedMeta.Version)
	}
	if loadedMeta.InteractionCount != 1000 {
		t.Errorf("InteractionCount = %d, want 1000", loadedMeta.InteractionCount)
	}
	if loadedMeta.Checksum == "" {
		t.Error("Checksum should not be empty")
	}
	if loadedMeta.SizeBytes == 0 {
		t.Error("SizeBytes should not be zero")
	}

	// Verify data
	if len(loaded.B) != 3 {
		t.Errorf("len(B) = %d, want 3", len(loaded.B))
	}
	if loaded.L2Regularization != 500.0 {
		t.Errorf("L2Regularization = %f, want 500.0", loaded.L2Regularization)
	}
	if len(loaded.ItemIndex) != 3 {
		t.Errorf("len(ItemIndex) = %d, want 3", len(loaded.ItemIndex))
	}
}

func TestStore_LoadLatest(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// Save multiple versions
	for v := 1; v <= 3; v++ {
		data := EASEModelState{
			L2Regularization: float64(v * 100),
		}
		meta := ModelMetadata{Version: v}
		if err := store.Save(ctx, "ease", v, data, meta); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Load latest (version 0 means latest)
	var loaded EASEModelState
	loadedMeta, err := store.Load(ctx, "ease", 0, &loaded)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loadedMeta.Version != 3 {
		t.Errorf("Version = %d, want 3 (latest)", loadedMeta.Version)
	}
	if loaded.L2Regularization != 300.0 {
		t.Errorf("L2Regularization = %f, want 300.0", loaded.L2Regularization)
	}
}

func TestStore_GetLatestVersion(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// No model saved yet
	_, ok := store.GetLatestVersion("ease")
	if ok {
		t.Error("GetLatestVersion() should return false for missing model")
	}

	// Save a model
	data := EASEModelState{}
	meta := ModelMetadata{Version: 5}
	if err := store.Save(ctx, "ease", 5, data, meta); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	version, ok := store.GetLatestVersion("ease")
	if !ok {
		t.Error("GetLatestVersion() should return true after saving")
	}
	if version != 5 {
		t.Errorf("Version = %d, want 5", version)
	}
}

func TestStore_ListModels(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// Save models for different algorithms
	algorithms := []string{"ease", "covisit", "content"}
	for _, alg := range algorithms {
		data := EASEModelState{}
		meta := ModelMetadata{
			Name:             alg,
			Version:          1,
			InteractionCount: 100,
		}
		if err := store.Save(ctx, alg, 1, data, meta); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	models, err := store.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(models) != 3 {
		t.Errorf("len(models) = %d, want 3", len(models))
	}

	// Verify all algorithms are present
	found := make(map[string]bool)
	for _, m := range models {
		found[m.Name] = true
	}
	for _, alg := range algorithms {
		if !found[alg] {
			t.Errorf("algorithm %s not found in list", alg)
		}
	}
}

func TestStore_Delete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// Save model
	data := EASEModelState{}
	meta := ModelMetadata{Version: 1}
	if err := store.Save(ctx, "ease", 1, data, meta); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify it exists
	_, ok := store.GetLatestVersion("ease")
	if !ok {
		t.Error("model should exist before delete")
	}

	// Delete it
	if err := store.Delete(ctx, "ease", 1); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, ok = store.GetLatestVersion("ease")
	if ok {
		t.Error("model should not exist after delete")
	}

	// Loading should fail
	var loaded EASEModelState
	_, err = store.Load(ctx, "ease", 1, &loaded)
	if err == nil {
		t.Error("Load() should fail after delete")
	}
}

func TestStore_Prune(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// Save 5 versions
	for v := 1; v <= 5; v++ {
		data := EASEModelState{L2Regularization: float64(v)}
		meta := ModelMetadata{Version: v}
		if err := store.Save(ctx, "ease", v, data, meta); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Prune, keeping only 2 versions
	if err := store.Prune(ctx, "ease", 2); err != nil {
		t.Fatalf("Prune() error = %v", err)
	}

	// Verify latest version is still there
	version, ok := store.GetLatestVersion("ease")
	if !ok {
		t.Error("latest version should still exist")
	}
	if version != 5 {
		t.Errorf("latest version = %d, want 5", version)
	}

	// Verify old versions are deleted
	var loaded EASEModelState
	for v := 1; v <= 3; v++ {
		_, err := store.Load(ctx, "ease", v, &loaded)
		if err == nil {
			t.Errorf("version %d should have been pruned", v)
		}
	}

	// Verify kept versions still work
	for v := 4; v <= 5; v++ {
		_, err := store.Load(ctx, "ease", v, &loaded)
		if err != nil {
			t.Errorf("version %d should still exist: %v", v, err)
		}
	}
}

func TestStore_ChecksumValidation(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// Save model
	data := EASEModelState{L2Regularization: 500.0}
	meta := ModelMetadata{Version: 1}
	if err := store.Save(ctx, "ease", 1, data, meta); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Corrupt the file
	filename := filepath.Join(dir, "ease_v1.gob.gz")
	f, err := os.OpenFile(filename, os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	// Write garbage at offset to corrupt data while keeping metadata intact
	f.Seek(100, 0)
	f.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Close()

	// Load should fail due to checksum mismatch or decompression error
	var loaded EASEModelState
	_, err = store.Load(ctx, "ease", 1, &loaded)
	if err == nil {
		t.Error("Load() should fail with corrupted data")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// Concurrent saves
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(v int) {
			data := EASEModelState{L2Regularization: float64(v)}
			meta := ModelMetadata{Version: v}
			store.Save(ctx, "ease", v, data, meta)
			done <- true
		}(i + 1)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify latest version is accessible
	version, ok := store.GetLatestVersion("ease")
	if !ok {
		t.Error("should have at least one version")
	}
	if version < 1 || version > 10 {
		t.Errorf("version = %d, want between 1 and 10", version)
	}
}

func TestStore_DifferentModelTypes(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()

	// Save CoVisit model
	covisitData := CoVisitModelState{
		CoOccurrence: map[int]map[int]float64{
			100: {101: 0.5, 102: 0.3},
		},
		ItemCounts:      map[int]int{100: 10, 101: 5, 102: 3},
		MinCoOccurrence: 2,
	}
	if err := store.Save(ctx, "covisit", 1, covisitData, ModelMetadata{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Save Content model
	contentData := ContentModelState{
		UserProfiles: map[int]UserProfile{
			1: {
				Genres:  map[string]float64{"Action": 0.8, "Comedy": 0.3},
				AvgYear: 2020,
			},
		},
		GenreWeight: 0.5,
	}
	if err := store.Save(ctx, "content", 1, contentData, ModelMetadata{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load and verify each type
	var loadedCovisit CoVisitModelState
	if _, err := store.Load(ctx, "covisit", 1, &loadedCovisit); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loadedCovisit.MinCoOccurrence != 2 {
		t.Errorf("MinCoOccurrence = %d, want 2", loadedCovisit.MinCoOccurrence)
	}

	var loadedContent ContentModelState
	if _, err := store.Load(ctx, "content", 1, &loadedContent); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loadedContent.GenreWeight != 0.5 {
		t.Errorf("GenreWeight = %f, want 0.5", loadedContent.GenreWeight)
	}
}
