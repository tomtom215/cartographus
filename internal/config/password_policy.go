// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package config provides password policy enforcement for production security.
// Phase 3: Enforces strong password requirements to prevent weak credentials.
package config

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// PasswordPolicy defines requirements for password strength.
// Implements NIST SP 800-63B guidelines for password security.
type PasswordPolicy struct {
	// MinLength is the minimum password length (NIST recommends 8+, we use 12 for admin passwords)
	MinLength int

	// RequireUppercase requires at least one uppercase letter
	RequireUppercase bool

	// RequireLowercase requires at least one lowercase letter
	RequireLowercase bool

	// RequireDigit requires at least one digit
	RequireDigit bool

	// RequireSpecial requires at least one special character
	RequireSpecial bool

	// MaxConsecutiveRepeats is the maximum allowed consecutive repeated characters (0 = disabled)
	MaxConsecutiveRepeats int

	// ForbidCommonPasswords blocks common/breached passwords
	ForbidCommonPasswords bool

	// ForbidUsernameSimilarity prevents passwords too similar to username
	ForbidUsernameSimilarity bool
}

// DefaultPasswordPolicy returns production-ready password policy.
// Follows NIST SP 800-63B with additional security measures for admin accounts.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:                12,
		RequireUppercase:         true,
		RequireLowercase:         true,
		RequireDigit:             true,
		RequireSpecial:           true,
		MaxConsecutiveRepeats:    3,
		ForbidCommonPasswords:    true,
		ForbidUsernameSimilarity: true,
	}
}

// RelaxedPasswordPolicy returns a less strict policy for non-admin users.
// Still secure but more user-friendly.
func RelaxedPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:                8,
		RequireUppercase:         false,
		RequireLowercase:         true,
		RequireDigit:             true,
		RequireSpecial:           false,
		MaxConsecutiveRepeats:    4,
		ForbidCommonPasswords:    true,
		ForbidUsernameSimilarity: true,
	}
}

// PasswordValidationResult contains details about password validation.
type PasswordValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
	Strength PasswordStrength
}

// PasswordStrength indicates the overall password strength.
type PasswordStrength int

const (
	PasswordStrengthWeak PasswordStrength = iota
	PasswordStrengthFair
	PasswordStrengthGood
	PasswordStrengthStrong
	PasswordStrengthExcellent
)

// String returns the string representation of password strength.
func (s PasswordStrength) String() string {
	switch s {
	case PasswordStrengthWeak:
		return "weak"
	case PasswordStrengthFair:
		return "fair"
	case PasswordStrengthGood:
		return "good"
	case PasswordStrengthStrong:
		return "strong"
	case PasswordStrengthExcellent:
		return "excellent"
	default:
		return "unknown"
	}
}

// charClasses holds the results of character class analysis.
type charClasses struct {
	hasUpper   bool
	hasLower   bool
	hasDigit   bool
	hasSpecial bool
}

// analyzeCharClasses examines a password and returns which character classes are present.
func analyzeCharClasses(password string) charClasses {
	var cc charClasses
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			cc.hasUpper = true
		case unicode.IsLower(r):
			cc.hasLower = true
		case unicode.IsDigit(r):
			cc.hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			cc.hasSpecial = true
		}
	}
	return cc
}

// maxConsecutiveRepeats returns the maximum number of consecutive repeated characters.
func maxConsecutiveRepeats(password string) int {
	if len(password) == 0 {
		return 0
	}
	maxRepeats := 1
	currentRepeats := 1
	var lastRune rune
	for i, r := range password {
		if i > 0 && r == lastRune {
			currentRepeats++
			if currentRepeats > maxRepeats {
				maxRepeats = currentRepeats
			}
		} else {
			currentRepeats = 1
		}
		lastRune = r
	}
	return maxRepeats
}

