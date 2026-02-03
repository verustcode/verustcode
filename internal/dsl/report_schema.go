// Package dsl provides DSL parsing for report configuration.
package dsl

// Note: AgentConfig is defined in schema.go and shared between Review DSL and Report DSL.

// ReportConfig represents a report DSL configuration.
// Each report type is defined in a separate YAML file with this structure.
type ReportConfig struct {
	// Version of the DSL schema
	Version string `yaml:"version,omitempty"`

	// ID is the unique identifier for this report type (e.g., "wiki", "security")
	ID string `yaml:"id"`

	// Name is the display name for this report type
	Name string `yaml:"name"`

	// Description provides details about what this report type generates
	Description string `yaml:"description,omitempty"`

	// Agent specifies the AI agent configuration (type and model)
	// Example:
	//   agent:
	//     type: cursor
	//     model: sonnet-4.5
	Agent AgentConfig `yaml:"agent,omitempty"`

	// Output defines output style and format configuration
	// This is where output.style.language is configured (unified with Review DSL)
	Output ReportOutputConfig `yaml:"output,omitempty"`

	// Structure defines Phase 1: structure generation configuration
	Structure StructurePhase `yaml:"structure"`

	// Section defines Phase 2: section content generation configuration
	Section SectionPhase `yaml:"section"`

	// Summary defines Phase 3: report summary generation configuration
	// This phase collects all section summaries and generates an overall report summary
	// The summary is in Markdown format, used for report introduction and abstract image generation
	Summary SummaryPhase `yaml:"summary"`
}

// ReportOutputConfig defines the output configuration for reports.
// This structure mirrors Review DSL's OutputConfig for consistency.
type ReportOutputConfig struct {
	// Style configures the output style (common fields with Review DSL + report-specific fields)
	Style ReportStyleConfig `yaml:"style,omitempty"`
}

// ReportStyleConfig defines style preferences for report generation.
// Common fields with Review DSL: tone, concise, no_emoji, language
// Report-specific fields: use_mermaid, heading_level, max_section_length, include_line_numbers
type ReportStyleConfig struct {
	// ===== Common fields (same naming as Review DSL OutputStyleConfig) =====

	// Tone specifies the writing tone: "professional", "friendly", "technical", "strict", "constructive"
	Tone string `yaml:"tone,omitempty"`

	// Concise enables concise output style
	Concise *bool `yaml:"concise,omitempty"`

	// NoEmoji disables emoji in output
	NoEmoji *bool `yaml:"no_emoji,omitempty"`

	// Language specifies the output language (e.g., "zh-cn", "en", "Chinese", "English")
	// This replaces the old top-level output_language field
	Language string `yaml:"language,omitempty"`

	// ===== Report-specific fields =====

	// UseMermaid enables Mermaid diagram generation
	UseMermaid *bool `yaml:"use_mermaid,omitempty"`

	// HeadingLevel specifies the starting heading level (1-4, default: 2)
	HeadingLevel int `yaml:"heading_level,omitempty"`

	// MaxSectionLength limits the maximum length of each section (in characters)
	// Minimum value is 500 if specified
	MaxSectionLength int `yaml:"max_section_length,omitempty"`

	// IncludeLineNumbers enables line number references in code citations
	IncludeLineNumbers *bool `yaml:"include_line_numbers,omitempty"`
}

// ReportGoalsConfig defines what a phase should focus on.
// Unlike Review DSL which uses predefined area IDs, Report DSL uses free-text topics
// to describe focus points for documentation generation.
type ReportGoalsConfig struct {
	// Topics lists the focus topics for this phase
	// Free text descriptions - allows flexible configuration for different report types
	// Example: ["Technology stack", "Project overview", "Quick start guide"]
	Topics []string `yaml:"topics,omitempty"`

	// Avoid lists things to avoid during this phase
	Avoid []string `yaml:"avoid,omitempty"`
}

// StructurePhase defines Phase 1 configuration for report structure generation.
// Phase 1 output is always JSON with a fixed schema.
type StructurePhase struct {
	// Description explains what this phase should accomplish
	Description string `yaml:"description,omitempty"`

	// Goals defines focus areas and things to avoid
	Goals ReportGoalsConfig `yaml:"goals,omitempty"`

	// Constraints lists constraints for structure generation
	Constraints []string `yaml:"constraints,omitempty"`

	// ReferenceDocs lists document files to include in the prompt
	// These files will be read and included as reference material for AI
	// File paths are relative to the project root
	// Example: ["README.md", "docs/architecture.md"]
	ReferenceDocs []string `yaml:"reference_docs,omitempty"`

	// Nested enables two-level section structure (parent sections + subsections)
	// When true, sections can have subsections (one level of nesting)
	// Phase 2 will only generate content for leaf nodes (sections without subsections)
	Nested bool `yaml:"nested,omitempty"`
}

// SectionPhase defines Phase 2 configuration for section content generation.
// Phase 2 output is always Markdown.
// Note: Style configuration has been moved to output.style for consistency with Review DSL.
type SectionPhase struct {
	// Description explains what this phase should accomplish
	Description string `yaml:"description,omitempty"`

	// Goals defines focus topics and things to avoid
	Goals ReportGoalsConfig `yaml:"goals,omitempty"`

	// Constraints lists constraints for section generation
	Constraints []string `yaml:"constraints,omitempty"`

	// ReferenceDocs lists document files to include in the prompt
	// These files will be read and included as reference material for AI
	// File paths are relative to the project root
	ReferenceDocs []string `yaml:"reference_docs,omitempty"`

	// Summary configures the section-level summary generation
	// Each section will generate a short summary alongside its content
	// These summaries are used as input for Phase 3 report summary generation
	Summary SectionSummaryConfig `yaml:"summary,omitempty"`
}

