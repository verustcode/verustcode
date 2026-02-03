// Package utils provides utility functions for the engine.
// This file contains unit tests for provider detection utilities.
package utils

import (
	"testing"
)

// TestDetectProviderFromURL tests the DetectProviderFromURL function
func TestDetectProviderFromURL(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "GitHub HTTPS URL",
			repoURL:  "https://github.com/owner/repo",
			expected: "github",
		},
		{
			name:     "GitHub HTTPS URL with .git",
			repoURL:  "https://github.com/owner/repo.git",
			expected: "github",
		},
		{
			name:     "GitHub enterprise URL",
			repoURL:  "https://github.company.com/owner/repo",
			expected: "github",
		},
		{
			name:     "GitLab HTTPS URL",
			repoURL:  "https://gitlab.com/owner/repo",
			expected: "gitlab",
		},
		{
			name:     "GitLab HTTPS URL with .git",
			repoURL:  "https://gitlab.com/owner/repo.git",
			expected: "gitlab",
		},
		{
			name:     "GitLab self-hosted URL",
			repoURL:  "https://gitlab.company.com/owner/repo",
			expected: "gitlab",
		},
		{
			name:     "Empty URL",
			repoURL:  "",
			expected: "",
		},
		{
			name:     "Unknown provider",
			repoURL:  "https://bitbucket.org/owner/repo",
			expected: "",
		},
		{
			name:     "Local path",
			repoURL:  "/path/to/repo",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectProviderFromURL(tt.repoURL)
			if result != tt.expected {
				t.Errorf("DetectProviderFromURL(%s) = %s, want %s", tt.repoURL, result, tt.expected)
			}
		})
	}
}

// TestDetectProviderFromURLCaseInsensitivity tests case handling
func TestDetectProviderFromURLCaseInsensitivity(t *testing.T) {
	// Note: The function uses strings.Contains which is case-sensitive
	// These tests document the current behavior
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "lowercase github.com",
			repoURL:  "https://github.com/owner/repo",
			expected: "github",
		},
		{
			name:     "mixed case GitHub.com",
			repoURL:  "https://GitHub.com/owner/repo",
			expected: "", // Case sensitive, won't match
		},
		{
			name:     "lowercase gitlab.com",
			repoURL:  "https://gitlab.com/owner/repo",
			expected: "gitlab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectProviderFromURL(tt.repoURL)
			if result != tt.expected {
				t.Errorf("DetectProviderFromURL(%s) = %s, want %s", tt.repoURL, result, tt.expected)
			}
		})
	}
}

// TestDetectProviderFromURLVariousFormats tests various URL formats
func TestDetectProviderFromURLVariousFormats(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "SSH format git@github.com",
			repoURL:  "git@github.com:owner/repo.git",
			expected: "github",
		},
		{
			name:     "SSH format git@gitlab.com",
			repoURL:  "git@gitlab.com:owner/repo.git",
			expected: "gitlab",
		},
		{
			name:     "GitHub with subdomain",
			repoURL:  "https://api.github.com/repos/owner/repo",
			expected: "github",
		},
		{
			name:     "GitLab with subdomain",
			repoURL:  "https://api.gitlab.com/projects/owner/repo",
			expected: "gitlab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectProviderFromURL(tt.repoURL)
			if result != tt.expected {
				t.Errorf("DetectProviderFromURL(%s) = %s, want %s", tt.repoURL, result, tt.expected)
			}
		})
	}
}

// BenchmarkDetectProviderFromURL benchmarks the provider detection function
func BenchmarkDetectProviderFromURL(b *testing.B) {
	urls := []string{
		"https://github.com/owner/repo",
		"https://gitlab.com/owner/repo",
		"https://bitbucket.org/owner/repo",
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			DetectProviderFromURL(url)
		}
	}
}