// Validate checks if a password meets the policy requirements.
// Returns a detailed validation result with all errors and warnings.
func (p PasswordPolicy) Validate(password string, username string) PasswordValidationResult {
	result := PasswordValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Check minimum length
	if len(password) < p.MinLength {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("password must be at least %d characters (got %d)", p.MinLength, len(password)))
	}

	// Analyze character classes
	cc := analyzeCharClasses(password)

	// Check character class requirements
	p.validateCharClasses(&result, cc)

	// Check consecutive repeated characters
	if p.MaxConsecutiveRepeats > 0 && maxConsecutiveRepeats(password) > p.MaxConsecutiveRepeats {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("password cannot have more than %d consecutive repeated characters", p.MaxConsecutiveRepeats))
	}

	// Check common passwords
	if p.ForbidCommonPasswords && isCommonPassword(password) {
		result.Valid = false
		result.Errors = append(result.Errors, "password is too common and easily guessable")
	}

	// Check similarity to username
	if p.ForbidUsernameSimilarity && username != "" && isSimilarToUsername(password, username) {
		result.Valid = false
		result.Errors = append(result.Errors, "password is too similar to username")
	}

	// Calculate password strength
	result.Strength = calculatePasswordStrength(password, cc.hasUpper, cc.hasLower, cc.hasDigit, cc.hasSpecial)

	// Add warnings for weak passwords that still pass policy
	if result.Valid && result.Strength < PasswordStrengthGood {
		result.Warnings = append(result.Warnings,
			"consider using a stronger password with more character variety")
	}

	return result
}

