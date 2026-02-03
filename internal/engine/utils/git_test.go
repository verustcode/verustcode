// Package utils provides utility functions for the engine.
// This file contains unit tests for Git utility functions.
package utils

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractOwnerRepo tests the ExtractOwnerRepo function
func TestExtractOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS GitHub URL",
			repoURL:   "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS GitHub URL with .git",
			repoURL:   "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTP GitHub URL",
			repoURL:   "http://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH GitHub URL",
			repoURL:   "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH GitHub URL without .git",
			repoURL:   "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "GitLab HTTPS URL",
			repoURL:   "https://gitlab.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "Owner/repo format",
			repoURL:   "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:    "Empty URL",
			repoURL: "",
			wantErr: true,
		},
		{
			name:    "Invalid URL - single word",
			repoURL: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ExtractOwnerRepo(tt.repoURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractOwnerRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("ExtractOwnerRepo() owner = %s, want %s", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("ExtractOwnerRepo() repo = %s, want %s", repo, tt.wantRepo)
				}
			}
		})
	}
}

// TestBuildPRURL tests the BuildPRURL function
func TestBuildPRURL(t *testing.T) {
	tests := []struct {
		name       string
		repoURL    string
		provider   string
		prNumber   int
		wantURL    string
	}{
		{
			name:     "GitHub PR URL",
			repoURL:  "https://github.com/owner/repo",
			provider: "github",
			prNumber: 123,
			wantURL:  "https://github.com/owner/repo/pull/123",
		},
		{
			name:     "GitHub PR URL with .git",
			repoURL:  "https://github.com/owner/repo.git",
			provider: "github",
			prNumber: 456,
			wantURL:  "https://github.com/owner/repo/pull/456",
		},
		{
			name:     "GitLab MR URL",
			repoURL:  "https://gitlab.com/owner/repo",
			provider: "gitlab",
			prNumber: 789,
			wantURL:  "https://gitlab.com/owner/repo/merge_requests/789",
		},
		{
			name:     "Auto-detect GitHub",
			repoURL:  "https://github.com/owner/repo",
			provider: "",
			prNumber: 100,
			wantURL:  "https://github.com/owner/repo/pull/100",
		},
		{
			name:     "Auto-detect GitLab",
			repoURL:  "https://gitlab.com/owner/repo",
			provider: "",
			prNumber: 200,
			wantURL:  "https://gitlab.com/owner/repo/merge_requests/200",
		},
		{
			name:     "Unknown provider with github in URL",
			repoURL:  "https://github.enterprise.com/owner/repo",
			provider: "unknown",
			prNumber: 300,
			wantURL:  "https://github.enterprise.com/owner/repo/pull/300", // auto-detects github from URL
		},
		{
			name:     "Empty repo URL",
			repoURL:  "",
			provider: "github",
			prNumber: 123,
			wantURL:  "",
		},
		{
			name:     "Invalid PR number (zero)",
			repoURL:  "https://github.com/owner/repo",
			provider: "github",
			prNumber: 0,
			wantURL:  "",
		},
		{
			name:     "Invalid PR number (negative)",
			repoURL:  "https://github.com/owner/repo",
			provider: "github",
			prNumber: -1,
			wantURL:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL := BuildPRURL(tt.repoURL, tt.provider, tt.prNumber)
			if gotURL != tt.wantURL {
				t.Errorf("BuildPRURL() = %s, want %s", gotURL, tt.wantURL)
			}
		})
	}
}

// TestDiffStats tests the DiffStats struct
func TestDiffStats(t *testing.T) {
	stats := &DiffStats{
		LinesAdded:   100,
		LinesDeleted: 50,
		FilesChanged: 10,
	}

	if stats.LinesAdded != 100 {
		t.Errorf("LinesAdded = %d, want 100", stats.LinesAdded)
	}
	if stats.LinesDeleted != 50 {
		t.Errorf("LinesDeleted = %d, want 50", stats.LinesDeleted)
	}
	if stats.FilesChanged != 10 {
		t.Errorf("FilesChanged = %d, want 10", stats.FilesChanged)
	}
}

