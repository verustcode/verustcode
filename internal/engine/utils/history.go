// Package utils provides utility functions for the review engine.
package utils

import (
	"encoding/json"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/pkg/logger"
)

// FindPreviousReviewResult finds the most recent completed review result for the same PR + rule.
// It searches for the last completed ReviewRule with the same pr_url and rule_id,
// excluding the current review.
//
// Parameters:
//   - db: database connection
//   - prURL: the PR/MR URL to match
//   - ruleID: the rule ID to match
//   - currentReviewID: the current review ID to exclude from results
//
// Returns:
//   - summary: the summary text from the previous review, empty string if not found
//   - found: true if a previous review was found
func FindPreviousReviewResult(db *gorm.DB, prURL, ruleID, currentReviewID string) (summary string, found bool) {
	if db == nil || prURL == "" || ruleID == "" {
		return "", false
	}

	// Find the most recent completed review for the same PR (excluding current review)
	var previousReview model.Review
	result := db.Where("pr_url = ? AND id != ? AND status = ?", prURL, currentReviewID, model.ReviewStatusCompleted).
		Order("created_at DESC").
		First(&previousReview)

	if result.Error != nil {
		if result.Error != gorm.ErrRecordNotFound {
			logger.Debug("Failed to find previous review",
				zap.String("pr_url", prURL),
				zap.String("current_review_id", currentReviewID),
				zap.Error(result.Error),
			)
		}
		return "", false
	}

	// Find the review rule with matching rule_id in the previous review
	var previousRule model.ReviewRule
	result = db.Where("review_id = ? AND rule_id = ? AND status = ?",
		previousReview.ID, ruleID, model.RuleStatusCompleted).
		First(&previousRule)

	if result.Error != nil {
		if result.Error != gorm.ErrRecordNotFound {
			logger.Debug("Failed to find previous review rule",
				zap.String("previous_review_id", previousReview.ID),
				zap.String("rule_id", ruleID),
				zap.Error(result.Error),
			)
		}
		return "", false
	}

	// Find the ReviewResult for this rule to get data
	var reviewResult model.ReviewResult
	result = db.Where("review_rule_id = ?", previousRule.ID).
		First(&reviewResult)

	if result.Error != nil {
		if result.Error != gorm.ErrRecordNotFound {
			logger.Debug("Failed to find previous review result",
				zap.String("previous_review_id", previousReview.ID),
				zap.String("rule_id", ruleID),
				zap.Error(result.Error),
			)
		}
		return "", false
	}

	// Extract summary from Data["summary"]
	var summaryText string
	if s, ok := reviewResult.Data["summary"].(string); ok && s != "" {
		summaryText = s
	} else if len(reviewResult.Data) > 0 {
		// If no summary field, return the entire Data as JSON string
		jsonBytes, err := json.Marshal(reviewResult.Data)
		if err == nil {
			summaryText = string(jsonBytes)
		}
	}

	if summaryText == "" {
		logger.Debug("Previous review result has empty data",
			zap.String("previous_review_id", previousReview.ID),
			zap.String("rule_id", ruleID),
		)
		return "", false
	}

	logger.Info("Found previous review result for comparison",
		zap.String("current_review_id", currentReviewID),
		zap.String("previous_review_id", previousReview.ID),
		zap.String("rule_id", ruleID),
		zap.Int("summary_length", len(summaryText)),
	)

	return summaryText, true
}