// validateCharClasses checks character class requirements and adds errors to result.
func (p PasswordPolicy) validateCharClasses(result *PasswordValidationResult, cc charClasses) {
	if p.RequireUppercase && !cc.hasUpper {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one uppercase letter")
	}
	if p.RequireLowercase && !cc.hasLower {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one lowercase letter")
	}
	if p.RequireDigit && !cc.hasDigit {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one digit")
	}
	if p.RequireSpecial && !cc.hasSpecial {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one special character (!@#$%^&*...)")
	}
}

// ValidateWithError is a convenience method that returns an error if validation fails.
func (p PasswordPolicy) ValidateWithError(password string, username string) error {
	result := p.Validate(password, username)
	if !result.Valid {
		return errors.New(strings.Join(result.Errors, "; "))
	}
	return nil
}

// calculatePasswordStrength estimates password strength based on various factors.
func calculatePasswordStrength(password string, hasUpper, hasLower, hasDigit, hasSpecial bool) PasswordStrength {
	score := 0

	// Length contributes to strength
	length := len(password)
	switch {
	case length >= 20:
		score += 4
	case length >= 16:
		score += 3
	case length >= 12:
		score += 2
	case length >= 8:
		score++
	}

	// Character variety
	charTypes := 0
	if hasUpper {
		charTypes++
	}
	if hasLower {
		charTypes++
	}
	if hasDigit {
		charTypes++
	}
	if hasSpecial {
		charTypes++
	}
	score += charTypes

	// Check for patterns that reduce strength
	if hasSequentialChars(password) {
		score--
	}
	if hasKeyboardPattern(password) {
		score--
	}

	// Map score to strength level
	switch {
	case score >= 8:
		return PasswordStrengthExcellent
	case score >= 6:
		return PasswordStrengthStrong
	case score >= 4:
		return PasswordStrengthGood
	case score >= 2:
		return PasswordStrengthFair
	default:
		return PasswordStrengthWeak
	}
}

// isCommonPassword checks if the password is in a list of common passwords.
// This list includes the top breached passwords that should never be used.
func isCommonPassword(password string) bool {
	lower := strings.ToLower(password)
	commonPasswords := map[string]bool{
		// Top 100 most common passwords (expanded list)
		"123456":           true,
		"password":         true,
		"123456789":        true,
		"12345678":         true,
		"12345":            true,
		"1234567":          true,
		"1234567890":       true,
		"qwerty":           true,
		"abc123":           true,
		"password1":        true,
		"password123":      true,
		"admin":            true,
		"admin123":         true,
		"letmein":          true,
		"welcome":          true,
		"monkey":           true,
		"dragon":           true,
		"master":           true,
		"login":            true,
		"princess":         true,
		"qwerty123":        true,
		"solo":             true,
		"passw0rd":         true,
		"starwars":         true,
		"iloveyou":         true,
		"sunshine":         true,
		"trustno1":         true,
		"111111":           true,
		"000000":           true,
		"654321":           true,
		"superman":         true,
		"qwerty1":          true,
		"michael":          true,
		"football":         true,
		"baseball":         true,
		"shadow":           true,
		"ashley":           true,
		"jessica":          true,
		"ninja":            true,
		"mustang":          true,
		"secret":           true,
		"changeme":         true,
		"default":          true,
		"test":             true,
		"guest":            true,
		"root":             true,
		"toor":             true,
		"pass":             true,
		"temp":             true,
		"server":           true,
		"database":         true,
		"administrator":    true,
		"letmein123":       true,
		"password!":        true,
		"p@ssw0rd":         true,
		"p@ssword":         true,
		"pa55word":         true,
		"passw0rd!":        true,
		"password1!":       true,
		"welcome1":         true,
		"welcome123":       true,
		"qwertyuiop":       true,
		"asdfghjkl":        true,
		"zxcvbnm":          true,
		"1qaz2wsx":         true,
		"qazwsx":           true,
		"abcd1234":         true,
		"1q2w3e4r":         true,
		"987654321":        true,
		"password1234":     true,
		"123qwe":           true,
		"123abc":           true,
		"123321":           true,
		"123123":           true,
		"112233":           true,
		"aaaaaa":           true,
		"123123123":        true,
		"11111111":         true,
		"00000000":         true,
		"test123":          true,
		"testing":          true,
		"testing123":       true,
		"cartographus":     true,
		"plex":             true,
		"plex123":          true,
		"media":            true,
		"mediaserver":      true,
		"tautulli":         true,
		"jellyfin":         true,
		"emby":             true,
		"streaming":        true,
		"homelab":          true,
		"server123":        true,
		"admin@123":        true,
		"admin#123":        true,
		"root123":          true,
		"root@123":         true,
		"dockeradmin":      true,
		"kubernetes":       true,
		"devops":           true,
		"sysadmin":         true,
		"password@123":     true,
		"welcome@123":      true,
		"administrator123": true,
	}
	return commonPasswords[lower]
}

// isSimilarToUsername checks if the password is too similar to the username.
func isSimilarToUsername(password, username string) bool {
	lowerPass := strings.ToLower(password)
	lowerUser := strings.ToLower(username)

	// Direct match or substring
	if strings.Contains(lowerPass, lowerUser) || strings.Contains(lowerUser, lowerPass) {
		return true
	}

	// Reverse of username
	reversed := reverseString(lowerUser)
	if strings.Contains(lowerPass, reversed) {
		return true
	}

	// Username with common substitutions
	substitutions := map[rune]rune{
		'a': '@', 'e': '3', 'i': '1', 'o': '0', 's': '$', 't': '7',
	}
	substituted := strings.Map(func(r rune) rune {
		if sub, ok := substitutions[r]; ok {
			return sub
		}
		return r
	}, lowerUser)
	return strings.Contains(lowerPass, substituted)
}

// hasSequentialChars checks for sequential character patterns (abc, 123, etc.)
func hasSequentialChars(password string) bool {
	if len(password) < 3 {
		return false
	}

	runes := []rune(strings.ToLower(password))
	sequenceCount := 0

	for i := 1; i < len(runes); i++ {
		// Check for ascending or descending sequences
		diff := int(runes[i]) - int(runes[i-1])
		if diff == 1 || diff == -1 {
			sequenceCount++
			// sequenceCount >= 2 means we have 3+ chars in sequence (e.g., "abc" = 2 diffs)
			if sequenceCount >= 2 {
				return true
			}
		} else {
			sequenceCount = 0
		}
	}

	return false
}

// hasKeyboardPattern checks for common keyboard patterns.
func hasKeyboardPattern(password string) bool {
	lower := strings.ToLower(password)
	patterns := []string{
		"qwerty", "asdf", "zxcv", "qazwsx", "1qaz", "2wsx",
		"!qaz", "@wsx", "qweasd", "asdzxc",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// reverseString reverses a string.
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
