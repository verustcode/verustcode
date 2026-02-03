// Package workspace provides workspace management for Git repositories.
package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/pkg/logger"
)

// PRRepositoryManager manages PR repository workspaces with update support
type PRRepositoryManager interface {
	// EnsurePRRepository ensures a PR repository is cloned and synchronized
	EnsurePRRepository(ctx context.Context, req *PRRepositoryRequest) (string, error)
}

// PRRepositoryRequest contains PR-specific information
type PRRepositoryRequest struct {
	Provider           provider.Provider
	Owner              string
	Repo               string
	PRNumber           int
	HeadSHA            string // PR head commit SHA
	Workspace          string // Base workspace directory
	Token              string // Authentication token for git operations
	InsecureSkipVerify bool   // Skip SSL certificate verification
}

// DefaultPRRepositoryManager is the default implementation of PRRepositoryManager
type DefaultPRRepositoryManager struct{}

// NewPRRepositoryManager creates a new PR repository manager
func NewPRRepositoryManager() PRRepositoryManager {
	return &DefaultPRRepositoryManager{}
}

// EnsurePRRepository ensures a PR repository is cloned and synchronized
// The clone path format is: {workspace}/{provider}-{owner}-{repo}
// This allows multiple PRs from the same repo to share the same directory,
// with PR switching handled by fetch + checkout operations.
func (m *DefaultPRRepositoryManager) EnsurePRRepository(ctx context.Context, req *PRRepositoryRequest) (string, error) {
	// Log request with authentication status for debugging
	if req.Token != "" {
		logger.Debug("EnsurePRRepository called with authentication",
			zap.String("provider", req.Provider.Name()),
			zap.String("owner", req.Owner),
			zap.String("repo", req.Repo),
			zap.Int("pr_number", req.PRNumber),
			zap.String("head_sha", req.HeadSHA),
			zap.String("token", MaskToken(req.Token)),
			zap.Bool("insecure_skip_verify", req.InsecureSkipVerify),
		)
	} else {
		logger.Debug("EnsurePRRepository called without authentication",
			zap.String("provider", req.Provider.Name()),
			zap.String("owner", req.Owner),
			zap.String("repo", req.Repo),
			zap.Int("pr_number", req.PRNumber),
			zap.String("head_sha", req.HeadSHA),
		)
	}

	// Build clone path: {workspace}/{provider}-{owner}-{repo}
	// Clean owner and repo, replace / with - (handle GitLab nested groups)
	owner := strings.ReplaceAll(req.Owner, "/", "-")
	repo := strings.ReplaceAll(req.Repo, "/", "-")
	cloneDirName := fmt.Sprintf("%s-%s-%s", req.Provider.Name(), owner, repo)
	clonePath := filepath.Join(req.Workspace, cloneDirName)

	// Create workspace directory if not exists
	if err := os.MkdirAll(req.Workspace, 0755); err != nil {
		logger.Error("Failed to create workspace directory",
			zap.String("path", req.Workspace),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Check if directory exists and whether it's a valid git repository
	needClone := false
	info, statErr := os.Stat(clonePath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			// Directory does not exist, need to clone
			needClone = true
		} else {
			// Other stat error
			return "", fmt.Errorf("failed to check clone path: %w", statErr)
		}
	} else if info.IsDir() {
		// Directory exists, check if it's a valid git repository by looking for .git
		gitDir := filepath.Join(clonePath, ".git")
		if _, gitErr := os.Stat(gitDir); os.IsNotExist(gitErr) {
			// Not a valid git repo (missing .git directory), remove and re-clone
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
		// If .git exists, it's a valid repo, needClone stays false
	}

	// Clone if needed
	if needClone {
		logger.Info("Cloning PR using refs",
			zap.String("owner", req.Owner),
			zap.String("repo", req.Repo),
			zap.Int("pr_number", req.PRNumber),
			zap.String("dest", clonePath),
		)

		if err := req.Provider.ClonePR(ctx, req.Owner, req.Repo, req.PRNumber, clonePath, nil); err != nil {
			logger.Error("Failed to clone PR",
				zap.String("owner", req.Owner),
				zap.String("repo", req.Repo),
				zap.Int("pr_number", req.PRNumber),
				zap.Error(err),
			)
			return "", fmt.Errorf("failed to clone PR: %w", err)
		}
		logger.Info("PR cloned successfully", zap.String("path", clonePath))
		return clonePath, nil
	}

	// Repository exists and is valid, check if it needs updating
	logger.Info("Using existing repository", zap.String("path", clonePath))

	// Check if local HEAD SHA matches PR HeadSHA, if not, fetch latest PR code
	localSHA, err := GetLocalHeadSHA(ctx, clonePath)
	if err != nil {
		logger.Warn("Failed to get local HEAD SHA, will fetch PR code anyway",
			zap.String("path", clonePath),
			zap.Error(err),
		)
		localSHA = ""
	}

	// Use PR-specific branch name for better isolation
	prBranchName := fmt.Sprintf("pr-%d", req.PRNumber)

	if localSHA != req.HeadSHA {
		logger.Info("Local PR code is outdated, fetching latest",
			zap.String("local_sha", localSHA),
			zap.String("pr_head_sha", req.HeadSHA),
			zap.Int("pr_number", req.PRNumber),
		)

		// Clean up stale git lock file if exists (from previous crashed process)
		if err := CleanupGitLock(clonePath); err != nil {
			logger.Warn("Failed to cleanup git lock file, continuing anyway",
				zap.String("path", clonePath),
				zap.Error(err),
			)
		}

		// Clean workspace before fetch (remove any local changes and untracked files)
		logger.Info("Cleaning workspace before fetch", zap.String("path", clonePath))
		if err := ResetAndClean(ctx, clonePath); err != nil {
			logger.Warn("Failed to clean workspace, continuing anyway",
				zap.String("path", clonePath),
				zap.Error(err),
			)
		}

		// Checkout to detached HEAD first to avoid "refusing to fetch into checked out branch" error
		if err := CheckoutDetached(ctx, clonePath); err != nil {
			// Log warning but continue, as this is not critical
			logger.Warn("Failed to checkout to detached HEAD, continuing anyway",
				zap.String("path", clonePath),
				zap.Error(err),
			)
		}

		// Fetch latest PR code to PR-specific branch
		prRef := req.Provider.GetPRRef(req.PRNumber)
		fetchOpts := &FetchOptions{
			Token:              req.Token,
			InsecureSkipVerify: req.InsecureSkipVerify,
			ProviderName:       req.Provider.Name(),
		}
		if err := FetchPRRefWithAuth(ctx, clonePath, prRef, prBranchName, fetchOpts); err != nil {
			logger.Error("Failed to fetch PR code",
				zap.String("owner", req.Owner),
				zap.String("repo", req.Repo),
				zap.Int("pr_number", req.PRNumber),
				zap.String("branch", prBranchName),
				zap.Error(err),
			)
			return "", fmt.Errorf("failed to fetch PR code: %w", err)
		}

		// Checkout to latest PR branch
		if err := CheckoutBranch(ctx, clonePath, prBranchName); err != nil {
			logger.Error("Failed to checkout PR branch",
				zap.String("path", clonePath),
				zap.String("branch", prBranchName),
				zap.Error(err),
			)
			return "", fmt.Errorf("failed to checkout PR branch: %w", err)
		}

		// Clean workspace after checkout to ensure pristine state
		logger.Info("Cleaning workspace after checkout", zap.String("path", clonePath))
		if err := ResetAndClean(ctx, clonePath); err != nil {
			logger.Warn("Failed to clean workspace after checkout",
				zap.String("path", clonePath),
				zap.Error(err),
			)
		}

		logger.Info("PR code updated successfully",
			zap.String("path", clonePath),
			zap.String("pr_head_sha", req.HeadSHA),
			zap.String("branch", prBranchName),
		)
	} else {
		logger.Debug("Local PR code is up to date",
			zap.String("sha", localSHA),
		)
	}

	return clonePath, nil
}
