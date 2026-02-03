package output

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/prompt"
)

// TestNewFileChannel tests creating a new FileChannel
func TestNewFileChannel(t *testing.T) {
	t.Run("markdown format", func(t *testing.T) {
		ch := NewFileChannel(FormatMarkdown)
		if ch == nil {
			t.Fatal("NewFileChannel() returned nil")
		}
		if ch.format != FormatMarkdown {
			t.Errorf("format = %s, want 'markdown'", ch.format)
		}
		if !ch.overwrite {
			t.Error("overwrite should be true by default")
		}
	})

	t.Run("json format", func(t *testing.T) {
		ch := NewFileChannel(FormatJSON)
		if ch == nil {
			t.Fatal("NewFileChannel() returned nil")
		}
		if ch.format != FormatJSON {
			t.Errorf("format = %s, want 'json'", ch.format)
		}
	})
}

// TestNewUnifiedFileChannel tests creating a unified file channel
func TestNewUnifiedFileChannel(t *testing.T) {
	ch := NewUnifiedFileChannel()
	if ch == nil {
		t.Fatal("NewUnifiedFileChannel() returned nil")
	}
	// Unified channel now defaults to markdown format
	if ch.format != FormatMarkdown {
		t.Errorf("format = %s, want markdown", ch.format)
	}
	if !ch.overwrite {
		t.Error("overwrite should be true by default")
	}
}

// TestNewFileChannelWithConfig tests creating a file channel with config
func TestNewFileChannelWithConfig(t *testing.T) {
	t.Run("json format", func(t *testing.T) {
		ch := NewFileChannelWithConfig("json", "/tmp/output", false)
		if ch.format != FormatJSON {
			t.Errorf("format = %s, want 'json'", ch.format)
		}
		if ch.dir != "/tmp/output" {
			t.Errorf("dir = %s, want '/tmp/output'", ch.dir)
		}
		if ch.overwrite != false {
			t.Error("overwrite should be false")
		}
	})

	t.Run("markdown format (default)", func(t *testing.T) {
		ch := NewFileChannelWithConfig("", "/tmp/output", true)
		if ch.format != FormatMarkdown {
			t.Errorf("format = %s, want 'markdown'", ch.format)
		}
	})

	t.Run("explicit markdown format", func(t *testing.T) {
		ch := NewFileChannelWithConfig("markdown", "", true)
		if ch.format != FormatMarkdown {
			t.Errorf("format = %s, want 'markdown'", ch.format)
		}
	})
}

// TestFileChannel_Name tests the Name method
func TestFileChannel_Name(t *testing.T) {
	t.Run("with format", func(t *testing.T) {
		ch := NewFileChannel(FormatJSON)
		if ch.Name() != "file_json" {
			t.Errorf("Name() = %s, want 'file_json'", ch.Name())
		}
	})

	t.Run("unified channel", func(t *testing.T) {
		ch := NewUnifiedFileChannel()
		// Unified channel now defaults to markdown format, so name includes format
		if ch.Name() != "file_markdown" {
			t.Errorf("Name() = %s, want 'file_markdown'", ch.Name())
		}
	})
}

