package prompt

import (
	"github.com/verustcode/verustcode/internal/dsl"
)

// Builder builds prompt specifications from DSL configuration
type Builder struct{}

// NewBuilder creates a new prompt builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Build converts a ReviewRuleConfig to a Spec
func (b *Builder) Build(rule *dsl.ReviewRuleConfig, ctx *BuildContext) *Spec {
	spec := &Spec{
		ReviewerID:  rule.ID,
		SystemRole:  b.buildSystemRole(rule),
		Goals:       b.buildGoals(rule),
		Constraints: b.buildConstraints(rule, ctx),
		Context:     b.buildContext(rule, ctx),
	}

	return spec
}

// BuildContext provides additional context for building prompts
type BuildContext struct {
	// RepoPath is the local repository path
	RepoPath string

	// RepoURL is the remote repository URL
	RepoURL string

	// Ref is the branch/tag/commit
	Ref string

	// CommitSHA is the specific commit (head commit of PR)
	CommitSHA string

	// BaseCommitSHA is the base commit SHA for PR diff range
	// Used to identify the commit range: BaseCommitSHA..CommitSHA
	BaseCommitSHA string

	// PRNumber is the PR/MR number
	PRNumber int

	// PRTitle is the PR/MR title
	PRTitle string

	// PRDescription is the PR/MR description
	PRDescription string

	// ChangedFiles lists changed files
	ChangedFiles []string

	// Commits lists all commit SHAs in the PR (from BaseCommitSHA to CommitSHA)
	// This is obtained using git rev-list base..head
	Commits []string

	// OutputLanguage is the language instruction for the AI to respond in
	// This is a human-readable instruction like "Please respond in Chinese."
	// This is used as a fallback if not specified in DSL output.style.language
	OutputLanguage string

	// PreviousReviewForComparison is the summary from the previous review of the same PR + rule
	// Used for historical comparison when history_compare is enabled
	PreviousReviewForComparison string
}

// buildSystemRole builds the system role specification (identity only)
func (b *Builder) buildSystemRole(rule *dsl.ReviewRuleConfig) SystemRoleSpec {
	return SystemRoleSpec{
		Description: rule.Description,
	}
}

// buildGoals builds the goals specification (focus areas with descriptions)
func (b *Builder) buildGoals(rule *dsl.ReviewRuleConfig) GoalsSpec {
	items := make([]AreaItem, 0, len(rule.Goals.Areas))
	for _, areaID := range rule.Goals.Areas {
		desc, _ := dsl.GetAreaDescription(areaID)
		items = append(items, AreaItem{ID: areaID, Description: desc})
	}
	return GoalsSpec{
		Areas: items,
	}
}

// buildConstraints builds the constraints specification
// Consolidates all constraints: scope control, avoid list, severity, and output style
func (b *Builder) buildConstraints(rule *dsl.ReviewRuleConfig, ctx *BuildContext) ConstraintsSpec {
	// Initialize with system-defined severity levels and default output style
	constraints := ConstraintsSpec{
		SeverityLevels:    dsl.SeverityLevels,
		FocusOnIssuesOnly: true,           // default: only report issues, don't explain changes
		Tone:              "constructive", // default tone
		Concise:           true,
		NoEmoji:           true,
		NoDate:            true,
	}

	// Apply avoid list from goals (moved from Goals to Constraints)
	if len(rule.Goals.Avoid) > 0 {
		constraints.Avoid = rule.Goals.Avoid
	}

	// Apply constraints settings
	if rule.Constraints != nil {
		// Apply scope control settings
		if len(rule.Constraints.ScopeControl) > 0 {
			constraints.ScopeControl = rule.Constraints.ScopeControl
		}

		// Apply focus on issues only setting
		if rule.Constraints.FocusOnIssuesOnly != nil {
			constraints.FocusOnIssuesOnly = *rule.Constraints.FocusOnIssuesOnly
		}

		// Apply severity settings (only min_report is configurable)
		if rule.Constraints.Severity != nil && rule.Constraints.Severity.MinReport != "" {
			constraints.MinSeverity = rule.Constraints.Severity.MinReport
		}
	}

	// Apply output style settings (including Tone, moved from SystemRole)
	if rule.Output != nil && rule.Output.Style != nil {
		if rule.Output.Style.Tone != "" {
			constraints.Tone = rule.Output.Style.Tone
		}
		if rule.Output.Style.Concise != nil {
			constraints.Concise = *rule.Output.Style.Concise
		}
		if rule.Output.Style.NoEmoji != nil {
			constraints.NoEmoji = *rule.Output.Style.NoEmoji
		}
		if rule.Output.Style.NoDate != nil {
			constraints.NoDate = *rule.Output.Style.NoDate
		}
		if rule.Output.Style.Language != "" {
			constraints.Language = rule.Output.Style.Language
		}
	}

	// Apply language from build context as fallback
	if constraints.Language == "" && ctx != nil && ctx.OutputLanguage != "" {
		constraints.Language = ctx.OutputLanguage
	}

	return constraints
}

// buildContext builds the context specification (PR/MR information)
func (b *Builder) buildContext(rule *dsl.ReviewRuleConfig, ctx *BuildContext) ContextSpec {
	spec := ContextSpec{}

	// Apply build context if provided (PR/MR information)
	if ctx != nil {
		spec.RepoPath = ctx.RepoPath
		spec.RepoURL = ctx.RepoURL
		spec.Ref = ctx.Ref
		spec.CommitSHA = ctx.CommitSHA
		spec.BaseCommitSHA = ctx.BaseCommitSHA
		spec.PRNumber = ctx.PRNumber
		spec.PRTitle = ctx.PRTitle
		spec.PRDescription = ctx.PRDescription
		spec.ChangedFiles = ctx.ChangedFiles
		spec.Commits = ctx.Commits
		spec.PreviousReviewForComparison = ctx.PreviousReviewForComparison
	}

	// Apply reference docs from rule configuration
	if len(rule.ReferenceDocs) > 0 {
		spec.ReferenceDocs = rule.ReferenceDocs
	}

	return spec
}

// BuildAll builds prompt specifications for all review rules
func (b *Builder) BuildAll(config *dsl.ReviewRulesConfig, ctx *BuildContext) []*Spec {
	specs := make([]*Spec, len(config.Rules))
	for i := range config.Rules {
		specs[i] = b.Build(&config.Rules[i], ctx)
	}
	return specs
}
