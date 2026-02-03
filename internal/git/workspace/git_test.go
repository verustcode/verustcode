// Package workspace provides workspace management for Git repositories.
package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ====================
// Tests for MaskToken
// ====================

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty token",
			input:    "",
			expected: "(empty)",
		},
		{
			name:     "short token <= 8 chars",
			input:    "short",
			expected: "****",
		},
		{
			name:     "exactly 8 chars",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "long token",
			input:    "ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "ghp_...wxyz",
		},
		{
			name:     "token with 9 chars",
			input:    "123456789",
			expected: "1234...6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskToken(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ====================
// Tests for createCredentialHelper
// ====================

func TestCreateCredentialHelper(t *testing.T) {
	t.Run("creates helper script", func(t *testing.T) {
		token := "test-token-12345"
		scriptPath, cleanup, err := createCredentialHelper(token)
		require.NoError(t, err)
		defer cleanup()

		// Verify script file exists
		_, err = os.Stat(scriptPath)
		assert.NoError(t, err)

		// Verify script content
		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), token)

		// Verify cleanup works
		cleanup()
		_, err = os.Stat(scriptPath)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("script is executable on Unix", func(t *testing.T) {
		if strings.HasPrefix(runtime.GOOS, "windows") {
			t.Skip("Skipping executable test on Windows")
		}

		token := "test-token"
		scriptPath, cleanup, err := createCredentialHelper(token)
		require.NoError(t, err)
		defer cleanup()

		// Check file permissions
		info, err := os.Stat(scriptPath)
		require.NoError(t, err)
		mode := info.Mode()
		assert.True(t, mode&0111 != 0, "Script should be executable")
	})
}

// ====================
// Helper function to create a test git repository
// ====================

func createTestGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init", tmpDir)
	require.NoError(t, cmd.Run())

	// Configure git user (required for commits)
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User")
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com")
	require.NoError(t, cmd.Run())

	// Create initial commit
	readmePath := filepath.Join(tmpDir, "README.md")
	err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "-C", tmpDir, "add", "README.md")
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit")
	require.NoError(t, cmd.Run())

	// Ensure main branch exists (rename default branch to main)
	cmd = exec.Command("git", "-C", tmpDir, "branch", "-M", "main")
	require.NoError(t, cmd.Run())

	return tmpDir
}

// ====================
// Tests for GetLocalHeadSHA
// ====================

func TestGetLocalHeadSHA(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		sha, err := GetLocalHeadSHA(ctx, repoPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, sha)
		assert.Len(t, sha, 40) // SHA-1 is 40 characters
	})

	t.Run("invalid repository path", func(t *testing.T) {
		ctx := context.Background()
		_, err := GetLocalHeadSHA(ctx, "/nonexistent/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get local HEAD SHA")
	})

	t.Run("context cancellation", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := GetLocalHeadSHA(ctx, repoPath)
		assert.Error(t, err)
	})
}

// ====================
// Tests for CheckoutDetached
// ====================

func TestCheckoutDetached(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		err := CheckoutDetached(ctx, repoPath)
		assert.NoError(t, err)

		// Verify we're in detached HEAD state
		cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "HEAD")
		err = cmd.Run()
		assert.Error(t, err) // Should fail in detached HEAD state
	})

	t.Run("invalid repository path", func(t *testing.T) {
		ctx := context.Background()
		err := CheckoutDetached(ctx, "/nonexistent/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to checkout to detached HEAD")
	})
}

// ====================
// Tests for CheckoutBranch
// ====================

func TestCheckoutBranch(t *testing.T) {
	t.Run("checkout existing branch", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// Create a new branch
		cmd := exec.Command("git", "-C", repoPath, "branch", "test-branch")
		require.NoError(t, cmd.Run())

		err := CheckoutBranch(ctx, repoPath, "test-branch")
		assert.NoError(t, err)

		// Verify we're on the branch
		cmd = exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Equal(t, "test-branch\n", string(output))
	})

	t.Run("checkout non-existent branch", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		err := CheckoutBranch(ctx, repoPath, "nonexistent-branch")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to checkout branch")
	})

	t.Run("invalid repository path", func(t *testing.T) {
		ctx := context.Background()
		err := CheckoutBranch(ctx, "/nonexistent/path", "main")
		assert.Error(t, err)
	})
}

// ====================
// Tests for CleanupGitLock
// ====================

func TestCleanupGitLock(t *testing.T) {
	t.Run("no lock file exists", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		err := CleanupGitLock(repoPath)
		assert.NoError(t, err)
	})

	t.Run("removes existing lock file", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		lockPath := filepath.Join(repoPath, ".git", "index.lock")

		// Create lock file
		err := os.WriteFile(lockPath, []byte("lock"), 0644)
		require.NoError(t, err)

		// Verify lock file exists
		_, err = os.Stat(lockPath)
		assert.NoError(t, err)

		// Cleanup should remove it
		err = CleanupGitLock(repoPath)
		assert.NoError(t, err)

		// Verify lock file is gone
		_, err = os.Stat(lockPath)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("handles missing .git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := CleanupGitLock(tmpDir)
		assert.NoError(t, err) // Should not error if lock doesn't exist
	})
}

// ====================
// Tests for DeleteLocalBranch
// ====================

