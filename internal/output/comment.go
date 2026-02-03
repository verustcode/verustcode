package output

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/pkg/logger"
)

const (
	// DefaultCommentMarkerPrefix is the default marker prefix for identifying VerustCode comments
	// The full marker format is: [{marker_prefix}:{rule.id}]
	DefaultCommentMarkerPrefix = "review_by_scopeview"
)

// CommentChannel outputs review results as comments on Git platforms
type CommentChannel struct {
	// Overwrite determines whether to remove existing VerustCode comments before posting
	// true: remove existing comments (overwrite mode)
	// false: add new comment without removing existing ones (append mode)
	Overwrite bool

	// MarkerPrefix is the prefix used to identify VerustCode comments
	MarkerPrefix string

	// Format specifies the output format (markdown or json)
	// For comments, markdown is typically used for human readability
	Format string
}

// NewCommentChannel creates a new CommentChannel with default settings
func NewCommentChannel() *CommentChannel {
	return &CommentChannel{
		Overwrite:    false,
		MarkerPrefix: DefaultCommentMarkerPrefix,
		Format:       "markdown",
	}
}

// NewCommentChannelWithConfig creates a new CommentChannel with custom settings
// overwrite: true to remove existing comments before posting, false to append
// markerPrefix: custom marker prefix for identifying VerustCode comments
// format: output format (markdown or json)
func NewCommentChannelWithConfig(overwrite bool, markerPrefix string, format string) *CommentChannel {
	if markerPrefix == "" {
		markerPrefix = DefaultCommentMarkerPrefix
	}
	if format == "" {
		format = "markdown"
	}
	return &CommentChannel{
		Overwrite:    overwrite,
		MarkerPrefix: markerPrefix,
		Format:       format,
	}
}

// Name returns the channel name
func (c *CommentChannel) Name() string {
	return "comment"
}

// Publish posts the review result as a comment on the PR/MR
func (c *CommentChannel) Publish(ctx context.Context, result *prompt.ReviewResult, opts *PublishOptions) error {
	if opts.PRNumber == 0 {
		logger.Warn("Comment channel: no PR number provided, skipping comment")
		return nil
	}

	if opts.Provider == nil {
		return fmt.Errorf("comment channel: provider not configured")
	}

	// Parse owner and repo from URL
	owner, repo, err := parseRepoURL(opts.RepoURL)
	if err != nil {
		logger.Error("Failed to parse repo URL",
			zap.String("url", opts.RepoURL),
			zap.Error(err),
		)
		return fmt.Errorf("failed to parse repo URL: %w", err)
	}

	logger.Debug("Parsed repo URL",
		zap.String("url", opts.RepoURL),
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("pr", opts.PRNumber),
	)

	// Determine overwrite mode and marker prefix from options or channel defaults
	overwrite := c.Overwrite
	markerPrefix := c.MarkerPrefix
	if opts.CommentMode != "" {
		// Support legacy mode string for backward compatibility
		overwrite = opts.CommentMode == "overwrite"
	}
	if opts.CommentMarker != "" {
		marker := opts.CommentMarker
		if strings.HasPrefix(marker, "[") && strings.HasSuffix(marker, "]") {
			marker = marker[1 : len(marker)-1]
			if idx := strings.Index(marker, ":"); idx > 0 {
				markerPrefix = marker[:idx]
			} else {
				markerPrefix = marker
			}
		}
	}

	// Build full marker: [{marker_prefix}:{rule.id}]
	ruleID := result.ReviewerID
	if ruleID == "" {
		ruleID = "unknown"
	}
	fullMarker := fmt.Sprintf("[%s:%s]", markerPrefix, ruleID)

	// Generate comment body with marker
	body := c.generateCommentBodyWithMarker(result, fullMarker, opts.MetadataConfig, opts.AgentName, opts.ModelName)

	// If overwrite mode, try to update existing comment instead of delete+create
	if overwrite {
		updated, err := c.updateOrCreateComment(ctx, opts.Provider, owner, repo, opts.PRNumber, fullMarker, body)
		if err != nil {
			return fmt.Errorf("failed to update or create comment: %w", err)
		}
		if updated {
			logger.Info("Updated existing review comment",
				zap.String("provider", opts.Provider.Name()),
				zap.String("repo", fmt.Sprintf("%s/%s", owner, repo)),
				zap.Int("pr", opts.PRNumber),
				zap.String("reviewer_id", result.ReviewerID),
			)
			return nil
		}
		// If no existing comment found, fall through to create new one
	}

	// Post new comment
	commentOpts := &provider.CommentOptions{
		PRNumber: opts.PRNumber,
	}
	if err := opts.Provider.PostComment(ctx, owner, repo, commentOpts, body); err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	logger.Info("Posted review comment",
		zap.String("provider", opts.Provider.Name()),
		zap.String("repo", fmt.Sprintf("%s/%s", owner, repo)),
		zap.Int("pr", opts.PRNumber),
		zap.String("reviewer_id", result.ReviewerID),
		zap.Bool("overwrite", overwrite),
	)

	return nil
}

