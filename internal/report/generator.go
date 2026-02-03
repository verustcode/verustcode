// Package report provides report generation functionality.
package report

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/llm"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/report/exporter"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Default retry configuration values
const (
	DefaultReportMaxRetries = 3
	DefaultReportRetryDelay = 10 * time.Second
	MaxReportRetryDelay     = 5 * time.Minute
)

// SectionGenerator generates content for individual report sections (Phase 2)
type SectionGenerator struct {
	agent          base.Agent
	configProvider config.ConfigProvider
}

// NewSectionGenerator creates a new section content generator
func NewSectionGenerator(agent base.Agent, configProvider config.ConfigProvider) *SectionGenerator {
	return &SectionGenerator{
		agent:          agent,
		configProvider: configProvider,
	}
}

// getReportConfig retrieves report configuration from database (real-time read)
func (g *SectionGenerator) getReportConfig() (*config.ReportConfig, error) {
	return g.configProvider.GetReportConfig()
}

// SectionResult contains the generated content and summary for a section
type SectionResult struct {
	Content string // Full section content in Markdown
	Summary string // Short summary for Phase 3 aggregation
}

// GenerateSection generates content for a single section
// Returns both content and a short summary (for Phase 3 report summary generation)
func (g *SectionGenerator) GenerateSection(
	ctx context.Context,
	report *model.Report,
	section *model.ReportSection,
	structure *ReportStructure,
	repoPath string,
) (*SectionResult, error) {
	logger.Info("Generating section content",
		zap.String("report_id", report.ID),
		zap.String("section_id", section.SectionID),
		zap.String("section_title", section.Title),
	)

	// Get report type definition and DSL config (dynamically load from filesystem)
	typeDef, err := ScanReportType(report.ReportType)
	if err != nil {
		return nil, fmt.Errorf("unknown report type: %w", err)
	}

	config := typeDef.Config
	if config == nil {
		return nil, fmt.Errorf("report type '%s' has no DSL configuration", report.ReportType)
	}

	// Build prompt for section generation using DSL config
	// Note: We only pass file paths as hints, the agent will read files as needed
	prompt := g.buildSectionPrompt(typeDef, config, section, structure)

	// Log complete prompt for debugging (Phase 2: section generation)
	logger.Debug("Section generation prompt",
		zap.String("section_id", section.SectionID),
		zap.String("prompt", prompt),
	)

	// Call agent to generate content with retry logic
	req := &base.ReviewRequest{
		RepoPath:  repoPath,
		RequestID: fmt.Sprintf("section-%s-%s", report.ID, section.SectionID),
		Model:     config.Agent.Model, // Use model from DSL configuration
	}

	// Get retry configuration from database (real-time read)
	reportCfg, err := g.getReportConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get report config: %w", err)
	}
	maxRetries := reportCfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = DefaultReportMaxRetries
	}
	retryDelay := time.Duration(reportCfg.RetryDelay) * time.Second
	if retryDelay <= 0 {
		retryDelay = DefaultReportRetryDelay
	}

	var result *base.ReviewResult
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying section generation",
				zap.String("report_id", report.ID),
				zap.String("section_id", section.SectionID),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Duration("delay", retryDelay),
			)

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}

			// Exponential backoff (capped at MaxReportRetryDelay)
			retryDelay = retryDelay * 2
			if retryDelay > MaxReportRetryDelay {
				retryDelay = MaxReportRetryDelay
			}
		}

		result, lastErr = g.agent.ExecuteWithPrompt(ctx, req, prompt)
		if lastErr == nil && result.Success {
			break
		}

		// Log the failure with raw response
		if lastErr != nil {
			logger.Warn("Section generation agent call failed",
				zap.String("report_id", report.ID),
				zap.String("section_id", section.SectionID),
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
				zap.String("raw_response", result.Text),
				zap.Int("response_length", len(result.Text)),
			)
		} else if !result.Success {
			logger.Warn("Section generation agent returned error",
				zap.String("report_id", report.ID),
				zap.String("section_id", section.SectionID),
				zap.Int("attempt", attempt),
				zap.String("error", result.Error),
				zap.String("raw_response", result.Text),
				zap.Int("response_length", len(result.Text)),
			)
			lastErr = fmt.Errorf("agent returned error: %s (raw response length: %d)", result.Error, len(result.Text))
		}

		// Check if error is retryable; stop retry if not
		if lastErr != nil && !llm.IsRetryable(lastErr) {
			logger.Warn("Non-retryable error, stopping retry",
				zap.String("report_id", report.ID),
				zap.String("section_id", section.SectionID),
				zap.Error(lastErr),
			)
			break
		}
	}

	if lastErr != nil {
		// Log final failure with raw response
		logger.Error("Section generation failed after retries",
			zap.String("report_id", report.ID),
			zap.String("section_id", section.SectionID),
			zap.Int("max_retries", maxRetries),
			zap.Error(lastErr),
			zap.String("raw_response", result.Text),
			zap.Int("response_length", len(result.Text)),
		)
		return nil, fmt.Errorf("agent call failed after %d retries (raw response length: %d): %w", maxRetries, len(result.Text), lastErr)
	}
	if !result.Success {
		// Log agent error with raw response
		logger.Error("Section generation agent returned error",
			zap.String("report_id", report.ID),
			zap.String("section_id", section.SectionID),
			zap.String("error", result.Error),
			zap.String("raw_response", result.Text),
			zap.Int("response_length", len(result.Text)),
		)
		return nil, fmt.Errorf("agent returned error: %s (raw response length: %d)", result.Error, len(result.Text))
	}

	// Get response text
	response := result.Text

	// Post-process and parse content + summary
	sectionResult := g.postProcessContentWithSummary(response, config)

	logger.Info("Section content generated",
		zap.String("section_id", section.SectionID),
		zap.Int("content_length", len(sectionResult.Content)),
		zap.Int("summary_length", len(sectionResult.Summary)),
	)

	return sectionResult, nil
}

