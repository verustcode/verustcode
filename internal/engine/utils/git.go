// Package utils provides utility functions for the engine.
// This file contains Git-related utility functions.
package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/pkg/logger"
)

// ExtractOwnerRepo extracts owner and repo name from a repository URL.
// Supports formats like:
// - https://github.com/owner/repo.git
// - https://github.com/owner/repo
// - git@github.com:owner/repo.git
// - git@github.com:owner/repo
//
// Deprecated: Use provider.Provider.ParseRepoPath instead.
// This function does not correctly handle GitLab multi-level namespaces.
// It will be removed in a future version.
func ExtractOwnerRepo(repoURL string) (owner, repo string, err error) {
	if repoURL == "" {
		return "", "", fmt.Errorf("empty repository URL")
	}

	// Remove protocol prefixes
	url := repoURL
	if strings.HasPrefix(url, "https://") {
		url = url[8:]
	} else if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "git@") {
		// Handle SSH format: git@host:owner/repo.git
		url = strings.TrimPrefix(url, "git@")
		// Replace colon with slash
		url = strings.Replace(url, ":", "/", 1)
	}

	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Find the path part (after domain)
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository URL format: %s", repoURL)
	}

	// Handle different URL formats
	// For github.com/owner/repo or gitlab.com/owner/repo
	// parts[0] = domain, parts[1] = owner, parts[2] = repo
	if len(parts) >= 3 {
		owner = parts[1]
		repo = parts[2]
		return owner, repo, nil
	}

	// If only 2 parts, assume it's owner/repo format
	if len(parts) == 2 {
		owner = parts[0]
		repo = parts[1]
		return owner, repo, nil
	}

	return "", "", fmt.Errorf("unable to extract owner/repo from URL: %s", repoURL)
}

// BuildPRURL builds a PR/MR URL from repository URL, provider name, and PR number.
// This is a fallback when provider API is not available.
func BuildPRURL(repoURL, providerName string, prNumber int) string {
	if repoURL == "" || prNumber <= 0 {
		return ""
	}

	// Remove .git suffix from repo URL
	baseURL := strings.TrimSuffix(repoURL, ".git")

	// Build PR/MR URL based on provider
	switch providerName {
	case "github":
		return fmt.Sprintf("%s/pull/%d", baseURL, prNumber)
	case "gitlab":
		return fmt.Sprintf("%s/merge_requests/%d", baseURL, prNumber)
	default:
		// Try to detect provider from URL if not specified
		if strings.Contains(repoURL, "github.com") || strings.Contains(repoURL, "github.") {
			return fmt.Sprintf("%s/pull/%d", baseURL, prNumber)
		}
		if strings.Contains(repoURL, "gitlab.com") || strings.Contains(repoURL, "gitlab.") {
			return fmt.Sprintf("%s/merge_requests/%d", baseURL, prNumber)
		}
		// Unknown provider, return empty
		return ""
	}
}

// GetCommitsInRange returns all commit SHAs between base and head (exclusive base, inclusive head)
// using git rev-list --reverse base..head.
// Returns empty slice if base or head is empty, or if the command fails.
func GetCommitsInRange(ctx context.Context, repoPath, baseCommit, headCommit string) []string {
	if baseCommit == "" || headCommit == "" {
		return nil
	}

	// Build git rev-list command: git rev-list --reverse base..head
	// --reverse: oldest commit first (chronological order)
	// base..head: commits reachable from head but not from base
	commitRange := fmt.Sprintf("%s..%s", baseCommit, headCommit)
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-list", "--reverse", commitRange)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Warn("Failed to get commits in range",
			zap.String("repo_path", repoPath),
			zap.String("base", baseCommit),
			zap.String("head", headCommit),
			zap.String("stderr", stderr.String()),
			zap.Error(err),
		)
		return nil
	}

	// Parse output: each line is a commit SHA
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return nil
	}

	commits := strings.Split(output, "\n")

	logger.Debug("Got commits in range",
		zap.String("repo_path", repoPath),
		zap.String("range", commitRange),
		zap.Int("count", len(commits)),
	)

	return commits
}

