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
	"github.com/verustcode/verustcode/pkg/logger"
)

// SummaryGenerator generates the overall report summary (Phase 3)
// It collects section summaries and generates a comprehensive report summary
type SummaryGenerator struct {
	agent          base.Agent
	configProvider config.ConfigProvider
}

// NewSummaryGenerator creates a new summary generator
func NewSummaryGenerator(agent base.Agent, configProvider config.ConfigProvider) *SummaryGenerator {
	return &SummaryGenerator{
		agent:          agent,
		configProvider: configProvider,
	}
}

// getReportConfig retrieves report configuration from database (real-time read)
func (g *SummaryGenerator) getReportConfig() (*config.ReportConfig, error) {
	return g.configProvider.GetReportConfig()
}

// GenerateSummary generates the overall report summary from section summaries
// The summary is in Markdown format, used for report introduction and abstract image generation
func (g *SummaryGenerator) GenerateSummary(
	ctx context.Context,
	report *model.Report,
	sections []model.ReportSection,
	structure *ReportStructure,
	repoPath string,
) (string, error) {
	logger.Info("Generating report summary (Phase 3)",
		zap.String("report_id", report.ID),
		zap.Int("section_count", len(sections)),
	)

	// Get report type definition and DSL config
	typeDef, err := ScanReportType(report.ReportType)
	if err != nil {
		return "", fmt.Errorf("unknown report type: %w", err)
	}

	reportConfig := typeDef.Config
	if reportConfig == nil {
		return "", fmt.Errorf("report type '%s' has no DSL configuration", report.ReportType)
	}

	// Collect section summaries
	sectionSummaries := g.collectSectionSummaries(sections, structure)

	// Build prompt for summary generation
	prompt := g.buildSummaryPrompt(typeDef, reportConfig, structure, sectionSummaries)

	logger.Debug("Summary generation prompt",
		zap.String("report_id", report.ID),
		zap.String("prompt", prompt),
	)

	// Call agent to generate summary with retry logic
	req := &base.ReviewRequest{
		RepoPath:  repoPath,
		RequestID: fmt.Sprintf("summary-%s", report.ID),
		Model:     reportConfig.Agent.Model,
	}

	// Get retry configuration from database (real-time read)
	reportCfg, err := g.getReportConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get report config: %w", err)
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
			logger.Info("Retrying summary generation",
				zap.String("report_id", report.ID),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Duration("delay", retryDelay),
			)

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelay):
			}

			retryDelay = retryDelay * 2
			if retryDelay > MaxReportRetryDelay {
				retryDelay = MaxReportRetryDelay
			}
		}

		result, lastErr = g.agent.ExecuteWithPrompt(ctx, req, prompt)
		if lastErr == nil && result.Success {
			break
		}

		if lastErr != nil {
			logger.Warn("Summary generation agent call failed",
				zap.String("report_id", report.ID),
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
				zap.String("raw_response", result.Text),
				zap.Int("response_length", len(result.Text)),
			)
		} else if !result.Success {
			logger.Warn("Summary generation agent returned error",
				zap.String("report_id", report.ID),
				zap.Int("attempt", attempt),
				zap.String("error", result.Error),
				zap.String("raw_response", result.Text),
				zap.Int("response_length", len(result.Text)),
			)
			lastErr = fmt.Errorf("agent returned error: %s (raw response length: %d)", result.Error, len(result.Text))
		}

		if lastErr != nil && !llm.IsRetryable(lastErr) {
			logger.Warn("Non-retryable error, stopping retry",
				zap.String("report_id", report.ID),
				zap.Error(lastErr),
			)
			break
		}
	}

	if lastErr != nil {
		// Log final failure with raw response
		logger.Error("Summary generation failed after retries",
			zap.String("report_id", report.ID),
			zap.Int("max_retries", maxRetries),
			zap.Error(lastErr),
			zap.String("raw_response", result.Text),
			zap.Int("response_length", len(result.Text)),
		)
		return "", fmt.Errorf("agent call failed after %d retries (raw response length: %d): %w", maxRetries, len(result.Text), lastErr)
	}
	if !result.Success {
		// Log agent error with raw response
		logger.Error("Summary generation agent returned error",
			zap.String("report_id", report.ID),
			zap.String("error", result.Error),
			zap.String("raw_response", result.Text),
			zap.Int("response_length", len(result.Text)),
		)
		return "", fmt.Errorf("agent returned error: %s (raw response length: %d)", result.Error, len(result.Text))
	}

	// Post-process summary
	summary := g.postProcessSummary(result.Text)
	return summary, nil
}

