// Package provider defines the interface for Git providers.
package provider

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ====================
// Tests for ShouldProcessPREvent
// ====================

func TestShouldProcessPREvent(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		expected bool
	}{
		{
			name:     "opened action",
			action:   PREventActionOpened,
			expected: true,
		},
		{
			name:     "synchronize action",
			action:   PREventActionSynchronize,
			expected: true,
		},
		{
			name:     "reopened action",
			action:   PREventActionReopened,
			expected: true,
		},
		{
			name:     "open (lowercase)",
			action:   "open",
			expected: true,
		},
		{
			name:     "update",
			action:   "update",
			expected: true,
		},
		{
			name:     "reopen",
			action:   "reopen",
			expected: true,
		},
		{
			name:     "OPEN (uppercase)",
			action:   "OPEN",
			expected: true,
		},
		{
			name:     "closed action",
			action:   "closed",
			expected: false,
		},
		{
			name:     "merged action",
			action:   "merged",
			expected: false,
		},
		{
			name:     "labeled action",
			action:   "labeled",
			expected: false,
		},
		{
			name:     "empty action",
			action:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldProcessPREvent(tt.action)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ====================
// Tests for IsPRMergedEvent
// ====================

func TestIsPRMergedEvent(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		expected bool
	}{
		{
			name:     "merged",
			action:   "merged",
			expected: true,
		},
		{
			name:     "merge",
			action:   "merge",
			expected: true,
		},
		{
			name:     "closed",
			action:   "closed",
			expected: true,
		},
		{
			name:     "close",
			action:   "close",
			expected: true,
		},
		{
			name:     "MERGED (uppercase)",
			action:   "MERGED",
			expected: true,
		},
		{
			name:     "opened",
			action:   "opened",
			expected: false,
		},
		{
			name:     "synchronize",
			action:   "synchronize",
			expected: false,
		},
		{
			name:     "empty action",
			action:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPRMergedEvent(tt.action)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ====================
// Tests for IsPRUpdateEvent
// ====================

func TestIsPRUpdateEvent(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		expected bool
	}{
		{
			name:     "synchronize",
			action:   PREventActionSynchronize,
			expected: true,
		},
		{
			name:     "update",
			action:   "update",
			expected: true,
		},
		{
			name:     "UPDATE (uppercase)",
			action:   "UPDATE",
			expected: true,
		},
		{
			name:     "opened",
			action:   "opened",
			expected: false,
		},
		{
			name:     "closed",
			action:   "closed",
			expected: false,
		},
		{
			name:     "merged",
			action:   "merged",
			expected: false,
		},
		{
			name:     "empty action",
			action:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPRUpdateEvent(tt.action)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ====================
// Tests for ProviderError
// ====================

func TestProviderError(t *testing.T) {
	t.Run("error without wrapped error", func(t *testing.T) {
		err := &ProviderError{
			Provider: "github",
			Message:  "test error",
		}
		assert.Equal(t, "[github] test error", err.Error())
	})

	t.Run("error with wrapped error", func(t *testing.T) {
		wrappedErr := errors.New("wrapped error")
		err := &ProviderError{
			Provider: "gitlab",
			Message:  "test error",
			Err:      wrappedErr,
		}
		assert.Equal(t, "[gitlab] test error: wrapped error", err.Error())
		assert.Equal(t, wrappedErr, err.Unwrap())
	})
}

// ====================
// Tests for Register and Create
// ====================

func TestRegisterAndCreate(t *testing.T) {
	// Save original registry
	originalRegistry := make(map[string]ProviderFactory)
	for k, v := range Registry {
		originalRegistry[k] = v
	}
	defer func() {
		Registry = originalRegistry
	}()

	// Clear registry for test
	Registry = make(map[string]ProviderFactory)

	t.Run("register and create provider", func(t *testing.T) {
		factory := func(opts *ProviderOptions) (Provider, error) {
			return &mockProvider{name: "test"}, nil
		}

		Register("test-provider", factory)

		provider, err := Create("test-provider", &ProviderOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "test", provider.Name())
	})

	t.Run("create non-existent provider", func(t *testing.T) {
		_, err := Create("nonexistent", &ProviderOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider not registered")
	})

	t.Run("factory returns error", func(t *testing.T) {
		factory := func(opts *ProviderOptions) (Provider, error) {
			return nil, errors.New("factory error")
		}

		Register("error-provider", factory)

		_, err := Create("error-provider", &ProviderOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "factory error")
	})
}

// ====================
// Mock Provider for testing
// ====================

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) GetBaseURL() string {
	return "https://example.com"
}

func (m *mockProvider) Clone(ctx context.Context, owner, repo, destPath string, opts *CloneOptions) error {
	return nil
}

func (m *mockProvider) ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *CloneOptions) error {
	return nil
}

func (m *mockProvider) GetPRRef(prNumber int) string {
	return "refs/pull/1/head"
}

func (m *mockProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	return nil, nil
}

func (m *mockProvider) ListPullRequests(ctx context.Context, owner, repo string) ([]*PullRequest, error) {
	return nil, nil
}

func (m *mockProvider) PostComment(ctx context.Context, owner, repo string, opts *CommentOptions, body string) error {
	return nil
}

func (m *mockProvider) ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*Comment, error) {
	return nil, nil
}

func (m *mockProvider) DeleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	return nil
}

func (m *mockProvider) UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error {
	return nil
}

func (m *mockProvider) ParseWebhook(r *http.Request, secret string) (*WebhookEvent, error) {
	return nil, nil
}

func (m *mockProvider) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	return "", nil
}

func (m *mockProvider) DeleteWebhook(ctx context.Context, owner, repo, webhookID string) error {
	return nil
}

func (m *mockProvider) ValidateToken(ctx context.Context) error {
	return nil
}

func (m *mockProvider) ParseRepoPath(repoURL string) (owner, repo string, err error) {
	return "", "", nil
}

func (m *mockProvider) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	return nil, nil
}

func (m *mockProvider) MatchesURL(repoURL string) bool {
	return true
}

// ====================
// Tests for buildFetchErrorMessage
// ====================

func TestBuildFetchErrorMessage(t *testing.T) {
	t.Run("remote ref not found - github", func(t *testing.T) {
		msg := buildFetchErrorMessage("github", 123, "couldn't find remote ref refs/pull/123/head")
		assert.Contains(t, msg, "PR #123 not found")
	})

	t.Run("remote ref not found - gitlab", func(t *testing.T) {
		msg := buildFetchErrorMessage("gitlab", 456, "couldn't find remote ref refs/merge-requests/456/head")
		assert.Contains(t, msg, "MR !456 not found")
		assert.Contains(t, msg, "merge_request_fetchable")
	})

	t.Run("remote ref not found - gitea", func(t *testing.T) {
		msg := buildFetchErrorMessage("gitea", 789, "couldn't find remote ref refs/pull/789/head")
		assert.Contains(t, msg, "PR #789 not found")
	})

	t.Run("authentication failed", func(t *testing.T) {
		msg := buildFetchErrorMessage("github", 123, "Authentication failed")
		assert.Contains(t, msg, "authentication failed")
		assert.Contains(t, msg, "SC_GITHUB_TOKEN")
	})

	t.Run("SSL certificate problem", func(t *testing.T) {
		msg := buildFetchErrorMessage("gitlab", 456, "SSL certificate problem")
		assert.Contains(t, msg, "SSL certificate verification failed")
		assert.Contains(t, msg, "insecure_skip_verify")
	})

	t.Run("generic error", func(t *testing.T) {
		msg := buildFetchErrorMessage("github", 123, "some other error")
		assert.Contains(t, msg, "failed to fetch PR #123")
	})
}

// ====================
// Tests for ClonePRWithRefs
// ====================

func TestClonePRWithRefs(t *testing.T) {
	t.Run("nil params", func(t *testing.T) {
		ctx := context.Background()
		err := ClonePRWithRefs(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ClonePRParams is nil")
	})

	t.Run("successful clone", func(t *testing.T) {
		ctx := context.Background()
		destPath := t.TempDir()

		// Create a test git repository to clone from
		sourceRepo := t.TempDir()
		cmd := exec.Command("git", "init", sourceRepo)
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", sourceRepo, "config", "user.name", "Test")
		cmd.Run()
		cmd = exec.Command("git", "-C", sourceRepo, "config", "user.email", "test@example.com")
		cmd.Run()
		readmePath := filepath.Join(sourceRepo, "README.md")
		os.WriteFile(readmePath, []byte("# Test\n"), 0644)
		cmd = exec.Command("git", "-C", sourceRepo, "add", "README.md")
		cmd.Run()
		cmd = exec.Command("git", "-C", sourceRepo, "commit", "-m", "Initial")
		cmd.Run()

		// Create a PR branch
		cmd = exec.Command("git", "-C", sourceRepo, "checkout", "-b", "pr-branch")
		cmd.Run()
		prFile := filepath.Join(sourceRepo, "pr-file.txt")
		os.WriteFile(prFile, []byte("PR content"), 0644)
		cmd = exec.Command("git", "-C", sourceRepo, "add", "pr-file.txt")
		cmd.Run()
		cmd = exec.Command("git", "-C", sourceRepo, "commit", "-m", "PR commit")
		cmd.Run()
		cmd = exec.Command("git", "-C", sourceRepo, "checkout", "main")
		cmd.Run()

		// Use file:// URL for local testing
		repoURL := "file://" + sourceRepo

		params := &ClonePRParams{
			ProviderName: "test",
			RepoURL:      repoURL,
			PRRef:        "refs/heads/pr-branch",
			PRNumber:     1,
			DestPath:     destPath,
		}

		err := ClonePRWithRefs(ctx, params)
		assert.NoError(t, err)

		// Verify repository was cloned
		gitDir := filepath.Join(destPath, ".git")
		_, err = os.Stat(gitDir)
		assert.NoError(t, err)

		// Verify we're on the PR branch
		cmd = exec.Command("git", "-C", destPath, "rev-parse", "--abbrev-ref", "HEAD")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Equal(t, "pr-1\n", string(output))
	})

	t.Run("with authentication token", func(t *testing.T) {
		ctx := context.Background()
		destPath := t.TempDir()

		sourceRepo := t.TempDir()
		cmd := exec.Command("git", "init", sourceRepo)
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", sourceRepo, "config", "user.name", "Test")
		cmd.Run()
		cmd = exec.Command("git", "-C", sourceRepo, "config", "user.email", "test@example.com")
		cmd.Run()
		readmePath := filepath.Join(sourceRepo, "README.md")
		os.WriteFile(readmePath, []byte("# Test\n"), 0644)
		cmd = exec.Command("git", "-C", sourceRepo, "add", "README.md")
		cmd.Run()
		cmd = exec.Command("git", "-C", sourceRepo, "commit", "-m", "Initial")
		cmd.Run()
		// Ensure main branch exists
		cmd = exec.Command("git", "-C", sourceRepo, "branch", "-M", "main")
		require.NoError(t, cmd.Run())

		repoURL := "file://" + sourceRepo

		params := &ClonePRParams{
			ProviderName: "test",
			RepoURL:      repoURL,
			Token:        "test-token-12345",
			PRRef:        "refs/heads/main",
			PRNumber:     1,
			DestPath:     destPath,
		}

		err := ClonePRWithRefs(ctx, params)
		assert.NoError(t, err)
	})

	t.Run("with insecure skip verify", func(t *testing.T) {
		ctx := context.Background()
		destPath := t.TempDir()

		sourceRepo := t.TempDir()
		cmd := exec.Command("git", "init", sourceRepo)
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", sourceRepo, "config", "user.name", "Test")
		cmd.Run()
		cmd = exec.Command("git", "-C", sourceRepo, "config", "user.email", "test@example.com")
		cmd.Run()
		readmePath := filepath.Join(sourceRepo, "README.md")
		os.WriteFile(readmePath, []byte("# Test\n"), 0644)
		cmd = exec.Command("git", "-C", sourceRepo, "add", "README.md")
		cmd.Run()
		cmd = exec.Command("git", "-C", sourceRepo, "commit", "-m", "Initial")
		cmd.Run()
		// Ensure main branch exists
		cmd = exec.Command("git", "-C", sourceRepo, "branch", "-M", "main")
		require.NoError(t, cmd.Run())

		repoURL := "file://" + sourceRepo

		params := &ClonePRParams{
			ProviderName:       "test",
			RepoURL:            repoURL,
			InsecureSkipVerify: true,
			PRRef:              "refs/heads/main",
			PRNumber:           1,
			DestPath:           destPath,
		}

		err := ClonePRWithRefs(ctx, params)
		assert.NoError(t, err)
	})

	t.Run("invalid destination path", func(t *testing.T) {
		ctx := context.Background()
		params := &ClonePRParams{
			ProviderName: "test",
			RepoURL:      "https://github.com/test/repo.git",
			PRRef:        "refs/pull/1/head",
			PRNumber:     1,
			DestPath:     "/nonexistent/invalid/path",
		}

		err := ClonePRWithRefs(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to init repository")
	})

	t.Run("invalid repository URL", func(t *testing.T) {
		ctx := context.Background()
		destPath := t.TempDir()

		// Initialize empty repo first
		cmd := exec.Command("git", "init", destPath)
		require.NoError(t, cmd.Run())

		params := &ClonePRParams{
			ProviderName: "test",
			RepoURL:      "https://nonexistent-repo-that-does-not-exist.com/test/repo.git",
			PRRef:        "refs/pull/1/head",
			PRNumber:     1,
			DestPath:     destPath,
		}

		err := ClonePRWithRefs(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch")
	})
}