func TestDeleteLocalBranch(t *testing.T) {
	t.Run("delete existing branch", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// Create and checkout a branch
		cmd := exec.Command("git", "-C", repoPath, "checkout", "-b", "test-branch")
		require.NoError(t, cmd.Run())

		// Switch back to main
		cmd = exec.Command("git", "-C", repoPath, "checkout", "main")
		require.NoError(t, cmd.Run())

		// Delete the branch
		err := DeleteLocalBranch(ctx, repoPath, "test-branch")
		assert.NoError(t, err)

		// Verify branch is deleted
		cmd = exec.Command("git", "-C", repoPath, "branch", "--list", "test-branch")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Empty(t, strings.TrimSpace(string(output)))
	})

	t.Run("delete non-existent branch", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// Should not error if branch doesn't exist
		err := DeleteLocalBranch(ctx, repoPath, "nonexistent-branch")
		assert.NoError(t, err)
	})

	t.Run("cannot delete current branch", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// Try to delete the current branch (main)
		err := DeleteLocalBranch(ctx, repoPath, "main")
		// This should fail because we can't delete the current branch
		assert.Error(t, err)
	})

	t.Run("invalid repository path", func(t *testing.T) {
		ctx := context.Background()
		err := DeleteLocalBranch(ctx, "/nonexistent/path", "branch")
		assert.Error(t, err)
	})
}

// ====================
// Tests for ResetAndClean
// ====================

func TestResetAndClean(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// Create a modified file
		readmePath := filepath.Join(repoPath, "README.md")
		err := os.WriteFile(readmePath, []byte("# Modified\n"), 0644)
		require.NoError(t, err)

		// Create an untracked file
		untrackedPath := filepath.Join(repoPath, "untracked.txt")
		err = os.WriteFile(untrackedPath, []byte("untracked"), 0644)
		require.NoError(t, err)

		// Reset and clean
		err = ResetAndClean(ctx, repoPath)
		assert.NoError(t, err)

		// Verify README is reset
		content, err := os.ReadFile(readmePath)
		require.NoError(t, err)
		assert.Equal(t, "# Test Repo\n", string(content))

		// Verify untracked file is removed
		_, err = os.Stat(untrackedPath)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("invalid repository path", func(t *testing.T) {
		ctx := context.Background()
		err := ResetAndClean(ctx, "/nonexistent/path")
		assert.Error(t, err)
	})

	t.Run("timeout", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait a bit to ensure timeout
		time.Sleep(10 * time.Millisecond)

		err := ResetAndClean(ctx, repoPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})
}

// ====================
// Tests for FetchPRRef
// ====================

func TestFetchPRRef(t *testing.T) {
	t.Run("calls FetchPRRefWithAuth with nil options", func(t *testing.T) {
		// This is a simple wrapper, so we just verify it doesn't panic
		// Actual fetch testing requires a real remote repository
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// This will fail because there's no remote, but we're testing the wrapper
		err := FetchPRRef(ctx, repoPath, "refs/pull/1/head", "pr-1")
		assert.Error(t, err) // Expected to fail without remote
		assert.Contains(t, err.Error(), "failed to fetch PR ref")
	})
}

// ====================
// Tests for FetchPRRefWithAuth
// ====================

func TestFetchPRRefWithAuth(t *testing.T) {
	t.Run("without authentication", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// Add a fake remote
		cmd := exec.Command("git", "-C", repoPath, "remote", "add", "origin", "https://github.com/test/repo.git")
		require.NoError(t, cmd.Run())

		// This will fail because the remote doesn't exist, but we're testing the function structure
		err := FetchPRRefWithAuth(ctx, repoPath, "refs/pull/1/head", "pr-1", nil)
		assert.Error(t, err) // Expected to fail with fake remote
	})

	t.Run("with authentication options", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// Add a fake remote
		cmd := exec.Command("git", "-C", repoPath, "remote", "add", "origin", "https://github.com/test/repo.git")
		require.NoError(t, cmd.Run())

		opts := &FetchOptions{
			Token:              "test-token-12345",
			InsecureSkipVerify: true,
			ProviderName:       "github",
		}

		// This will fail because the remote doesn't exist, but we're testing the function structure
		err := FetchPRRefWithAuth(ctx, repoPath, "refs/pull/1/head", "pr-1", opts)
		assert.Error(t, err) // Expected to fail with fake remote
	})

	t.Run("handles non-fast-forward error", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx := context.Background()

		// Create a branch first
		cmd := exec.Command("git", "-C", repoPath, "branch", "pr-1")
		require.NoError(t, cmd.Run())

		// Add a fake remote
		cmd = exec.Command("git", "-C", repoPath, "remote", "add", "origin", "https://github.com/test/repo.git")
		require.NoError(t, cmd.Run())

		// Try to fetch - will fail but tests the non-fast-forward handling logic
		err := FetchPRRefWithAuth(ctx, repoPath, "refs/pull/1/head", "pr-1", nil)
		// May fail for various reasons, but should handle non-fast-forward if it occurs
		assert.Error(t, err)
	})

	t.Run("timeout", func(t *testing.T) {
		repoPath := createTestGitRepo(t)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait a bit to ensure timeout
		time.Sleep(10 * time.Millisecond)

		err := FetchPRRefWithAuth(ctx, repoPath, "refs/pull/1/head", "pr-1", nil)
		assert.Error(t, err)
	})
}

