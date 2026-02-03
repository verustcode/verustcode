// Package task defines task-related types for the engine.
// These types are used throughout the engine package for review task management.
package task

import (
	"time"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/model"
)

// Task represents a review task that is submitted to the engine for processing.
// It contains all the information needed to execute a code review.
// The task is identified by Review.ID (UUID), there is no separate task ID.
type Task struct {
	// Review is the database model for this review (ID is used as task identifier)
	Review *model.Review

	// Request contains the review request parameters
	Request *base.ReviewRequest

	// ProviderName is the Git provider name (github, gitlab, etc.)
	ProviderName string

	// AgentName is the AI agent name to use for this review
	AgentName string

	// CreatedAt is when this task was created
	CreatedAt time.Time

	// BaseCommitSHA is the base commit SHA for PR diff range (from webhook)
	BaseCommitSHA string

	// ReviewRulesConfig is the DSL configuration for this review
	ReviewRulesConfig *dsl.ReviewRulesConfig

	// OutputDir is the output directory for file channels
	OutputDir string
}

// ReviewRequest represents a request to run a DSL-driven review.
// This is used internally by the engine to pass review parameters.
type ReviewRequest struct {
	// RepoPath is the local path to the repository
	RepoPath string

	// RepoURL is the remote repository URL (optional)
	RepoURL string

	// Ref is the branch/tag/commit
	Ref string

	// CommitSHA is the specific commit (head commit of PR)
	CommitSHA string

	// BaseCommitSHA is the base commit SHA for PR diff range
	// Used to identify the commit range: BaseCommitSHA..CommitSHA
	BaseCommitSHA string

	// PRNumber is the PR/MR number (optional)
	PRNumber int

	// PRTitle is the PR/MR title
	PRTitle string

	// PRDescription is the PR/MR description
	PRDescription string

	// ChangedFiles lists changed files (for diff-based context)
	ChangedFiles []string

	// ReviewRulesConfig is the DSL configuration
	ReviewRulesConfig *dsl.ReviewRulesConfig

	// OutputDir is the output directory for file channels
	OutputDir string
}

// PRInfo contains additional PR information from webhook.
// This is used to pass PR metadata when submitting a review task.
type PRInfo struct {
	// Title is the PR/MR title
	Title string

	// Description is the PR/MR description/body
	Description string

	// BaseSHA is the base commit SHA
	BaseSHA string

	// ChangedFiles lists the files changed in this PR
	ChangedFiles []string
}

// RunResult represents a single review run result.
// Used internally for multi-run review aggregation.
type RunResult struct {
	// Index is the run index (1-based)
	Index int

	// Model is the AI model used for this run
	Model string

	// Data is the complete AI response as JSON
	Data map[string]interface{}

	// Text is the raw text output from the agent
	Text string

	// Duration is how long the run took
	Duration time.Duration

	// Err is any error that occurred during the run
	Err error
}
