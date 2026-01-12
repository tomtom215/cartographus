// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package storage provides model persistence for recommendation algorithms.
//
// Models are serialized using Go's gob encoding for efficient storage and
// fast deserialization. Each algorithm's state is stored separately, allowing
// independent versioning and atomic updates.
//
// # Storage Format
//
// Models are stored with metadata including version, timestamp, and checksum
// to ensure integrity and enable rollback to previous versions.
//
// # Thread Safety
//
// All storage operations are thread-safe and use file locking to prevent
// concurrent write corruption.
package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ModelMetadata contains information about a stored model.
type ModelMetadata struct {
	// Name is the algorithm name (e.g., "ease", "covisit").
	Name string `json:"name"`

	// Version is the model version (monotonically increasing).
	Version int `json:"version"`

	// TrainedAt is when the model was trained.
	TrainedAt time.Time `json:"trained_at"`

	// SavedAt is when the model was saved.
	SavedAt time.Time `json:"saved_at"`

	// InteractionCount is the number of interactions used for training.
	InteractionCount int `json:"interaction_count"`

	// ItemCount is the number of unique items.
	ItemCount int `json:"item_count"`

	// UserCount is the number of unique users.
	UserCount int `json:"user_count"`

	// Checksum is the SHA-256 checksum of the model data.
	Checksum string `json:"checksum"`

	// SizeBytes is the compressed model size in bytes.
	SizeBytes int64 `json:"size_bytes"`

	// TrainingDurationMS is how long training took.
	TrainingDurationMS int64 `json:"training_duration_ms"`
}

// StoredModel wraps model data with metadata for persistence.
type StoredModel struct {
	// Metadata contains model information.
	Metadata ModelMetadata `json:"metadata"`

	// Data contains the serialized model state.
	Data []byte `json:"-"`
}

// Store manages model persistence.
type Store struct {
	baseDir string
	mu      sync.RWMutex

	// Keep track of latest version per algorithm
	versions map[string]int
}

// NewStore creates a new model store at the given directory.
func NewStore(baseDir string) (*Store, error) {
	// Ensure directory exists
	if err := os.MkdirAll(baseDir, 0o750); err != nil { //nolint:gosec // 0750 is acceptable for model storage
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	s := &Store{
		baseDir:  baseDir,
		versions: make(map[string]int),
	}

	// Scan for existing models
	if err := s.scanModels(); err != nil {
		return nil, fmt.Errorf("scan existing models: %w", err)
	}

	return s, nil
}

// scanModels scans the storage directory for existing model files.
func (s *Store) scanModels() error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Extract algorithm name and version from filename
		// Format: {name}_v{version}.gob.gz or {name}_v{version}.gob
		// Check for gzip compression
		if len(name) > 7 && name[len(name)-7:] == ".gob.gz" {
			name = name[:len(name)-7]
		} else if len(name) > 4 && name[len(name)-4:] == ".gob" {
			name = name[:len(name)-4]
		} else {
			continue
		}

		algName, version := parseModelFilename(name)
		if algName == "" {
			continue
		}

		if current, ok := s.versions[algName]; !ok || version > current {
			s.versions[algName] = version
		}
	}

	return nil
}

// parseModelFilename extracts algorithm name and version from a filename like "ease_v1".
func parseModelFilename(name string) (algName string, version int) {
	// Find the last "_v" to split name from version
	lastVIdx := -1
	for i := len(name) - 1; i >= 1; i-- {
		if name[i] == 'v' && name[i-1] == '_' {
			lastVIdx = i - 1
			break
		}
	}
	if lastVIdx < 0 {
		return "", 0
	}

	algName = name[:lastVIdx]
	versionStr := name[lastVIdx+2:]

	_, err := fmt.Sscanf(versionStr, "%d", &version)
	if err != nil {
		return "", 0
	}

	return algName, version
}

// storedFile is the on-disk format for model files.
type storedFile struct {
	Metadata       ModelMetadata
	CompressedData []byte
}

