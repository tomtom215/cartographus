// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

// =====================================================
// Token Encryption Tests
// ADR-0015: Zero Trust Authentication - Phase 4D.2
// =====================================================

func TestNewTokenEncryptor_NilConfig(t *testing.T) {
	enc, err := NewTokenEncryptor(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enc != nil {
		t.Error("encryptor should be nil when config is nil")
	}
}

func TestNewTokenEncryptor_EmptyKey(t *testing.T) {
	enc, err := NewTokenEncryptor(&TokenEncryptorConfig{
		MasterKey: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enc != nil {
		t.Error("encryptor should be nil when key is empty")
	}
}

func TestNewTokenEncryptor_ValidKey(t *testing.T) {
	// Generate a valid key
	key, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey error: %v", err)
	}

	enc, err := NewTokenEncryptor(&TokenEncryptorConfig{
		MasterKey: key,
	})
	if err != nil {
		t.Fatalf("NewTokenEncryptor error: %v", err)
	}
	if enc == nil {
		t.Fatal("encryptor should not be nil")
	}
	if !enc.IsEnabled() {
		t.Error("encryptor should be enabled")
	}
}

func TestNewTokenEncryptor_ShortKey(t *testing.T) {
	// Create a key that's too short (less than 16 bytes)
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))

	_, err := NewTokenEncryptor(&TokenEncryptorConfig{
		MasterKey: shortKey,
	})
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestNewTokenEncryptor_InvalidBase64(t *testing.T) {
	_, err := NewTokenEncryptor(&TokenEncryptorConfig{
		MasterKey: "not-valid-base64!!!",
	})
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestTokenEncryptor_EncryptDecrypt(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	enc, err := NewTokenEncryptor(&TokenEncryptorConfig{
		MasterKey: key,
	})
	if err != nil {
		t.Fatalf("NewTokenEncryptor error: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple token", "access_token_12345"},
		{"JWT-like", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature"},
		{"empty string", ""},
		{"unicode", "token-with-unicode-\u4e2d\u6587"},
		{"special chars", "token+with/special=chars&more"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt error: %v", err)
			}

			// Ciphertext should be different from plaintext (unless empty)
			if tt.plaintext != "" && ciphertext == tt.plaintext {
				t.Error("ciphertext should be different from plaintext")
			}

			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt error: %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("decrypted = %s, want %s", decrypted, tt.plaintext)
			}
		})
	}
}

func TestTokenEncryptor_EncryptProducesDifferentCiphertexts(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	plaintext := "same-token"

	// Encrypt same plaintext multiple times
	ciphertext1, _ := enc.Encrypt(plaintext)
	ciphertext2, _ := enc.Encrypt(plaintext)
	ciphertext3, _ := enc.Encrypt(plaintext)

	// Each ciphertext should be different (due to random nonce)
	if ciphertext1 == ciphertext2 {
		t.Error("ciphertexts should be different (random nonce)")
	}
	if ciphertext2 == ciphertext3 {
		t.Error("ciphertexts should be different (random nonce)")
	}

	// But all should decrypt to same plaintext
	decrypted1, _ := enc.Decrypt(ciphertext1)
	decrypted2, _ := enc.Decrypt(ciphertext2)
	decrypted3, _ := enc.Decrypt(ciphertext3)

	if decrypted1 != plaintext || decrypted2 != plaintext || decrypted3 != plaintext {
		t.Error("all ciphertexts should decrypt to same plaintext")
	}
}

func TestTokenEncryptor_DecryptInvalidCiphertext(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"not base64", "not-valid-base64!!!"},
		{"too short", base64.StdEncoding.EncodeToString([]byte("x"))},
		{"corrupted", base64.StdEncoding.EncodeToString([]byte("this-is-not-encrypted-at-all-but-long-enough"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("expected decryption error")
			}
		})
	}
}

func TestTokenEncryptor_NilEncryptor(t *testing.T) {
	var enc *TokenEncryptor

	// Encrypt should return plaintext as-is
	plaintext := "test-token"
	result, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != plaintext {
		t.Errorf("result = %s, want %s", result, plaintext)
	}

	// Decrypt should return ciphertext as-is
	result, err = enc.Decrypt(plaintext)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != plaintext {
		t.Errorf("result = %s, want %s", result, plaintext)
	}

	// IsEnabled should return false
	if enc.IsEnabled() {
		t.Error("nil encryptor should not be enabled")
	}
}

func TestTokenEncryptor_EncryptSessionMetadata(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	metadata := map[string]string{
		"access_token":  "access-123",
		"refresh_token": "refresh-456",
		"id_token":      "id-789",
		"user_id":       "user-abc", // Should not be encrypted
	}

	encrypted, err := enc.EncryptSessionMetadata(metadata)
	if err != nil {
		t.Fatalf("EncryptSessionMetadata error: %v", err)
	}

	// Verify tokens are encrypted (different from original)
	if encrypted["access_token"] == metadata["access_token"] {
		t.Error("access_token should be encrypted")
	}
	if encrypted["refresh_token"] == metadata["refresh_token"] {
		t.Error("refresh_token should be encrypted")
	}
	if encrypted["id_token"] == metadata["id_token"] {
		t.Error("id_token should be encrypted")
	}

	// Verify non-sensitive fields are unchanged
	if encrypted["user_id"] != metadata["user_id"] {
		t.Error("user_id should not be modified")
	}

	// Verify original map is not modified
	if metadata["access_token"] != "access-123" {
		t.Error("original map should not be modified")
	}
}

