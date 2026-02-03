// Package prompt provides structured prompt generation from DSL configuration.
// It transforms reviewer DSL into structured prompts for AI agents.
package prompt

// Spec represents a structured prompt specification
// This is the intermediate representation between DSL and the final prompt text
// Note: Output format is handled by llm client layer (via ResponseSchema or MarkdownOutputPrompt)
type Spec struct {
	// ReviewerID is the identifier of the reviewer
	ReviewerID string

	// SystemRole defines the AI's role and persona
	SystemRole SystemRoleSpec

	// Goals defines what the AI should achieve
	Goals GoalsSpec

	// Constraints defines rules and constraints for the review
	Constraints ConstraintsSpec

	// Context provides repository and code context (PR/MR information)
	Context ContextSpec
}

// SystemRoleSpec defines the AI's role and persona (identity only)
type SystemRoleSpec struct {
	// Description describes the reviewer's role and focus areas
	Description string
}

// AreaItem represents an area with its ID and description
type AreaItem struct {
	// ID is the area identifier
	ID string
	// Description is the human-readable description of the area
	Description string
}

// GoalsSpec defines what the AI should achieve (focus areas only)
type GoalsSpec struct {
	// Areas are the focus areas for review with their descriptions
	Areas []AreaItem
}

// ConstraintsSpec defines rules and constraints for review behavior and output
type ConstraintsSpec struct {
	// ScopeControl defines rules about what scope to review
	// e.g., "Review only code changed in this PR"
	ScopeControl []string

	// Avoid lists things to avoid during review (moved from Goals)
	Avoid []string

	// FocusOnIssuesOnly when true, tells the reviewer to only report issues
	// without explaining what the code changes do or praising them
	FocusOnIssuesOnly bool

	// SeverityLevels are the available severity levels (system constant)
	SeverityLevels []string

	// MinSeverity is the minimum severity to report
	// If empty, all findings are reported
	MinSeverity string

	// Tone defines the review tone (e.g., "constructive", "strict")
	// Moved from SystemRole as it's an output constraint
	Tone string

	// Concise enables concise output style
	Concise bool

	// NoEmoji disables emoji in output
	NoEmoji bool

	// NoDate disables date/timestamp in output
	NoDate bool

	// Language specifies the response language (e.g., "Chinese", "English")
	Language string
}

// ContextSpec provides repository and code context (PR/MR information)
type ContextSpec struct {
	// RepoPath is the local repository path
	RepoPath string

	// RepoURL is the remote repository URL
	RepoURL string

	// Ref is the branch/tag/commit being reviewed
	Ref string

	// CommitSHA is the specific commit (head commit of PR)
	CommitSHA string

	// BaseCommitSHA is the base commit SHA for PR diff range
	// Used to identify the commit range: BaseCommitSHA..CommitSHA
	BaseCommitSHA string

	// PRNumber is the PR/MR number if applicable
	PRNumber int

	// PRTitle is the PR/MR title
	PRTitle string

	// PRDescription is the PR/MR description
	PRDescription string

	// ChangedFiles lists files changed in the PR
	ChangedFiles []string

	// Commits lists all commit SHAs in the PR (from BaseCommitSHA to CommitSHA)
	Commits []string

	// Languages lists programming languages in scope
	Languages []string

	// ReferenceDocs lists documentation files to reference during review
	// These files provide additional context for the AI reviewer
	ReferenceDocs []string

	// PreviousReviewForComparison contains the previous review result for historical comparison
	// Only populated when history_compare is enabled
	// The AI will compare current findings with previous ones and mark status
	PreviousReviewForComparison string
}

// ReviewResult represents the raw AI response for a reviewer
// Supports two output modes:
// - JSON Schema mode: structured data in Data field
// - Markdown mode: raw text in Text field
type ReviewResult struct {
	// ReviewerID identifies which reviewer produced this result
	ReviewerID string `json:"reviewer_id"`

	// Data contains the complete AI response as dynamic JSON (for JSON Schema mode)
	// Structure is defined by JSON Schema, not hardcoded
	Data map[string]any `json:"data"`

	// Text stores raw text output (for markdown mode)
	Text string `json:"text,omitempty"`

	// AgentName is the name of the agent used
	AgentName string `json:"agent_name,omitempty"`

	// ModelName is the name of the model used
	ModelName string `json:"model_name,omitempty"`

	// Error contains any error message during review execution
	Error string `json:"error,omitempty"`
}

// NewReviewResult creates a new ReviewResult for a reviewer
func NewReviewResult(reviewerID string) *ReviewResult {
	return &ReviewResult{
		ReviewerID: reviewerID,
		Data:       make(map[string]any),
	}
}
