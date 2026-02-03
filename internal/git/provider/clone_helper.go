// Package provider provides helper functions for Git provider implementations.
package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ClonePRParams contains parameters for ClonePRWithRefs helper
type ClonePRParams struct {
	ProviderName       string // Provider name for error messages (e.g., "github", "gitlab")
	RepoURL            string // Git clone URL WITHOUT embedded credentials (P1-2 security improvement)
	Token              string // Authentication token (passed via credential helper, not in URL)
	PRRef              string // PR ref path (e.g., "refs/pull/123/head" or "refs/merge-requests/123/head")
	PRNumber           int    // PR/MR number
	DestPath           string // Destination directory path
	InsecureSkipVerify bool   // Skip SSL certificate verification
}

// createCredentialHelper creates a temporary credential helper script that provides the token
// Returns the script path and a cleanup function that should be deferred
// P1-2 Security improvement: Use credential helper instead of embedding token in URL
func createCredentialHelper(token string) (string, func(), error) {
	// Create temporary script file
	tmpFile, err := os.CreateTemp("", "git-credential-helper-*.sh")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create credential helper: %w", err)
	}

	// Script content that outputs password when git asks for credentials
	// The script simply echoes the token as password
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

// ClonePRWithRefs executes the 4-step PR clone flow using Git refs.
// This is a helper function that providers can optionally use to reduce code duplication.
//
// The 4 steps are:
//  1. git init
//  2. git remote set-url/add origin
//  3. git fetch --no-tags origin {ref}:{branch}
//  4. git checkout {branch}
//
// P1-2 Security improvement: Token is passed via credential helper, not embedded in URL.
// Each provider maintains its own ClonePR method and calls this helper internally.
func ClonePRWithRefs(ctx context.Context, params *ClonePRParams) error {
	if params == nil {
		return &ProviderError{
			Provider: "unknown",
			Message:  "ClonePRParams is nil",
		}
	}

	// Build environment for git commands
	var gitEnv []string
	if params.InsecureSkipVerify {
		gitEnv = append(gitEnv, "GIT_SSL_NO_VERIFY=true")
	}

	// Always set GIT_TERMINAL_PROMPT=0 to prevent interactive prompts
	// This is especially important for tests that don't provide tokens
	gitEnv = append(gitEnv, "GIT_TERMINAL_PROMPT=0")

	// P1-2 Security improvement: Create credential helper for secure token passing
	// Token is NOT embedded in URL anymore
	if params.Token != "" {
		helperPath, cleanup, err := createCredentialHelper(params.Token)
		if err != nil {
			return &ProviderError{
				Provider: params.ProviderName,
				Message:  "failed to create credential helper",
				Err:      err,
			}
		}
		defer cleanup()

		// Use GIT_ASKPASS to provide credentials without embedding in URL
		gitEnv = append(gitEnv,
			"GIT_ASKPASS="+helperPath,
			"GIT_USERNAME=oauth2",
		)
	}

	var stderrBuf strings.Builder

	// Step 1: git init
	cmdInit := exec.CommandContext(ctx, "git", "init", params.DestPath)
	if len(gitEnv) > 0 {
		cmdInit.Env = append(cmdInit.Environ(), gitEnv...)
	}
	cmdInit.Stdout = io.Discard
	cmdInit.Stderr = &stderrBuf
	if err := cmdInit.Run(); err != nil {
		return &ProviderError{
			Provider: params.ProviderName,
			Message:  "failed to init repository",
			Err:      fmt.Errorf("%w: %s", err, stderrBuf.String()),
		}
	}

	// Step 2: Set or add remote origin
	// First try to set-url (in case remote already exists), then fall back to add
	stderrBuf.Reset()
	cmdSetURL := exec.CommandContext(ctx, "git", "-C", params.DestPath, "remote", "set-url", "origin", params.RepoURL)
	if len(gitEnv) > 0 {
		cmdSetURL.Env = append(cmdSetURL.Environ(), gitEnv...)
	}
	cmdSetURL.Stdout = io.Discard
	cmdSetURL.Stderr = &stderrBuf
	if err := cmdSetURL.Run(); err != nil {
		// set-url failed, remote doesn't exist, try to add it
		stderrBuf.Reset()
		cmdRemote := exec.CommandContext(ctx, "git", "-C", params.DestPath, "remote", "add", "origin", params.RepoURL)
		if len(gitEnv) > 0 {
			cmdRemote.Env = append(cmdRemote.Environ(), gitEnv...)
		}
		cmdRemote.Stdout = io.Discard
		cmdRemote.Stderr = &stderrBuf
		if err := cmdRemote.Run(); err != nil {
			return &ProviderError{
				Provider: params.ProviderName,
				Message:  "failed to add remote",
				Err:      fmt.Errorf("%w: %s", err, stderrBuf.String()),
			}
		}
	}

	// Step 3: git fetch --no-tags origin {ref}:{branch}
	// Use PR-specific branch name for better isolation
	prBranchName := fmt.Sprintf("pr-%d", params.PRNumber)
	stderrBuf.Reset()
	cmdFetch := exec.CommandContext(ctx, "git", "-C", params.DestPath, "fetch", "--no-tags", "origin", params.PRRef+":"+prBranchName)
	if len(gitEnv) > 0 {
		cmdFetch.Env = append(cmdFetch.Environ(), gitEnv...)
	}
	cmdFetch.Stdout = io.Discard
	cmdFetch.Stderr = &stderrBuf
	if err := cmdFetch.Run(); err != nil {
		stderrOutput := stderrBuf.String()
		errMsg := buildFetchErrorMessage(params.ProviderName, params.PRNumber, stderrOutput)
		return &ProviderError{
			Provider: params.ProviderName,
			Message:  errMsg,
			Err:      fmt.Errorf("%w: %s", err, stderrOutput),
		}
	}

	// Step 4: git checkout {branch}
	stderrBuf.Reset()
	cmdCheckout := exec.CommandContext(ctx, "git", "-C", params.DestPath, "checkout", prBranchName)
	if len(gitEnv) > 0 {
		cmdCheckout.Env = append(cmdCheckout.Environ(), gitEnv...)
	}
	cmdCheckout.Stdout = io.Discard
	cmdCheckout.Stderr = &stderrBuf
	if err := cmdCheckout.Run(); err != nil {
		return &ProviderError{
			Provider: params.ProviderName,
			Message:  "failed to checkout PR branch",
			Err:      fmt.Errorf("%w: %s", err, stderrBuf.String()),
		}
	}

	return nil
}

// buildFetchErrorMessage builds a user-friendly error message based on stderr output
func buildFetchErrorMessage(providerName string, prNumber int, stderrOutput string) string {
	if strings.Contains(stderrOutput, "couldn't find remote ref") {
		switch providerName {
		case "gitlab":
			return fmt.Sprintf("MR !%d not found or not accessible. For self-hosted GitLab, ensure 'merge_request_fetchable' is enabled in repository settings", prNumber)
		case "gitea":
			return fmt.Sprintf("PR #%d not found or not accessible. Ensure the PR exists and you have access", prNumber)
		default:
			return fmt.Sprintf("PR #%d not found or not accessible. Ensure the PR exists and you have access", prNumber)
		}
	}

	if strings.Contains(stderrOutput, "Authentication failed") {
		tokenEnvVar := fmt.Sprintf("SC_%s_TOKEN", strings.ToUpper(providerName))
		return fmt.Sprintf("authentication failed: check your %s", tokenEnvVar)
	}

	if strings.Contains(stderrOutput, "SSL certificate problem") {
		return "SSL certificate verification failed: consider setting insecure_skip_verify: true"
	}

	return fmt.Sprintf("failed to fetch PR #%d", prNumber)
}