func TestTokenEncryptor_DecryptSessionMetadata(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	originalMetadata := map[string]string{
		"access_token":  "access-123",
		"refresh_token": "refresh-456",
		"id_token":      "id-789",
		"user_id":       "user-abc",
	}

	// Encrypt
	encrypted, err := enc.EncryptSessionMetadata(originalMetadata)
	if err != nil {
		t.Fatalf("EncryptSessionMetadata error: %v", err)
	}

	// Decrypt
	decrypted, err := enc.DecryptSessionMetadata(encrypted)
	if err != nil {
		t.Fatalf("DecryptSessionMetadata error: %v", err)
	}

	// Verify all values match original
	for k, v := range originalMetadata {
		if decrypted[k] != v {
			t.Errorf("decrypted[%s] = %s, want %s", k, decrypted[k], v)
		}
	}
}

func TestTokenEncryptor_DecryptSessionMetadata_BackwardCompatibility(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	// Simulate old unencrypted session with JWT tokens
	metadata := map[string]string{
		"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature",
		"id_token":     "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyIn0.sig",
	}

	// Decryption should not fail for JWT-like tokens
	decrypted, err := enc.DecryptSessionMetadata(metadata)
	if err != nil {
		t.Fatalf("DecryptSessionMetadata error: %v", err)
	}

	// Values should remain unchanged (backward compatible)
	if decrypted["access_token"] != metadata["access_token"] {
		t.Error("JWT access_token should remain unchanged")
	}
	if decrypted["id_token"] != metadata["id_token"] {
		t.Error("JWT id_token should remain unchanged")
	}
}

func TestTokenEncryptor_NilMetadata(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	// Encrypt nil
	encrypted, err := enc.EncryptSessionMetadata(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if encrypted != nil {
		t.Error("encrypted should be nil")
	}

	// Decrypt nil
	decrypted, err := enc.DecryptSessionMetadata(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decrypted != nil {
		t.Error("decrypted should be nil")
	}
}

func TestTokenEncryptor_DisabledEncryption(t *testing.T) {
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: ""})

	metadata := map[string]string{
		"access_token": "token-123",
	}

	// With disabled encryption, metadata should pass through unchanged
	encrypted, err := enc.EncryptSessionMetadata(metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if encrypted["access_token"] != metadata["access_token"] {
		t.Error("with encryption disabled, tokens should pass through unchanged")
	}
}

func TestTokenEncryptor_CustomContext(t *testing.T) {
	key, _ := GenerateEncryptionKey()

	enc1, _ := NewTokenEncryptor(&TokenEncryptorConfig{
		MasterKey: key,
		Context:   "context-1",
	})

	enc2, _ := NewTokenEncryptor(&TokenEncryptorConfig{
		MasterKey: key,
		Context:   "context-2",
	})

	plaintext := "test-token"

	// Encrypt with enc1
	ciphertext, _ := enc1.Encrypt(plaintext)

	// Try to decrypt with enc2 (different context = different derived key)
	_, err := enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("decryption with different context should fail")
	}

	// But enc1 should still work
	decrypted, err := enc1.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decryption with same context should succeed: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("decrypted = %s, want %s", decrypted, plaintext)
	}
}

func TestGenerateEncryptionKey(t *testing.T) {
	key1, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey error: %v", err)
	}

	key2, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey error: %v", err)
	}

	// Keys should be different
	if key1 == key2 {
		t.Error("generated keys should be different")
	}

	// Keys should be valid base64
	decoded, err := base64.StdEncoding.DecodeString(key1)
	if err != nil {
		t.Fatalf("key should be valid base64: %v", err)
	}

	// Key should be 32 bytes
	if len(decoded) != 32 {
		t.Errorf("key length = %d, want 32", len(decoded))
	}
}

func TestLooksLikeJWT(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature", true},
		{"a.b.c", true},
		{"..", true},   // 2 dots = 3 parts (like JWT)
		{"...", false}, // 3 dots = 4 parts (not like JWT)
		{"not-a-jwt", false},
		{"one.part", false},
		{"", false},
		{"no.dots.at.all.four", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := looksLikeJWT(tt.input)
			if result != tt.expected {
				t.Errorf("looksLikeJWT(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTokenEncryptor_EmptyTokenFields(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	metadata := map[string]string{
		"access_token": "", // Empty should not be encrypted
		"user_id":      "user-abc",
	}

	encrypted, err := enc.EncryptSessionMetadata(metadata)
	if err != nil {
		t.Fatalf("EncryptSessionMetadata error: %v", err)
	}

	// Empty token should remain empty
	if encrypted["access_token"] != "" {
		t.Error("empty token should remain empty")
	}
}

func BenchmarkTokenEncryptor_Encrypt(b *testing.B) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	token := strings.Repeat("x", 1000) // 1KB token

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//nolint:errcheck
		enc.Encrypt(token)
	}
}

func BenchmarkTokenEncryptor_Decrypt(b *testing.B) {
	key, _ := GenerateEncryptionKey()
	enc, _ := NewTokenEncryptor(&TokenEncryptorConfig{MasterKey: key})

	token := strings.Repeat("x", 1000)
	ciphertext, _ := enc.Encrypt(token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//nolint:errcheck
		enc.Decrypt(ciphertext)
	}
}
