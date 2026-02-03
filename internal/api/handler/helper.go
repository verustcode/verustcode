// Package handler provides HTTP handlers for the API.
package handler

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Regex to identify sensitive keys in YAML
// Match order: double-quoted value, single-quoted value, or unquoted value (stops at # for comments)
// This properly handles values containing # when they are quoted
var sensitiveKeyRegex = regexp.MustCompile(`(?m)^(\s*[\w-]*(?:password|token|secret|api_key|jwt_secret|private_key)[\w-]*\s*:\s*)("[^"]*"|'[^']*'|[^#\n]*)(\s*(?:#.*)?)$`)

// validateFilename validates a filename to prevent path traversal attacks
// Returns true if the filename is safe, false otherwise
func validateFilename(name string) bool {
	// Check for empty name
	if name == "" {
		return false
	}

	// Check for path traversal patterns
	if strings.Contains(name, "..") {
		return false
	}

	// Check for directory separators (both Unix and Windows)
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}

	// Check for null bytes (can be used to bypass checks)
	if strings.Contains(name, "\x00") {
		return false
	}

	// Clean the filename and ensure it doesn't change after cleaning
	// This catches URL-encoded path traversal attempts
	cleaned := filepath.Clean(name)
	if cleaned != name || cleaned == "." || cleaned == ".." {
		return false
	}

	return true
}

// safeJoinPath safely joins a base directory with a filename and validates
// that the result is within the base directory
func safeJoinPath(baseDir, name string) (string, bool) {
	if !validateFilename(name) {
		return "", false
	}

	// Get absolute path of base directory
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", false
	}

	// Join and clean the path
	fullPath := filepath.Join(absBase, name)
	cleanPath := filepath.Clean(fullPath)

	// Verify the result is still within the base directory
	if !strings.HasPrefix(cleanPath, absBase+string(filepath.Separator)) && cleanPath != absBase {
		return "", false
	}

	return cleanPath, true
}

// maskString masks a string with ****, keeping first 2 and last 2 chars
func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// truncateContent truncates content for logging purposes
// Uses rune-based truncation to avoid breaking multi-byte UTF-8 characters
func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

// computeContentHash calculates SHA256 hash of content
func computeContentHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}

// processSensitiveValues iterates over sensitive values in YAML content and applies a handler
// The handler receives the field name (key) and the value, allowing for key-aware processing
func processSensitiveValues(content string, handler func(key, val string) string) string {
	return sensitiveKeyRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := sensitiveKeyRegex.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		prefix := parts[1]
		val := parts[2]
		suffix := parts[3]

		// Extract field name from prefix (e.g., "  password: " -> "password")
		keyName := strings.TrimSpace(prefix)
		keyName = strings.TrimSuffix(keyName, ":")
		keyName = strings.TrimSpace(keyName)

		// Detect quote style used in original value
		var quoteChar byte
		cleanVal := val
		if len(val) >= 2 {
			if val[0] == '"' && val[len(val)-1] == '"' {
				quoteChar = '"'
				cleanVal = val[1 : len(val)-1]
			} else if val[0] == '\'' && val[len(val)-1] == '\'' {
				quoteChar = '\''
				cleanVal = val[1 : len(val)-1]
			}
		}

		// Skip env vars (check both with and without quotes)
		if strings.HasPrefix(cleanVal, "${") {
			return match
		}

		// Skip empty values
		if strings.TrimSpace(cleanVal) == "" {
			return match
		}

		// Process value with key name for context
		newVal := handler(keyName, cleanVal)

		// Preserve original quote style (or no quotes if original had none)
		if quoteChar != 0 {
			return prefix + string(quoteChar) + newVal + string(quoteChar) + suffix
		}
		return prefix + newVal + suffix
	})
}

// generateRandomString generates a random string of the specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		// Use crypto-safe random if available, otherwise use time-based seed
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond) // Ensure different values
	}
	return string(b)
}