// updateOrCreateComment finds existing comment with marker and updates it, or returns false if not found
// If multiple matching comments exist, updates the first one and deletes the rest
func (c *CommentChannel) updateOrCreateComment(ctx context.Context, prov provider.Provider, owner, repo string, prNumber int, marker, body string) (bool, error) {
	logger.Debug("Looking for existing VerustCode comment to update",
		zap.String("provider", prov.Name()),
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("pr", prNumber),
		zap.String("marker", marker),
	)

	comments, err := prov.ListComments(ctx, owner, repo, prNumber)
	if err != nil {
		logger.Error("Failed to list PR comments for update",
			zap.String("provider", prov.Name()),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("pr", prNumber),
			zap.Error(err),
		)
		return false, err
	}

	// Find all comments matching the marker
	var matchingComments []*provider.Comment
	for _, comment := range comments {
		if strings.Contains(comment.Body, marker) {
			matchingComments = append(matchingComments, comment)
		}
	}

	if len(matchingComments) == 0 {
		logger.Debug("No existing comment found with marker, will create new one",
			zap.String("marker", marker),
		)
		return false, nil
	}

	// Update the first matching comment
	firstComment := matchingComments[0]
	if err := prov.UpdateComment(ctx, owner, repo, firstComment.ID, prNumber, body); err != nil {
		logger.Error("Failed to update existing comment",
			zap.Int64("comment_id", firstComment.ID),
			zap.Error(err),
		)
		return false, err
	}

	logger.Debug("Updated existing comment",
		zap.Int64("comment_id", firstComment.ID),
	)

	// Delete any extra matching comments (cleanup duplicates)
	if len(matchingComments) > 1 {
		deletedCount := 0
		for _, comment := range matchingComments[1:] {
			if err := prov.DeleteComment(ctx, owner, repo, comment.ID); err != nil {
				logger.Warn("Failed to delete duplicate comment",
					zap.Int64("comment_id", comment.ID),
					zap.Error(err),
				)
				continue
			}
			deletedCount++
		}
		if deletedCount > 0 {
			logger.Info("Cleaned up duplicate VerustCode comments",
				zap.String("repo", fmt.Sprintf("%s/%s", owner, repo)),
				zap.Int("pr", prNumber),
				zap.Int("count", deletedCount),
			)
		}
	}

	return true, nil
}

// generateCommentBodyWithMarker generates the comment body with the specified marker
// Uses channel's Format setting to determine output format
func (c *CommentChannel) generateCommentBodyWithMarker(result *prompt.ReviewResult, marker string, metadataConfig *config.OutputMetadataConfig, agentName, modelName string) string {
	// JSON format: output raw JSON data with marker
	if c.Format == "json" {
		var sb strings.Builder
		sb.WriteString(marker)
		sb.WriteString("\n\n")
		if len(result.Data) > 0 {
			sb.WriteString("```json\n")
			jsonStr, _ := ConvertToJSON(result)
			sb.WriteString(jsonStr)
			sb.WriteString("\n```\n")
		} else {
			sb.WriteString("**No review data available.**\n")
		}
		return sb.String()
	}

	// Markdown format: use common converter
	markdownOpts := CommentMarkdownOptions(marker, metadataConfig, agentName, modelName)
	return ConvertToMarkdown(result, markdownOpts)
}

// parseRepoURL parses owner and repo from a repository URL
func parseRepoURL(url string) (owner, repo string, err error) {
	url = strings.TrimSuffix(url, ".git")

	// HTTPS format
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		parts := strings.Split(url, "/")
		if len(parts) < 5 {
			return "", "", fmt.Errorf("invalid HTTPS URL format: %s", url)
		}

		domainIdx := 2
		ownerIdx := domainIdx + 1
		repoIdx := domainIdx + 2

		if len(parts) <= repoIdx {
			return "", "", fmt.Errorf("missing owner/repo in URL: %s", url)
		}

		owner = parts[ownerIdx]
		repo = parts[repoIdx]

		if owner == "" || repo == "" {
			return "", "", fmt.Errorf("empty owner or repo in URL: %s", url)
		}

		return owner, repo, nil
	}

	// SSH format
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			repoParts := strings.Split(parts[1], "/")
			if len(repoParts) >= 2 {
				return repoParts[len(repoParts)-2], repoParts[len(repoParts)-1], nil
			}
		}
		return "", "", fmt.Errorf("invalid SSH URL format: %s", url)
	}

	// Simple format: owner/repo
	parts := strings.Split(url, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("unable to parse repo URL: %s", url)
}
