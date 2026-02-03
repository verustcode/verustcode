// Package dsl provides DSL (Domain Specific Language) parsing and validation
// for review rule configuration. The DSL allows declarative definition of
// code review rules, goals, constraints, and output formats.
package dsl

// Default DSL configuration values
const (
	defaultAgentType       = "cursor"
	defaultSimilarityScore = 0.88
	defaultTone            = "constructive"
)

// AgentConfig defines the AI agent configuration.
// Used by both Review DSL and Report DSL.
// Example YAML:
//
//	agent:
//	  type: cursor
//	  model: sonnet-4.5
type AgentConfig struct {
	// Type specifies which AI agent to use (e.g., "cursor", "gemini")
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Model specifies which model to use (optional, uses agent's default if not specified)
	// Examples: "sonnet-4.5", "gemini-2.5-pro"
	Model string `yaml:"model,omitempty" json:"model,omitempty"`
}

// GetType returns the agent type with default fallback.
func (a *AgentConfig) GetType() string {
	if a.Type == "" {
		return defaultAgentType
	}
	return a.Type
}

// SeverityLevels defines the standard severity levels (from low to high)
// This is a system constant and cannot be customized by users
var SeverityLevels = []string{"info", "low", "medium", "high", "critical"}

// ReviewRulesConfig represents the root configuration containing multiple review rules
type ReviewRulesConfig struct {
	// Version of the DSL schema (for future compatibility)
	Version string `yaml:"version,omitempty"`

	// RuleBase provides base configuration inherited by all review rules
	RuleBase *RuleBaseConfig `yaml:"rule_base,omitempty"`

	// Rules is the list of review rule configurations
	Rules []ReviewRuleConfig `yaml:"rules"`
}

// RuleBaseConfig provides base configuration that can be inherited by review rules
type RuleBaseConfig struct {
	// Agent specifies the default AI agent configuration (type and model)
	// Example:
	//   agent:
	//     type: cursor
	//     model: sonnet-4.5
	Agent AgentConfig `yaml:"agent,omitempty"`

	// Constraints provides default constraints configuration
	Constraints *ConstraintsConfig `yaml:"constraints,omitempty"`

	// Output provides default output configuration
	Output *OutputConfig `yaml:"output,omitempty"`
}

// ReviewRuleConfig represents a single review rule configuration
type ReviewRuleConfig struct {
	// ID is the unique identifier for this review rule
	ID string `yaml:"id" json:"id"`

	// Description provides a detailed description of what this review rule does
	// This describes the reviewer's role and focus areas
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Agent specifies the AI agent configuration (type and model)
	// Example:
	//   agent:
	//     type: cursor
	//     model: sonnet-4.5
	Agent AgentConfig `yaml:"agent,omitempty" json:"agent,omitempty"`

	// ReferenceDocs specifies documentation files to reference during review
	// These files provide additional context for the AI reviewer
	// Paths are relative to the repository root
	// Maximum 5 files allowed
	ReferenceDocs []string `yaml:"reference_docs,omitempty" json:"reference_docs,omitempty"`

	// Goals defines what this review rule should achieve
	Goals GoalsConfig `yaml:"goals" json:"goals"`

	// Constraints defines all constraints for the review (scope, severity)
	Constraints *ConstraintsConfig `yaml:"constraints,omitempty" json:"constraints,omitempty"`

	// Output defines how results should be output
	Output *OutputConfig `yaml:"output,omitempty" json:"output,omitempty"`

	// MultiRun configures multiple review runs for this rule
	MultiRun *MultiRunConfig `yaml:"multi_run,omitempty" json:"multi_run,omitempty"`

	// HistoryCompare configures historical review comparison
	// When enabled, includes previous review result in prompt for comparison
	HistoryCompare *HistoryCompareConfig `yaml:"history_compare,omitempty" json:"history_compare,omitempty"`
}

// MultiRunConfig configures multiple review runs for a single rule
// Multi-run is automatically enabled when Runs >= 2
type MultiRunConfig struct {
	// Runs is the number of times to run the review
	// Multi-run is automatically enabled when Runs >= 2 (max 3 runs)
	Runs int `yaml:"runs,omitempty" json:"runs,omitempty"`

	// Models is a list of models to use for each run
	// If empty, uses the same agent's default model for all runs
	// If specified, must match the number of runs (or will cycle)
	Models []string `yaml:"models,omitempty" json:"models,omitempty"`

	// MergeModel is the model to use for merging results (optional)
	// If not specified, uses the same agent's default model
	MergeModel string `yaml:"merge_model,omitempty" json:"merge_model,omitempty"`
}

// HistoryCompareConfig configures historical review comparison
// When enabled, the prompt will include the last review result for the same PR + rule,
// allowing the AI to compare and indicate status of each issue (FIXED/NEW/PERSISTS)
type HistoryCompareConfig struct {
	// Enabled enables comparison with previous review results
	// When true, the prompt will include the last review result for the same PR + rule
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// GoalsConfig defines what a reviewer should achieve
type GoalsConfig struct {
	// Areas are the focus areas for review (e.g., ["business-logic", "edge-cases"])
	Areas []string `yaml:"areas,omitempty" json:"areas,omitempty"`

	// Avoid lists things the reviewer should avoid (e.g., "formatting-only nitpicks")
	Avoid []string `yaml:"avoid,omitempty" json:"avoid,omitempty"`
}

// ConstraintsConfig defines all constraints for the review
type ConstraintsConfig struct {
	// ScopeControl defines rules about what scope to review
	// e.g., ["Review only code changed in this PR", "Do NOT comment on unrelated legacy code"]
	ScopeControl []string `yaml:"scope_control,omitempty" json:"scope_control,omitempty"`

	// FocusOnIssuesOnly when true, tells the reviewer to only report issues
	// without explaining what the code changes do or praising them
	// Defaults to true
	FocusOnIssuesOnly *bool `yaml:"focus_on_issues_only,omitempty" json:"focus_on_issues_only,omitempty"`

	// Severity configures severity levels
	Severity *SeverityConfig `yaml:"severity,omitempty" json:"severity,omitempty"`

	// Duplicates configures duplicate suppression
	Duplicates *DuplicatesConfig `yaml:"duplicates,omitempty" json:"duplicates,omitempty"`
}

// SeverityConfig configures severity filtering for findings
// Severity levels are system constants: info, low, medium, high, critical
type SeverityConfig struct {
	// MinReport is the minimum severity level to report
	// Findings below this level will be filtered out
	// If empty, all findings are reported
	MinReport string `yaml:"min_report,omitempty" json:"min_report,omitempty"`
}

// OutputStyleConfig configures the output style
type OutputStyleConfig struct {
	// Tone defines the tone of the review (e.g., "strict", "constructive")
	Tone string `yaml:"tone,omitempty" json:"tone,omitempty"`

	// Concise enables concise output style
	// Use pointer to distinguish between unset (nil) and explicitly set to false
	Concise *bool `yaml:"concise,omitempty" json:"concise,omitempty"`

	// NoEmoji disables emoji in output
	// Use pointer to distinguish between unset (nil) and explicitly set to false
	NoEmoji *bool `yaml:"no_emoji,omitempty" json:"no_emoji,omitempty"`

	// NoDate disables date/timestamp in output
	// Use pointer to distinguish between unset (nil) and explicitly set to false
	NoDate *bool `yaml:"no_date,omitempty" json:"no_date,omitempty"`

	// Language specifies the response language (e.g., "Chinese", "English")
	Language string `yaml:"language,omitempty" json:"language,omitempty"`
}

// DuplicatesConfig configures duplicate finding suppression
type DuplicatesConfig struct {
	// SuppressSimilar enables suppression of similar findings
	// Use pointer to distinguish between unset (nil) and explicitly set to false
	SuppressSimilar *bool `yaml:"suppress_similar,omitempty" json:"suppress_similar,omitempty"`

	// Similarity is the threshold for considering findings similar (0.0-1.0)
	Similarity float64 `yaml:"similarity,omitempty" json:"similarity,omitempty"`
}

// OutputConfig defines how results should be output
type OutputConfig struct {
	// Style configures the output style (tone, concise, emoji, date, language)
	Style *OutputStyleConfig `yaml:"style,omitempty" json:"style,omitempty"`

	// Schema defines the output structure using JSON Schema
	// LLM always returns JSON structured data, channels convert to their format as needed
	Schema *OutputSchemaConfig `yaml:"schema,omitempty" json:"schema,omitempty"`

	// Channels is the list of output channels (file, comment, webhook)
	// Each channel can specify its own format (markdown or json)
	Channels []OutputItemConfig `yaml:"channels" json:"channels,omitempty"`
}

// OutputSchemaConfig defines the output schema configuration
// The base schema (summary, findings with core fields) is embedded in code and immutable.
// Users can only add extra fields to findings items via ExtraFields.
// Note: ExtraFields can only be defined at rule level, not in rule_base (no inheritance).
type OutputSchemaConfig struct {
	// ExtraFields defines additional fields to add to each finding item
	// These fields extend the base schema's findings.items.properties
	ExtraFields []ExtraFieldConfig `yaml:"extra_fields,omitempty" json:"extra_fields,omitempty"`
}

// ExtraFieldConfig defines an additional field for findings items
type ExtraFieldConfig struct {
	// Name is the field name (required)
	Name string `yaml:"name" json:"name"`

	// Type is the field type: string, integer, boolean, array (required)
	Type string `yaml:"type" json:"type"`

	// Description provides context for the field (required)
	Description string `yaml:"description" json:"description"`

	// Required indicates if the field must be present in each finding
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`

	// Enum specifies allowed values for string fields (optional)
	Enum []string `yaml:"enum,omitempty" json:"enum,omitempty"`
}

// OutputItemConfig defines a single output configuration
type OutputItemConfig struct {
	// Type specifies the output type: "file", "comment", or "webhook"
	Type string `yaml:"type" json:"type"`

	// Format specifies the output format for this channel: "markdown" or "json"
	// Default values by channel type:
	// - file, comment: "markdown"
	// - webhook: "json"
	Format string `yaml:"format,omitempty" json:"format,omitempty"`

	// File output options (used when Type is "file")

	// Dir is the output directory path where review result files will be saved
	// Can be either a relative or absolute path
	// - If absolute: used directly as the output directory
	// - If relative: combined with opts.OutputDir if provided, otherwise used as-is
	// Files are automatically named based on review ID or timestamp
	Dir string `yaml:"dir,omitempty" json:"dir,omitempty"`

	// Overwrite controls overwrite behavior for file and comment channels
	// For file: allows overwriting existing files
	// For comment: removes existing VerustCode comments before posting a new one
	// Defaults to false (append mode for comments, no overwrite for files)
	Overwrite bool `yaml:"overwrite,omitempty" json:"overwrite,omitempty"`

	// MarkerPrefix is the marker prefix used to identify VerustCode comments
	// The full marker format is: [{marker_prefix}:{rule.id}]
	// Used in overwrite mode to find and delete existing comments for the same rule
	// Defaults to "review_by_scopeview"
	MarkerPrefix string `yaml:"marker_prefix,omitempty" json:"marker_prefix,omitempty"`

	// Webhook output options (used when Type is "webhook")

	// URL is the webhook endpoint URL (required for webhook type)
	URL string `yaml:"url,omitempty" json:"url,omitempty"`

	// HeaderSecret is the secret value for X-SCOPEVIEW-KEY header
	// Used for webhook authentication
	HeaderSecret string `yaml:"header_secret,omitempty" json:"header_secret,omitempty"`

	// Timeout is the HTTP request timeout in seconds (default: 60)
	Timeout int `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// MaxRetries is the maximum number of retry attempts with exponential backoff (default: 6)
	MaxRetries int `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
}

// DefaultReviewRuleConfig returns a ReviewRuleConfig with sensible defaults
func DefaultReviewRuleConfig() *ReviewRuleConfig {
	trueVal := true
	return &ReviewRuleConfig{
		Agent: AgentConfig{
			Type: defaultAgentType,
		},
		Constraints: &ConstraintsConfig{
			// No default severity config - report all levels by default
			FocusOnIssuesOnly: &trueVal,
			Duplicates: &DuplicatesConfig{
				SuppressSimilar: &trueVal,
				Similarity:      defaultSimilarityScore,
			},
		},
		Output: &OutputConfig{
			Style: &OutputStyleConfig{
				Tone:    defaultTone,
				Concise: &trueVal,
				NoEmoji: &trueVal,
				NoDate:  &trueVal,
			},
			// No default channels - user must explicitly configure at least one
			Channels: nil,
		},
	}
}
