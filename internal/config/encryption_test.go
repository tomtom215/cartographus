// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestNewCredentialEncryptor(t *testing.T) {
	tests := []struct {
		name      string
		jwtSecret string
		wantErr   error
	}{
		{
			name:      "valid secret",
			jwtSecret: "my-super-secret-jwt-key",
			wantErr:   nil,
		},
		{
			name:      "empty secret",
			jwtSecret: "",
			wantErr:   ErrEmptySecret,
		},
		{
			name:      "short secret",
			jwtSecret: "x",
			wantErr:   nil, // HKDF can derive from any length
		},
		{
			name:      "long secret",
			jwtSecret: strings.Repeat("a", 1000),
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewCredentialEncryptor(tt.jwtSecret)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewCredentialEncryptor() error = %v, wantErr %v", err, tt.wantErr)
				}
				if enc != nil {
					t.Error("NewCredentialEncryptor() returned encryptor on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewCredentialEncryptor() unexpected error = %v", err)
				}
				if enc == nil {
					t.Error("NewCredentialEncryptor() returned nil encryptor")
				}
			}
		})
	}
}

func TestCredentialEncryptor_Encrypt(t *testing.T) {
	enc, err := NewCredentialEncryptor("test-secret")
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
		wantErr   error
	}{
		{
			name:      "valid plaintext",
			plaintext: "my-api-token",
			wantErr:   nil,
		},
		{
			name:      "empty plaintext",
			plaintext: "",
			wantErr:   ErrEmptyPlaintext,
		},
		{
			name:      "special characters",
			plaintext: "token!@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr:   nil,
		},
		{
			name:      "unicode",
			plaintext: "token-日本語-emoji-🔐",
			wantErr:   nil,
		},
		{
			name:      "very long plaintext",
			plaintext: strings.Repeat("x", 10000),
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := enc.Encrypt(tt.plaintext)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				}
				if ciphertext != "" {
					t.Error("Encrypt() returned ciphertext on error")
				}
			} else {
				if err != nil {
					t.Errorf("Encrypt() unexpected error = %v", err)
				}
				if ciphertext == "" {
					t.Error("Encrypt() returned empty ciphertext")
				}

				// Verify it's valid base64
				_, decodeErr := base64.StdEncoding.DecodeString(ciphertext)
				if decodeErr != nil {
					t.Errorf("Encrypt() output is not valid base64: %v", decodeErr)
				}
			}
		})
	}
}

func TestCredentialEncryptor_Decrypt(t *testing.T) {
	enc, err := NewCredentialEncryptor("test-secret")
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Create a valid ciphertext for testing
	validCiphertext, err := enc.Encrypt("test-token")
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	tests := []struct {
		name       string
		ciphertext string
		wantErr    error
	}{
		{
			name:       "valid ciphertext",
			ciphertext: validCiphertext,
			wantErr:    nil,
		},
		{
			name:       "empty ciphertext",
			ciphertext: "",
			wantErr:    ErrEmptyCiphertext,
		},
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!!!",
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "too short ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString([]byte("short")),
			wantErr:    ErrCiphertextTooShort,
		},
		{
			name:       "tampered ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString(make([]byte, 50)),
			wantErr:    ErrDecryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plaintext, err := enc.Decrypt(tt.ciphertext)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Decrypt() expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
				}
				if plaintext != "" {
					t.Error("Decrypt() returned plaintext on error")
				}
			} else {
				if err != nil {
					t.Errorf("Decrypt() unexpected error = %v", err)
				}
				if plaintext == "" {
					t.Error("Decrypt() returned empty plaintext")
				}
			}
		})
	}
}

func TestCredentialEncryptor_RoundTrip(t *testing.T) {
	enc, err := NewCredentialEncryptor("test-secret-for-roundtrip")
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	testCases := []string{
		"simple-token",
		"token with spaces",
		"token!@#$%^&*()",
		"日本語トークン",
		"🔐🔑🗝️",
		strings.Repeat("a", 1000),
		"plex-token-XXXX-YYYY-ZZZZ",
		"jellyfin-api-key-1234567890",
		"emby-token-with-dashes-and-numbers-12345",
	}

	for _, original := range testCases {
		t.Run(original[:min(len(original), 20)], func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(original)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify
			if decrypted != original {
				t.Errorf("Round trip failed: got %q, want %q", decrypted, original)
			}
		})
	}
}

