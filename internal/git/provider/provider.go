// Package provider defines the interface for Git providers.
// Different Git hosting services (GitHub, GitLab, etc.) implement this interface.
package provider

import (
	"context"
	"net/http"
	"strings"
)

// PullRequest represents a pull/merge request
type PullRequest struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"` // open, closed, merged
	HeadBranch  string `json:"head_branch"`
	HeadSHA     string `json:"head_sha"`
	BaseBranch  string `json:"base_branch"`
	BaseSHA     string `json:"base_sha"` // Base commit SHA for diff range
	Author      string `json:"author"`
	URL         string `json:"url"`
}

// WebhookEventType represents the type of webhook event
type WebhookEventType string

const (
	EventTypePush         WebhookEventType = "push"
	EventTypePullRequest  WebhookEventType = "pull_request"
	EventTypeMergeRequest WebhookEventType = "merge_request"
	EventTypeComment      WebhookEventType = "comment"
)

// PREventAction represents PR/MR webhook event actions
// These are normalized action names that should trigger code review
const (
	// PREventActionOpened indicates a PR/MR was opened/created
	PREventActionOpened = "opened"
	// PREventActionSynchronize indicates a PR/MR was updated with new commits
	PREventActionSynchronize = "synchronize"
	// PREventActionReopened indicates a PR/MR was reopened after being closed
	PREventActionReopened = "reopened"
)

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	Type          WebhookEventType `json:"type"`
	Provider      string           `json:"provider"`
	Owner         string           `json:"owner"`
	Repo          string           `json:"repo"`
	Ref           string           `json:"ref"` // branch or tag name
	CommitSHA     string           `json:"commit_sha"`
	PRNumber      int              `json:"pr_number,omitempty"`
	Action        string           `json:"action,omitempty"` // opened, closed, synchronized, etc.
	Sender        string           `json:"sender"`
	PRTitle       string           `json:"pr_title,omitempty"`        // PR/MR title
	PRDescription string           `json:"pr_description,omitempty"`  // PR/MR description/body
	BaseCommitSHA string           `json:"base_commit_sha,omitempty"` // Base commit SHA for PR diff range
	ChangedFiles  []string         `json:"changed_files,omitempty"`   // Files changed in PR/MR
	RawPayload    []byte           `json:"-"`
}

// CloneOptions holds options for cloning a repository
type CloneOptions struct {
	Depth   int    // shallow clone depth (0 for full clone)
	Branch  string // branch to checkout
	Timeout int    // timeout in seconds
}

// CommentOptions holds options for posting a comment
type CommentOptions struct {
	// For PR comments
	PRNumber int
	// For commit comments
	CommitSHA string
	// For line comments
	FilePath  string
	StartLine int
	EndLine   int
}

// Comment represents a comment on a PR or commit
type Comment struct {
	// ID is the unique identifier for the comment
	ID int64 `json:"id"`
	// Body is the comment content
	Body string `json:"body"`
	// Author is the username of the comment author
	Author string `json:"author"`
	// CreatedAt is the creation timestamp
	CreatedAt string `json:"created_at"`
}

