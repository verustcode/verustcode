// Package handler provides HTTP handlers for the API.
package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	pkgerrors "github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
)

// WebhookHandler handles webhook-related HTTP requests
type WebhookHandler struct {
	engine *engine.Engine
	store  store.Store
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(e *engine.Engine, s store.Store) *WebhookHandler {
	return &WebhookHandler{engine: e, store: s}
}

// HandleWebhook handles POST /api/v1/webhooks/:provider
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	providerName := c.Param("provider")

	// Get provider
	prov, ok := h.engine.GetProvider(providerName)
	if !ok {
		logger.Warn("Unknown webhook provider", zap.String("provider", providerName))
		c.JSON(http.StatusNotFound, gin.H{
			"code":    pkgerrors.ErrCodeGitNotFound,
			"message": "Unknown provider: " + providerName,
		})
		return
	}

	// Get webhook secret from provider configuration only
	// Note: URL query parameter support removed for security (secrets in URLs get logged)
	var secret string
	if cfg, ok := h.engine.GetProviderConfig(providerName); ok && cfg.WebhookSecret != "" {
		secret = cfg.WebhookSecret
	}

	// P0-2 Security improvement: Warn if webhook secret is not configured
	if secret == "" {
		logger.Warn("Webhook secret not configured, signature validation skipped",
			zap.String("provider", providerName),
			zap.String("hint", "Configure webhook_secret in git.providers for security"),
		)
	}

	// Parse webhook
	event, err := prov.ParseWebhook(c.Request, secret)
	if err != nil {
		logger.Warn("Failed to parse webhook",
			zap.String("provider", providerName),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    pkgerrors.ErrCodeGitWebhook,
			"message": "Failed to parse webhook: " + err.Error(),
		})
		return
	}

	logger.Info("Webhook received",
		zap.String("provider", providerName),
		zap.String("type", string(event.Type)),
		zap.String("repo", event.Owner+"/"+event.Repo),
		zap.String("ref", event.Ref),
		zap.String("action", event.Action),
		zap.String("sender", event.Sender),
		zap.Int("pr_number", event.PRNumber),
	)

	// Handle event based on type
	switch event.Type {
	case provider.EventTypePush:
		h.handlePushEvent(c, event)
	case provider.EventTypePullRequest, provider.EventTypeMergeRequest:
		h.handlePREvent(c, event)
	default:
		// Acknowledge but don't process
		c.JSON(http.StatusOK, gin.H{
			"message": "Event received but not processed",
			"type":    event.Type,
		})
	}
}

// handlePushEvent handles push webhook events
func (h *WebhookHandler) handlePushEvent(c *gin.Context, event *provider.WebhookEvent) {
	// Build repository URL from provider configuration or default
	repoURL := h.buildRepoURL(event)

	// Create review with unique ID
	review := &model.Review{
		ID:          idgen.NewReviewID(),
		RepoURL:     repoURL,
		Ref:         event.Ref,
		CommitSHA:   event.CommitSHA,
		Status:      model.ReviewStatusPending,
		Source:      "webhook",
		TriggeredBy: event.Sender,
	}

	if err := h.store.Review().Create(review); err != nil {
		logger.Error("Failed to create review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    pkgerrors.ErrCodeDBQuery,
			"message": "Failed to create review",
		})
		return
	}

	// Auto-create repository config if not exists
	// This ensures the repo appears in the Repositories list
	if _, err := h.store.RepositoryConfig().EnsureConfig(repoURL); err != nil {
		// Log warning but don't fail the webhook - this is a non-critical operation
		logger.Warn("Failed to ensure repository config for push event",
			zap.String("repo_url", repoURL),
			zap.Error(err),
		)
	}

	// Submit to engine (push event, no PR info)
	_, err := h.engine.Submit(review, nil)
	if err != nil {
		if dbErr := h.store.Review().UpdateStatusWithError(review.ID, model.ReviewStatusFailed, err.Error()); dbErr != nil {
			logger.Error("Failed to update review status after engine submission failure", zap.Error(dbErr))
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    pkgerrors.ErrCodeReviewFailed,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "Review triggered",
		"review_id": review.ID,
	})
}