func TestCredentialEncryptor_UniqueNonce(t *testing.T) {
	enc, err := NewCredentialEncryptor("test-secret")
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := "same-token"
	ciphertexts := make(map[string]bool)

	// Encrypt the same plaintext multiple times
	for i := 0; i < 100; i++ {
		ciphertext, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}

		// Each ciphertext should be unique due to random nonce
		if ciphertexts[ciphertext] {
			t.Error("Encrypt() produced duplicate ciphertext")
		}
		ciphertexts[ciphertext] = true
	}
}

func TestCredentialEncryptor_DifferentSecrets(t *testing.T) {
	enc1, err := NewCredentialEncryptor("secret-one")
	if err != nil {
		t.Fatalf("Failed to create encryptor 1: %v", err)
	}

	enc2, err := NewCredentialEncryptor("secret-two")
	if err != nil {
		t.Fatalf("Failed to create encryptor 2: %v", err)
	}

	plaintext := "my-token"

	// Encrypt with encryptor 1
	ciphertext, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Try to decrypt with encryptor 2 (should fail)
	_, err = enc2.Decrypt(ciphertext)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("Decrypt() with wrong secret: expected %v, got %v", ErrDecryptionFailed, err)
	}

	// Decrypt with correct encryptor (should succeed)
	decrypted, err := enc1.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() with correct secret: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypt() returned wrong plaintext: got %q, want %q", decrypted, plaintext)
	}
}

func TestCredentialEncryptor_ValidateEncryptionSetup(t *testing.T) {
	tests := []struct {
		name      string
		jwtSecret string
		wantErr   bool
	}{
		{
			name:      "valid setup",
			jwtSecret: "valid-secret",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewCredentialEncryptor(tt.jwtSecret)
			if err != nil {
				t.Fatalf("Failed to create encryptor: %v", err)
			}

			err = enc.ValidateEncryptionSetup()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEncryptionSetup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaskCredential(t *testing.T) {
	tests := []struct {
		name       string
		credential string
		want       string
	}{
		{
			name:       "normal credential",
			credential: "plex-token-12345678",
			want:       "****...5678",
		},
		{
			name:       "short credential (4 chars)",
			credential: "1234",
			want:       "****",
		},
		{
			name:       "very short credential",
			credential: "ab",
			want:       "****",
		},
		{
			name:       "empty credential",
			credential: "",
			want:       "",
		},
		{
			name:       "exactly 5 chars",
			credential: "12345",
			want:       "****...2345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskCredential(tt.credential)
			if got != tt.want {
				t.Errorf("MaskCredential() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeriveKey(t *testing.T) {
	// Test that the same secret always produces the same key (deterministic)
	key1, err := deriveKey("test-secret")
	if err != nil {
		t.Fatalf("deriveKey() error = %v", err)
	}

	key2, err := deriveKey("test-secret")
	if err != nil {
		t.Fatalf("deriveKey() error = %v", err)
	}

	if string(key1) != string(key2) {
		t.Error("deriveKey() is not deterministic")
	}

	// Test that different secrets produce different keys
	key3, err := deriveKey("different-secret")
	if err != nil {
		t.Fatalf("deriveKey() error = %v", err)
	}

	if string(key1) == string(key3) {
		t.Error("deriveKey() produced same key for different secrets")
	}

	// Verify key length
	if len(key1) != aesKeySize {
		t.Errorf("deriveKey() key length = %d, want %d", len(key1), aesKeySize)
	}
}

// Benchmark tests

func BenchmarkEncrypt(b *testing.B) {
	enc, _ := NewCredentialEncryptor("benchmark-secret")
	plaintext := "plex-token-XXXX-YYYY-ZZZZ-1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Encrypt(plaintext)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	enc, _ := NewCredentialEncryptor("benchmark-secret")
	plaintext := "plex-token-XXXX-YYYY-ZZZZ-1234567890"
	ciphertext, _ := enc.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Decrypt(ciphertext)
	}
}

func BenchmarkRoundTrip(b *testing.B) {
	enc, _ := NewCredentialEncryptor("benchmark-secret")
	plaintext := "plex-token-XXXX-YYYY-ZZZZ-1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, _ := enc.Encrypt(plaintext)
		_, _ = enc.Decrypt(ciphertext)
	}
}
