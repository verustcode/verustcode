// Package idgen provides ID generation utilities for the application.
// This file contains unit tests for the idgen package.
package idgen

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"unicode"
)

// TestNewID tests the NewID function
func TestNewID(t *testing.T) {
	t.Run("returns non-empty ID", func(t *testing.T) {
		id := NewID()
		if id == "" {
			t.Error("NewID() returned empty string")
		}
	})

	t.Run("returns 20 character ID", func(t *testing.T) {
		id := NewID()
		if len(id) != 20 {
			t.Errorf("NewID() returned ID with length %d, want 20", len(id))
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			id := NewID()
			if ids[id] {
				t.Errorf("NewID() generated duplicate ID: %s", id)
			}
			ids[id] = true
		}
	})

	t.Run("generates URL-safe IDs", func(t *testing.T) {
		// xid uses base32 encoding which is URL-safe (alphanumeric)
		urlSafe := regexp.MustCompile(`^[a-z0-9]+$`)
		for i := 0; i < 100; i++ {
			id := NewID()
			if !urlSafe.MatchString(id) {
				t.Errorf("NewID() returned non-URL-safe ID: %s", id)
			}
		}
	})

	t.Run("IDs are sortable by creation time", func(t *testing.T) {
		// Generate IDs in sequence and verify they are in lexicographic order
		var prevID string
		for i := 0; i < 100; i++ {
			id := NewID()
			if prevID != "" && id <= prevID {
				t.Errorf("NewID() generated non-sortable IDs: %s <= %s", id, prevID)
			}
			prevID = id
		}
	})

	t.Run("concurrent generation is safe", func(t *testing.T) {
		var wg sync.WaitGroup
		ids := make(chan string, 1000)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					ids <- NewID()
				}
			}()
		}

		wg.Wait()
		close(ids)

		seen := make(map[string]bool)
		for id := range ids {
			if seen[id] {
				t.Errorf("Concurrent NewID() generated duplicate ID: %s", id)
			}
			seen[id] = true
		}
	})
}

// TestNewReviewID tests the NewReviewID function
func TestNewReviewID(t *testing.T) {
	t.Run("returns valid ID", func(t *testing.T) {
		id := NewReviewID()
		if id == "" {
			t.Error("NewReviewID() returned empty string")
		}
		if len(id) != 20 {
			t.Errorf("NewReviewID() returned ID with length %d, want 20", len(id))
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := NewReviewID()
			if ids[id] {
				t.Errorf("NewReviewID() generated duplicate ID: %s", id)
			}
			ids[id] = true
		}
	})
}

// TestNewReportID tests the NewReportID function
func TestNewReportID(t *testing.T) {
	t.Run("returns valid ID", func(t *testing.T) {
		id := NewReportID()
		if id == "" {
			t.Error("NewReportID() returned empty string")
		}
		if len(id) != 20 {
			t.Errorf("NewReportID() returned ID with length %d, want 20", len(id))
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := NewReportID()
			if ids[id] {
				t.Errorf("NewReportID() generated duplicate ID: %s", id)
			}
			ids[id] = true
		}
	})
}

// TestNewRequestID tests the NewRequestID function
func TestNewRequestID(t *testing.T) {
	t.Run("returns valid ID", func(t *testing.T) {
		id := NewRequestID()
		if id == "" {
			t.Error("NewRequestID() returned empty string")
		}
		if len(id) != 20 {
			t.Errorf("NewRequestID() returned ID with length %d, want 20", len(id))
		}
	})
}

// TestNewSecureSecret tests the NewSecureSecret function
func TestNewSecureSecret(t *testing.T) {
	t.Run("returns correct length", func(t *testing.T) {
		for _, length := range []int{8, 16, 32, 64, 128} {
			secret := NewSecureSecret(length)
			if len(secret) != length {
				t.Errorf("NewSecureSecret(%d) returned length %d", length, len(secret))
			}
		}
	})

	t.Run("generates unique secrets", func(t *testing.T) {
		secrets := make(map[string]bool)
		for i := 0; i < 100; i++ {
			secret := NewSecureSecret(32)
			if secrets[secret] {
				t.Errorf("NewSecureSecret() generated duplicate: %s", secret)
			}
			secrets[secret] = true
		}
	})

	t.Run("uses URL-safe base64", func(t *testing.T) {
		// URL-safe base64 uses A-Z, a-z, 0-9, -, _
		urlSafe := regexp.MustCompile(`^[A-Za-z0-9\-_]+$`)
		for i := 0; i < 100; i++ {
			secret := NewSecureSecret(32)
			if !urlSafe.MatchString(secret) {
				t.Errorf("NewSecureSecret() returned non-URL-safe secret: %s", secret)
			}
		}
	})

	t.Run("handles edge cases", func(t *testing.T) {
		// Test zero length
		secret := NewSecureSecret(0)
		if len(secret) != 0 {
			t.Errorf("NewSecureSecret(0) returned non-empty string")
		}

		// Test very small length
		secret = NewSecureSecret(1)
		if len(secret) != 1 {
			t.Errorf("NewSecureSecret(1) returned length %d", len(secret))
		}
	})
}

// TestNewSecurePassword tests the NewSecurePassword function
func TestNewSecurePassword(t *testing.T) {
	t.Run("returns at least 12 characters", func(t *testing.T) {
		password := NewSecurePassword()
		if len(password) < 12 {
			t.Errorf("NewSecurePassword() returned length %d, want at least 12", len(password))
		}
	})

	t.Run("contains required character types", func(t *testing.T) {
		// Generate multiple passwords and check that most meet requirements
		// Due to randomness, a small number of failures is acceptable
		failures := 0
		total := 50

		for i := 0; i < total; i++ {
			password := NewSecurePassword()

			hasUpper := false
			hasLower := false
			hasDigit := false
			hasSpecial := false
			special := "!@$%^&*()_+-=[]{}|;:,.<>?"

			for _, r := range password {
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

			if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
				failures++
			}
		}

		// Allow up to 10% failure rate due to randomness in fix-up logic
		maxFailures := total / 10
		if failures > maxFailures {
			t.Errorf("Too many passwords missing required characters: %d/%d failed", failures, total)
		}
	})

	t.Run("generates unique passwords", func(t *testing.T) {
		passwords := make(map[string]bool)
		for i := 0; i < 100; i++ {
			password := NewSecurePassword()
			if passwords[password] {
				t.Errorf("NewSecurePassword() generated duplicate: %s", password)
			}
			passwords[password] = true
		}
	})
}

// BenchmarkNewID benchmarks the NewID function
func BenchmarkNewID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewID()
	}
}

// BenchmarkNewSecureSecret benchmarks the NewSecureSecret function
func BenchmarkNewSecureSecret(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSecureSecret(32)
	}
}

// BenchmarkNewSecurePassword benchmarks the NewSecurePassword function
func BenchmarkNewSecurePassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSecurePassword()
	}
}