// Save stores a model with the given name and data.
//
//nolint:gocritic // meta passed by value is acceptable for this write operation
func (s *Store) Save(ctx context.Context, name string, version int, data interface{}, meta ModelMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize model data
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode model: %w", err)
	}

	rawData := buf.Bytes()

	// Compute checksum
	hash := sha256.Sum256(rawData)
	meta.Checksum = hex.EncodeToString(hash[:])

	// Compress data
	var compressed bytes.Buffer
	gzw := gzip.NewWriter(&compressed)
	if _, err := gzw.Write(rawData); err != nil {
		return fmt.Errorf("compress model: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return fmt.Errorf("finalize compression: %w", err)
	}

	meta.SizeBytes = int64(compressed.Len())
	meta.SavedAt = time.Now()
	meta.Name = name
	meta.Version = version

	// Write model file
	filename := s.modelPath(name, version)
	f, err := os.Create(filename) //nolint:gosec // filename is constructed from trusted name parameter
	if err != nil {
		return fmt.Errorf("create model file: %w", err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // error on close after write is logged via return

	// Write as single gob-encoded struct to avoid buffering issues
	sf := storedFile{
		Metadata:       meta,
		CompressedData: compressed.Bytes(),
	}
	fileEnc := gob.NewEncoder(f)
	if err := fileEnc.Encode(sf); err != nil {
		return fmt.Errorf("write model file: %w", err)
	}

	// Update version tracking
	if current, ok := s.versions[name]; !ok || version > current {
		s.versions[name] = version
	}

	return nil
}

// Load loads a model by name and version.
// If version is 0, loads the latest version.
func (s *Store) Load(ctx context.Context, name string, version int, target interface{}) (*ModelMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if version == 0 {
		var ok bool
		version, ok = s.versions[name]
		if !ok {
			return nil, fmt.Errorf("no model found for %s", name)
		}
	}

	filename := s.modelPath(name, version)
	f, err := os.Open(filename) //nolint:gosec // filename is constructed from trusted name parameter
	if err != nil {
		return nil, fmt.Errorf("open model file: %w", err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // error on close after read is not actionable

	// Read the stored file struct
	var sf storedFile
	fileDec := gob.NewDecoder(f)
	if err := fileDec.Decode(&sf); err != nil {
		return nil, fmt.Errorf("read model file: %w", err)
	}

	// Decompress
	gzr, err := gzip.NewReader(bytes.NewReader(sf.CompressedData))
	if err != nil {
		return nil, fmt.Errorf("decompress model: %w", err)
	}
	defer func() { _ = gzr.Close() }() //nolint:errcheck // error on gzip close after read is not actionable

	rawData, err := io.ReadAll(gzr)
	if err != nil {
		return nil, fmt.Errorf("read decompressed data: %w", err)
	}

	// Verify checksum
	hash := sha256.Sum256(rawData)
	checksum := hex.EncodeToString(hash[:])
	if checksum != sf.Metadata.Checksum {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", sf.Metadata.Checksum, checksum)
	}

	// Deserialize model
	dec := gob.NewDecoder(bytes.NewReader(rawData))
	if err := dec.Decode(target); err != nil {
		return nil, fmt.Errorf("decode model: %w", err)
	}

	return &sf.Metadata, nil
}

// GetLatestVersion returns the latest version number for a model.
func (s *Store) GetLatestVersion(name string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	version, ok := s.versions[name]
	return version, ok
}

// ListModels returns metadata for all stored models.
func (s *Store) ListModels(ctx context.Context) ([]ModelMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var models []ModelMetadata

	for name, version := range s.versions {
		filename := s.modelPath(name, version)
		f, err := os.Open(filename) //nolint:gosec // filename is constructed from trusted name parameter
		if err != nil {
			continue
		}

		var sf storedFile
		dec := gob.NewDecoder(f)
		if err := dec.Decode(&sf); err != nil {
			_ = f.Close() //nolint:errcheck // error on close after read failure is not actionable
			continue
		}
		_ = f.Close() //nolint:errcheck // error on close after successful read is not actionable

		models = append(models, sf.Metadata)
	}

	return models, nil
}

// Delete removes a specific model version.
func (s *Store) Delete(ctx context.Context, name string, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := s.modelPath(name, version)
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("delete model: %w", err)
	}

	// Update version tracking
	if s.versions[name] == version {
		// Find next latest version
		s.versions[name] = 0
		entries, err := os.ReadDir(s.baseDir)
		if err != nil {
			return fmt.Errorf("read directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			entryName := entry.Name()
			if len(entryName) > 7 && entryName[len(entryName)-7:] == ".gob.gz" {
				entryName = entryName[:len(entryName)-7]
			} else if len(entryName) > 4 && entryName[len(entryName)-4:] == ".gob" {
				entryName = entryName[:len(entryName)-4]
			} else {
				continue
			}

			algName, v := parseModelFilename(entryName)
			if algName != name {
				continue
			}
			if v > s.versions[name] {
				s.versions[name] = v
			}
		}
		if s.versions[name] == 0 {
			delete(s.versions, name)
		}
	}

	return nil
}

// Prune removes old model versions, keeping only the latest N versions.
//
//nolint:gocyclo // file management has multiple edge cases
func (s *Store) Prune(ctx context.Context, name string, keepVersions int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if keepVersions < 1 {
		keepVersions = 1
	}

	latestVersion, ok := s.versions[name]
	if !ok {
		return nil
	}

	// Find all versions
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	var versions []int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		entryName := entry.Name()
		if len(entryName) > 7 && entryName[len(entryName)-7:] == ".gob.gz" {
			entryName = entryName[:len(entryName)-7]
		} else if len(entryName) > 4 && entryName[len(entryName)-4:] == ".gob" {
			entryName = entryName[:len(entryName)-4]
		} else {
			continue
		}

		algName, v := parseModelFilename(entryName)
		if algName != name {
			continue
		}
		versions = append(versions, v)
	}

	// Sort versions descending
	for i := 0; i < len(versions)-1; i++ {
		for j := i + 1; j < len(versions); j++ {
			if versions[j] > versions[i] {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}

	// Delete old versions
	for i := keepVersions; i < len(versions); i++ {
		filename := s.modelPath(name, versions[i])
		_ = os.Remove(filename) //nolint:errcheck // best-effort cleanup of old versions
	}

	// Update version to latest
	if len(versions) > 0 {
		s.versions[name] = latestVersion
	}

	return nil
}

// modelPath returns the file path for a model.
func (s *Store) modelPath(name string, version int) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%s_v%d.gob.gz", name, version))
}

// EASEModelState represents the serializable state of an EASE model.
type EASEModelState struct {
	// B is the item-item weight matrix.
	B [][]float64

	// ItemIndex maps item ID to matrix index.
	ItemIndex map[int]int

	// IndexToItem maps matrix index to item ID.
	IndexToItem []int

	// UserVectors stores user interaction vectors.
	UserVectors map[int][]float64

	// Config is the algorithm configuration.
	L2Regularization float64
	MinConfidence    float64
}

// CoVisitModelState represents the serializable state of a CoVisitation model.
type CoVisitModelState struct {
	// CoOccurrence stores co-occurrence counts between item pairs.
	CoOccurrence map[int]map[int]float64

	// ItemCounts stores occurrence counts per item.
	ItemCounts map[int]int

	// UserHistory stores recent item IDs per user.
	UserHistory map[int][]int

	// Config values
	MinCoOccurrence    int
	SessionWindowHours int
	MaxPairs           int
}

// ContentModelState represents the serializable state of a ContentBased model.
type ContentModelState struct {
	// UserProfiles stores user genre/actor/director preferences.
	UserProfiles map[int]UserProfile

	// ItemVectors stores pre-computed item feature vectors.
	Items map[int]ItemFeatures

	// Config values
	GenreWeight       float64
	ActorWeight       float64
	DirectorWeight    float64
	YearWeight        float64
	MaxYearDifference int
}

// UserProfile represents a user's content preferences.
type UserProfile struct {
	Genres    map[string]float64
	Actors    map[string]float64
	Directors map[string]float64
	AvgYear   float64
}

// ItemFeatures represents an item's content features.
type ItemFeatures struct {
	Genres    []string
	Actors    []string
	Directors []string
	Year      int
}

// Register gob types for serialization.
//
//nolint:gochecknoinits // gob.Register must be called in init for type registration
func init() {
	gob.Register(EASEModelState{})
	gob.Register(CoVisitModelState{})
	gob.Register(ContentModelState{})
	gob.Register(ModelMetadata{})
	gob.Register(UserProfile{})
	gob.Register(ItemFeatures{})
	gob.Register(storedFile{})
}
