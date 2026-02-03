package output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Format defines the output file format
type Format string

const (
	// FormatMarkdown outputs as Markdown
	FormatMarkdown Format = "markdown"

	// FormatJSON outputs as JSON
	FormatJSON Format = "json"
)

// FileChannel outputs review results to files
type FileChannel struct {
	format    Format
	dir       string
	overwrite bool
}

// NewFileChannel creates a new FileChannel with a specific format
func NewFileChannel(format Format) *FileChannel {
	return &FileChannel{
		format:    format,
		overwrite: true,
	}
}

// NewUnifiedFileChannel creates a new FileChannel with default markdown format
func NewUnifiedFileChannel() *FileChannel {
	return &FileChannel{
		format:    FormatMarkdown,
		overwrite: true,
	}
}

// NewFileChannelWithConfig creates a new FileChannel with full configuration
func NewFileChannelWithConfig(format, dir string, overwrite bool) *FileChannel {
	f := FormatMarkdown
	if format == "json" {
		f = FormatJSON
	} else if format != "" {
		f = Format(format)
	}
	return &FileChannel{
		format:    f,
		dir:       dir,
		overwrite: overwrite,
	}
}

// Name returns the channel name
func (c *FileChannel) Name() string {
	if c.format != "" {
		return fmt.Sprintf("file_%s", c.format)
	}
	return "file"
}

// Publish writes the review result to a file
func (c *FileChannel) Publish(ctx context.Context, result *prompt.ReviewResult, opts *PublishOptions) error {
	// Use channel's format (already set during channel creation)
	format := c.format
	if format == "" {
		format = FormatMarkdown
	}

	// Extract rule ID from result
	ruleID := result.ReviewerID
	if ruleID == "" {
		ruleID = "unknown"
	}

	// Determine output path
	outputPath := c.determineOutputPath(opts, format, ruleID)

	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine overwrite setting (channel config takes priority)
	overwrite := c.overwrite || opts.Overwrite

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil && !overwrite {
		return fmt.Errorf("file already exists: %s (use overwrite option)", outputPath)
	}

	// Generate content
	var content []byte

	switch format {
	case FormatMarkdown:
		markdownOpts := FileMarkdownOptions(opts.PRInfo, opts.MetadataConfig, opts.AgentName, opts.ModelName)
		content = []byte(ConvertToMarkdown(result, markdownOpts))
		logger.Debug("File channel: generating markdown report",
			zap.String("reviewer_id", result.ReviewerID),
		)
	case FormatJSON:
		// Output the complete Data as JSON
		jsonStr, jsonErr := ConvertToJSON(result)
		if jsonErr != nil {
			return jsonErr
		}
		content = []byte(jsonStr)
		logger.Debug("File channel: outputting JSON format",
			zap.String("reviewer_id", result.ReviewerID),
		)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Write file
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info("Review result written to file",
		zap.String("path", outputPath),
		zap.String("format", string(format)),
		zap.String("reviewer_id", result.ReviewerID),
	)

	return nil
}

// extractWorkspaceName extracts workspace name from repository path
func extractWorkspaceName(repoPath string) string {
	if repoPath == "" {
		logger.Debug("Empty repo path, using 'unknown' as workspace name")
		return "unknown"
	}
	// Clean path and get the last directory name
	cleanPath := filepath.Clean(repoPath)
	baseName := filepath.Base(cleanPath)

	// Replace special characters with hyphens
	baseName = strings.ReplaceAll(baseName, " ", "-")
	baseName = strings.ReplaceAll(baseName, "/", "-")

	logger.Debug("Extracted workspace name from repo path",
		zap.String("repo_path", repoPath),
		zap.String("workspace_name", baseName),
	)

	return baseName
}

// determineOutputPath determines the output file path
func (c *FileChannel) determineOutputPath(opts *PublishOptions, format Format, ruleID string) string {
	// Generate filename based on review ID or timestamp
	ext := ".md"
	if format == FormatJSON {
		ext = ".json"
	}

	var filename string
	if opts.FileName != "" {
		// Custom filename has highest priority
		filename = opts.FileName
	} else {
		// Extract workspace name from RepoPath
		workspaceName := extractWorkspaceName(opts.RepoPath)

		// Determine filename format based on PR number
		if opts.PRNumber > 0 {
			// With PR: review-{workspace_name}-{pr_num}-{rule.id}.{ext}
			filename = fmt.Sprintf("review-%s-%d-%s%s", workspaceName, opts.PRNumber, ruleID, ext)
			logger.Debug("Generated filename with PR number",
				zap.String("workspace", workspaceName),
				zap.Int("pr_number", opts.PRNumber),
				zap.String("rule_id", ruleID),
				zap.String("filename", filename),
			)
		} else {
			// Without PR: review-{workspace_name}-{rule.id}.{ext}
			filename = fmt.Sprintf("review-%s-%s%s", workspaceName, ruleID, ext)
			logger.Debug("Generated filename without PR number",
				zap.String("workspace", workspaceName),
				zap.String("rule_id", ruleID),
				zap.String("filename", filename),
			)
		}
	}

	// Determine output directory
	var outputDir string
	if c.dir != "" {
		if filepath.IsAbs(c.dir) {
			outputDir = c.dir
		} else if opts.OutputDir != "" {
			outputDir = filepath.Join(opts.OutputDir, c.dir)
		} else {
			outputDir = c.dir
		}
	} else if opts.OutputDir != "" {
		outputDir = opts.OutputDir
	}

	// Combine directory and filename
	if outputDir != "" {
		return filepath.Join(outputDir, filename)
	}

	return filename
}