// collectSectionSummaries collects summaries from all sections, organized by structure
func (g *SummaryGenerator) collectSectionSummaries(sections []model.ReportSection, structure *ReportStructure) []sectionSummaryItem {
	// Create a map for quick lookup
	summaryMap := make(map[string]string)
	for _, s := range sections {
		if s.Summary != "" {
			summaryMap[s.SectionID] = s.Summary
		}
	}

	var items []sectionSummaryItem

	// Iterate through structure to maintain order
	for i, sec := range structure.Sections {
		if len(sec.Subsections) > 0 {
			// Parent section with subsections
			for j, sub := range sec.Subsections {
				if summary, ok := summaryMap[sub.ID]; ok {
					items = append(items, sectionSummaryItem{
						Index:       fmt.Sprintf("%d.%d", i+1, j+1),
						Title:       sub.Title,
						ParentTitle: sec.Title,
						Summary:     summary,
					})
				}
			}
		} else {
			// Flat section
			if summary, ok := summaryMap[sec.ID]; ok {
				items = append(items, sectionSummaryItem{
					Index:   fmt.Sprintf("%d", i+1),
					Title:   sec.Title,
					Summary: summary,
				})
			}
		}
	}

	return items
}

// sectionSummaryItem represents a section summary for prompt building
type sectionSummaryItem struct {
	Index       string
	Title       string
	ParentTitle string
	Summary     string
}

// buildSummaryPrompt builds the prompt for report summary generation
func (g *SummaryGenerator) buildSummaryPrompt(
	typeDef *ReportTypeDefinition,
	reportConfig *dsl.ReportConfig,
	structure *ReportStructure,
	sectionSummaries []sectionSummaryItem,
) string {
	var sb strings.Builder

	// Phase description from DSL
	if reportConfig.Summary.Description != "" {
		sb.WriteString("## Role\n")
		sb.WriteString(reportConfig.Summary.Description)
		sb.WriteString("\n\n")
	}

	// Report context
	sb.WriteString("## Report Information\n")
	sb.WriteString(fmt.Sprintf("- **Report Type**: %s\n", typeDef.Name))
	sb.WriteString(fmt.Sprintf("- **Report Title**: %s\n", structure.Title))
	sb.WriteString(fmt.Sprintf("- **Report Description**: %s\n\n", structure.Summary))

	// Section summaries
	sb.WriteString("## Section Summaries\n\n")
	sb.WriteString("The following are brief summaries of each section in the report:\n\n")

	for _, item := range sectionSummaries {
		if item.ParentTitle != "" {
			sb.WriteString(fmt.Sprintf("### %s. %s > %s\n", item.Index, item.ParentTitle, item.Title))
		} else {
			sb.WriteString(fmt.Sprintf("### %s. %s\n", item.Index, item.Title))
		}
		sb.WriteString(item.Summary)
		sb.WriteString("\n\n")
	}

	// Goals from DSL
	hasTopics := len(reportConfig.Summary.Goals.Topics) > 0
	hasAvoid := len(reportConfig.Summary.Goals.Avoid) > 0
	if hasTopics || hasAvoid {
		sb.WriteString("## Goals\n\n")

		if hasTopics {
			sb.WriteString("### Focus Topics\n")
			for _, topic := range reportConfig.Summary.Goals.Topics {
				sb.WriteString(fmt.Sprintf("- %s\n", topic))
			}
			sb.WriteString("\n")
		}

		if hasAvoid {
			sb.WriteString("### Things to Avoid\n")
			for _, avoid := range reportConfig.Summary.Goals.Avoid {
				sb.WriteString(fmt.Sprintf("- %s\n", avoid))
			}
			sb.WriteString("\n")
		}
	}

	// Constraints from DSL
	if len(reportConfig.Summary.Constraints) > 0 {
		sb.WriteString("## Constraints\n\n")
		for _, constraint := range reportConfig.Summary.Constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", constraint))
		}
		sb.WriteString("\n")
	}

	// Output requirements
	sb.WriteString("## Output Requirements\n\n")
	sb.WriteString("Generate a comprehensive report summary in Markdown format. The summary should:\n\n")
	sb.WriteString("1. **Provide an executive overview** of the entire report\n")
	sb.WriteString("2. **Highlight key findings** from across all sections\n")
	sb.WriteString("3. **Be concise** but comprehensive - typically 200-500 words\n")
	sb.WriteString("4. **Use proper Markdown formatting**: headings, lists, emphasis\n")
	sb.WriteString("5. **Be suitable for display** at the beginning of the report or for generating abstract images\n\n")

	// Language instruction
	outputLanguage := reportConfig.Output.Style.Language
	if outputLanguage == "" {
		// Get from database config (real-time read)
		if reportCfg, err := g.configProvider.GetReportConfig(); err == nil && reportCfg != nil {
			outputLanguage = reportCfg.OutputLanguage
		}
	}
	if outputLanguage != "" {
		if langCfg, err := config.ParseLanguage(outputLanguage); err == nil {
			sb.WriteString(fmt.Sprintf("**Output Language**: All content must be in %s.\n\n", langCfg.PromptInstruction()))
		} else {
			sb.WriteString(fmt.Sprintf("**Output Language**: All content must be in %s.\n\n", outputLanguage))
		}
	}

	sb.WriteString("**IMPORTANT:**\n")
	sb.WriteString("- Do NOT write any files to disk. Return ONLY the summary content.\n")
	sb.WriteString("- Output as plain text Markdown. No file operations.\n")
	sb.WriteString("- Start directly with the summary content. No preamble.\n")

	return sb.String()
}