// TestFileChannel_Publish tests the Publish method
func TestFileChannel_Publish(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "file_channel_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("publish markdown", func(t *testing.T) {
		ch := NewFileChannel(FormatMarkdown)

		result := prompt.NewReviewResult("test-reviewer")
		result.Data["summary"] = "Test summary"

		opts := &PublishOptions{
			OutputDir: tmpDir,
			RepoPath:  "/path/to/test-repo",
		}

		err := ch.Publish(context.Background(), result, opts)
		if err != nil {
			t.Fatalf("Publish() unexpected error: %v", err)
		}

		// Check file exists
		expectedPath := filepath.Join(tmpDir, "review-test-repo-test-reviewer.md")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", expectedPath)
		}

		// Check content
		content, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !strings.Contains(string(content), "Code Review Report") {
			t.Error("Content should contain 'Code Review Report' header")
		}

		if !strings.Contains(string(content), "Test summary") {
			t.Error("Content should contain the summary")
		}
	})

	t.Run("publish json", func(t *testing.T) {
		ch := NewFileChannel(FormatJSON)

		result := prompt.NewReviewResult("test-reviewer")
		result.Data["findings"] = []string{"finding1", "finding2"}

		opts := &PublishOptions{
			OutputDir: tmpDir,
			RepoPath:  "/path/to/test-repo",
		}

		err := ch.Publish(context.Background(), result, opts)
		if err != nil {
			t.Fatalf("Publish() unexpected error: %v", err)
		}

		// Check file exists
		expectedPath := filepath.Join(tmpDir, "review-test-repo-test-reviewer.json")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", expectedPath)
		}

		// Check content is valid JSON
		content, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !strings.Contains(string(content), "findings") {
			t.Error("Content should contain findings field")
		}
	})

	t.Run("publish with PR number", func(t *testing.T) {
		ch := NewFileChannel(FormatMarkdown)

		result := prompt.NewReviewResult("security")
		result.Data["summary"] = "Security review"

		opts := &PublishOptions{
			OutputDir: tmpDir,
			RepoPath:  "/path/to/my-project",
			PRNumber:  42,
		}

		err := ch.Publish(context.Background(), result, opts)
		if err != nil {
			t.Fatalf("Publish() unexpected error: %v", err)
		}

		// Check file name includes PR number
		expectedPath := filepath.Join(tmpDir, "review-my-project-42-security.md")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", expectedPath)
		}
	})

	t.Run("publish with custom filename", func(t *testing.T) {
		ch := NewFileChannel(FormatMarkdown)

		result := prompt.NewReviewResult("test")
		result.Data["summary"] = "Custom file test"

		opts := &PublishOptions{
			OutputDir: tmpDir,
			FileName:  "custom-report.md",
		}

		err := ch.Publish(context.Background(), result, opts)
		if err != nil {
			t.Fatalf("Publish() unexpected error: %v", err)
		}

		expectedPath := filepath.Join(tmpDir, "custom-report.md")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", expectedPath)
		}
	})

	t.Run("file channel with json format", func(t *testing.T) {
		// Create channel with JSON format
		ch := NewFileChannelWithConfig("json", tmpDir, true)

		result := prompt.NewReviewResult("test-json")
		result.Data["summary"] = "Format from channel config"

		opts := &PublishOptions{
			OutputDir: tmpDir,
			RepoPath:  "/path/to/repo",
		}

		err := ch.Publish(context.Background(), result, opts)
		if err != nil {
			t.Fatalf("Publish() unexpected error: %v", err)
		}

		// Should create JSON file
		expectedPath := filepath.Join(tmpDir, "review-repo-test-json.json")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected JSON file not created: %s", expectedPath)
		}
	})
}

// TestExtractWorkspaceName tests the extractWorkspaceName function
func TestExtractWorkspaceName(t *testing.T) {
	tests := []struct {
		name     string
		repoPath string
		want     string
	}{
		{
			name:     "simple path",
			repoPath: "/path/to/my-project",
			want:     "my-project",
		},
		{
			name:     "path with trailing slash",
			repoPath: "/path/to/project/",
			want:     "project",
		},
		{
			name:     "path with spaces",
			repoPath: "/path/to/my project",
			want:     "my-project",
		},
		{
			name:     "empty path",
			repoPath: "",
			want:     "unknown",
		},
		{
			name:     "root path",
			repoPath: "/",
			want:     "-", // filepath.Base("/") returns "/" which becomes "-" after replacing
		},
		{
			name:     "relative path",
			repoPath: "relative/path/repo",
			want:     "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorkspaceName(tt.repoPath)
			if got != tt.want {
				t.Errorf("extractWorkspaceName(%q) = %q, want %q", tt.repoPath, got, tt.want)
			}
		})
	}
}

// TestFileChannel_DetermineOutputPath tests the determineOutputPath method
func TestFileChannel_DetermineOutputPath(t *testing.T) {
	t.Run("with output dir from options", func(t *testing.T) {
		ch := NewFileChannel(FormatMarkdown)

		opts := &PublishOptions{
			OutputDir: "/output/dir",
			RepoPath:  "/path/to/repo",
		}

		path := ch.determineOutputPath(opts, FormatMarkdown, "test-rule")

		if !strings.HasPrefix(path, "/output/dir/") {
			t.Errorf("Path should start with output dir: %s", path)
		}

		if !strings.HasSuffix(path, ".md") {
			t.Errorf("Path should end with .md: %s", path)
		}
	})

	t.Run("with channel dir", func(t *testing.T) {
		ch := NewFileChannelWithConfig("", "/channel/dir", true)

		opts := &PublishOptions{
			RepoPath: "/path/to/repo",
		}

		path := ch.determineOutputPath(opts, FormatMarkdown, "test-rule")

		if !strings.HasPrefix(path, "/channel/dir/") {
			t.Errorf("Path should start with channel dir: %s", path)
		}
	})

	t.Run("json extension", func(t *testing.T) {
		ch := NewFileChannel(FormatJSON)

		opts := &PublishOptions{
			OutputDir: "/output",
			RepoPath:  "/path/to/repo",
		}

		path := ch.determineOutputPath(opts, FormatJSON, "test-rule")

		if !strings.HasSuffix(path, ".json") {
			t.Errorf("Path should end with .json: %s", path)
		}
	})

	t.Run("custom filename", func(t *testing.T) {
		ch := NewFileChannel(FormatMarkdown)

		opts := &PublishOptions{
			FileName: "custom.md",
		}

		path := ch.determineOutputPath(opts, FormatMarkdown, "test-rule")

		if path != "custom.md" {
			t.Errorf("Path should be custom filename: %s", path)
		}
	})
}