// buildSectionPrompt builds the prompt for section content generation using DSL config.
// Phase 2 prompt composition:
// 1. Role (section.description)
// 2. Report context (title, summary)
// 3. Current section information
// 4. Full report structure (for reference)
// 5. Goals (section.goals.areas and section.goals.avoid)
// 6. Output style (section.style)
// 7. Constraints (section.constraints)
// 8. Output format instructions (fixed Markdown)
//
// Note: We don't include file contents in the prompt. The agent will analyze the codebase
// and read files as needed based on the section requirements and file path hints.
func (g *SectionGenerator) buildSectionPrompt(
	typeDef *ReportTypeDefinition,
	reportConfig *dsl.ReportConfig,
	section *model.ReportSection,
	structure *ReportStructure,
) string {
	var sb strings.Builder

	// Phase description from DSL
	if reportConfig.Section.Description != "" {
		sb.WriteString("## Role\n")
		sb.WriteString(reportConfig.Section.Description)
		sb.WriteString("\n\n")
	}

	// Report context
	sb.WriteString("## Report Context\n")
	sb.WriteString(fmt.Sprintf("- **Report Type**: %s\n", typeDef.Name))
	sb.WriteString(fmt.Sprintf("- **Report Title**: %s\n", structure.Title))
	sb.WriteString(fmt.Sprintf("- **Report Summary**: %s\n\n", structure.Summary))

	// Section information
	sb.WriteString("## Section to Generate\n")
	sb.WriteString(fmt.Sprintf("- **Section ID**: %s\n", section.SectionID))
	sb.WriteString(fmt.Sprintf("- **Section Title**: %s\n", section.Title))
	sb.WriteString(fmt.Sprintf("- **Section Description**: %s\n", section.Description))
	// Show parent section info if this is a subsection (nested structure)
	if section.ParentSectionID != nil {
		sb.WriteString(fmt.Sprintf("- **Parent Section ID**: %s\n", *section.ParentSectionID))
		// Find parent section to get its title
		for _, s := range structure.Sections {
			if s.ID == *section.ParentSectionID {
				sb.WriteString(fmt.Sprintf("- **Parent Section Title**: %s\n", s.Title))
				break
			}
		}
	}
	sb.WriteString("\n")

	// Report structure (for context) - supports hierarchical display
	sb.WriteString("### Full Report Structure (for reference)\n")
	for i, s := range structure.Sections {
		marker := ""
		if s.ID == section.SectionID {
			marker = " <-- CURRENT SECTION"
		}
		sb.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, s.Title, marker))

		// Display subsections with indentation if hierarchical
		if len(s.Subsections) > 0 {
			for j, sub := range s.Subsections {
				subMarker := ""
				if sub.ID == section.SectionID {
					subMarker = " <-- CURRENT SECTION"
				}
				sb.WriteString(fmt.Sprintf("   %d.%d. %s%s\n", i+1, j+1, sub.Title, subMarker))
			}
		}
	}
	sb.WriteString("\n")

	// Goals section (combining topics and avoid)
	hasTopics := len(reportConfig.Section.Goals.Topics) > 0
	hasAvoid := len(reportConfig.Section.Goals.Avoid) > 0
	if hasTopics || hasAvoid {
		sb.WriteString("## Goals\n\n")

		// Focus topics from DSL section.goals.topics
		if hasTopics {
			sb.WriteString("### Focus Topics\n")
			sb.WriteString("When generating section content, focus on:\n\n")
			for _, topic := range reportConfig.Section.Goals.Topics {
				sb.WriteString(fmt.Sprintf("- %s\n", topic))
			}
			sb.WriteString("\n")
		}

		// Avoid list from DSL section.goals.avoid
		if hasAvoid {
			sb.WriteString("### Things to Avoid\n")
			sb.WriteString("Do NOT include:\n\n")
			for _, avoid := range reportConfig.Section.Goals.Avoid {
				sb.WriteString(fmt.Sprintf("- %s\n", avoid))
			}
			sb.WriteString("\n")
		}
	}

	// Output format section - unified from Output Style, Output Requirements, and Section Summary
	style := &reportConfig.Output.Style
	headingLevel := style.GetHeadingLevel()
	headingPrefix := strings.Repeat("#", headingLevel)
	maxSummaryLength := reportConfig.Section.Summary.GetMaxLength()

	sb.WriteString("## Output Format\n\n")

	// Format requirements
	sb.WriteString("### Format\n\n")
	sb.WriteString(fmt.Sprintf("- Start with a level-%d heading (%s) matching the section title\n", headingLevel, headingPrefix))
	sb.WriteString("- Use proper Markdown formatting: headings, lists, code blocks, tables\n")
	sb.WriteString("- Reference specific files and line numbers when discussing code\n")
	sb.WriteString("- Be accurate - only describe what actually exists in the codebase\n\n")

	// Style requirements
	sb.WriteString("### Style\n\n")
	if style.Tone != "" {
		sb.WriteString(fmt.Sprintf("- **Tone**: %s\n", style.Tone))
	}
	if style.GetUseMermaid() {
		sb.WriteString("- Use Mermaid diagrams where helpful (flowcharts, sequence diagrams, class diagrams)\n")
	}
	if style.GetConcise() {
		sb.WriteString("- Keep content concise and focused\n")
	}
	if style.GetNoEmoji() {
		sb.WriteString("- Do not use emoji in the content\n")
	}
	if style.GetIncludeLineNumbers() {
		sb.WriteString("- Include line numbers when referencing code\n")
	}
	if style.MaxSectionLength > 0 {
		sb.WriteString(fmt.Sprintf("- Keep section under %d characters\n", style.MaxSectionLength))
	}
	sb.WriteString("\n")

	// Language instruction - use PromptInstruction() to get human-readable language name
	// Priority: report yaml output.style.language > config.Report.OutputLanguage > auto-detect
	outputLanguage := style.Language
	if outputLanguage == "" {
		// Get from database config (real-time read)
		if reportCfg, err := g.configProvider.GetReportConfig(); err == nil && reportCfg != nil {
			outputLanguage = reportCfg.OutputLanguage
		}
	}
	if outputLanguage != "" {
		sb.WriteString("### Language\n\n")
		if langCfg, err := config.ParseLanguage(outputLanguage); err == nil {
			sb.WriteString(fmt.Sprintf("All content must be in %s.\n\n", langCfg.PromptInstruction()))
		} else {
			// Fallback to original value if parsing fails
			logger.Warn("Failed to parse output language, using raw value",
				zap.String("language", outputLanguage),
				zap.Error(err),
			)
			sb.WriteString(fmt.Sprintf("All content must be in %s.\n\n", outputLanguage))
		}
	}

	// Response structure (formerly Section Summary)
	sb.WriteString("### Response Structure\n\n")
	sb.WriteString("Format your response as follows:\n\n")
	sb.WriteString("```\n")
	sb.WriteString("[CONTENT]\n")
	sb.WriteString("(Your section content in Markdown here)\n\n")
	sb.WriteString("[SUMMARY]\n")
	sb.WriteString(fmt.Sprintf("(Short summary within %d characters, capturing key points)\n", maxSummaryLength))
	sb.WriteString("```\n\n")

	// Constraints (combining critical constraints and DSL constraints)
	sb.WriteString("### Constraints\n\n")
	sb.WriteString("- Do NOT write any files to disk. Return ONLY the document content.\n")
	sb.WriteString("- Do NOT create, save, or write any files (e.g., .md, .json, .txt).\n")
	sb.WriteString("- Do NOT mention file paths or say \"saved to\" or \"created file\".\n")
	sb.WriteString(fmt.Sprintf("- Start directly with the section heading (%s). No preamble or meta-commentary.\n", headingPrefix))
	// Add DSL constraints if any
	for _, constraint := range reportConfig.Section.Constraints {
		sb.WriteString(fmt.Sprintf("- %s\n", constraint))
	}
	sb.WriteString("\n")

	return sb.String()
}