// handlePREvent handles pull request / merge request webhook events
func (h *WebhookHandler) handlePREvent(c *gin.Context, event *provider.WebhookEvent) {
	// Build PR URL for lookup
	prURL := h.buildPRURL(event)

	// Handle merged/closed events - update MergedAt for statistics
	if provider.IsPRMergedEvent(event.Action) {
		h.handlePRMergedEvent(c, event, prURL)
		return
	}

	// Check if this action should trigger a code review
	if !provider.ShouldProcessPREvent(event.Action) {
		logger.Info("PR/MR action skipped, not triggering review",
			zap.String("provider", event.Provider),
			zap.String("action", event.Action),
			zap.String("repo", event.Owner+"/"+event.Repo),
			zap.Int("pr_number", event.PRNumber),
		)
		c.JSON(http.StatusOK, gin.H{
			"message": "PR action not supported for code review, skipping",
			"action":  event.Action,
		})
		return
	}

	// Build repository URL from provider configuration or default
	repoURL := h.buildRepoURL(event)

	// Check if review already exists for this PR URL + Commit SHA combination
	// This prevents duplicate reviews for the same commit
	if prURL != "" && event.CommitSHA != "" {
		existingReview, err := h.store.Review().GetByPRURLAndCommit(prURL, event.CommitSHA)
		if err == nil && existingReview != nil {
			// Review already exists for this PR + commit combination
			logger.Info("Review already exists for PR + commit combination, skipping creation",
				zap.String("review_id", existingReview.ID),
				zap.String("provider", event.Provider),
				zap.String("repo", event.Owner+"/"+event.Repo),
				zap.Int("pr_number", event.PRNumber),
				zap.String("commit_sha", event.CommitSHA),
				zap.String("pr_url", prURL),
			)
			c.JSON(http.StatusOK, gin.H{
				"message":   "Review already exists for this PR + commit",
				"review_id": existingReview.ID,
				"pr_number": event.PRNumber,
				"action":    event.Action,
			})
			return
		}
		// If error is not "record not found", log it but continue to create new review
		if err != nil && err != gorm.ErrRecordNotFound {
			logger.Warn("Failed to check for existing review, continuing with creation",
				zap.String("provider", event.Provider),
				zap.String("repo", event.Owner+"/"+event.Repo),
				zap.Int("pr_number", event.PRNumber),
				zap.Error(err),
			)
		}
	}

	// Calculate revision count for this MR
	// For opened events, revision = 1
	// For synchronize events, revision = max existing revision + 1
	revisionCount := 1
	if provider.IsPRUpdateEvent(event.Action) && prURL != "" {
		maxRevision, err := h.store.Review().GetMaxRevisionByPRURL(prURL)
		if err != nil {
			logger.Warn("Failed to get max revision count, using default",
				zap.String("pr_url", prURL),
				zap.Error(err),
			)
		} else {
			revisionCount = maxRevision + 1
		}
		logger.Debug("Calculated revision count for MR update",
			zap.String("pr_url", prURL),
			zap.Int("revision_count", revisionCount),
		)
	}

	// Create review with unique ID
	review := &model.Review{
		ID:            idgen.NewReviewID(),
		RepoURL:       repoURL,
		Ref:           event.Ref,
		CommitSHA:     event.CommitSHA,
		PRNumber:      event.PRNumber,
		PRURL:         prURL,
		Status:        model.ReviewStatusPending,
		Source:        "webhook",
		TriggeredBy:   event.Sender,
		RevisionCount: revisionCount,
	}

	if err := h.store.Review().Create(review); err != nil {
		// Check if error is due to unique constraint violation
		// SQLite returns "UNIQUE constraint failed" error
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// Try to find the existing record
			existing, findErr := h.store.Review().GetByPRURLAndCommit(prURL, event.CommitSHA)
			if findErr == nil && existing != nil {
				logger.Info("Review already exists (detected via constraint violation), returning existing review",
					zap.String("review_id", existing.ID),
					zap.String("provider", event.Provider),
					zap.String("repo", event.Owner+"/"+event.Repo),
					zap.Int("pr_number", event.PRNumber),
					zap.String("commit_sha", event.CommitSHA),
				)
				c.JSON(http.StatusOK, gin.H{
					"message":   "Review already exists for this PR + commit",
					"review_id": existing.ID,
					"pr_number": event.PRNumber,
					"action":    event.Action,
				})
				return
			}
		}

		logger.Error("Failed to create review record",
			zap.String("provider", event.Provider),
			zap.String("repo", event.Owner+"/"+event.Repo),
			zap.Int("pr_number", event.PRNumber),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    pkgerrors.ErrCodeDBQuery,
			"message": "Failed to create review",
		})
		return
	}

	// Auto-create repository config if not exists
	// This ensures the repo appears in the Repositories list
	if _, err := h.store.RepositoryConfig().EnsureConfig(repoURL); err != nil {
		// Log warning but don't fail the webhook - this is a non-critical operation
		logger.Warn("Failed to ensure repository config for PR event",
			zap.String("repo_url", repoURL),
			zap.Error(err),
		)
	}

	// Prepare PR information from webhook event
	prInfo := &engine.PRInfo{
		Title:        event.PRTitle,
		Description:  event.PRDescription,
		BaseSHA:      event.BaseCommitSHA,
		ChangedFiles: event.ChangedFiles,
	}

	// Submit to engine with PR information
	_, err := h.engine.Submit(review, prInfo)
	if err != nil {
		logger.Error("Failed to submit review to engine",
			zap.String("review_id", review.ID),
			zap.String("provider", event.Provider),
			zap.String("repo", event.Owner+"/"+event.Repo),
			zap.Int("pr_number", event.PRNumber),
			zap.Error(err),
		)
		if dbErr := h.store.Review().UpdateStatusWithError(review.ID, model.ReviewStatusFailed, err.Error()); dbErr != nil {
			logger.Error("Failed to update review status after engine submission failure", zap.Error(dbErr))
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    pkgerrors.ErrCodeReviewFailed,
			"message": err.Error(),
		})
		return
	}

	logger.Info("Review successfully triggered",
		zap.String("review_id", review.ID),
		zap.String("provider", event.Provider),
		zap.String("repo", event.Owner+"/"+event.Repo),
		zap.Int("pr_number", event.PRNumber),
		zap.String("action", event.Action),
	)

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "Review triggered",
		"review_id": review.ID,
		"pr_number": event.PRNumber,
		"action":    event.Action,
	})
}