// CleanupWorkspace removes the task workspace directory.
func CleanupWorkspace(path string) {
	if err := os.RemoveAll(path); err != nil {
		logger.Warn("Failed to cleanup workspace",
			zap.String("path", path),
			zap.Error(err),
		)
	}
}

// DiffStats holds statistics about changes between two commits.
type DiffStats struct {
	LinesAdded   int
	LinesDeleted int
	FilesChanged int
}

// GetBranchCreatedAt returns the timestamp of the first commit in the range base..head.
// This represents when the branch diverged from the base.
// Returns nil if base or head is empty, or if the command fails.
func GetBranchCreatedAt(ctx context.Context, repoPath, baseCommit, headCommit string) *int64 {
	if baseCommit == "" || headCommit == "" {
		return nil
	}

	// Get the first commit in the range (oldest commit on the branch)
	// git log --reverse --format=%ct base..head | head -1
	commitRange := fmt.Sprintf("%s..%s", baseCommit, headCommit)
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "log", "--reverse", "--format=%ct", commitRange)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Warn("Failed to get branch created at timestamp",
			zap.String("repo_path", repoPath),
			zap.String("base", baseCommit),
			zap.String("head", headCommit),
			zap.String("stderr", stderr.String()),
			zap.Error(err),
		)
		return nil
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return nil
	}

	// Get the first line (oldest commit timestamp)
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return nil
	}

	var timestamp int64
	if _, err := fmt.Sscanf(lines[0], "%d", &timestamp); err != nil {
		logger.Warn("Failed to parse commit timestamp",
			zap.String("timestamp_str", lines[0]),
			zap.Error(err),
		)
		return nil
	}

	logger.Debug("Got branch created at timestamp",
		zap.String("repo_path", repoPath),
		zap.String("range", commitRange),
		zap.Int64("timestamp", timestamp),
	)

	return &timestamp
}

// GetDiffStats returns the diff statistics (lines added, deleted, files changed) between base and head.
// Returns nil if base or head is empty, or if the command fails.
func GetDiffStats(ctx context.Context, repoPath, baseCommit, headCommit string) *DiffStats {
	if baseCommit == "" || headCommit == "" {
		return nil
	}

	// Use git diff --numstat to get per-file stats, then sum them up
	// --numstat gives: added<TAB>deleted<TAB>filename
	commitRange := fmt.Sprintf("%s..%s", baseCommit, headCommit)
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--numstat", commitRange)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Warn("Failed to get diff stats",
			zap.String("repo_path", repoPath),
			zap.String("base", baseCommit),
			zap.String("head", headCommit),
			zap.String("stderr", stderr.String()),
			zap.Error(err),
		)
		return nil
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		// No changes
		return &DiffStats{}
	}

	stats := &DiffStats{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: added<TAB>deleted<TAB>filename
		// Binary files show as "-" for both added and deleted
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		stats.FilesChanged++

		// Parse added lines (skip binary files marked as "-")
		if parts[0] != "-" {
			var added int
			if _, err := fmt.Sscanf(parts[0], "%d", &added); err == nil {
				stats.LinesAdded += added
			}
		}

		// Parse deleted lines (skip binary files marked as "-")
		if parts[1] != "-" {
			var deleted int
			if _, err := fmt.Sscanf(parts[1], "%d", &deleted); err == nil {
				stats.LinesDeleted += deleted
			}
		}
	}

	logger.Debug("Got diff stats",
		zap.String("repo_path", repoPath),
		zap.String("range", commitRange),
		zap.Int("lines_added", stats.LinesAdded),
		zap.Int("lines_deleted", stats.LinesDeleted),
		zap.Int("files_changed", stats.FilesChanged),
	)

	return stats
}
