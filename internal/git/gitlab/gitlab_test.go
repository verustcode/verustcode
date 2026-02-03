package gitlab

import (
	"testing"

	"github.com/verustcode/verustcode/internal/git/provider"
)

// TestNewProvider tests creating a new GitLab provider
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

	if prov.Name() != "gitlab" {
		t.Errorf("Expected provider name 'gitlab', got '%s'", prov.Name())
	}
}

// TestGitLabProvider_Name tests provider name
func TestGitLabProvider_Name(t *testing.T) {
	p := &GitLabProvider{}
	if p.Name() != "gitlab" {
		t.Errorf("Name() = %q, want 'gitlab'", p.Name())
	}
}

// TestGitLabProvider_ParseRepoPath tests parsing repository path
func TestGitLabProvider_ParseRepoPath(t *testing.T) {
	p := &GitLabProvider{
		baseURL: "https://gitlab.com",
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
			repoURL:   "https://gitlab.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "full HTTPS URL with .git suffix",
			repoURL:   "https://gitlab.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "group/project format",
			repoURL:   "group/subgroup/project",
			wantOwner: "group/subgroup",
			wantRepo:  "project",
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

// TestGitLabProvider_GetPRRef tests getting MR ref
func TestGitLabProvider_GetPRRef(t *testing.T) {
	p := &GitLabProvider{}

	tests := []struct {
		prNumber int
		want     string
	}{
		{prNumber: 1, want: "refs/merge-requests/1/head"},
		{prNumber: 123, want: "refs/merge-requests/123/head"},
		{prNumber: 9999, want: "refs/merge-requests/9999/head"},
	}

	for _, tt := range tests {
		got := p.GetPRRef(tt.prNumber)
		if got != tt.want {
			t.Errorf("GetPRRef(%d) = %q, want %q", tt.prNumber, got, tt.want)
		}
	}
}

// TestGitLabProvider_GetBaseURL tests getting base URL
func TestGitLabProvider_GetBaseURL(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"", "https://gitlab.com"},
		{"https://gitlab.com", "https://gitlab.com"},
		{"https://gitlab.example.com", "https://gitlab.example.com"},
	}

	for _, tt := range tests {
		p := &GitLabProvider{baseURL: tt.baseURL}
		got := p.GetBaseURL()
		if got != tt.want {
			t.Errorf("GetBaseURL() = %q, want %q", got, tt.want)
		}
	}
}