// buildRepoURL builds repository URL from webhook event
// Uses provider configuration URL if available, otherwise uses default public URL
func (h *WebhookHandler) buildRepoURL(event *provider.WebhookEvent) string {
	// Try to get base URL from provider configuration
	baseURL := ""
	if cfg, ok := h.engine.GetProviderConfig(event.Provider); ok && cfg.URL != "" {
		baseURL = cfg.URL
	}

	// Build URL based on provider type
	switch event.Provider {
	case "github":
		if baseURL != "" && baseURL != "https://github.com" {
			// GitHub Enterprise - use configured URL
			baseURL = strings.TrimSuffix(baseURL, "/")
			return fmt.Sprintf("%s/%s/%s", baseURL, event.Owner, event.Repo)
		}
		// Default GitHub.com
		return fmt.Sprintf("https://github.com/%s/%s", event.Owner, event.Repo)

	case "gitlab":
		if baseURL != "" && baseURL != "https://gitlab.com" {
			// Self-hosted GitLab - use configured URL
			baseURL = strings.TrimSuffix(baseURL, "/")
			// Remove /api/v4 suffix if present
			baseURL = strings.TrimSuffix(baseURL, "/api/v4")
			return fmt.Sprintf("%s/%s/%s", baseURL, event.Owner, event.Repo)
		}
		// Default GitLab.com
		return fmt.Sprintf("https://gitlab.com/%s/%s", event.Owner, event.Repo)

	default:
		// Fallback: use provider name as domain
		return fmt.Sprintf("https://%s.com/%s/%s", event.Provider, event.Owner, event.Repo)
	}
}

// buildPRURL builds PR/MR URL from webhook event
// Uses provider configuration URL if available, otherwise uses default public URL
func (h *WebhookHandler) buildPRURL(event *provider.WebhookEvent) string {
	if event.PRNumber <= 0 {
		return ""
	}

	// Get base repo URL first
	repoURL := h.buildRepoURL(event)

	// Build PR/MR URL based on provider type
	switch event.Provider {
	case "github":
		return fmt.Sprintf("%s/pull/%d", repoURL, event.PRNumber)
	case "gitlab":
		return fmt.Sprintf("%s/-/merge_requests/%d", repoURL, event.PRNumber)
	default:
		// Fallback: try common patterns
		if strings.Contains(repoURL, "github") {
			return fmt.Sprintf("%s/pull/%d", repoURL, event.PRNumber)
		}
		if strings.Contains(repoURL, "gitlab") {
			return fmt.Sprintf("%s/-/merge_requests/%d", repoURL, event.PRNumber)
		}
		return ""
	}
}

// handlePRMergedEvent handles merged/closed PR events
// Updates MergedAt timestamp for all reviews with the same PR URL
func (h *WebhookHandler) handlePRMergedEvent(c *gin.Context, event *provider.WebhookEvent, prURL string) {
	if prURL == "" {
		logger.Warn("Cannot update MergedAt: PR URL is empty",
			zap.String("provider", event.Provider),
			zap.Int("pr_number", event.PRNumber),
		)
		c.JSON(http.StatusOK, gin.H{
			"message": "Merged event received but PR URL is empty",
			"action":  event.Action,
		})
		return
	}

	now := time.Now()

	// Update MergedAt for all reviews with the same PR URL
	rowsAffected, err := h.store.Review().UpdateMergedAtByPRURL(prURL, now)

	if err != nil {
		logger.Error("Failed to update MergedAt for PR",
			zap.String("provider", event.Provider),
			zap.String("pr_url", prURL),
			zap.Int("pr_number", event.PRNumber),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    pkgerrors.ErrCodeDBQuery,
			"message": "Failed to update merged time",
		})
		return
	}

	logger.Info("Updated MergedAt for PR reviews",
		zap.String("provider", event.Provider),
		zap.String("pr_url", prURL),
		zap.Int("pr_number", event.PRNumber),
		zap.String("action", event.Action),
		zap.Int64("affected_rows", rowsAffected),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Merged event processed",
		"action":        event.Action,
		"pr_number":     event.PRNumber,
		"affected_rows": rowsAffected,
	})
}