// SectionSummaryConfig configures summary generation for individual sections.
// Each section generates a short summary during Phase 2 content generation.
type SectionSummaryConfig struct {
	// MaxLength specifies the maximum length (in characters) for each section summary
	// Default: 200 characters
	MaxLength int `yaml:"max_length,omitempty"`
}

// SummaryPhase defines Phase 3 configuration for report summary generation.
// Phase 3 collects all section summaries and generates an overall report summary.
// The output is always Markdown, suitable for report introduction and abstract image generation.
type SummaryPhase struct {
	// Description explains what this phase should accomplish
	Description string `yaml:"description,omitempty"`

	// Goals defines focus topics and things to avoid for summary generation
	Goals ReportGoalsConfig `yaml:"goals,omitempty"`

	// Constraints lists constraints for summary generation
	Constraints []string `yaml:"constraints,omitempty"`
}

// DefaultReportConfig returns a ReportConfig with sensible defaults.
func DefaultReportConfig() *ReportConfig {
	useMermaid := true
	concise := true
	noEmoji := true
	includeLineNumbers := true

	return &ReportConfig{
		Version: "1.0",
		Agent: AgentConfig{
			Type: "cursor",
		},
		Output: ReportOutputConfig{
			Style: ReportStyleConfig{
				Tone:               "professional",
				Concise:            &concise,
				NoEmoji:            &noEmoji,
				Language:           "Chinese",
				UseMermaid:         &useMermaid,
				HeadingLevel:       2,
				IncludeLineNumbers: &includeLineNumbers,
			},
		},
		Structure: StructurePhase{
			Description: "分析项目并生成文档结构框架。",
		},
		Section: SectionPhase{
			Description: "为每个章节生成详细的文档内容。",
			Summary: SectionSummaryConfig{
				MaxLength: 200,
			},
		},
		Summary: SummaryPhase{
			Description: "基于各章节摘要，生成项目的整体概览。",
		},
	}
}

// ApplyDefaults applies default values to empty fields.
func (c *ReportConfig) ApplyDefaults() {
	defaults := DefaultReportConfig()

	// Apply agent defaults
	if c.Agent.Type == "" {
		c.Agent.Type = defaults.Agent.Type
	}

	// Apply output.style defaults
	if c.Output.Style.Tone == "" {
		c.Output.Style.Tone = defaults.Output.Style.Tone
	}
	if c.Output.Style.Concise == nil {
		c.Output.Style.Concise = defaults.Output.Style.Concise
	}
	if c.Output.Style.NoEmoji == nil {
		c.Output.Style.NoEmoji = defaults.Output.Style.NoEmoji
	}
	if c.Output.Style.Language == "" {
		c.Output.Style.Language = defaults.Output.Style.Language
	}
	if c.Output.Style.UseMermaid == nil {
		c.Output.Style.UseMermaid = defaults.Output.Style.UseMermaid
	}
	if c.Output.Style.HeadingLevel == 0 {
		c.Output.Style.HeadingLevel = defaults.Output.Style.HeadingLevel
	}
	if c.Output.Style.IncludeLineNumbers == nil {
		c.Output.Style.IncludeLineNumbers = defaults.Output.Style.IncludeLineNumbers
	}

	// Apply phase defaults
	if c.Structure.Description == "" {
		c.Structure.Description = defaults.Structure.Description
	}
	if c.Section.Description == "" {
		c.Section.Description = defaults.Section.Description
	}
	if c.Section.Summary.MaxLength == 0 {
		c.Section.Summary.MaxLength = defaults.Section.Summary.MaxLength
	}
	if c.Summary.Description == "" {
		c.Summary.Description = defaults.Summary.Description
	}
}

// GetUseMermaid returns whether to use Mermaid diagrams (default: true).
func (s *ReportStyleConfig) GetUseMermaid() bool {
	if s.UseMermaid == nil {
		return true
	}
	return *s.UseMermaid
}

// GetConcise returns whether to use concise style (default: true).
func (s *ReportStyleConfig) GetConcise() bool {
	if s.Concise == nil {
		return true
	}
	return *s.Concise
}

// GetNoEmoji returns whether to disable emoji (default: true).
func (s *ReportStyleConfig) GetNoEmoji() bool {
	if s.NoEmoji == nil {
		return true
	}
	return *s.NoEmoji
}

// GetHeadingLevel returns the starting heading level (default: 2).
func (s *ReportStyleConfig) GetHeadingLevel() int {
	if s.HeadingLevel == 0 {
		return 2
	}
	return s.HeadingLevel
}

// GetIncludeLineNumbers returns whether to include line numbers (default: true).
func (s *ReportStyleConfig) GetIncludeLineNumbers() bool {
	if s.IncludeLineNumbers == nil {
		return true
	}
	return *s.IncludeLineNumbers
}

// GetLanguage returns the output language (default: "Chinese").
func (s *ReportStyleConfig) GetLanguage() string {
	if s.Language == "" {
		return "Chinese"
	}
	return s.Language
}

// GetMaxLength returns the max length for section summary (default: 200).
func (c *SectionSummaryConfig) GetMaxLength() int {
	if c.MaxLength == 0 {
		return 200
	}
	return c.MaxLength
}