// postProcessSummary cleans up the generated summary
func (g *SummaryGenerator) postProcessSummary(summary string) string {
	summary = strings.TrimSpace(summary)

	// Remove markdown code fence wrapper if present
	if strings.HasPrefix(summary, "```") {
		wrapperPatterns := []string{"```markdown\n", "```md\n", "```\n"}
		for _, prefix := range wrapperPatterns {
			if strings.HasPrefix(summary, prefix) {
				summary = strings.TrimPrefix(summary, prefix)
				if idx := strings.LastIndex(summary, "```"); idx > 0 {
					remaining := strings.TrimSpace(summary[idx+3:])
					if remaining == "" {
						summary = summary[:idx]
					}
				}
				summary = strings.TrimSpace(summary)
				break
			}
		}
	}

	// Remove leading meta-commentary patterns
	// Common patterns like "Here is the summary:\n\n" or "Summary:\n\n"
	leadingPatterns := []string{
		"Here is the summary:",
		"Summary:",
		"Here's the summary:",
		"The summary is:",
		"Summary of the report:",
		"报告摘要：",
		"摘要：",
		"总结：",
	}
	for _, pattern := range leadingPatterns {
		if strings.HasPrefix(summary, pattern) {
			// Remove the pattern and any following newlines
			summary = strings.TrimPrefix(summary, pattern)
			summary = strings.TrimLeft(summary, " \n\r\t")
			break
		}
		// Also check for pattern followed by newlines
		patternWithNewline := pattern + "\n\n"
		if strings.HasPrefix(summary, patternWithNewline) {
			summary = strings.TrimPrefix(summary, patternWithNewline)
			break
		}
	}

	// Remove trailing patterns (file save messages, etc.)
	suffixPatterns := []string{
		// Chinese patterns - file operations
		"文档已保存",
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

	lines := strings.Split(summary, "\n")
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
	summary = strings.Join(lines, "\n")

	return strings.TrimSpace(summary)
}
