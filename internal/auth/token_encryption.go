// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including token encryption.
// ADR-0015: Zero Trust Authentication & Authorization
// Phase 4D.2: Token Encryption at Rest
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Token encryption errors
var (
	// ErrEncryptionKeyMissing indicates no encryption key was configured.
	ErrEncryptionKeyMissing = errors.New("encryption key not configured")

	// ErrDecryptionFailed indicates the decryption operation failed.
	ErrDecryptionFailed = errors.New("decryption failed")

	// ErrInvalidCiphertext indicates the ciphertext is malformed.
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// TokenEncryptor provides AES-GCM encryption for sensitive tokens.
// It uses HKDF for key derivation to support different encryption contexts.
type TokenEncryptor struct {
	masterKey []byte
	aead      cipher.AEAD
}

// TokenEncryptorConfig holds configuration for token encryption.
type TokenEncryptorConfig struct {
	// MasterKey is the base64-encoded master encryption key.
	// Should be at least 32 bytes (256 bits) of entropy.
	MasterKey string

	// Context is used for key derivation (default: "cartographus-token-encryption").
	Context string
}

// NewTokenEncryptor creates a new token encryptor.
// Returns nil if masterKey is empty (encryption disabled).
func NewTokenEncryptor(config *TokenEncryptorConfig) (*TokenEncryptor, error) {
	if config == nil || config.MasterKey == "" {
		return nil, nil // Encryption disabled
	}

	// Decode master key
	masterKey, err := base64.StdEncoding.DecodeString(config.MasterKey)
	if err != nil {
		return nil, fmt.Errorf("decode master key: %w", err)
	}

	if len(masterKey) < 16 {
		return nil, errors.New("master key must be at least 16 bytes")
	}

	// Derive encryption key using HKDF
	context := config.Context
	if context == "" {
		context = "cartographus-token-encryption"
	}

	derivedKey, err := deriveKey(masterKey, []byte(context), 32)
	if err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM cipher: %w", err)
	}

	return &TokenEncryptor{
		masterKey: masterKey,
		aead:      aead,
	}, nil
}

// deriveKey derives a key using HKDF-SHA256.
func deriveKey(secret, context []byte, keyLen int) ([]byte, error) {
	reader := hkdf.New(sha256.New, secret, nil, context)
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// Encrypt encrypts the plaintext and returns base64-encoded ciphertext.
// The nonce is prepended to the ciphertext.
// Empty strings are returned as-is (no encryption needed).
func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	if e == nil || e.aead == nil {
		return plaintext, nil // Encryption disabled, return as-is
	}

	// Empty strings don't need encryption
	if plaintext == "" {
		return "", nil
	}

	// Generate random nonce
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := e.aead.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
// Empty strings are returned as-is.
func (e *TokenEncryptor) Decrypt(ciphertext string) (string, error) {
	if e == nil || e.aead == nil {
		return ciphertext, nil // Encryption disabled, return as-is
	}

	// Empty strings don't need decryption
	if ciphertext == "" {
		return "", nil
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: base64 decode failed", ErrInvalidCiphertext)
	}

	// Check minimum length (nonce + at least 1 byte + auth tag)
	nonceSize := e.aead.NonceSize()
	if len(data) < nonceSize+1+e.aead.Overhead() {
		return "", fmt.Errorf("%w: data too short", ErrInvalidCiphertext)
	}

	// Extract nonce and ciphertext
	nonce := data[:nonceSize]
	encryptedData := data[nonceSize:]

	// Decrypt
	plaintext, err := e.aead.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrDecryptionFailed, err.Error())
	}

	return string(plaintext), nil
}

// IsEnabled returns true if encryption is enabled.
func (e *TokenEncryptor) IsEnabled() bool {
	return e != nil && e.aead != nil
}

// Sensitive token field names that should be encrypted
const (
	MetadataKeyAccessToken  = "access_token"
	MetadataKeyRefreshToken = "refresh_token"
	MetadataKeyIDToken      = "id_token"
)

// EncryptSessionMetadata encrypts sensitive fields in session metadata.
// Returns a new map with encrypted values. Original map is not modified.
func (e *TokenEncryptor) EncryptSessionMetadata(metadata map[string]string) (map[string]string, error) {
	if e == nil || !e.IsEnabled() {
		return metadata, nil
	}

	if metadata == nil {
		return nil, nil
	}

	// Copy metadata
	encrypted := make(map[string]string, len(metadata))
	for k, v := range metadata {
		encrypted[k] = v
	}

	// Encrypt sensitive fields
	sensitiveFields := []string{MetadataKeyAccessToken, MetadataKeyRefreshToken, MetadataKeyIDToken}
	for _, field := range sensitiveFields {
		if value, ok := encrypted[field]; ok && value != "" {
			encValue, err := e.Encrypt(value)
			if err != nil {
				return nil, fmt.Errorf("encrypt %s: %w", field, err)
			}
			encrypted[field] = encValue
		}
	}

	return encrypted, nil
}

// DecryptSessionMetadata decrypts sensitive fields in session metadata.
// Returns a new map with decrypted values. Original map is not modified.
func (e *TokenEncryptor) DecryptSessionMetadata(metadata map[string]string) (map[string]string, error) {
	if e == nil || !e.IsEnabled() {
		return metadata, nil
	}

	if metadata == nil {
		return nil, nil
	}

	// Copy metadata
	decrypted := make(map[string]string, len(metadata))
	for k, v := range metadata {
		decrypted[k] = v
	}

	// Decrypt sensitive fields
	sensitiveFields := []string{MetadataKeyAccessToken, MetadataKeyRefreshToken, MetadataKeyIDToken}
	for _, field := range sensitiveFields {
		if value, ok := decrypted[field]; ok && value != "" {
			// Try to decrypt - if it fails, the value might not be encrypted
			// (for backward compatibility with existing unencrypted sessions)
			decValue, err := e.Decrypt(value)
			if err != nil {
				// Check if it looks like a JWT (not encrypted)
				if looksLikeJWT(value) {
					continue // Keep original value
				}
				return nil, fmt.Errorf("decrypt %s: %w", field, err)
			}
			decrypted[field] = decValue
		}
	}

	return decrypted, nil
}

// looksLikeJWT checks if a string looks like a JWT token (for backward compatibility).
func looksLikeJWT(s string) bool {
	// JWTs have format: header.payload.signature
	parts := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts++
		}
	}
	return parts == 2
}

// GenerateEncryptionKey generates a cryptographically secure encryption key.
// Returns the key as a base64-encoded string suitable for configuration.
func GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("generate random key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
