// Package workspace provides workspace management for Git repositories.
package workspace

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/git/provider"
)

// ====================
// Mock Provider for testing
// ====================

type mockProvider struct {
	name           string
	clonePRErr     error
	getPRRefResult string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) GetBaseURL() string {
	return "https://github.com"
}

func (m *mockProvider) Clone(ctx context.Context, owner, repo, destPath string, opts *provider.CloneOptions) error {
	return m.ClonePR(ctx, owner, repo, 0, destPath, opts)
}

func (m *mockProvider) ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *provider.CloneOptions) error {
	if m.clonePRErr != nil {
		return m.clonePRErr
	}
	// Create a minimal git repo at destPath
	cmd := exec.Command("git", "init", destPath)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("git", "-C", destPath, "config", "user.name", "Test")
	cmd.Run()
	cmd = exec.Command("git", "-C", destPath, "config", "user.email", "test@example.com")
	cmd.Run()
	readmePath := filepath.Join(destPath, "README.md")
	os.WriteFile(readmePath, []byte("# Test\n"), 0644)
	cmd = exec.Command("git", "-C", destPath, "add", "README.md")
	cmd.Run()
	cmd = exec.Command("git", "-C", destPath, "commit", "-m", "Initial")
	cmd.Run()
	return nil
}

func (m *mockProvider) GetPRRef(prNumber int) string {
	if m.getPRRefResult != "" {
		return m.getPRRefResult
	}
	return "refs/pull/1/head"
}

func (m *mockProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*provider.PullRequest, error) {
	return nil, nil
}

func (m *mockProvider) ListPullRequests(ctx context.Context, owner, repo string) ([]*provider.PullRequest, error) {
	return nil, nil
}

func (m *mockProvider) PostComment(ctx context.Context, owner, repo string, opts *provider.CommentOptions, body string) error {
	return nil
}

func (m *mockProvider) ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*provider.Comment, error) {
	return nil, nil
}

func (m *mockProvider) DeleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	return nil
}

func (m *mockProvider) UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error {
	return nil
}

