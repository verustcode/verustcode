package store

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// FindingWithRepoInfo represents a review result with its associated review information
// Used for the findings list endpoint to display findings across all repositories
type FindingWithRepoInfo struct {
	ReviewID  string        `json:"review_id"`
	RepoURL   string        `json:"repo_url"`
	Data      model.JSONMap `json:"data"`
	CreatedAt time.Time     `json:"created_at"`
}

// ReviewStore defines operations for Review, ReviewRule, ReviewRuleRun, and ReviewResult models.
type ReviewStore interface {
	// Review CRUD
	Create(review *model.Review) error
	GetByID(id string) (*model.Review, error)
	GetByIDWithDetails(id string) (*model.Review, error)
	GetByIDWithRules(id string) (*model.Review, error)
	Update(review *model.Review) error
	Save(review *model.Review) error
	Delete(id string) error

	// Review status updates
	UpdateStatus(id string, status model.ReviewStatus) error
	UpdateStatusWithError(id string, status model.ReviewStatus, errMsg string) error
	UpdateStatusWithErrorAndCompletedAt(id string, status model.ReviewStatus, errMsg string) error
	UpdateStatusToRunningIfPending(id string, startedAt time.Time) (bool, error)
	UpdateStatusIfAllowed(id string, newStatus model.ReviewStatus, allowedStatuses []model.ReviewStatus) (int64, error)
	UpdateProgress(id string, currentRuleIndex int) error
	UpdateCurrentRuleIndex(reviewID string, index int) error
	UpdateRepoPath(reviewID, repoPath string) error
	UpdateMetadata(reviewID string, updates map[string]interface{}) error
	IncrementRetryCount(id string) error

	// Review queries
	List(statusFilter string, limit, offset int) ([]model.Review, int64, error)
	ListByRepository(repoURL string, limit, offset int) ([]model.Review, int64, error)
	ListByStatus(status model.ReviewStatus) ([]model.Review, error)
	ListPendingOrRunning() ([]model.Review, error)
	GetByPRURLAndCommit(prURL, commitSHA string) (*model.Review, error)

	// ReviewRule operations
	CreateRule(rule *model.ReviewRule) error
	BatchCreateRules(rules []model.ReviewRule) error
	GetRuleByID(id uint) (*model.ReviewRule, error)
	GetRulesByReviewID(reviewID string) ([]model.ReviewRule, error)
	UpdateRule(rule *model.ReviewRule) error
	UpdateRuleStatus(id uint, status model.RuleStatus) error
	UpdateRuleStatusWithError(id uint, status model.RuleStatus, errMsg string) error

	// ReviewRuleRun operations
	CreateRun(run *model.ReviewRuleRun) error
	GetRunByID(id uint) (*model.ReviewRuleRun, error)
	GetRunsByRuleID(ruleID uint) ([]model.ReviewRuleRun, error)
	DeleteReviewRuleRunsByRuleID(ruleID uint) error
	UpdateRun(run *model.ReviewRuleRun) error
	UpdateRunStatus(id uint, status model.RunStatus) error

	// ReviewResult operations
	CreateResult(result *model.ReviewResult) error
	DeleteReviewResultsByRuleID(ruleID uint) error
	GetResultsByRuleID(ruleID uint) ([]model.ReviewResult, error)
	GetResultsByReviewID(reviewID string) ([]model.ReviewResult, error)

	// Webhook log operations
	CreateWebhookLog(log *model.ReviewResultWebhookLog) error
	UpdateWebhookLog(log *model.ReviewResultWebhookLog) error
	GetPendingWebhookLogs() ([]model.ReviewResultWebhookLog, error)

	// Statistics queries
	CountByStatusAndDateRange(status model.ReviewStatus, start, end time.Time) (int64, error)
	GetReviewsWithResultsByRepository(repoURL string, limit, offset int) ([]model.Review, error)

	// Admin statistics queries
	CountAll() (int64, error)
	CountCreatedAfter(start time.Time) (int64, error)
	CountByStatusOnly(status model.ReviewStatus) (int64, error)
	CountByStatusAndCompletedAfter(status model.ReviewStatus, start time.Time) (int64, error)
	CountCompletedOrFailedAfter(start time.Time) (int64, error)
	CountCompletedAfter(start time.Time) (int64, error)
	GetAverageDurationAfter(start time.Time) (float64, error)

	// Repo statistics queries (for stats handler)
	ListCompletedByRepoAndDateRange(repoURL string, start time.Time) ([]model.Review, error)
	GetReviewResultsByReviewIDs(reviewIDs []string) ([]model.ReviewResult, error)

	// Findings queries (for findings handler)
	// GetAllFindingsWithRepoInfo returns all review results with their associated review info
	GetAllFindingsWithRepoInfo(repoURL string) ([]FindingWithRepoInfo, error)

	// Webhook-specific queries
	GetMaxRevisionByPRURL(prURL string) (int, error)
	UpdateMergedAtByPRURL(prURL string, mergedAt time.Time) (int64, error)
	// FindPreviousReviewResult returns the complete result JSON from previous review
	FindPreviousReviewResult(prURL, ruleID, currentReviewID string) (string, bool, error)
	ResetReviewState(reviewID string, retryCount int) error
	ResetRuleState(ruleID string, reviewID string, ruleRetryCount, reviewRetryCount int) error
}