// Provider defines the interface for Git hosting providers
type Provider interface {
	// Name returns the provider name (github, gitlab, etc.)
	Name() string

	// GetBaseURL returns the base URL of the provider
	// For public providers: https://github.com, https://gitlab.com
	// For self-hosted: the configured base URL
	GetBaseURL() string

	// Clone clones a repository to the specified path
	Clone(ctx context.Context, owner, repo, destPath string, opts *CloneOptions) error

	// ClonePR clones a specific PR/MR to the destination path using refs
	// GitHub uses refs/pull/<pr>/head, GitLab uses refs/merge-requests/<mr>/head
	// This method works for both fork and non-fork PRs
	ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *CloneOptions) error

	// GetPRRef returns the Git ref for a PR/MR number
	// GitHub: "refs/pull/{pr}/head"
	// GitLab: "refs/merge-requests/{mr}/head"
	GetPRRef(prNumber int) string

	// GetPullRequest retrieves pull request details
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error)

	// ListPullRequests lists open pull requests for a repository
	ListPullRequests(ctx context.Context, owner, repo string) ([]*PullRequest, error)

	// PostComment posts a comment on a PR or commit
	PostComment(ctx context.Context, owner, repo string, opts *CommentOptions, body string) error

	// ListComments lists comments on a PR
	ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*Comment, error)

	// DeleteComment deletes a comment by ID
	DeleteComment(ctx context.Context, owner, repo string, commentID int64) error

	// UpdateComment updates an existing comment by ID
	// For GitLab, prNumber is required to identify the MR containing the note
	UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error

	// ParseWebhook parses an incoming webhook request
	ParseWebhook(r *http.Request, secret string) (*WebhookEvent, error)

	// CreateWebhook creates a webhook for the repository
	CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error)

	// DeleteWebhook deletes a webhook from the repository
	DeleteWebhook(ctx context.Context, owner, repo, webhookID string) error

	// ValidateToken validates the provider token
	ValidateToken(ctx context.Context) error

	// ParseRepoPath parses owner and repo from a repository URL or path.
	// Each provider implements its own parsing logic based on URL format:
	// - GitHub: "owner/repo" (two-level)
	// - GitLab: "group/subgroup/project" (multi-level namespaces supported)
	ParseRepoPath(repoURL string) (owner, repo string, err error)

	// ListBranches lists all branches for a repository
	ListBranches(ctx context.Context, owner, repo string) ([]string, error)

	// MatchesURL checks if the given repository URL matches this provider
	// Each provider implements its own matching logic based on URL patterns
	// Returns true if the URL matches this provider's domain or configured base URL
	MatchesURL(repoURL string) bool
}

// ProviderOptions holds options for creating a provider
type ProviderOptions struct {
	Token              string // access token
	BaseURL            string // base URL for self-hosted instances
	InsecureSkipVerify bool   // skip SSL certificate verification
}

// ProviderFactory creates a provider instance
type ProviderFactory func(opts *ProviderOptions) (Provider, error)

// Registry holds registered provider factories
var Registry = make(map[string]ProviderFactory)

// Register registers a provider factory
func Register(name string, factory ProviderFactory) {
	Registry[name] = factory
}

// Create creates a provider by name
func Create(name string, opts *ProviderOptions) (Provider, error) {
	factory, ok := Registry[name]
	if !ok {
		return nil, &ProviderError{
			Provider: name,
			Message:  "provider not registered",
		}
	}
	return factory(opts)
}

// ProviderError represents a provider-related error
type ProviderError struct {
	Provider string
	Message  string
	Err      error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return "[" + e.Provider + "] " + e.Message + ": " + e.Err.Error()
	}
	return "[" + e.Provider + "] " + e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// ShouldProcessPREvent determines if a PR/MR webhook event action should trigger a code review
// For code review scenarios, we process:
// - opened: PR/MR was just created, needs review
// - synchronize: PR/MR has new commits, needs re-review
// - reopened: PR/MR was reopened, may need review if code was updated
// We skip actions like closed, merged, labeled, etc. as they don't require code review
func ShouldProcessPREvent(action string) bool {
	// Normalize action to lowercase for comparison
	normalized := strings.ToLower(action)

	switch normalized {
	case PREventActionOpened, PREventActionSynchronize, PREventActionReopened:
		return true
	// Handle provider-specific action names
	case "open", "update", "reopen":
		return true
	default:
		return false
	}
}

// IsPRMergedEvent checks if the action indicates a PR/MR was merged or closed
// Used for updating MergedAt timestamp in statistics tracking
func IsPRMergedEvent(action string) bool {
	normalized := strings.ToLower(action)
	switch normalized {
	case "merged", "merge", "closed", "close":
		return true
	default:
		return false
	}
}

// IsPRUpdateEvent checks if the action indicates a PR/MR was updated with new commits
// Used for tracking revision count in statistics
func IsPRUpdateEvent(action string) bool {
	normalized := strings.ToLower(action)
	switch normalized {
	case PREventActionSynchronize, "update":
		return true
	default:
		return false
	}
}
