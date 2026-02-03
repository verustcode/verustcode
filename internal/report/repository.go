// Package report provides report generation functionality.
// This includes repository management, structure analysis, and content generation.
package report

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/git/workspace"
	"github.com/verustcode/verustcode/pkg/logger"
)

const (
	// GitOperationTimeout is the default timeout for git operations
	GitOperationTimeout = 5 * time.Minute
)

// RepositoryManager manages repository workspaces for report generation.
// It uses a separate workspace directory from PR review to avoid conflicts.
type RepositoryManager interface {
	// EnsureRepository ensures a repository is cloned and checked out to the specified ref
	EnsureRepository(ctx context.Context, req *RepositoryRequest) (string, error)
}

// RepositoryRequest contains repository information for report generation
type RepositoryRequest struct {
	Provider  provider.Provider // Git provider
	Owner     string            // Repository owner
	Repo      string            // Repository name
	Ref       string            // Branch, tag, or commit to checkout
	Workspace string            // Base workspace directory for reports
}

// DefaultRepositoryManager is the default implementation of RepositoryManager
type DefaultRepositoryManager struct{}

// NewRepositoryManager creates a new repository manager for reports
func NewRepositoryManager() RepositoryManager {
	return &DefaultRepositoryManager{}
}

// EnsureRepository ensures a repository is cloned and checked out to the specified ref.
// Unlike PRRepositoryManager, this manages full repository clones for report generation.
// The clone path format is: {workspace}/{provider}-{owner}-{repo}
func (m *DefaultRepositoryManager) EnsureRepository(ctx context.Context, req *RepositoryRequest) (string, error) {
	// Build clone path: {workspace}/{provider}-{owner}-{repo}
	// Clean owner and repo, replace / with - (handle GitLab nested groups)
	owner := strings.ReplaceAll(req.Owner, "/", "-")
	repo := strings.ReplaceAll(req.Repo, "/", "-")
	cloneDirName := fmt.Sprintf("%s-%s-%s", req.Provider.Name(), owner, repo)
	clonePath := filepath.Join(req.Workspace, cloneDirName)

	// Create workspace directory if not exists
	if err := os.MkdirAll(req.Workspace, 0755); err != nil {
		logger.Error("Failed to create report workspace directory",
			zap.String("path", req.Workspace),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to create report workspace directory: %w", err)
	}

	// Check if directory exists and whether it's a valid git repository
	needClone := false
	info, statErr := os.Stat(clonePath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			needClone = true
		} else {
			return "", fmt.Errorf("failed to check clone path: %w", statErr)
		}
	} else if info.IsDir() {
		// Directory exists, check if it's a valid git repository
		gitDir := filepath.Join(clonePath, ".git")
		if _, gitErr := os.Stat(gitDir); os.IsNotExist(gitErr) {
			// Not a valid git repo, remove and re-clone
			logger.Warn("Directory exists but is not a valid git repository, removing",
				zap.String("path", clonePath),
			)
			if err := os.RemoveAll(clonePath); err != nil {
				logger.Error("Failed to remove invalid directory",
					zap.String("path", clonePath),
					zap.Error(err),
				)
				return "", fmt.Errorf("failed to remove invalid directory: %w", err)
			}
			needClone = true
		}
	}

	// Clone if needed
	if needClone {
		logger.Info("Cloning repository for report",
			zap.String("owner", req.Owner),
			zap.String("repo", req.Repo),
			zap.String("ref", req.Ref),
			zap.String("dest", clonePath),
		)

		if err := m.cloneRepository(ctx, req, clonePath); err != nil {
			logger.Error("Failed to clone repository",
				zap.String("owner", req.Owner),
				zap.String("repo", req.Repo),
				zap.Error(err),
			)
			return "", fmt.Errorf("failed to clone repository: %w", err)
		}
		logger.Info("Repository cloned successfully", zap.String("path", clonePath))
	} else {
		// Repository exists, update it
		logger.Info("Using existing repository, updating to latest",
			zap.String("path", clonePath),
			zap.String("ref", req.Ref),
		)

		// Clean up stale git lock file if exists
		if err := workspace.CleanupGitLock(clonePath); err != nil {
			logger.Warn("Failed to cleanup git lock file, continuing anyway",
				zap.String("path", clonePath),
				zap.Error(err),
			)
		}

		// Fetch latest changes
		if err := m.fetchRepository(ctx, clonePath); err != nil {
			logger.Warn("Failed to fetch repository, will try to continue",
				zap.String("path", clonePath),
				zap.Error(err),
			)
		}
	}

	// Checkout to the specified ref
	if err := m.checkoutRef(ctx, clonePath, req.Ref); err != nil {
		logger.Error("Failed to checkout ref",
			zap.String("path", clonePath),
			zap.String("ref", req.Ref),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to checkout ref %s: %w", req.Ref, err)
	}

	// Clean workspace to ensure pristine state
	if err := workspace.ResetAndClean(ctx, clonePath); err != nil {
		logger.Warn("Failed to clean workspace after checkout",
			zap.String("path", clonePath),
			zap.Error(err),
		)
	}

	logger.Info("Repository ready for report generation",
		zap.String("path", clonePath),
		zap.String("ref", req.Ref),
	)

	return clonePath, nil
}

// cloneRepository clones a repository using provider's Clone method
func (m *DefaultRepositoryManager) cloneRepository(ctx context.Context, req *RepositoryRequest, dest string) error {
	opts := &provider.CloneOptions{
		Branch: req.Ref,
	}
	return req.Provider.Clone(ctx, req.Owner, req.Repo, dest, opts)
}

// fetchRepository fetches latest changes from remote
func (m *DefaultRepositoryManager) fetchRepository(ctx context.Context, repoPath string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, GitOperationTimeout)
	defer cancel()

	// Update fetch config to include all branches
	// This fixes issues where single-branch clone prevents fetching other branches
	configCmd := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	if err := configCmd.Run(); err != nil {
		logger.Warn("Failed to update git fetch config, continuing anyway",
			zap.String("path", repoPath),
			zap.Error(err),
		)
	}

	cmd := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "fetch", "--all", "--prune")
	// Prevent interactive prompts for credentials
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w (stderr: %s)", err, stderrBuf.String())
	}

	return nil
}