func (m *mockProvider) ParseWebhook(r *http.Request, secret string) (*provider.WebhookEvent, error) {
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

func (m *mockProvider) ShouldProcessPREvent(event string) bool {
	return true
}

// ====================
// Tests for NewPRRepositoryManager
// ====================

func TestNewPRRepositoryManager(t *testing.T) {
	manager := NewPRRepositoryManager()
	assert.NotNil(t, manager)
	assert.IsType(t, &DefaultPRRepositoryManager{}, manager)
}

// ====================
// Tests for EnsurePRRepository
// ====================

func TestDefaultPRRepositoryManager_EnsurePRRepository(t *testing.T) {
	t.Run("clones new repository", func(t *testing.T) {
		manager := NewPRRepositoryManager()
		ctx := context.Background()
		workspace := t.TempDir()

		req := &PRRepositoryRequest{
			Provider:  &mockProvider{name: "github"},
			Owner:     "test",
			Repo:      "repo",
			PRNumber:  1,
			HeadSHA:   "abc123",
			Workspace: workspace,
		}

		clonePath, err := manager.EnsurePRRepository(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, clonePath)
		assert.Contains(t, clonePath, "github-test-repo")

		// Verify repository was cloned
		gitDir := filepath.Join(clonePath, ".git")
		_, err = os.Stat(gitDir)
		assert.NoError(t, err)
	})

	t.Run("uses existing repository", func(t *testing.T) {
		manager := NewPRRepositoryManager()
		ctx := context.Background()
		workspace := t.TempDir()

		// Create existing repo
		cloneDirName := "github-test-repo"
		clonePath := filepath.Join(workspace, cloneDirName)
		cmd := exec.Command("git", "init", clonePath)
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", clonePath, "config", "user.name", "Test")
		cmd.Run()
		cmd = exec.Command("git", "-C", clonePath, "config", "user.email", "test@example.com")
		cmd.Run()
		readmePath := filepath.Join(clonePath, "README.md")
		os.WriteFile(readmePath, []byte("# Test\n"), 0644)
		cmd = exec.Command("git", "-C", clonePath, "add", "README.md")
		cmd.Run()
		cmd = exec.Command("git", "-C", clonePath, "commit", "-m", "Initial")
		cmd.Run()

		// Get current HEAD SHA
		cmd = exec.Command("git", "-C", clonePath, "rev-parse", "HEAD")
		output, err := cmd.Output()
		require.NoError(t, err)
		currentSHA := string(output[:40]) // First 40 chars

		req := &PRRepositoryRequest{
			Provider:  &mockProvider{name: "github"},
			Owner:     "test",
			Repo:      "repo",
			PRNumber:  1,
			HeadSHA:   currentSHA, // Same SHA, so no fetch needed
			Workspace: workspace,
		}

		resultPath, err := manager.EnsurePRRepository(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, clonePath, resultPath)
	})

	t.Run("handles nested owner/repo paths", func(t *testing.T) {
		manager := NewPRRepositoryManager()
		ctx := context.Background()
		workspace := t.TempDir()

		req := &PRRepositoryRequest{
			Provider:  &mockProvider{name: "gitlab"},
			Owner:     "group/subgroup",
			Repo:      "project/subproject",
			PRNumber:  1,
			HeadSHA:   "abc123",
			Workspace: workspace,
		}

		clonePath, err := manager.EnsurePRRepository(ctx, req)
		assert.NoError(t, err)
		assert.Contains(t, clonePath, "gitlab-group-subgroup-project-subproject")
	})

	t.Run("handles clone failure", func(t *testing.T) {
		manager := NewPRRepositoryManager()
		ctx := context.Background()
		workspace := t.TempDir()

		req := &PRRepositoryRequest{
			Provider: &mockProvider{
				name:       "github",
				clonePRErr: assert.AnError,
			},
			Owner:     "test",
			Repo:      "repo",
			PRNumber:  1,
			HeadSHA:   "abc123",
			Workspace: workspace,
		}

		_, err := manager.EnsurePRRepository(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clone PR")
	})

	t.Run("removes invalid git directory", func(t *testing.T) {
		manager := NewPRRepositoryManager()
		ctx := context.Background()
		workspace := t.TempDir()

		// Create directory without .git
		cloneDirName := "github-test-repo"
		clonePath := filepath.Join(workspace, cloneDirName)
		err := os.MkdirAll(clonePath, 0755)
		require.NoError(t, err)

		// Create a file to verify directory is removed
		testFile := filepath.Join(clonePath, "test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		req := &PRRepositoryRequest{
			Provider:  &mockProvider{name: "github"},
			Owner:     "test",
			Repo:      "repo",
			PRNumber:  1,
			HeadSHA:   "abc123",
			Workspace: workspace,
		}

		clonePath, err = manager.EnsurePRRepository(ctx, req)
		assert.NoError(t, err)

		// Verify directory was recreated as a valid git repo
		gitDir := filepath.Join(clonePath, ".git")
		_, err = os.Stat(gitDir)
		assert.NoError(t, err)

		// Verify old file is gone
		_, err = os.Stat(testFile)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("fetches when SHA differs", func(t *testing.T) {
		manager := NewPRRepositoryManager()
		ctx := context.Background()
		workspace := t.TempDir()

		// Create existing repo
		cloneDirName := "github-test-repo"
		clonePath := filepath.Join(workspace, cloneDirName)
		cmd := exec.Command("git", "init", clonePath)
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", clonePath, "config", "user.name", "Test")
		cmd.Run()
		cmd = exec.Command("git", "-C", clonePath, "config", "user.email", "test@example.com")
		cmd.Run()
		readmePath := filepath.Join(clonePath, "README.md")
		os.WriteFile(readmePath, []byte("# Test\n"), 0644)
		cmd = exec.Command("git", "-C", clonePath, "add", "README.md")
		cmd.Run()
		cmd = exec.Command("git", "-C", clonePath, "commit", "-m", "Initial")
		cmd.Run()

		// Add remote (required for fetch)
		cmd = exec.Command("git", "-C", clonePath, "remote", "add", "origin", "https://github.com/test/repo.git")
		cmd.Run()

		req := &PRRepositoryRequest{
			Provider: &mockProvider{
				name:           "github",
				getPRRefResult: "refs/pull/1/head",
			},
			Owner:     "test",
			Repo:      "repo",
			PRNumber:  1,
			HeadSHA:   "different-sha-1234567890123456789012345678901234567890",
			Workspace: workspace,
		}

		// This will attempt to fetch, which may fail without a real remote,
		// but we're testing the logic flow
		resultPath, err := manager.EnsurePRRepository(ctx, req)
		// May fail due to fetch, but should handle the flow correctly
		if err != nil {
			assert.Contains(t, err.Error(), "failed to fetch PR code")
		} else {
			assert.Equal(t, clonePath, resultPath)
		}
	})

	t.Run("creates workspace directory if not exists", func(t *testing.T) {
		manager := NewPRRepositoryManager()
		ctx := context.Background()
		workspace := filepath.Join(t.TempDir(), "nonexistent", "workspace")

		req := &PRRepositoryRequest{
			Provider:  &mockProvider{name: "github"},
			Owner:     "test",
			Repo:      "repo",
			PRNumber:  1,
			HeadSHA:   "abc123",
			Workspace: workspace,
		}

		clonePath, err := manager.EnsurePRRepository(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, clonePath)

		// Verify workspace directory was created
		_, err = os.Stat(workspace)
		assert.NoError(t, err)
	})
}