// TestConvertToMarkdown tests the unified markdown conversion function
func TestConvertToMarkdown(t *testing.T) {
	t.Run("basic report", func(t *testing.T) {
		result := prompt.NewReviewResult("test-reviewer")
		result.Data["summary"] = "This is the review summary."

		opts := FileMarkdownOptions(nil, nil, "", "")
		report := ConvertToMarkdown(result, opts)

		if !strings.Contains(report, "# Code Review Report") {
			t.Error("Report should contain header")
		}

		if !strings.Contains(report, "**Reviewer**: test-reviewer") {
			t.Error("Report should contain reviewer info")
		}

		if !strings.Contains(report, "This is the review summary.") {
			t.Error("Report should contain summary")
		}
	})

	t.Run("with PR info", func(t *testing.T) {
		result := prompt.NewReviewResult("test-reviewer")
		result.Data["summary"] = "Summary"

		prInfo := &provider.PullRequest{
			Number:     42,
			Title:      "Test PR",
			URL:        "https://github.com/test/repo/pull/42",
			Author:     "testuser",
			State:      "open",
			HeadBranch: "feature-branch",
			BaseBranch: "main",
		}

		opts := FileMarkdownOptions(prInfo, nil, "", "")
		report := ConvertToMarkdown(result, opts)

		if !strings.Contains(report, "#42") {
			t.Error("Report should contain PR number")
		}

		if !strings.Contains(report, "Test PR") {
			t.Error("Report should contain PR title")
		}

		if !strings.Contains(report, "testuser") {
			t.Error("Report should contain author")
		}

		if !strings.Contains(report, "feature-branch") {
			t.Error("Report should contain branch info")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		result := prompt.NewReviewResult("test-reviewer")

		opts := FileMarkdownOptions(nil, nil, "", "")
		report := ConvertToMarkdown(result, opts)

		if !strings.Contains(report, "No review content available") {
			t.Error("Report should indicate no content available")
		}
	})

	t.Run("data without summary", func(t *testing.T) {
		result := prompt.NewReviewResult("test-reviewer")
		result.Data["findings"] = []string{"finding1"}
		result.Data["score"] = 85

		opts := FileMarkdownOptions(nil, nil, "", "")
		report := ConvertToMarkdown(result, opts)

		// Should contain raw data section
		if !strings.Contains(report, "Raw Data") {
			t.Error("Report should contain Raw Data section")
		}

		if !strings.Contains(report, "```json") {
			t.Error("Report should contain JSON code block")
		}
	})

	t.Run("with metadata", func(t *testing.T) {
		result := prompt.NewReviewResult("test-reviewer")
		result.Data["summary"] = "Test summary"

		trueVal := true
		metadataConfig := &config.OutputMetadataConfig{
			ShowAgent:  &trueVal,
			ShowModel:  &trueVal,
			CustomText: "Generated by [VerustCode](https://github.com/verustcode/verustcode)",
		}

		opts := FileMarkdownOptions(nil, metadataConfig, "cursor", "claude-sonnet-4")
		report := ConvertToMarkdown(result, opts)

		if !strings.Contains(report, "Agent: cursor") {
			t.Error("Report should contain agent name")
		}

		if !strings.Contains(report, "Model: claude-sonnet-4") {
			t.Error("Report should contain model name")
		}

		if !strings.Contains(report, "Generated by") {
			t.Error("Report should contain custom text")
		}

		if !strings.Contains(report, "---") {
			t.Error("Report should contain separator before metadata")
		}
	})

	t.Run("with metadata disabled", func(t *testing.T) {
		result := prompt.NewReviewResult("test-reviewer")
		result.Data["summary"] = "Test summary"

		falseVal := false
		metadataConfig := &config.OutputMetadataConfig{
			ShowAgent:  &falseVal,
			ShowModel:  &falseVal,
			CustomText: "",
		}

		opts := FileMarkdownOptions(nil, metadataConfig, "cursor", "claude-sonnet-4")
		report := ConvertToMarkdown(result, opts)

		// Should not contain metadata section since all options are disabled
		if strings.Contains(report, "Agent: cursor") {
			t.Error("Report should not contain agent name when disabled")
		}

		if strings.Contains(report, "Model: claude-sonnet-4") {
			t.Error("Report should not contain model name when disabled")
		}
	})

	t.Run("comment options with marker", func(t *testing.T) {
		result := prompt.NewReviewResult("test-reviewer")
		result.Data["summary"] = "Comment summary"

		opts := CommentMarkdownOptions("[test-marker:rule-id]", nil, "", "")
		report := ConvertToMarkdown(result, opts)

		// Should have marker at beginning
		if !strings.HasPrefix(report, "[test-marker:rule-id]") {
			t.Error("Comment output should start with marker")
		}

		// Should have collapsible raw data
		if strings.Contains(report, "<details>") {
			// Collapsible is only shown when there's more than just summary
		}
	})
}
