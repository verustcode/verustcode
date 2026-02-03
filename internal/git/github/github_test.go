package github

import (
	"testing"

	"github.com/verustcode/verustcode/internal/git/provider"
)

// TestNormalizeURL tests URL normalization
func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS URL with .git",
			input:    "https://github.com/owner/repo.git",
			expected: "github.com/owner/repo",
		},
		{
			name:     "HTTPS URL without .git",
			input:    "https://github.com/owner/repo",
			expected: "github.com/owner/repo",
		},
		{
			name:     "git@ format",
			input:    "git@github.com:owner/repo.git",
			expected: "github.com/owner/repo",
		},
		{
			name:     "URL with trailing slash",
			input:    "https://github.com/owner/repo/",
			expected: "github.com/owner/repo",
		},
		{
			name:     "HTTP URL",
			input:    "http://github.com/owner/repo",
			expected: "github.com/owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNewProvider tests creating a new GitHub provider
func TestNewProvider(t *testing.T) {
	opts := &provider.ProviderOptions{
		Token:   "test-token",
		BaseURL: "",
	}

	prov, err := NewProvider(opts)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}

	if prov == nil {
		t.Fatal("NewProvider() returned nil")
	}

	if prov.Name() != "github" {
		t.Errorf("Expected provider name 'github', got '%s'", prov.Name())
	}
}

// TestGitHubProvider_Name tests provider name
func TestGitHubProvider_Name(t *testing.T) {
	p := &GitHubProvider{}
	if p.Name() != "github" {
		t.Errorf("Name() = %q, want 'github'", p.Name())
	}
}

// TestGitHubProvider_ParseRepoPath tests parsing repository path
func TestGitHubProvider_ParseRepoPath(t *testing.T) {
	p := &GitHubProvider{
		baseURL: "https://github.com",
	}

	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "simple owner/repo format",
			repoURL:   "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "full HTTPS URL",
			repoURL:   "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "full HTTPS URL with .git suffix",
			repoURL:   "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:    "empty URL",
			repoURL: "",
			wantErr: true,
		},
		{
			name:    "invalid format - single segment",
			repoURL: "owner",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := p.ParseRepoPath(tt.repoURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRepoPath() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRepoPath() unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

// TestGitHubProvider_GetPRRef tests getting PR ref
func TestGitHubProvider_GetPRRef(t *testing.T) {
	p := &GitHubProvider{}

	tests := []struct {
		prNumber int
		want     string
	}{
		{prNumber: 1, want: "refs/pull/1/head"},
		{prNumber: 123, want: "refs/pull/123/head"},
		{prNumber: 9999, want: "refs/pull/9999/head"},
	}

	for _, tt := range tests {
		got := p.GetPRRef(tt.prNumber)
		if got != tt.want {
			t.Errorf("GetPRRef(%d) = %q, want %q", tt.prNumber, got, tt.want)
		}
	}
}

// TestGitHubProvider_GetBaseURL tests getting base URL
func TestGitHubProvider_GetBaseURL(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"", "https://github.com"},
		{"https://github.com", "https://github.com"},
		{"https://github.example.com", "https://github.example.com"},
	}

	for _, tt := range tests {
		p := &GitHubProvider{baseURL: tt.baseURL}
		got := p.GetBaseURL()
		if got != tt.want {
			t.Errorf("GetBaseURL() = %q, want %q", got, tt.want)
		}
	}
}
