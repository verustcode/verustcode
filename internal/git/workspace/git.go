// Package workspace provides workspace management for Git repositories.
// It handles cloning, updating, and synchronizing local repository workspaces.
package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/pkg/logger"
)

const (
	// GitOperationTimeout is the default timeout for git operations
	// This prevents operations like 'git clean' from blocking indefinitely
	GitOperationTimeout = 5 * time.Minute
)

// FetchOptions contains options for fetch operations with authentication support
type FetchOptions struct {
	// Token is the authentication token for the git operation
	// It will be passed via GIT_ASKPASS credential helper
	Token string

	// InsecureSkipVerify skips SSL certificate verification
	InsecureSkipVerify bool

	// ProviderName is used for logging and error messages (e.g., "gitlab", "github")
	ProviderName string
}

// MaskToken masks a token for safe logging, showing first 4 and last 4 characters
// Returns "****" for tokens <= 8 characters, or "xxxx...xxxx" format for longer tokens
func MaskToken(token string) string {
	if token == "" {
		return "(empty)"
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// createCredentialHelper creates a temporary credential helper script that provides the token
// Returns the script path and a cleanup function that should be deferred
// This uses GIT_ASKPASS mechanism for secure token passing
func createCredentialHelper(token string) (string, func(), error) {
	// Create temporary script file
	tmpFile, err := os.CreateTemp("", "git-credential-helper-*.sh")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create credential helper: %w", err)
	}

	// Script content that outputs password when git asks for credentials
	var scriptContent string
	if runtime.GOOS == "windows" {
		// Windows batch script
		scriptContent = fmt.Sprintf("@echo off\necho password=%s\n", token)
	} else {
		// Unix shell script
		scriptContent = fmt.Sprintf("#!/bin/sh\necho \"password=%s\"\n", token)
	}

	if _, err := tmpFile.WriteString(scriptContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("failed to write credential helper: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("failed to close credential helper: %w", err)
	}

	// Make script executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
			os.Remove(tmpFile.Name())
			return "", nil, fmt.Errorf("failed to make credential helper executable: %w", err)
		}
	}

	// Cleanup function to remove the temporary script
	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup, nil
}

// GetLocalHeadSHA gets the current HEAD SHA of the local repository
func GetLocalHeadSHA(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get local HEAD SHA: %w (stderr: %s)", err, stderr.String())
	}

	sha := strings.TrimSpace(stdout.String())
	return sha, nil
}

// CheckoutDetached checks out to a detached HEAD state
// This is useful before fetching into a branch that is currently checked out
func CheckoutDetached(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "--detach")
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		logger.Warn("Failed to checkout to detached HEAD",
			zap.String("path", repoPath),
			zap.String("stderr", stderrBuf.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to checkout to detached HEAD: %w (stderr: %s)", err, stderrBuf.String())
	}

	return nil
}

// FetchPRRef fetches a PR ref from the remote repository without authentication.
// DEPRECATED: Use FetchPRRefWithAuth for repositories requiring authentication.
// If the fetch fails due to non-fast-forward (e.g., PR was rebased or force-pushed),
// it will delete the local branch and retry the fetch.
func FetchPRRef(ctx context.Context, repoPath, prRef, localBranch string) error {
	return FetchPRRefWithAuth(ctx, repoPath, prRef, localBranch, nil)
}

// FetchPRRefWithAuth fetches a PR ref from the remote repository with optional authentication.
// If the fetch fails due to non-fast-forward (e.g., PR was rebased or force-pushed),
// it will delete the local branch and retry the fetch.
// The FetchOptions parameter provides authentication via GIT_ASKPASS credential helper.
func FetchPRRefWithAuth(ctx context.Context, repoPath, prRef, localBranch string, opts *FetchOptions) error {
	// Log authentication status for debugging
	if opts != nil && opts.Token != "" {
		logger.Debug("Fetching PR ref with authentication",
			zap.String("path", repoPath),
			zap.String("prRef", prRef),
			zap.String("localBranch", localBranch),
			zap.String("provider", opts.ProviderName),
			zap.String("token", MaskToken(opts.Token)),
		)
	} else {
		logger.Debug("Fetching PR ref without authentication",
			zap.String("path", repoPath),
			zap.String("prRef", prRef),
			zap.String("localBranch", localBranch),
		)
	}

	err := fetchPRRefOnce(ctx, repoPath, prRef, localBranch, opts)
	if err == nil {
		return nil
	}

	// Check if the error is due to non-fast-forward
	if strings.Contains(err.Error(), "non-fast-forward") {
		logger.Info("Fetch failed due to non-fast-forward, deleting local branch and retrying",
			zap.String("path", repoPath),
			zap.String("prRef", prRef),
			zap.String("localBranch", localBranch),
		)

		// Delete the local branch and retry
		if delErr := DeleteLocalBranch(ctx, repoPath, localBranch); delErr != nil {
			return fmt.Errorf("failed to delete local branch before retry: %w (original error: %v)", delErr, err)
		}

		// Retry the fetch
		if retryErr := fetchPRRefOnce(ctx, repoPath, prRef, localBranch, opts); retryErr != nil {
			return fmt.Errorf("failed to fetch PR ref after retry: %w", retryErr)
		}

		logger.Info("Successfully fetched PR ref after deleting local branch",
			zap.String("path", repoPath),
			zap.String("prRef", prRef),
		)
		return nil
	}

	return err
}