// reviewStore implements ReviewStore using GORM.
type reviewStore struct {
	db *gorm.DB
}

func newReviewStore(db *gorm.DB) ReviewStore {
	return &reviewStore{db: db}
}

// Review CRUD implementations

func (s *reviewStore) Create(review *model.Review) error {
	return s.db.Create(review).Error
}

func (s *reviewStore) GetByID(id string) (*model.Review, error) {
	var review model.Review
	err := s.db.First(&review, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func (s *reviewStore) GetByIDWithDetails(id string) (*model.Review, error) {
	var review model.Review
	err := s.db.Preload("Rules").Preload("Rules.Runs").Preload("Rules.Results").First(&review, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func (s *reviewStore) GetByIDWithRules(id string) (*model.Review, error) {
	var review model.Review
	err := s.db.Preload("Rules").First(&review, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func (s *reviewStore) Update(review *model.Review) error {
	return s.db.Model(review).Updates(review).Error
}

func (s *reviewStore) Save(review *model.Review) error {
	return s.db.Save(review).Error
}

func (s *reviewStore) Delete(id string) error {
	return s.db.Delete(&model.Review{}, "id = ?", id).Error
}

// Review status updates

func (s *reviewStore) UpdateStatus(id string, status model.ReviewStatus) error {
	return s.db.Model(&model.Review{}).Where("id = ?", id).Update("status", status).Error
}

func (s *reviewStore) UpdateStatusWithError(id string, status model.ReviewStatus, errMsg string) error {
	return s.db.Model(&model.Review{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"error_message": errMsg,
	}).Error
}

func (s *reviewStore) UpdateStatusWithErrorAndCompletedAt(id string, status model.ReviewStatus, errMsg string) error {
	return s.db.Model(&model.Review{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"error_message": errMsg,
		"completed_at":  time.Now(),
	}).Error
}

func (s *reviewStore) UpdateStatusToRunningIfPending(id string, startedAt time.Time) (bool, error) {
	result := s.db.Model(&model.Review{}).
		Where("id = ?", id).
		Where("status IN ?", []model.ReviewStatus{model.ReviewStatusPending, model.ReviewStatusRunning}).
		Updates(map[string]interface{}{
			"status":     model.ReviewStatusRunning,
			"started_at": startedAt,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (s *reviewStore) UpdateProgress(id string, currentRuleIndex int) error {
	return s.db.Model(&model.Review{}).Where("id = ?", id).Update("current_rule_index", currentRuleIndex).Error
}

func (s *reviewStore) UpdateCurrentRuleIndex(reviewID string, index int) error {
	return s.db.Model(&model.Review{}).Where("id = ?", reviewID).Update("current_rule_index", index).Error
}

func (s *reviewStore) UpdateRepoPath(reviewID, repoPath string) error {
	return s.db.Model(&model.Review{}).Where("id = ?", reviewID).Update("repo_path", repoPath).Error
}

func (s *reviewStore) UpdateMetadata(reviewID string, updates map[string]interface{}) error {
	return s.db.Model(&model.Review{}).Where("id = ?", reviewID).Updates(updates).Error
}

func (s *reviewStore) IncrementRetryCount(id string) error {
	return s.db.Model(&model.Review{}).Where("id = ?", id).UpdateColumn("retry_count", gorm.Expr("retry_count + 1")).Error
}

func (s *reviewStore) UpdateStatusIfAllowed(id string, newStatus model.ReviewStatus, allowedStatuses []model.ReviewStatus) (int64, error) {
	result := s.db.Model(&model.Review{}).Where("id = ? AND status IN ?", id, allowedStatuses).Update("status", newStatus)
	return result.RowsAffected, result.Error
}

// Review queries

func (s *reviewStore) List(statusFilter string, limit, offset int) ([]model.Review, int64, error) {
	var reviews []model.Review
	var total int64

	query := s.db.Model(&model.Review{})
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&reviews).Error
	return reviews, total, err
}

func (s *reviewStore) ListByRepository(repoURL string, limit, offset int) ([]model.Review, int64, error) {
	var reviews []model.Review
	var total int64

	query := s.db.Model(&model.Review{}).Where("repo_url = ?", repoURL)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&reviews).Error
	return reviews, total, err
}

func (s *reviewStore) ListByStatus(status model.ReviewStatus) ([]model.Review, error) {
	var reviews []model.Review
	err := s.db.Where("status = ?", status).Find(&reviews).Error
	return reviews, err
}

func (s *reviewStore) ListPendingOrRunning() ([]model.Review, error) {
	var reviews []model.Review
	err := s.db.Where("status IN ?", []model.ReviewStatus{
		model.ReviewStatusPending,
		model.ReviewStatusRunning,
	}).Order("created_at ASC").Find(&reviews).Error
	return reviews, err
}

func (s *reviewStore) GetByPRURLAndCommit(prURL, commitSHA string) (*model.Review, error) {
	var review model.Review
	err := s.db.Where("pr_url = ? AND commit_sha = ?", prURL, commitSHA).First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// ReviewRule operations

func (s *reviewStore) CreateRule(rule *model.ReviewRule) error {
	return s.db.Create(rule).Error
}

func (s *reviewStore) BatchCreateRules(rules []model.ReviewRule) error {
	if len(rules) == 0 {
		return nil
	}
	return s.db.Create(&rules).Error
}

func (s *reviewStore) GetRuleByID(id uint) (*model.ReviewRule, error) {
	var rule model.ReviewRule
	err := s.db.First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *reviewStore) GetRulesByReviewID(reviewID string) ([]model.ReviewRule, error) {
	var rules []model.ReviewRule
	err := s.db.Where("review_id = ?", reviewID).Order("rule_index ASC").Find(&rules).Error
	return rules, err
}

func (s *reviewStore) UpdateRule(rule *model.ReviewRule) error {
	return s.db.Save(rule).Error
}

func (s *reviewStore) UpdateRuleStatus(id uint, status model.RuleStatus) error {
	return s.db.Model(&model.ReviewRule{}).Where("id = ?", id).Update("status", status).Error
}

func (s *reviewStore) UpdateRuleStatusWithError(id uint, status model.RuleStatus, errMsg string) error {
	return s.db.Model(&model.ReviewRule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"error_message": errMsg,
	}).Error
}

// ReviewRuleRun operations

func (s *reviewStore) CreateRun(run *model.ReviewRuleRun) error {
	return s.db.Create(run).Error
}

func (s *reviewStore) GetRunByID(id uint) (*model.ReviewRuleRun, error) {
	var run model.ReviewRuleRun
	err := s.db.First(&run, id).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *reviewStore) GetRunsByRuleID(ruleID uint) ([]model.ReviewRuleRun, error) {
	var runs []model.ReviewRuleRun
	err := s.db.Where("review_rule_id = ?", ruleID).Order("run_index ASC").Find(&runs).Error
	return runs, err
}

func (s *reviewStore) DeleteReviewRuleRunsByRuleID(ruleID uint) error {
	return s.db.Where("review_rule_id = ?", ruleID).Delete(&model.ReviewRuleRun{}).Error
}

func (s *reviewStore) UpdateRun(run *model.ReviewRuleRun) error {
	return s.db.Save(run).Error
}

func (s *reviewStore) UpdateRunStatus(id uint, status model.RunStatus) error {
	return s.db.Model(&model.ReviewRuleRun{}).Where("id = ?", id).Update("status", status).Error
}

// ReviewResult operations

func (s *reviewStore) CreateResult(result *model.ReviewResult) error {
	return s.db.Create(result).Error
}

func (s *reviewStore) DeleteReviewResultsByRuleID(ruleID uint) error {
	return s.db.Where("review_rule_id = ?", ruleID).Delete(&model.ReviewResult{}).Error
}

func (s *reviewStore) GetResultsByRuleID(ruleID uint) ([]model.ReviewResult, error) {
	var results []model.ReviewResult
	err := s.db.Where("review_rule_id = ?", ruleID).Find(&results).Error
	return results, err
}

func (s *reviewStore) GetResultsByReviewID(reviewID string) ([]model.ReviewResult, error) {
	var results []model.ReviewResult
	err := s.db.Table("review_results").
		Joins("JOIN review_rules ON review_results.review_rule_id = review_rules.id").
		Where("review_rules.review_id = ?", reviewID).
		Find(&results).Error
	return results, err
}

// Webhook log operations

func (s *reviewStore) CreateWebhookLog(log *model.ReviewResultWebhookLog) error {
	return s.db.Create(log).Error
}

func (s *reviewStore) UpdateWebhookLog(log *model.ReviewResultWebhookLog) error {
	return s.db.Save(log).Error
}

func (s *reviewStore) GetPendingWebhookLogs() ([]model.ReviewResultWebhookLog, error) {
	var logs []model.ReviewResultWebhookLog
	err := s.db.Where("status = ?", model.WebhookStatusPending).Find(&logs).Error
	return logs, err
}

// Statistics queries

func (s *reviewStore) CountByStatusAndDateRange(status model.ReviewStatus, start, end time.Time) (int64, error) {
	var count int64
	err := s.db.Model(&model.Review{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", status, start, end).
		Count(&count).Error
	return count, err
}

func (s *reviewStore) GetReviewsWithResultsByRepository(repoURL string, limit, offset int) ([]model.Review, error) {
	var reviews []model.Review
	err := s.db.Where("repo_url = ?", repoURL).
		Preload("Rules.Results").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reviews).Error
	return reviews, err
}

// Admin statistics queries

func (s *reviewStore) CountAll() (int64, error) {
	var count int64
	err := s.db.Model(&model.Review{}).Count(&count).Error
	return count, err
}

func (s *reviewStore) CountCreatedAfter(start time.Time) (int64, error) {
	var count int64
	err := s.db.Model(&model.Review{}).Where("created_at >= ?", start).Count(&count).Error
	return count, err
}

func (s *reviewStore) CountByStatusOnly(status model.ReviewStatus) (int64, error) {
	var count int64
	err := s.db.Model(&model.Review{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (s *reviewStore) CountByStatusAndCompletedAfter(status model.ReviewStatus, start time.Time) (int64, error) {
	var count int64
	err := s.db.Model(&model.Review{}).
		Where("status = ? AND completed_at >= ?", status, start).
		Count(&count).Error
	return count, err
}

func (s *reviewStore) CountCompletedOrFailedAfter(start time.Time) (int64, error) {
	var count int64
	err := s.db.Model(&model.Review{}).
		Where("completed_at >= ? AND status IN ?", start,
			[]model.ReviewStatus{model.ReviewStatusCompleted, model.ReviewStatusFailed}).
		Count(&count).Error
	return count, err
}

func (s *reviewStore) CountCompletedAfter(start time.Time) (int64, error) {
	var count int64
	err := s.db.Model(&model.Review{}).
		Where("completed_at >= ? AND status = ?", start, model.ReviewStatusCompleted).
		Count(&count).Error
	return count, err
}

func (s *reviewStore) GetAverageDurationAfter(start time.Time) (float64, error) {
	var avgDuration float64
	err := s.db.Model(&model.Review{}).
		Where("completed_at >= ? AND status = ? AND duration > 0", start, model.ReviewStatusCompleted).
		Select("AVG(duration)").Row().Scan(&avgDuration)
	return avgDuration, err
}

// Repo statistics queries (for stats handler)

func (s *reviewStore) ListCompletedByRepoAndDateRange(repoURL string, start time.Time) ([]model.Review, error) {
	var reviews []model.Review
	query := s.db.Model(&model.Review{}).
		Where("created_at >= ?", start).
		Where("status = ?", model.ReviewStatusCompleted)

	if repoURL != "" {
		query = query.Where("repo_url = ?", repoURL)
	}

	err := query.Order("created_at ASC").Find(&reviews).Error
	return reviews, err
}

func (s *reviewStore) GetReviewResultsByReviewIDs(reviewIDs []string) ([]model.ReviewResult, error) {
	if len(reviewIDs) == 0 {
		return []model.ReviewResult{}, nil
	}

	var results []model.ReviewResult
	err := s.db.Table("review_results").
		Joins("JOIN review_rules ON review_rules.id = review_results.review_rule_id").
		Where("review_rules.review_id IN ?", reviewIDs).
		Where("review_rules.deleted_at IS NULL").
		Find(&results).Error
	return results, err
}

// Webhook-specific queries

func (s *reviewStore) GetMaxRevisionByPRURL(prURL string) (int, error) {
	var maxRevision int
	err := s.db.Model(&model.Review{}).
		Where("pr_url = ?", prURL).
		Select("COALESCE(MAX(revision_count), 0)").
		Row().Scan(&maxRevision)
	return maxRevision, err
}

func (s *reviewStore) UpdateMergedAtByPRURL(prURL string, mergedAt time.Time) (int64, error) {
	result := s.db.Model(&model.Review{}).
		Where("pr_url = ?", prURL).
		Where("merged_at IS NULL").
		Update("merged_at", mergedAt)
	return result.RowsAffected, result.Error
}

// FindPreviousReviewResult finds the previous review result for the same PR and rule.
// Returns the complete result data as JSON string for historical comparison.
func (s *reviewStore) FindPreviousReviewResult(prURL, ruleID, currentReviewID string) (string, bool, error) {
	// Find the most recent completed review for the same PR (excluding current review)
	var previousReview model.Review
	err := s.db.Where("pr_url = ? AND id != ? AND status = ?", prURL, currentReviewID, model.ReviewStatusCompleted).
		Order("created_at DESC").
		First(&previousReview).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", false, nil
		}
		return "", false, err
	}

	// Find the review rule with matching rule_id in the previous review
	var previousRule model.ReviewRule
	err = s.db.Where("review_id = ? AND rule_id = ? AND status = ?",
		previousReview.ID, ruleID, model.RuleStatusCompleted).
		First(&previousRule).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", false, nil
		}
		return "", false, err
	}

	// Find the ReviewResult for this rule to get data
	var reviewResult model.ReviewResult
	err = s.db.Where("review_rule_id = ?", previousRule.ID).
		First(&reviewResult).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", false, nil
		}
		return "", false, err
	}

	// Return the complete Data as JSON string for historical comparison
	// This allows the AI to compare with each finding's details including status
	if len(reviewResult.Data) > 0 {
		jsonBytes, err := json.Marshal(reviewResult.Data)
		if err == nil {
			return string(jsonBytes), true, nil
		}
	}

	return "", false, nil
}

func (s *reviewStore) ResetReviewState(reviewID string, retryCount int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Update review: reset status, increment retry count, clear error message
		updates := map[string]interface{}{
			"status":             model.ReviewStatusPending,
			"retry_count":        retryCount,
			"error_message":      "",
			"current_rule_index": 0,
			"completed_at":       nil,
		}
		if err := tx.Model(&model.Review{}).Where("id = ?", reviewID).Updates(updates).Error; err != nil {
			return err
		}

		// 2. Reset all ReviewRule records to pending status
		var rules []model.ReviewRule
		if err := tx.Where("review_id = ?", reviewID).Find(&rules).Error; err != nil {
			return err
		}

		for _, rule := range rules {
			// Reset rule status
			if err := tx.Model(&rule).Updates(map[string]interface{}{
				"status":        model.RuleStatusPending,
				"error_message": "",
				"started_at":    nil,
				"completed_at":  nil,
			}).Error; err != nil {
				return err
			}

			// Delete old ReviewRuleRun records
			if err := tx.Where("review_rule_id = ?", rule.ID).Delete(&model.ReviewRuleRun{}).Error; err != nil {
				return err
			}

			// Delete old ReviewResult records
			if err := tx.Where("review_rule_id = ?", rule.ID).Delete(&model.ReviewResult{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *reviewStore) ResetRuleState(ruleID string, reviewID string, ruleRetryCount, reviewRetryCount int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Update rule: reset status, increment retry count, clear error message
		ruleUpdates := map[string]interface{}{
			"status":        model.RuleStatusRunning,
			"retry_count":   ruleRetryCount,
			"error_message": "",
			"started_at":    time.Now(),
			"completed_at":  nil,
		}
		if err := tx.Model(&model.ReviewRule{}).Where("rule_id = ? AND review_id = ?", ruleID, reviewID).Updates(ruleUpdates).Error; err != nil {
			return err
		}

		// Get the rule internal ID to delete related records
		var rule model.ReviewRule
		if err := tx.Where("rule_id = ? AND review_id = ?", ruleID, reviewID).First(&rule).Error; err != nil {
			return err
		}

		// Update review: increment retry count if needed, set status to running
		reviewUpdates := map[string]interface{}{
			"retry_count": reviewRetryCount,
		}

		// Check current review status
		var review model.Review
		if err := tx.First(&review, "id = ?", reviewID).Error; err != nil {
			return err
		}

		if review.Status == model.ReviewStatusFailed {
			reviewUpdates["status"] = model.ReviewStatusRunning
			reviewUpdates["error_message"] = ""
		}

		if err := tx.Model(&model.Review{}).Where("id = ?", reviewID).Updates(reviewUpdates).Error; err != nil {
			return err
		}

		// Delete old ReviewRuleRun records
		if err := tx.Where("review_rule_id = ?", rule.ID).Delete(&model.ReviewRuleRun{}).Error; err != nil {
			return err
		}

		// Delete old ReviewResult records
		if err := tx.Where("review_rule_id = ?", rule.ID).Delete(&model.ReviewResult{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetAllFindingsWithRepoInfo returns all review results with their associated review information
// Used for the findings list endpoint to display findings across all repositories
func (s *reviewStore) GetAllFindingsWithRepoInfo(repoURL string) ([]FindingWithRepoInfo, error) {
	var results []FindingWithRepoInfo

	query := s.db.Table("review_results").
		Select("reviews.id as review_id, reviews.repo_url, review_results.data, review_results.created_at").
		Joins("JOIN review_rules ON review_rules.id = review_results.review_rule_id").
		Joins("JOIN reviews ON reviews.id = review_rules.review_id").
		Where("review_rules.deleted_at IS NULL").
		Where("reviews.deleted_at IS NULL").
		Where("reviews.status = ?", model.ReviewStatusCompleted)

	if repoURL != "" {
		query = query.Where("reviews.repo_url = ?", repoURL)
	}

	err := query.Order("review_results.created_at DESC").Find(&results).Error
	return results, err
}
