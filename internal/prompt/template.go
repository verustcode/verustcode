package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// Renderer renders prompt specifications into prompt text
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer creates a new prompt renderer
func NewRenderer() *Renderer {
	r := &Renderer{}
	r.initTemplates()
	return r
}

// initTemplates initializes the prompt templates
func (r *Renderer) initTemplates() {
	funcMap := template.FuncMap{
		"join":     strings.Join,
		"indent":   indent,
		"bullet":   bullet,
		"numbered": numbered,
		"quote":    quote,
		"add":      func(a, b int) int { return a + b },
	}

	r.tmpl = template.New("prompt").Funcs(funcMap)

	// Parse all templates
	// Note: Output format is handled by llm client layer (via ResponseSchema or MarkdownOutputPrompt)
	template.Must(r.tmpl.New("main").Parse(mainTemplate))
	template.Must(r.tmpl.New("system_role").Parse(systemRoleTemplate))
	template.Must(r.tmpl.New("goals").Parse(goalsTemplate))
	template.Must(r.tmpl.New("constraints").Parse(constraintsTemplate))
	template.Must(r.tmpl.New("context").Parse(contextTemplate))
}

// Render renders a Spec into prompt text
func (r *Renderer) Render(spec *Spec) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpl.ExecuteTemplate(&buf, "main", spec); err != nil {
		return "", fmt.Errorf("failed to render prompt: %w", err)
	}
	promptText := buf.String()

	return promptText, nil
}

// RenderSystemPrompt renders only the system prompt portion
func (r *Renderer) RenderSystemPrompt(spec *Spec) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpl.ExecuteTemplate(&buf, "system_role", spec.SystemRole); err != nil {
		return "", fmt.Errorf("failed to render system prompt: %w", err)
	}
	return buf.String(), nil
}

// Helper functions for templates
func indent(spaces int, s string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

func bullet(items []string) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString("- ")
		sb.WriteString(item)
		sb.WriteString("\n")
	}
	return sb.String()
}

// numbered formats items as a numbered list (1. 2. 3. etc.)
func numbered(items []string) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, item := range items {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
	}
	return sb.String()
}

// quote formats text as a markdown blockquote
func quote(s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	var sb strings.Builder
	for i, line := range lines {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("> ")
		sb.WriteString(line)
	}
	return sb.String()
}

// Template definitions
// Note: Output format is handled by llm client layer (via ResponseSchema or MarkdownOutputPrompt)
const mainTemplate = `{{template "system_role" .SystemRole}}

{{template "goals" .Goals}}

{{template "constraints" .Constraints}}

{{template "context" .Context}}`

const systemRoleTemplate = `## Role

{{.Description}}
`

const goalsTemplate = `## Goals

Review the following areas in priority order:
{{range $i, $area := .Areas}}{{add $i 1}}. {{$area.ID}}: {{$area.Description}}
{{end}}
For each area, identify and check all relevant detection points based on industry best practices (e.g., OWASP Top 10, CWE, performance anti-patterns).`

const constraintsTemplate = `## Constraints
{{- if .ScopeControl}}

### Scope Control
{{bullet .ScopeControl}}
{{- end}}
{{- if .FocusOnIssuesOnly}}

### Focus
- Focus ONLY on reporting issues/problems found in the code
- Do NOT explain what the code changes do or what problem they fix
- Do NOT describe the intent or purpose of the changes
- Do NOT praise or commend the changes
{{- end}}
{{- if .Avoid}}

### Don't
{{bullet .Avoid}}
{{- end}}

### Severity
Levels: {{range $i, $level := .SeverityLevels}}{{if $i}}, {{end}}{{$level}}{{end}}
{{- if .MinSeverity}}
Minimum severity to report: {{.MinSeverity}}

- Severity must be assessed objectively based on actual impact.
- Do NOT inflate or escalate severity solely to satisfy this threshold.
- If no issues genuinely meet medium or higher severity, output no findings.
{{- end}}

### Output Style
- Output review as **plain text only**. Do NOT create any files.
{{- if .Tone}}
- Tone: {{.Tone}}
{{- end}}
{{- if .Concise}}
- Be concise.
{{- end}}
{{- if .NoEmoji}}
- Do NOT use emojis.
{{- end}}
{{- if .NoDate}}
- Do NOT include report date or timestamp.
{{- end}}
{{- if .Language}}
- Response language: {{.Language}}
{{- end}}`

const contextTemplate = `## Context
{{- if gt .PRNumber 0}}

This is a code review for a Pull Request / Merge Request.

### PR/MR Info
PR #{{.PRNumber}}{{if .PRTitle}}: {{.PRTitle}}{{end}}
{{- if .Ref}}
Branch: {{.Ref}}
{{- end}}
{{- if and .BaseCommitSHA .CommitSHA}}
Commit Range: {{.BaseCommitSHA}}..{{.CommitSHA}}{{if and .Commits (gt (len .Commits) 0)}} ({{len .Commits}} commits){{end}}
{{- else if .CommitSHA}}
Commit: {{.CommitSHA}}
{{- end}}
{{- if .PRDescription}}

Description:

{{quote .PRDescription}}
{{- end}}
{{- end}}

{{- if .ChangedFiles}}

### Changed Files
{{bullet .ChangedFiles}}
{{- end}}

{{- if .Languages}}

### Languages
Focus on files written in: {{join .Languages ", "}}
{{- end}}

{{- if .ReferenceDocs}}

### Reference Documentation
Review the code based on the following documentation files:
{{bullet .ReferenceDocs}}
{{- end}}

{{- if .PreviousReviewForComparison}}

### Previous Review Result (Historical Comparison)

The following is the complete result JSON from the previous review of this PR for the same rule.
Compare your current findings with the previous ones and set the "status" field for each finding:

- **"fixed"** - Issue was present in previous review but has been resolved in current code
- **"new"** - New issue found in current review that was not present before  
- **"persists"** - Issue still exists from previous review and has not been fixed

**Important**: The "status" field is required for each finding when history comparison is enabled.
For issues that persist, you may update the description if the context has changed.
If a previous issue cannot be verified (e.g., related code was removed), mark it as "fixed".

Previous review result (JSON):

{{quote .PreviousReviewForComparison}}
{{- end}}`

// QuickRender is a convenience function to render a spec with default settings
func QuickRender(spec *Spec) (string, error) {
	return NewRenderer().Render(spec)
}
