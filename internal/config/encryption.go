// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package config provides configuration management for the application.
// This file implements credential encryption for secure storage of API tokens and secrets.
//
// Encryption Algorithm:
//   - AES-256-GCM (authenticated encryption)
//   - 12-byte random nonce per encryption
//   - Key derived from JWT_SECRET using HKDF-SHA256
//
// Security Properties:
//   - Confidentiality: AES-256 encryption
//   - Integrity: GCM authentication tag
//   - Uniqueness: Random nonce prevents ciphertext analysis
//
// Example Usage:
//
//	encryptor, err := NewCredentialEncryptor("jwt-secret-key")
//	if err != nil {
//	    log.Fatal("Failed to create encryptor:", err)
//	}
//
//	ciphertext, err := encryptor.Encrypt("api-token")
//	plaintext, err := encryptor.Decrypt(ciphertext)
package config

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

const (
	// credentialEncryptionSalt is the salt used for HKDF key derivation.
	// This is a fixed, application-specific salt that ensures keys are
	// uniquely bound to this application's credential encryption use case.
	credentialEncryptionSalt = "cartographus-server-credentials"

	// credentialEncryptionInfo is the HKDF info parameter for key derivation.
	credentialEncryptionInfo = "credential-encryption-v1"

	// aesKeySize is the size of the AES key in bytes (256 bits).
	aesKeySize = 32

	// gcmNonceSize is the size of the GCM nonce in bytes.
	gcmNonceSize = 12
)

var (
	// ErrEmptySecret is returned when an empty JWT secret is provided.
	ErrEmptySecret = errors.New("JWT secret cannot be empty")

	// ErrEmptyPlaintext is returned when attempting to encrypt empty data.
	ErrEmptyPlaintext = errors.New("plaintext cannot be empty")

	// ErrEmptyCiphertext is returned when attempting to decrypt empty data.
	ErrEmptyCiphertext = errors.New("ciphertext cannot be empty")

	// ErrDecryptionFailed is returned when decryption fails (invalid ciphertext or tampered data).
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or authentication tag")

	// ErrInvalidCiphertext is returned when the ciphertext format is invalid.
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")

	// ErrCiphertextTooShort is returned when the ciphertext is shorter than the minimum length.
	ErrCiphertextTooShort = errors.New("ciphertext too short")
)

// CredentialEncryptor provides AES-256-GCM encryption for sensitive credentials.
// It derives an encryption key from the application's JWT secret using HKDF,
// ensuring that credential encryption is tied to the application's identity.
type CredentialEncryptor struct {
	key    []byte
	cipher cipher.AEAD
}

// NewCredentialEncryptor creates a new credential encryptor using the provided JWT secret.
// The JWT secret is used to derive a 256-bit AES key using HKDF-SHA256.
//
// Parameters:
//   - jwtSecret: The application's JWT secret (must not be empty)
//
// Returns:
//   - *CredentialEncryptor: The encryptor instance
//   - error: ErrEmptySecret if the secret is empty, or any key derivation error
func NewCredentialEncryptor(jwtSecret string) (*CredentialEncryptor, error) {
	if jwtSecret == "" {
		return nil, ErrEmptySecret
	}

	// Derive encryption key from JWT secret using HKDF
	key, err := deriveKey(jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &CredentialEncryptor{
		key:    key,
		cipher: gcm,
	}, nil
}

// Encrypt encrypts a plaintext string and returns a base64-encoded ciphertext.
// The ciphertext format is: base64(nonce || ciphertext || tag)
//
// Parameters:
//   - plaintext: The credential to encrypt (must not be empty)
//
// Returns:
//   - string: Base64-encoded ciphertext
//   - error: ErrEmptyPlaintext if plaintext is empty, or any encryption error
func (e *CredentialEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", ErrEmptyPlaintext
	}

	// Generate random nonce
	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt with GCM (includes authentication tag)
	ciphertext := e.cipher.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded result
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded ciphertext and returns the plaintext.
//
// Parameters:
//   - ciphertext: Base64-encoded ciphertext (as returned by Encrypt)
//
// Returns:
//   - string: The decrypted plaintext
//   - error: Various errors for invalid input or decryption failure
func (e *CredentialEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", ErrEmptyCiphertext
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: base64 decode failed: %s", ErrInvalidCiphertext, err.Error())
	}

	// Minimum length: nonce (12) + at least 1 byte + tag (16) = 29 bytes
	minLength := gcmNonceSize + 1 + e.cipher.Overhead()
	if len(data) < minLength {
		return "", ErrCiphertextTooShort
	}

	// Extract nonce and ciphertext
	nonce := data[:gcmNonceSize]
	encryptedData := data[gcmNonceSize:]

	// Decrypt and verify
	plaintext, err := e.cipher.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// MaskCredential returns a masked version of a credential for display purposes.
// Shows only the last 4 characters preceded by asterisks.
//
// Parameters:
//   - credential: The credential to mask
//
// Returns:
//   - string: Masked credential (e.g., "****...abc1")
func MaskCredential(credential string) string {
	if credential == "" {
		return ""
	}

	if len(credential) <= 4 {
		return "****"
	}

	// Show last 4 characters
	return "****..." + credential[len(credential)-4:]
}

// deriveKey derives a 256-bit AES key from the JWT secret using HKDF-SHA256.
func deriveKey(jwtSecret string) ([]byte, error) {
	// Create HKDF reader
	hkdfReader := hkdf.New(
		sha256.New,
		[]byte(jwtSecret),
		[]byte(credentialEncryptionSalt),
		[]byte(credentialEncryptionInfo),
	)

	// Read key bytes
	key := make([]byte, aesKeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("failed to read HKDF output: %w", err)
	}

	return key, nil
}

// ValidateEncryptionSetup validates that encryption is properly configured.
// This performs a round-trip encrypt/decrypt test to ensure the encryptor is working.
//
// Returns:
//   - error: nil if encryption is working, error otherwise
func (e *CredentialEncryptor) ValidateEncryptionSetup() error {
	testData := "encryption-validation-test"

	encrypted, err := e.Encrypt(testData)
	if err != nil {
		return fmt.Errorf("encryption test failed: %w", err)
	}

	decrypted, err := e.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("decryption test failed: %w", err)
	}

	if decrypted != testData {
		return errors.New("round-trip validation failed: data mismatch")
	}

	return nil
}