// TestGetCommitsInRangeWithEmptyArgs tests GetCommitsInRange with empty arguments
func TestGetCommitsInRangeWithEmptyArgs(t *testing.T) {
	t.Run("empty base commit", func(t *testing.T) {
		result := GetCommitsInRange(nil, "/some/path", "", "abc123")
		if result != nil {
			t.Error("Should return nil for empty base commit")
		}
	})

	t.Run("empty head commit", func(t *testing.T) {
		result := GetCommitsInRange(nil, "/some/path", "abc123", "")
		if result != nil {
			t.Error("Should return nil for empty head commit")
		}
	})

	t.Run("both empty", func(t *testing.T) {
		result := GetCommitsInRange(nil, "/some/path", "", "")
		if result != nil {
			t.Error("Should return nil for empty commits")
		}
	})
}

// TestGetBranchCreatedAtWithEmptyArgs tests GetBranchCreatedAt with empty arguments
func TestGetBranchCreatedAtWithEmptyArgs(t *testing.T) {
	t.Run("empty base commit", func(t *testing.T) {
		result := GetBranchCreatedAt(nil, "/some/path", "", "abc123")
		if result != nil {
			t.Error("Should return nil for empty base commit")
		}
	})

	t.Run("empty head commit", func(t *testing.T) {
		result := GetBranchCreatedAt(nil, "/some/path", "abc123", "")
		if result != nil {
			t.Error("Should return nil for empty head commit")
		}
	})
}

// TestGetDiffStatsWithEmptyArgs tests GetDiffStats with empty arguments
func TestGetDiffStatsWithEmptyArgs(t *testing.T) {
	t.Run("empty base commit", func(t *testing.T) {
		result := GetDiffStats(nil, "/some/path", "", "abc123")
		if result != nil {
			t.Error("Should return nil for empty base commit")
		}
	})

	t.Run("empty head commit", func(t *testing.T) {
		result := GetDiffStats(nil, "/some/path", "abc123", "")
		if result != nil {
			t.Error("Should return nil for empty head commit")
		}
	})
}

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, string, string) {
	// Create temporary directory
	repoPath := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "-C", repoPath, "init")
	require.NoError(t, cmd.Run())

	// Configure git user (required for commits)
	exec.Command("git", "-C", repoPath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", repoPath, "config", "user.email", "test@example.com").Run()

	// Create initial commit
	file1 := filepath.Join(repoPath, "file1.txt")
	require.NoError(t, os.WriteFile(file1, []byte("initial content"), 0644))
	exec.Command("git", "-C", repoPath, "add", "file1.txt").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Initial commit").Run()

	// Get base commit SHA
	baseCmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	baseOutput, err := baseCmd.Output()
	require.NoError(t, err)
	baseCommit := string(baseOutput[:40]) // First 40 chars (full SHA)

	// Create second commit
	file2 := filepath.Join(repoPath, "file2.txt")
	require.NoError(t, os.WriteFile(file2, []byte("new file content"), 0644))
	exec.Command("git", "-C", repoPath, "add", "file2.txt").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add file2").Run()

	// Get head commit SHA
	headCmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	headOutput, err := headCmd.Output()
	require.NoError(t, err)
	headCommit := string(headOutput[:40])

	return repoPath, baseCommit, headCommit
}

// TestGetCommitsInRange_WithRealRepo tests GetCommitsInRange with a real git repository
func TestGetCommitsInRange_WithRealRepo(t *testing.T) {
	repoPath, baseCommit, headCommit := setupTestRepo(t)
	ctx := context.Background()

	commits := GetCommitsInRange(ctx, repoPath, baseCommit, headCommit)
	require.NotNil(t, commits)
	assert.Len(t, commits, 1) // Should have one commit between base and head
	assert.Equal(t, headCommit, commits[0])
}