// postProcessContentWithSummary parses the response to extract content and summary
func (g *SectionGenerator) postProcessContentWithSummary(response string, config *dsl.ReportConfig) *SectionResult {
	response = strings.TrimSpace(response)

	// Try to parse structured response with [CONTENT] and [SUMMARY] markers
	content, summary := g.parseStructuredResponse(response)

	// Post-process content
	content = g.postProcessContent(content)

	// Truncate summary if it exceeds max length
	// Use rune-based truncation to avoid breaking multi-byte UTF-8 characters
	maxLen := config.Section.Summary.GetMaxLength()
	summaryRunes := []rune(summary)
	if len(summaryRunes) > maxLen {
		// Reserve space for "..." (3 characters)
		truncateLen := maxLen - 3
		if truncateLen < 0 {
			truncateLen = 0
		}
		truncatedRunes := summaryRunes[:truncateLen]
		truncated := string(truncatedRunes)
		// Truncate at word boundary if possible
		if idx := strings.LastIndex(truncated, " "); idx > len(truncated)*2/3 {
			summary = truncated[:idx] + "..."
		} else {
			summary = truncated + "..."
		}
	}

	return &SectionResult{
		Content: content,
		Summary: strings.TrimSpace(summary),
	}
}

// parseStructuredResponse parses [CONTENT] and [SUMMARY] markers from response
func (g *SectionGenerator) parseStructuredResponse(response string) (content, summary string) {
	// Check for [CONTENT] and [SUMMARY] markers (case-insensitive)
	contentMarker := "[CONTENT]"
	summaryMarker := "[SUMMARY]"

	// Try to find markers (case-insensitive)
	upperResponse := strings.ToUpper(response)
	contentIdx := strings.Index(upperResponse, contentMarker)
	summaryIdx := strings.Index(upperResponse, summaryMarker)

	if contentIdx >= 0 && summaryIdx > contentIdx {
		// Found both markers in correct order
		// Extract content between [CONTENT] and [SUMMARY]
		contentStart := contentIdx + len(contentMarker)
		content = strings.TrimSpace(response[contentStart:summaryIdx])

		// Extract summary after [SUMMARY]
		summaryStart := summaryIdx + len(summaryMarker)
		summary = strings.TrimSpace(response[summaryStart:])
	} else if summaryIdx >= 0 {
		// Only [SUMMARY] found - content is everything before it
		content = strings.TrimSpace(response[:summaryIdx])
		summaryStart := summaryIdx + len(summaryMarker)
		summary = strings.TrimSpace(response[summaryStart:])
	} else {
		// No markers found - use entire response as content
		// Generate a simple summary from the content
		content = response
		summary = g.generateFallbackSummary(response)
	}

	return content, summary
}

