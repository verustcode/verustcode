// Package config provides configuration management for the application.
// This file contains validation functions for configuration values.
package config

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/verustcode/verustcode/pkg/errors"
)

// MinJWTSecretLength is the minimum required length for JWT secret (256 bits for HS256)
const MinJWTSecretLength = 32

// PasswordRequirements defines the password complexity requirements
type PasswordRequirements struct {
	MinLength        int    // Minimum password length
	RequireUppercase bool   // Require at least one uppercase letter
	RequireLowercase bool   // Require at least one lowercase letter
	RequireDigit     bool   // Require at least one digit
	RequireSpecial   bool   // Require at least one special character
	SpecialChars     string // Allowed special characters
}

// DefaultPasswordRequirements returns the default password complexity requirements
func DefaultPasswordRequirements() PasswordRequirements {
	return PasswordRequirements{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
		SpecialChars:     "!@#$%^&*()_+-=[]{}|;:,.<>?",
	}
}

// ValidatePassword validates a password against the complexity requirements
// Returns nil if password is valid, otherwise returns an error describing the failure
func ValidatePassword(password string, req PasswordRequirements) error {
	var failures []string

	// Check minimum length
	if len(password) < req.MinLength {
		failures = append(failures, fmt.Sprintf("at least %d characters", req.MinLength))
	}

	// Check for uppercase letter
	if req.RequireUppercase {
		hasUpper := false
		for _, r := range password {
			if unicode.IsUpper(r) {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			failures = append(failures, "at least one uppercase letter (A-Z)")
		}
	}

	// Check for lowercase letter
	if req.RequireLowercase {
		hasLower := false
		for _, r := range password {
			if unicode.IsLower(r) {
				hasLower = true
				break
			}
		}
		if !hasLower {
			failures = append(failures, "at least one lowercase letter (a-z)")
		}
	}

	// Check for digit
	if req.RequireDigit {
		hasDigit := false
		for _, r := range password {
			if unicode.IsDigit(r) {
				hasDigit = true
				break
			}
		}
		if !hasDigit {
			failures = append(failures, "at least one digit (0-9)")
		}
	}

	// Check for special character
	if req.RequireSpecial {
		hasSpecial := false
		for _, r := range password {
			if strings.ContainsRune(req.SpecialChars, r) {
				hasSpecial = true
				break
			}
		}
		if !hasSpecial {
			failures = append(failures, fmt.Sprintf("at least one special character (%s)", req.SpecialChars))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("password must contain: %s", strings.Join(failures, ", "))
	}

	return nil
}

// ValidateAdminConfig validates the admin configuration
// Returns an error if admin is enabled but credentials are invalid
// Note: password_hash is NOT validated here - it can be set via Web UI after server starts
func ValidateAdminConfig(cfg *AdminConfig) *errors.AppError {
	// Skip validation if admin is not enabled
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	// Check if username is empty
	if strings.TrimSpace(cfg.Username) == "" {
		return errors.New(errors.ErrCodeAdminCredentialsEmpty,
			"admin username cannot be empty when admin console is enabled")
	}

	// Note: password_hash validation is intentionally skipped
	// Users can set password via Web UI after server starts
	// This enables a better first-run experience

	// Validate JWT secret (required for secure token signing)
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return errors.New(errors.ErrCodeJWTSecretInvalid,
			"jwt_secret cannot be empty when admin console is enabled")
	}

	if len(cfg.JWTSecret) < MinJWTSecretLength {
		return errors.New(errors.ErrCodeJWTSecretInvalid,
			fmt.Sprintf("jwt_secret must be at least %d characters long for security (HS256 requires 256 bits)", MinJWTSecretLength))
	}

	return nil
}

// IsValidBcryptHash checks if a string is a valid bcrypt hash
// Bcrypt hashes start with $2a$, $2b$, or $2y$ followed by cost factor
func IsValidBcryptHash(hash string) bool {
	if len(hash) < 60 {
		return false
	}
	// Bcrypt hash format: $2a$XX$... or $2b$XX$... or $2y$XX$...
	// where XX is the cost factor (2 digits)
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		return false
	}
	return true
}

// FormatPasswordRequirements returns a human-readable description of password requirements
func FormatPasswordRequirements() string {
	req := DefaultPasswordRequirements()
	var requirements []string

	requirements = append(requirements, fmt.Sprintf("- At least %d characters long", req.MinLength))

	if req.RequireUppercase {
		requirements = append(requirements, "- Contains at least one uppercase letter (A-Z)")
	}
	if req.RequireLowercase {
		requirements = append(requirements, "- Contains at least one lowercase letter (a-z)")
	}
	if req.RequireDigit {
		requirements = append(requirements, "- Contains at least one digit (0-9)")
	}
	if req.RequireSpecial {
		requirements = append(requirements, fmt.Sprintf("- Contains at least one special character (%s)", req.SpecialChars))
	}

	return strings.Join(requirements, "\n")
}
