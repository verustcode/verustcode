// Package idgen provides ID generation utilities for the application.
// It encapsulates the ID generation implementation, making it easy to change
// the underlying ID generation strategy in the future.
package idgen

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"
	"unicode"

	"github.com/rs/xid"
)

// NewID generates a new globally unique, sortable identifier.
// Returns a 20-character string using xid format.
// The generated ID is:
// - Globally unique
// - Sortable by creation time
// - URL-safe (base32 encoded)
// - 20 characters long
func NewID() string {
	return xid.New().String()
}

// NewReviewID generates a unique ID for Review entities.
// Currently an alias for NewID, but can be customized in the future
// (e.g., adding a prefix like "rev_" for better identification).
func NewReviewID() string {
	return NewID()
}

// NewReportID generates a unique ID for Report entities.
// Currently an alias for NewID, but can be customized in the future
// (e.g., adding a prefix like "rpt_" for better identification).
func NewReportID() string {
	return NewID()
}

// NewRequestID generates a unique ID for request tracking.
// Currently an alias for NewID, but can be customized in the future
// (e.g., adding a prefix like "req_" for better identification).
func NewRequestID() string {
	return NewID()
}

// NewSecureSecret generates a cryptographically secure random string of specified length.
// Uses URL-safe base64 encoding. Useful for JWT secrets and other security tokens.
func NewSecureSecret(length int) string {
	// Calculate the number of bytes needed (base64 encoding expands by ~4/3)
	byteLength := (length*3 + 3) / 4
	bytes := make([]byte, byteLength)

	if _, err := rand.Read(bytes); err != nil {
		// Fallback should never happen with crypto/rand, but just in case
		return "please-generate-a-secure-random-secret"
	}

	// Use URL-safe base64 encoding and trim to exact length
	encoded := base64.URLEncoding.EncodeToString(bytes)
	if len(encoded) > length {
		encoded = encoded[:length]
	}
	return encoded
}

// NewSecurePassword generates a cryptographically secure password that meets complexity requirements.
// Requirements: at least 12 characters, contains uppercase, lowercase, digit, and special character.
func NewSecurePassword() string {
	const (
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		digits    = "0123456789"
		special   = "!@$%^&*()_+-=[]{}|;:,.<>?"
		minLength = 12 // Use 12 characters for better security
	)

	// Character sets for password generation
	allChars := uppercase + lowercase + digits + special

	// Generate random password
	password := make([]byte, minLength)
	for i := 0; i < minLength; i++ {
		randomBytes := make([]byte, 1)
		if _, err := rand.Read(randomBytes); err != nil {
			// Fallback: use time-based seed if crypto/rand fails (should never happen)
			password[i] = allChars[int(time.Now().UnixNano())%len(allChars)]
		} else {
			password[i] = allChars[int(randomBytes[0])%len(allChars)]
		}
	}

	// Ensure password meets all requirements by replacing characters if needed
	passwordStr := string(password)
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, r := range passwordStr {
		if unicode.IsUpper(r) {
			hasUpper = true
		} else if unicode.IsLower(r) {
			hasLower = true
		} else if unicode.IsDigit(r) {
			hasDigit = true
		} else if strings.ContainsRune(special, r) {
			hasSpecial = true
		}
	}

	// Fix missing requirements by replacing random positions
	passwordBytes := []byte(passwordStr)
	if !hasUpper {
		randomBytes := make([]byte, 1)
		rand.Read(randomBytes)
		passwordBytes[int(randomBytes[0])%len(passwordBytes)] = uppercase[int(randomBytes[0])%len(uppercase)]
	}
	if !hasLower {
		randomBytes := make([]byte, 1)
		rand.Read(randomBytes)
		passwordBytes[int(randomBytes[0])%len(passwordBytes)] = lowercase[int(randomBytes[0])%len(lowercase)]
	}
	if !hasDigit {
		randomBytes := make([]byte, 1)
		rand.Read(randomBytes)
		passwordBytes[int(randomBytes[0])%len(passwordBytes)] = digits[int(randomBytes[0])%len(digits)]
	}
	if !hasSpecial {
		randomBytes := make([]byte, 1)
		rand.Read(randomBytes)
		passwordBytes[int(randomBytes[0])%len(passwordBytes)] = special[int(randomBytes[0])%len(special)]
	}

	return string(passwordBytes)
}