// generateFallbackSummary generates a simple summary when the LLM doesn't provide one
// Uses rune-based truncation to avoid breaking multi-byte UTF-8 characters
func (g *SectionGenerator) generateFallbackSummary(content string) string {
	// Take the first meaningful paragraph (skip headings)
	lines := strings.Split(content, "\n")
	var summaryLines []string
	charCount := 0
	maxChars := 200

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Remove markdown formatting for summary
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "*", "")
		line = strings.ReplaceAll(line, "`", "")

		lineRunes := []rune(line)
		if charCount+len(lineRunes) > maxChars {
			if charCount == 0 {
				// First line is too long, truncate it
				if len(lineRunes) > maxChars-3 {
					line = string(lineRunes[:maxChars-3]) + "..."
				}
				summaryLines = append(summaryLines, line)
			}
			break
		}
		summaryLines = append(summaryLines, line)
		charCount += len(lineRunes)
	}

	return strings.Join(summaryLines, " ")
}

// postProcessContent cleans up the generated content
func (g *SectionGenerator) postProcessContent(content string) string {
	// Remove common LLM artifacts
	content = strings.TrimSpace(content)

	// Remove markdown code fence if the entire response is wrapped
	// Handle various patterns: ```markdown, ```md, ``` (plain)
	if strings.HasPrefix(content, "```") {
		// Check for common wrapper patterns
		wrapperPatterns := []string{"```markdown\n", "```md\n", "```\n"}
		for _, prefix := range wrapperPatterns {
			if strings.HasPrefix(content, prefix) {
				content = strings.TrimPrefix(content, prefix)
				// Remove the closing fence
				if idx := strings.LastIndex(content, "```"); idx > 0 {
					// Make sure this is a closing fence (at end of content)
					remaining := strings.TrimSpace(content[idx+3:])
					if remaining == "" {
						content = content[:idx]
					}
				}
				content = strings.TrimSpace(content)
				break
			}
		}
	}

	// Remove leading lines until we hit a markdown heading
	// Expected output should start with ## (section heading)
	for {
		content = strings.TrimSpace(content)
		if content == "" || strings.HasPrefix(content, "#") {
			break
		}
		// Remove the first line (meta-commentary)
		if idx := strings.Index(content, "\n"); idx > 0 {
			content = content[idx+1:]
		} else {
			break
		}
	}

	// Remove file save and meta-commentary suffix patterns (Chinese and English)
	// These often appear at the end of generated content
	suffixPatterns := []string{
		// Chinese patterns - file operations
		"文档已保存到",
		"已保存到",
		"保存在",
		"已生成",
		"已创建",
		"写入到",
		"已写入",
		"可直接使用",
		"可直接用于",
		"可以直接复制",
		"生成完成",
		"创建完成",
		"docs/",
		".md",
		// English patterns - file operations
		"Document saved to",
		"Saved to",
		"Written to",
		"Created file",
		"Generated file",
		"Ready to use",
		"You can use this",
		"Generation complete",
		"File created",
	}

	lines := strings.Split(content, "\n")
	// Remove trailing lines that match suffix patterns (check last few lines)
	maxLinesToCheck := 5
	for i := 0; i < maxLinesToCheck && len(lines) > 0; i++ {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if lastLine == "" {
			lines = lines[:len(lines)-1]
			continue
		}

		shouldRemove := false
		for _, p := range suffixPatterns {
			if strings.Contains(lastLine, p) {
				shouldRemove = true
				break
			}
		}

		if shouldRemove {
			lines = lines[:len(lines)-1]
		} else {
			break
		}
	}
	content = strings.Join(lines, "\n")

	return strings.TrimSpace(content)
}

// MergeSections merges all section contents into a complete report
// Supports hierarchical structure with parent sections and subsections
// MergeSections merges report sections into a single Markdown document (backward compatibility wrapper)
func MergeSections(report *model.Report, sections []model.ReportSection) string {
	return exporter.MergeSections(report, sections)
}