// TestGetCommitsInRange_MultipleCommits tests GetCommitsInRange with multiple commits
func TestGetCommitsInRange_MultipleCommits(t *testing.T) {
	repoPath := t.TempDir()

	// Initialize git repository
	exec.Command("git", "-C", repoPath, "init").Run()
	exec.Command("git", "-C", repoPath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", repoPath, "config", "user.email", "test@example.com").Run()

	// Create base commit
	file1 := filepath.Join(repoPath, "file1.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	exec.Command("git", "-C", repoPath, "add", "file1.txt").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Commit 1").Run()

	// Get base commit
	baseCmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	baseOutput, _ := baseCmd.Output()
	baseCommit := string(baseOutput[:40])

	// Create multiple commits
	for i := 2; i <= 4; i++ {
		file := filepath.Join(repoPath, "file"+string(rune('0'+i))+".txt")
		os.WriteFile(file, []byte("content"), 0644)
		exec.Command("git", "-C", repoPath, "add", file).Run()
		exec.Command("git", "-C", repoPath, "commit", "-m", "Commit "+string(rune('0'+i))).Run()
	}

	// Get head commit
	headCmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	headOutput, _ := headCmd.Output()
	headCommit := string(headOutput[:40])

	ctx := context.Background()
	commits := GetCommitsInRange(ctx, repoPath, baseCommit, headCommit)
	assert.Len(t, commits, 3) // Should have 3 commits between base and head
}

// TestGetCommitsInRange_InvalidPath tests GetCommitsInRange with invalid repository path
func TestGetCommitsInRange_InvalidPath(t *testing.T) {
	ctx := context.Background()
	commits := GetCommitsInRange(ctx, "/nonexistent/path", "abc123", "def456")
	assert.Nil(t, commits)
}

// TestGetCommitsInRange_InvalidCommits tests GetCommitsInRange with invalid commit SHAs
func TestGetCommitsInRange_InvalidCommits(t *testing.T) {
	repoPath := t.TempDir()
	ctx := context.Background()

	// Initialize git repository
	exec.Command("git", "-C", repoPath, "init").Run()

	commits := GetCommitsInRange(ctx, repoPath, "invalid", "alsoinvalid")
	assert.Nil(t, commits)
}

// TestGetBranchCreatedAt_WithRealRepo tests GetBranchCreatedAt with a real git repository
func TestGetBranchCreatedAt_WithRealRepo(t *testing.T) {
	repoPath, baseCommit, headCommit := setupTestRepo(t)
	ctx := context.Background()

	timestamp := GetBranchCreatedAt(ctx, repoPath, baseCommit, headCommit)
	require.NotNil(t, timestamp)
	assert.Greater(t, *timestamp, int64(0))
	assert.Less(t, *timestamp, time.Now().Unix())
}

// TestGetBranchCreatedAt_InvalidPath tests GetBranchCreatedAt with invalid repository path
func TestGetBranchCreatedAt_InvalidPath(t *testing.T) {
	ctx := context.Background()
	timestamp := GetBranchCreatedAt(ctx, "/nonexistent/path", "abc123", "def456")
	assert.Nil(t, timestamp)
}

// TestGetBranchCreatedAt_InvalidCommits tests GetBranchCreatedAt with invalid commit SHAs
func TestGetBranchCreatedAt_InvalidCommits(t *testing.T) {
	repoPath := t.TempDir()
	ctx := context.Background()

	// Initialize git repository
	exec.Command("git", "-C", repoPath, "init").Run()

	timestamp := GetBranchCreatedAt(ctx, repoPath, "invalid", "alsoinvalid")
	assert.Nil(t, timestamp)
}

// TestGetDiffStats_WithRealRepo tests GetDiffStats with a real git repository
func TestGetDiffStats_WithRealRepo(t *testing.T) {
	repoPath := t.TempDir()

	// Initialize git repository
	exec.Command("git", "-C", repoPath, "init").Run()
	exec.Command("git", "-C", repoPath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", repoPath, "config", "user.email", "test@example.com").Run()

	// Create base commit
	file1 := filepath.Join(repoPath, "file1.txt")
	os.WriteFile(file1, []byte("line1\nline2\nline3\n"), 0644)
	exec.Command("git", "-C", repoPath, "add", "file1.txt").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Base commit").Run()

	// Get base commit
	baseCmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	baseOutput, _ := baseCmd.Output()
	baseCommit := string(baseOutput[:40])

	// Modify file and add new file
	os.WriteFile(file1, []byte("line1\nline2\nline3\nline4\nline5\n"), 0644)
	file2 := filepath.Join(repoPath, "file2.txt")
	os.WriteFile(file2, []byte("new file\ncontent\n"), 0644)
	exec.Command("git", "-C", repoPath, "add", "file1.txt", "file2.txt").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Head commit").Run()

	// Get head commit
	headCmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	headOutput, _ := headCmd.Output()
	headCommit := string(headOutput[:40])

	ctx := context.Background()
	stats := GetDiffStats(ctx, repoPath, baseCommit, headCommit)
	require.NotNil(t, stats)
	assert.Equal(t, 2, stats.FilesChanged) // file1.txt and file2.txt
	assert.Greater(t, stats.LinesAdded, 0)
	assert.GreaterOrEqual(t, stats.LinesDeleted, 0)
}

// TestGetDiffStats_NoChanges tests GetDiffStats when there are no changes
func TestGetDiffStats_NoChanges(t *testing.T) {
	repoPath, baseCommit, _ := setupTestRepo(t)
	ctx := context.Background()

	// Use same commit for base and head (no changes)
	stats := GetDiffStats(ctx, repoPath, baseCommit, baseCommit)
	require.NotNil(t, stats)
	assert.Equal(t, 0, stats.FilesChanged)
	assert.Equal(t, 0, stats.LinesAdded)
	assert.Equal(t, 0, stats.LinesDeleted)
}

// TestGetDiffStats_InvalidPath tests GetDiffStats with invalid repository path
func TestGetDiffStats_InvalidPath(t *testing.T) {
	ctx := context.Background()
	stats := GetDiffStats(ctx, "/nonexistent/path", "abc123", "def456")
	assert.Nil(t, stats)
}

// TestGetDiffStats_InvalidCommits tests GetDiffStats with invalid commit SHAs
func TestGetDiffStats_InvalidCommits(t *testing.T) {
	repoPath := t.TempDir()
	ctx := context.Background()

	// Initialize git repository
	exec.Command("git", "-C", repoPath, "init").Run()

	stats := GetDiffStats(ctx, repoPath, "invalid", "alsoinvalid")
	assert.Nil(t, stats)
}

// TestGetDiffStats_BinaryFiles tests GetDiffStats with binary files
func TestGetDiffStats_BinaryFiles(t *testing.T) {
	repoPath := t.TempDir()

	// Initialize git repository
	exec.Command("git", "-C", repoPath, "init").Run()
	exec.Command("git", "-C", repoPath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", repoPath, "config", "user.email", "test@example.com").Run()

	// Create base commit
	exec.Command("git", "-C", repoPath, "commit", "--allow-empty", "-m", "Base").Run()
	baseCmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	baseOutput, _ := baseCmd.Output()
	baseCommit := string(baseOutput[:40])

	// Add binary file (git will mark as binary in diff)
	file := filepath.Join(repoPath, "binary.bin")
	os.WriteFile(file, []byte{0x00, 0x01, 0x02, 0x03}, 0644)
	exec.Command("git", "-C", repoPath, "add", "binary.bin").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add binary").Run()

	headCmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	headOutput, _ := headCmd.Output()
	headCommit := string(headOutput[:40])

	ctx := context.Background()
	stats := GetDiffStats(ctx, repoPath, baseCommit, headCommit)
	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.FilesChanged)
	// Binary files should not contribute to lines added/deleted
}

// TestCleanupWorkspace tests CleanupWorkspace function
func TestCleanupWorkspace(t *testing.T) {
	// Create temporary directory
	workspacePath := t.TempDir()

	// Create some files
	file1 := filepath.Join(workspacePath, "file1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	// Verify directory exists
	_, err := os.Stat(workspacePath)
	require.NoError(t, err)

	// Cleanup
	CleanupWorkspace(workspacePath)

	// Verify directory is removed
	_, err = os.Stat(workspacePath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

// TestCleanupWorkspace_NonExistent tests CleanupWorkspace with non-existent path
func TestCleanupWorkspace_NonExistent(t *testing.T) {
	// Should not panic
	CleanupWorkspace("/nonexistent/path")
}

// TestGetCommitsInRange_ContextCancellation tests GetCommitsInRange with cancelled context
func TestGetCommitsInRange_ContextCancellation(t *testing.T) {
	repoPath, baseCommit, headCommit := setupTestRepo(t)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	commits := GetCommitsInRange(ctx, repoPath, baseCommit, headCommit)
	// Should return nil or empty slice when context is cancelled
	assert.Nil(t, commits)
}

// TestGetBranchCreatedAt_ContextCancellation tests GetBranchCreatedAt with cancelled context
func TestGetBranchCreatedAt_ContextCancellation(t *testing.T) {
	repoPath, baseCommit, headCommit := setupTestRepo(t)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	timestamp := GetBranchCreatedAt(ctx, repoPath, baseCommit, headCommit)
	assert.Nil(t, timestamp)
}

// TestGetDiffStats_ContextCancellation tests GetDiffStats with cancelled context
func TestGetDiffStats_ContextCancellation(t *testing.T) {
	repoPath, baseCommit, headCommit := setupTestRepo(t)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	stats := GetDiffStats(ctx, repoPath, baseCommit, headCommit)
	assert.Nil(t, stats)
}
