// Package output provides output channels for publishing review results.
// This file contains the common markdown conversion logic.
package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/prompt"
)

// MarkdownOptions controls markdown generation behavior
type MarkdownOptions struct {
	// IncludeHeader adds "## VerustCode Code Review" header
	IncludeHeader bool

	// IncludeReviewer adds reviewer info
	IncludeReviewer bool

	// IncludePRInfo adds PR/MR info
	IncludePRInfo bool

	// IncludeRawData adds raw JSON data section
	IncludeRawData bool

	// CollapsibleData uses <details> tag for raw data
	CollapsibleData bool

	// Marker is the comment marker prefix (for comment channel)
	Marker string

	// MetadataConfig controls output metadata (agent/model info footer)
	MetadataConfig *config.OutputMetadataConfig

	// PRInfo contains PR/MR information
	PRInfo *provider.PullRequest

	// AgentName is the agent name used
	AgentName string

	// ModelName is the model name used
	ModelName string
}

// FileMarkdownOptions returns options suitable for file output
func FileMarkdownOptions(prInfo *provider.PullRequest, meta *config.OutputMetadataConfig, agent, model string) *MarkdownOptions {
	return &MarkdownOptions{
		IncludeHeader:   true,
		IncludeReviewer: true,
		IncludePRInfo:   true,
		IncludeRawData:  true,
		CollapsibleData: false,
		PRInfo:          prInfo,
		MetadataConfig:  meta,
		AgentName:       agent,
		ModelName:       model,
	}
}

// CommentMarkdownOptions returns options for Git comment output
func CommentMarkdownOptions(marker string, meta *config.OutputMetadataConfig, agent, model string) *MarkdownOptions {
	return &MarkdownOptions{
		IncludeHeader:   true,
		IncludeReviewer: true,
		IncludePRInfo:   false,
		IncludeRawData:  true,
		CollapsibleData: true,
		Marker:          marker,
		MetadataConfig:  meta,
		AgentName:       agent,
		ModelName:       model,
	}
}

// ConvertToMarkdown converts structured JSON result to markdown
// This is the unified conversion function used by all channels
func ConvertToMarkdown(result *prompt.ReviewResult, opts *MarkdownOptions) string {
	if opts == nil {
		opts = &MarkdownOptions{}
	}

	var sb strings.Builder

	// Add marker at the beginning (for comment channel)
	if opts.Marker != "" {
		sb.WriteString(opts.Marker)
		sb.WriteString("\n\n")
	}

	// Add header
	if opts.IncludeHeader {
		sb.WriteString("# Code Review Report\n\n")
	}

	// Add reviewer information
	if opts.IncludeReviewer && result.ReviewerID != "" {
		sb.WriteString(fmt.Sprintf("**Reviewer**: %s\n\n", result.ReviewerID))
	}

	// Add PR/MR information
	if opts.IncludePRInfo && opts.PRInfo != nil && opts.PRInfo.URL != "" {
		prInfo := opts.PRInfo
		if prInfo.Title != "" {
			sb.WriteString(fmt.Sprintf("**PR/MR**: [#%d - %s](%s)\n\n", prInfo.Number, prInfo.Title, prInfo.URL))
		} else {
			sb.WriteString(fmt.Sprintf("**PR/MR**: [#%d](%s)\n\n", prInfo.Number, prInfo.URL))
		}

		// Add additional PR metadata if available
		if prInfo.Author != "" || prInfo.State != "" || (prInfo.HeadBranch != "" && prInfo.BaseBranch != "") {
			var metaParts []string
			if prInfo.Author != "" {
				metaParts = append(metaParts, fmt.Sprintf("**Author**: %s", prInfo.Author))
			}
			if prInfo.State != "" {
				metaParts = append(metaParts, fmt.Sprintf("**Status**: %s", prInfo.State))
			}
			if prInfo.HeadBranch != "" && prInfo.BaseBranch != "" {
				metaParts = append(metaParts, fmt.Sprintf("**Branch**: %s → %s", prInfo.HeadBranch, prInfo.BaseBranch))
			}
			if len(metaParts) > 0 {
				sb.WriteString(strings.Join(metaParts, " | "))
				sb.WriteString("\n\n")
			}
		}
	}

	// Check if we have any content
	hasContent := false

	// Try to get summary from Data (LLM always returns JSON)
	if len(result.Data) > 0 {
		if summary, ok := result.Data["summary"].(string); ok && summary != "" {
			sb.WriteString(summary)
			sb.WriteString("\n\n")
			hasContent = true
		}

		// Add raw data section if enabled
		if opts.IncludeRawData {
			// Check if there's more than just summary
			hasSummary := false
			if _, ok := result.Data["summary"]; ok {
				hasSummary = true
			}

			if len(result.Data) > 1 || !hasSummary {
				if opts.CollapsibleData {
					sb.WriteString("<details>\n<summary>Raw Data</summary>\n\n")
				} else {
					sb.WriteString("---\n\n")
					sb.WriteString("## Raw Data\n\n")
				}

				sb.WriteString("```json\n")
				jsonData, err := json.MarshalIndent(result.Data, "", "  ")
				if err == nil {
					sb.WriteString(string(jsonData))
				}
				sb.WriteString("\n```\n")

				if opts.CollapsibleData {
					sb.WriteString("\n</details>\n\n")
				} else {
					sb.WriteString("\n")
				}
				hasContent = true
			}
		}
	}

	// Handle no content case
	if !hasContent {
		sb.WriteString("**No review content available.**\n\n")
		if result.Error != "" {
			sb.WriteString(fmt.Sprintf("**Error**: %s\n\n", result.Error))
		} else {
			sb.WriteString("The review did not return any content. Please check the execution logs.\n")
		}
	}

	// Append metadata footer if configured
	if opts.MetadataConfig != nil {
		metadata := BuildMetadataString(opts.MetadataConfig, opts.AgentName, opts.ModelName)
		if metadata != "" {
			sb.WriteString("---\n\n")
			sb.WriteString(metadata)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// ConvertToJSON converts structured result to JSON string
func ConvertToJSON(result *prompt.ReviewResult) (string, error) {
	if len(result.Data) == 0 {
		return "{}", nil
	}
	jsonData, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}
	return string(jsonData), nil
}