// fetchPRRefOnce performs a single fetch attempt for a PR ref with optional authentication
func fetchPRRefOnce(ctx context.Context, repoPath, prRef, localBranch string, opts *FetchOptions) error {
	// Create a timeout context to prevent long-running fetch operations
	timeoutCtx, cancel := context.WithTimeout(ctx, GitOperationTimeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "fetch", "--no-tags", "origin", prRef+":"+localBranch)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// Build environment with authentication if provided
	var gitEnv []string
	// Always set GIT_TERMINAL_PROMPT=0 to prevent interactive prompts
	// This is especially important for tests that don't provide tokens
	gitEnv = append(gitEnv, "GIT_TERMINAL_PROMPT=0")

	if opts != nil {
		if opts.InsecureSkipVerify {
			gitEnv = append(gitEnv, "GIT_SSL_NO_VERIFY=true")
		}

		// Set up credential helper for secure token passing
		if opts.Token != "" {
			helperPath, cleanup, err := createCredentialHelper(opts.Token)
			if err != nil {
				logger.Error("Failed to create credential helper for fetch",
					zap.Error(err),
					zap.String("path", repoPath),
					zap.String("token", MaskToken(opts.Token)),
				)
				return fmt.Errorf("failed to create credential helper: %w", err)
			}
			defer cleanup()

			// Use GIT_ASKPASS to provide credentials without embedding in URL
			gitEnv = append(gitEnv,
				"GIT_ASKPASS="+helperPath,
				"GIT_USERNAME=oauth2",
			)

			logger.Debug("Credential helper configured for fetch",
				zap.String("path", repoPath),
				zap.String("token", MaskToken(opts.Token)),
			)
		}
	}

	// Apply environment variables to the command
	if len(gitEnv) > 0 {
		cmd.Env = append(cmd.Environ(), gitEnv...)
	}

	if err := cmd.Run(); err != nil {
		stderrOutput := stderrBuf.String()
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git fetch timed out after %v: %w", GitOperationTimeout, err)
		}

		// Provide detailed error logging for authentication failures
		if strings.Contains(stderrOutput, "could not read Username") ||
			strings.Contains(stderrOutput, "Authentication failed") ||
			strings.Contains(stderrOutput, "not found or you don't have permission") {
			tokenStatus := "(no token provided)"
			if opts != nil && opts.Token != "" {
				tokenStatus = MaskToken(opts.Token)
			}
			logger.Error("Git fetch authentication failed",
				zap.String("path", repoPath),
				zap.String("prRef", prRef),
				zap.String("token", tokenStatus),
				zap.String("stderr", stderrOutput),
			)
		}

		return fmt.Errorf("failed to fetch PR ref: %w (stderr: %s)", err, stderrOutput)
	}

	return nil
}

// CheckoutBranch checks out to a specific branch
func CheckoutBranch(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", branch)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w (stderr: %s)", branch, err, stderrBuf.String())
	}

	return nil
}

// CleanupGitLock removes stale .git/index.lock file if it exists.
// This handles cases where a previous git process crashed and left the lock behind.
// In serial-per-repo scenarios, a lock file likely indicates an abnormal exit.
func CleanupGitLock(repoPath string) error {
	lockPath := filepath.Join(repoPath, ".git", "index.lock")
	if _, err := os.Stat(lockPath); err == nil {
		logger.Warn("Removing stale git index lock file",
			zap.String("path", lockPath),
		)
		if err := os.Remove(lockPath); err != nil {
			return fmt.Errorf("failed to remove git lock file: %w", err)
		}
	}
	return nil
}

// DeleteLocalBranch forcefully deletes a local branch.
// This is useful when we need to re-fetch a PR ref that has been rebased or force-pushed.
func DeleteLocalBranch(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "-D", branch)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		stderrStr := stderrBuf.String()
		// Branch not existing is not an error for our use case
		// Check for both English and Chinese error messages
		if strings.Contains(stderrStr, "not found") || strings.Contains(stderrStr, "未发现") {
			return nil
		}
		return fmt.Errorf("failed to delete branch %s: %w (stderr: %s)", branch, err, stderrStr)
	}

	logger.Debug("Deleted local branch",
		zap.String("path", repoPath),
		zap.String("branch", branch),
	)
	return nil
}

// ResetAndClean performs git reset --hard and git clean -fd
// This ensures the working directory is clean before operations
func ResetAndClean(ctx context.Context, repoPath string) error {
	// Create a timeout context to prevent long-running operations
	timeoutCtx, cancel := context.WithTimeout(ctx, GitOperationTimeout)
	defer cancel()

	// git reset --hard
	resetCmd := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "reset", "--hard")
	var resetStderr bytes.Buffer
	resetCmd.Stderr = &resetStderr
	if err := resetCmd.Run(); err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git reset --hard timed out after %v: %w", GitOperationTimeout, err)
		}
		return fmt.Errorf("git reset --hard failed: %w (stderr: %s)", err, resetStderr.String())
	}

	// git clean -fd (remove untracked files and directories)
	cleanCmd := exec.CommandContext(timeoutCtx, "git", "-C", repoPath, "clean", "-fd")
	var cleanStderr bytes.Buffer
	cleanCmd.Stderr = &cleanStderr
	if err := cleanCmd.Run(); err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git clean -fd timed out after %v: %w", GitOperationTimeout, err)
		}
		return fmt.Errorf("git clean -fd failed: %w (stderr: %s)", err, cleanStderr.String())
	}

	logger.Info("Workspace reset and cleaned",
		zap.String("path", repoPath),
	)
	return nil
}