// checkoutRef checks out to a specific ref (branch, tag, or commit)
func (m *DefaultRepositoryManager) checkoutRef(ctx context.Context, repoPath, ref string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, GitOperationTimeout)
	defer cancel()

	// First try to checkout as a local branch
	cmd := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "checkout", ref)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		logger.Debug("Direct checkout failed, trying origin/ref",
			zap.String("ref", ref),
			zap.String("stderr", stderrBuf.String()),
		)

		// If branch checkout fails, try to create local branch tracking origin/ref
		originRef := "origin/" + ref
		cmd2 := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "checkout", "-B", ref, originRef)
		var stderr2 bytes.Buffer
		cmd2.Stderr = &stderr2
		if err2 := cmd2.Run(); err2 != nil {
			logger.Debug("Branch tracking checkout failed, trying direct origin/ref checkout",
				zap.String("ref", ref),
				zap.String("stderr", stderr2.String()),
			)

			// If origin/branch fails, try to checkout origin/ref directly (auto detaches HEAD)
			cmd3 := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "checkout", originRef)
			var stderr3 bytes.Buffer
			cmd3.Stderr = &stderr3
			if err3 := cmd3.Run(); err3 != nil {
				logger.Debug("Origin ref checkout failed, trying as commit/tag",
					zap.String("ref", ref),
					zap.String("stderr", stderr3.String()),
				)

				// Last resort: try to checkout as a commit SHA or tag (FETCH_HEAD or direct ref)
				cmd4 := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "checkout", ref, "--")
				var stderr4 bytes.Buffer
				cmd4.Stderr = &stderr4
				if err4 := cmd4.Run(); err4 != nil {
					return fmt.Errorf("git checkout failed for ref %s: %w (stderr: %s)", ref, err4, stderr4.String())
				}
			}
		}
	}

	// Pull latest if on a branch (not detached)
	pullCmd := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "pull", "--ff-only")
	// Ignore pull errors (might be detached HEAD or no upstream)
	_ = pullCmd.Run()

	return nil
}
