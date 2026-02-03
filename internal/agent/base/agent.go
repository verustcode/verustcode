// Package base defines the Agent interface for AI-powered code review.
// Different AI CLI tools (Cursor, Gemini, etc.) implement this interface.
package base

import (
	"context"
	"time"

	"github.com/verustcode/verustcode/internal/store"
)

// ReviewRequest represents a request to review code
type ReviewRequest struct {
	// Repository information
	RepoPath  string `json:"repo_path"`  // Local path to cloned repository
	RepoURL   string `json:"repo_url"`   // Remote repository URL
	Owner     string `json:"owner"`      // Repository owner
	RepoName  string `json:"repo_name"`  // Repository name
	Ref       string `json:"ref"`        // Branch, tag, or commit
	CommitSHA string `json:"commit_sha"` // Specific commit to review

	// Context for focused review
	PRNumber     int      `json:"pr_number,omitempty"`     // PR number for context
	PRTitle      string   `json:"pr_title,omitempty"`      // PR title for context
	PRBody       string   `json:"pr_body,omitempty"`       // PR description
	ChangedFiles []string `json:"changed_files,omitempty"` // Files changed in PR

	// Metadata
	RequestID string `json:"request_id"`          // Unique request identifier
	RuleID    string `json:"rule_id,omitempty"`   // Review rule ID (optional, for tracking)
	ReviewID  string `json:"review_id,omitempty"` // Review ID (optional, for tracking)

	// Model is an optional model override for this request
	Model string `json:"model,omitempty"`
}

// ReviewResult represents the raw AI response for a code review
// The Data field contains the complete AI response as dynamic JSON
// Structure is entirely determined by JSON Schema - no assumptions about fields
type ReviewResult struct {
	// Request reference
	RequestID string `json:"request_id"`

	// Data contains the complete AI response as dynamic JSON (for JSON Schema mode)
	// Structure is defined by JSON Schema, not hardcoded
	Data map[string]any `json:"data"`

	// Text stores raw text output (for markdown mode)
	Text string `json:"text,omitempty"`

	// Timing
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Duration    time.Duration `json:"duration"`

	// Agent information
	AgentName    string `json:"agent_name"`
	AgentVersion string `json:"agent_version,omitempty"`
	ModelName    string `json:"model_name,omitempty"` // Model name used for this review

	// Error handling
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Agent defines the interface for AI code review agents
type Agent interface {
	// Name returns the agent identifier
	Name() string

	// Version returns the agent version
	Version() string

	// Available checks if the agent is available for use
	Available() bool

	// ExecuteWithPrompt performs code review using a custom prompt (DSL mode)
	ExecuteWithPrompt(ctx context.Context, req *ReviewRequest, prompt string) (*ReviewResult, error)

	// SetStore sets the database store for reading runtime configuration.
	// Agent reads configuration from database on each execution.
	SetStore(s store.Store)
}

// AgentFactory creates an Agent instance
type AgentFactory func() (Agent, error)

// Registry holds registered agent factories
var Registry = make(map[string]AgentFactory)

// Register registers an agent factory
func Register(name string, factory AgentFactory) {
	Registry[name] = factory
}

// Create creates an agent by name
func Create(name string) (Agent, error) {
	factory, ok := Registry[name]
	if !ok {
		return nil, &AgentError{
			Agent:   name,
			Message: "agent not registered",
		}
	}
	return factory()
}

// List returns all registered agent names
func List() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}

// AgentError represents an agent-related error
type AgentError struct {
	Agent   string
	Message string
	Err     error
}

func (e *AgentError) Error() string {
	if e.Err != nil {
		return "[agent:" + e.Agent + "] " + e.Message + ": " + e.Err.Error()
	}
	return "[agent:" + e.Agent + "] " + e.Message
}

func (e *AgentError) Unwrap() error {
	return e.Err
}

// NewResult creates a new ReviewResult with initialized fields
func NewResult(requestID, agentName string) *ReviewResult {
	return &ReviewResult{
		RequestID: requestID,
		Data:      make(map[string]any),
		AgentName: agentName,
		Success:   true,
	}
}
