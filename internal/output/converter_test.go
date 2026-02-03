package output

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/prompt"
)

func TestFileMarkdownOptions(t *testing.T) {
	prInfo := &provider.PullRequest{
		Number:     123,
		Title:      "Test PR",
		URL:        "https://github.com/test/repo/pull/123",
		Author:     "testuser",
		State:      "open",
		HeadBranch: "feature-branch",
		BaseBranch: "main",
	}
	meta := &config.OutputMetadataConfig{
		ShowAgent: boolPtr(true),
		ShowModel: boolPtr(true),
	}

	opts := FileMarkdownOptions(prInfo, meta, "cursor", "gpt-4")

	assert.True(t, opts.IncludeHeader)
	assert.True(t, opts.IncludeReviewer)
	assert.True(t, opts.IncludePRInfo)
	assert.True(t, opts.IncludeRawData)
	assert.False(t, opts.CollapsibleData)
	assert.Equal(t, prInfo, opts.PRInfo)
	assert.Equal(t, meta, opts.MetadataConfig)
	assert.Equal(t, "cursor", opts.AgentName)
	assert.Equal(t, "gpt-4", opts.ModelName)
}

func TestCommentMarkdownOptions(t *testing.T) {
	meta := &config.OutputMetadataConfig{
		ShowAgent: boolPtr(true),
		ShowModel: boolPtr(true),
	}

	opts := CommentMarkdownOptions("[review:test]", meta, "qoder", "auto")

	assert.True(t, opts.IncludeHeader)
	assert.True(t, opts.IncludeReviewer)
	assert.False(t, opts.IncludePRInfo)
	assert.True(t, opts.IncludeRawData)
	assert.True(t, opts.CollapsibleData)
	assert.Equal(t, "[review:test]", opts.Marker)
	assert.Equal(t, meta, opts.MetadataConfig)
	assert.Equal(t, "qoder", opts.AgentName)
	assert.Equal(t, "auto", opts.ModelName)
}

func TestConvertToMarkdown_WithSummary(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data: map[string]any{
			"summary": "This is a test summary",
		},
	}

	opts := &MarkdownOptions{
		IncludeHeader:   true,
		IncludeReviewer: true,
		IncludeRawData:  false,
	}

	markdown := ConvertToMarkdown(result, opts)

	assert.Contains(t, markdown, "# Code Review Report")
	assert.Contains(t, markdown, "**Reviewer**: test-reviewer")
	assert.Contains(t, markdown, "This is a test summary")
}

func TestConvertToMarkdown_WithPRInfo(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data: map[string]any{
			"summary": "Test summary",
		},
	}

	prInfo := &provider.PullRequest{
		Number:     456,
		Title:      "Feature PR",
		URL:        "https://github.com/test/repo/pull/456",
		Author:     "developer",
		State:      "open",
		HeadBranch: "feature",
		BaseBranch: "main",
	}

	opts := &MarkdownOptions{
		IncludeHeader:   true,
		IncludeReviewer: true,
		IncludePRInfo:   true,
		PRInfo:          prInfo,
		IncludeRawData:  false,
	}

	markdown := ConvertToMarkdown(result, opts)

	assert.Contains(t, markdown, "**PR/MR**: [#456 - Feature PR](https://github.com/test/repo/pull/456)")
	assert.Contains(t, markdown, "**Author**: developer")
	assert.Contains(t, markdown, "**Status**: open")
	assert.Contains(t, markdown, "**Branch**: feature → main")
}

func TestConvertToMarkdown_WithRawData(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data: map[string]any{
			"summary": "Test summary",
			"findings": []map[string]any{
				{"severity": "high", "message": "Issue 1"},
			},
		},
	}

	opts := &MarkdownOptions{
		IncludeHeader:   true,
		IncludeRawData:  true,
		CollapsibleData: false,
	}

	markdown := ConvertToMarkdown(result, opts)

	assert.Contains(t, markdown, "## Raw Data")
	assert.Contains(t, markdown, "```json")
	assert.Contains(t, markdown, "\"summary\"")
	assert.Contains(t, markdown, "\"findings\"")
}

func TestConvertToMarkdown_WithCollapsibleData(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data: map[string]any{
			"summary": "Test summary",
			"findings": []map[string]any{
				{"severity": "high", "message": "Issue 1"},
			},
		},
	}

	opts := &MarkdownOptions{
		IncludeHeader:   true,
		IncludeRawData:  true,
		CollapsibleData: true,
	}

	markdown := ConvertToMarkdown(result, opts)

	assert.Contains(t, markdown, "<details>")
	assert.Contains(t, markdown, "<summary>Raw Data</summary>")
	assert.Contains(t, markdown, "</details>")
}

func TestConvertToMarkdown_WithMarker(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data: map[string]any{
			"summary": "Test summary",
		},
	}

	opts := &MarkdownOptions{
		Marker:        "[review:test]",
		IncludeHeader: true,
	}

	markdown := ConvertToMarkdown(result, opts)

	assert.True(t, strings.HasPrefix(markdown, "[review:test]\n\n"))
}

func TestConvertToMarkdown_NoContent(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{},
	}

	opts := &MarkdownOptions{
		IncludeHeader: true,
	}

	markdown := ConvertToMarkdown(result, opts)

	assert.Contains(t, markdown, "**No review content available.**")
}

func TestConvertToMarkdown_NoContentWithError(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{},
		Error:      "Execution failed",
	}

	opts := &MarkdownOptions{
		IncludeHeader: true,
	}

	markdown := ConvertToMarkdown(result, opts)

	assert.Contains(t, markdown, "**No review content available.**")
	assert.Contains(t, markdown, "**Error**: Execution failed")
}

func TestConvertToMarkdown_WithMetadata(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data: map[string]any{
			"summary": "Test summary",
		},
	}

	meta := &config.OutputMetadataConfig{
		ShowAgent:  boolPtr(true),
		ShowModel:  boolPtr(true),
		CustomText: "Generated by VerustCode",
	}

	opts := &MarkdownOptions{
		IncludeHeader:  true,
		MetadataConfig: meta,
		AgentName:      "cursor",
		ModelName:      "gpt-4",
	}

	markdown := ConvertToMarkdown(result, opts)

	assert.Contains(t, markdown, "---")
	assert.Contains(t, markdown, "Generated by VerustCode")
	assert.Contains(t, markdown, "Agent: cursor")
	assert.Contains(t, markdown, "Model: gpt-4")
}

func TestConvertToMarkdown_NilOptions(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data: map[string]any{
			"summary": "Test summary",
		},
	}

	markdown := ConvertToMarkdown(result, nil)

	// Should still generate markdown without options
	assert.Contains(t, markdown, "Test summary")
}

func TestConvertToJSON(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data: map[string]any{
			"summary": "Test summary",
			"findings": []map[string]any{
				{"severity": "high", "message": "Issue 1"},
			},
		},
	}

	jsonStr, err := ConvertToJSON(result)
	require.NoError(t, err)

	assert.Contains(t, jsonStr, "\"summary\"")
	assert.Contains(t, jsonStr, "\"findings\"")
	assert.Contains(t, jsonStr, "Test summary")
}

func TestConvertToJSON_EmptyData(t *testing.T) {
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{},
	}

	jsonStr, err := ConvertToJSON(result)
	require.NoError(t, err)

	assert.Equal(t, "{}", jsonStr)
}
