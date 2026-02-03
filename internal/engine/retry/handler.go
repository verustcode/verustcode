// Package retry provides handlers for retrying failed reviews and rules.
package retry

import (
	"context"
	"fmt"
	"path/filepath"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/engine/runner"
	"github.com/verustcode/verustcode/internal/engine/task"
	"github.com/verustcode/verustcode/internal/engine/utils"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// TaskEnqueuer allows components to enqueue tasks for processing.
type TaskEnqueuer interface {
	Enqueue(task *task.Task) bool
	HasTask(reviewID string) bool
}

// ProviderResolver provides access to git providers.
type ProviderResolver interface {
	Get(name string) provider.Provider
	DetectFromURL(url string) string
}

// TaskBuilder builds recovery tasks from reviews.
type TaskBuilder interface {
	BuildRecoveryTask(ctx context.Context, review *model.Review) *task.Task
}

// Handler handles retry operations for failed reviews and rules.
type Handler struct {
	cfg              *config.Config
	configProvider   config.ConfigProvider
	store            store.Store
	providerResolver ProviderResolver
	taskEnqueuer     TaskEnqueuer
	taskBuilder      TaskBuilder
	runner           *runner.Runner
	dslLoader        *dsl.Loader
	ctx              context.Context
}

// NewHandler creates a new RetryHandler instance.
func NewHandler(
	cfg *config.Config,
	s store.Store,
	providerResolver ProviderResolver,
	taskEnqueuer TaskEnqueuer,
	taskBuilder TaskBuilder,
	rnr *runner.Runner,
	ctx context.Context,
) *Handler {
	return &Handler{
		cfg:              cfg,
		configProvider:   config.NewDBConfigProvider(s),
		store:            s,
		providerResolver: providerResolver,
		taskEnqueuer:     taskEnqueuer,
		taskBuilder:      taskBuilder,
		runner:           rnr,
		dslLoader:        dsl.NewLoader(),
		ctx:              ctx,
	}
}

// getReviewConfig retrieves review configuration from database with fallback to cached config.
func (h *Handler) getReviewConfig() *config.ReviewConfig {
	if h.configProvider != nil {
		reviewCfg, err := h.configProvider.GetReviewConfig()
		if err == nil && reviewCfg != nil {
			return reviewCfg
		}
	}
	// Fallback to cached config
	if h.cfg != nil {
		return &h.cfg.Review
	}
	return nil
}

// loadDSLConfigWithPriority loads review configuration with priority order:
// 1. .verust-review.yaml at repository root (highest priority)
// 2. Database configured review file for this repository
// 3. config/reviews/default.yaml (fallback)
func (h *Handler) loadDSLConfigWithPriority(repoPath, repoURL string) (*dsl.ReviewRulesConfig, error) {
	// Priority 1: Check for .verust-review.yaml at repository root
	if repoPath != "" {
		rootConfig, err := h.dslLoader.LoadFromRepoRoot(repoPath)
		if err != nil {
			logger.Warn("Failed to load review config from repository root",
				zap.String("repo_path", repoPath),
				zap.Error(err),
			)
			// Continue to next priority
		} else if rootConfig != nil {
			logger.Info("Using review config from repository root (.verust-review.yaml)",
				zap.String("repo_path", repoPath),
				zap.Int("rules", len(rootConfig.Rules)),
			)
			return rootConfig, nil
		}
	}

	// Priority 2: Database configured review file
	repoConfig, err := h.store.RepositoryConfig().GetByRepoURL(repoURL)
	if err == nil && repoConfig.ReviewFile != "" {
		reviewFilePath := filepath.Join(config.ReviewsDir, repoConfig.ReviewFile)
		cfg, loadErr := h.dslLoader.Load(reviewFilePath)
		if loadErr != nil {
			logger.Warn("Failed to load configured review file, falling back to default",
				zap.String("review_file", repoConfig.ReviewFile),
				zap.Error(loadErr),
			)
			// Fall through to load default
		} else {
			logger.Info("Using repository-specific review config",
				zap.String("repo_url", repoURL),
				zap.String("review_file", repoConfig.ReviewFile),
			)
			return cfg, nil
		}
	}

	// Priority 3: Default review configuration
	logger.Debug("Using default review config",
		zap.String("repo_url", repoURL),
	)
	return h.dslLoader.LoadDefaultReviewConfig()
}

// Retry retries a failed review by resetting its state and re-enqueuing.
// This method:
// 1. Validates that the review exists and is in a failed state
// 2. Resets the review status to pending and increments retry count
// 3. Cleans up or resets associated ReviewRule and ReviewRuleRun records
// 4. Re-enqueues the review task
//
// Note: Manual retry has no limit. The MaxRetries config only applies to
// automatic retries during execution (in executor.go).
func (h *Handler) Retry(reviewID string) error {
	// Load the review
	review, err := h.store.Review().GetByID(reviewID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.Wrap(errors.ErrCodeReviewNotFound, "review not found", err)
		}
		return errors.Wrap(errors.ErrCodeDBQuery, "failed to load review", err)
	}

	// Validate review status - only failed reviews can be retried
	if review.Status != model.ReviewStatusFailed {
		return errors.New(errors.ErrCodeValidation, fmt.Sprintf("cannot retry review with status '%s', only failed reviews can be retried", review.Status))
	}

	// Check if review is already in the queue (prevent duplicate submissions)
	if h.taskEnqueuer.HasTask(reviewID) {
		return errors.New(errors.ErrCodeValidation, "review is already in the queue")
	}

	logger.Info("Retrying failed review",
		zap.String("review_id", reviewID),
		zap.Int("retry_count", review.RetryCount+1),
	)

	// Reset review state
	if err := h.store.Review().ResetReviewState(reviewID, review.RetryCount+1); err != nil {
		return errors.Wrap(errors.ErrCodeDBQuery, "failed to reset review state", err)
	}

	// Reload review with updated state
	review, err = h.store.Review().GetByID(reviewID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeDBQuery, "failed to reload review", err)
	}

	// Build recovery task and re-enqueue
	t := h.taskBuilder.BuildRecoveryTask(h.ctx, review)
	if t == nil {
		// Revert review status to failed if task building fails
		h.store.Review().UpdateStatusWithError(review.ID, model.ReviewStatusFailed, "retry failed: could not build recovery task")
		return errors.New(errors.ErrCodeInternal, "failed to build recovery task for retry")
	}

	// Submit to memory queue
	if !h.taskEnqueuer.Enqueue(t) {
		// Task already exists in queue (should not happen after our check above)
		logger.Warn("Task already in queue during retry",
			zap.String("review_id", reviewID),
		)
		return nil
	}

	logger.Info("Review retry submitted to queue",
		zap.String("review_id", reviewID),
		zap.Int("retry_count", review.RetryCount),
	)

	return nil
}

// RetryRule retries a single failed rule within a review.
// This method allows parallel execution - the rule retry runs in a separate goroutine
// while other rules may still be executing.
// Parameters:
//   - reviewID: the ID of the review containing the rule
//   - ruleID: the rule_id (not the database ID) of the rule to retry
func (h *Handler) RetryRule(reviewID string, ruleID string) error {
	// Load the review with its rules
	review, err := h.store.Review().GetByIDWithRules(reviewID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Error("Failed to load review for rule retry",
				zap.String("review_id", reviewID),
				zap.Error(err),
			)
			return errors.Wrap(errors.ErrCodeReviewNotFound, "review not found", err)
		}
		logger.Error("Failed to load review for rule retry",
			zap.String("review_id", reviewID),
			zap.Error(err),
		)
		return errors.Wrap(errors.ErrCodeDBQuery, "failed to load review", err)
	}

	// Find the rule by rule_id
	var targetRule *model.ReviewRule
	for i := range review.Rules {
		if review.Rules[i].RuleID == ruleID {
			targetRule = &review.Rules[i]
			break
		}
	}

	if targetRule == nil {
		logger.Error("Rule not found in review",
			zap.String("review_id", reviewID),
			zap.String("rule_id", ruleID),
		)
		return errors.New(errors.ErrCodeValidation, fmt.Sprintf("rule '%s' not found in review", ruleID))
	}

	// Validate rule status - only failed rules can be retried
	if targetRule.Status != model.RuleStatusFailed {
		logger.Warn("Attempted to retry non-failed rule",
			zap.String("review_id", reviewID),
			zap.String("rule_id", ruleID),
			zap.String("status", string(targetRule.Status)),
		)
		return errors.New(errors.ErrCodeValidation, fmt.Sprintf("cannot retry rule with status '%s', only failed rules can be retried", targetRule.Status))
	}

	// Note: Manual retry has no limit. The MaxRetries config only applies to
	// automatic retries during execution (in executor.go).

	logger.Info("Retrying single rule",
		zap.String("review_id", reviewID),
		zap.String("rule_id", ruleID),
		zap.Int("rule_retry_count", targetRule.RetryCount+1),
		zap.Int("review_retry_count", review.RetryCount),
	)

	// Reset rule state in a transaction
	if err := h.store.Review().ResetRuleState(ruleID, reviewID, targetRule.RetryCount+1, review.RetryCount+1); err != nil {
		logger.Error("Failed to reset rule state for retry",
			zap.String("review_id", reviewID),
			zap.String("rule_id", ruleID),
			zap.Error(err),
		)
		return errors.Wrap(errors.ErrCodeDBQuery, "failed to reset rule state", err)
	}

	// Reload the rule with updated state
	targetRule, err = h.store.Review().GetRuleByID(targetRule.ID)
	if err != nil {
		logger.Error("Failed to reload rule after reset",
			zap.String("review_id", reviewID),
			zap.String("rule_id", ruleID),
			zap.Error(err),
		)
		return errors.Wrap(errors.ErrCodeDBQuery, "failed to reload rule", err)
	}

	// Reload review with updated state
	review, err = h.store.Review().GetByID(reviewID)
	if err != nil {
		logger.Error("Failed to reload review after reset",
			zap.String("review_id", reviewID),
			zap.Error(err),
		)
		return errors.Wrap(errors.ErrCodeDBQuery, "failed to reload review", err)
	}

	// Start rule execution in a separate goroutine
	// This allows parallel execution with other rules
	go h.executeRuleRetry(reviewID, ruleID, review, targetRule)

	logger.Info("Rule retry initiated",
		zap.String("review_id", reviewID),
		zap.String("rule_id", ruleID),
	)

	return nil
}

// executeRuleRetry executes a single rule retry in a goroutine.
// This method handles the actual execution after RetryRule has validated and reset the state.
func (h *Handler) executeRuleRetry(reviewID, ruleID string, review *model.Review, reviewRule *model.ReviewRule) {
	ctx := h.ctx

	logger.Info("Starting rule retry execution",
		zap.String("review_id", reviewID),
		zap.String("rule_id", ruleID),
	)

	// Dynamically load DSL config with priority:
	// 1. .verust-review.yaml at repository root (if repo is cloned)
	// 2. Database configured review file
	// 3. config/reviews/default.yaml
	dslConfig, err := h.loadDSLConfigWithPriority(review.RepoPath, review.RepoURL)
	if err != nil {
		logger.Error("Failed to load DSL config for rule retry",
			zap.String("review_id", reviewID),
			zap.String("rule_id", ruleID),
			zap.Error(err),
		)
		// Mark rule as failed
		h.store.Review().UpdateRuleStatusWithError(reviewRule.ID, model.RuleStatusFailed, fmt.Sprintf("failed to load DSL config: %v", err))
		h.runner.UpdateReviewStatusAfterRuleExecution(review)
		return
	}

	// Find the rule config
	var ruleConfig *dsl.ReviewRuleConfig
	for i := range dslConfig.Rules {
		if dslConfig.Rules[i].ID == ruleID {
			ruleConfig = &dslConfig.Rules[i]
			break
		}
	}

	if ruleConfig == nil {
		logger.Error("Rule config not found in DSL config",
			zap.String("review_id", reviewID),
			zap.String("rule_id", ruleID),
		)
		h.store.Review().UpdateRuleStatusWithError(reviewRule.ID, model.RuleStatusFailed, "rule config not found in DSL config")
		h.runner.UpdateReviewStatusAfterRuleExecution(review)
		return
	}

	// Build execution context
	// Get output language configuration from database
	outputLanguage := ""
	execReviewCfg := h.getReviewConfig()
	if execReviewCfg != nil {
		if langCfg, err := execReviewCfg.GetOutputLanguage(); err == nil {
			outputLanguage = langCfg.PromptInstruction()
		}
	}

	// Build context for prompt generation
	buildCtx := &prompt.BuildContext{
		RepoPath:       review.RepoPath,
		RepoURL:        review.RepoURL,
		Ref:            review.Ref,
		CommitSHA:      review.CommitSHA,
		PRNumber:       review.PRNumber,
		OutputLanguage: outputLanguage,
	}

	// Determine provider
	var prov provider.Provider
	providerName := ""
	if review.RepoURL != "" {
		providerName = h.providerResolver.DetectFromURL(review.RepoURL)
		if providerName != "" {
			prov = h.providerResolver.Get(providerName)
		}
	}

	// Get PR info
	var prInfo *provider.PullRequest
	if prov != nil && review.PRNumber > 0 && review.RepoURL != "" {
		owner, repo, err := prov.ParseRepoPath(review.RepoURL)
		if err == nil && owner != "" && repo != "" {
			pr, err := prov.GetPullRequest(ctx, owner, repo, review.PRNumber)
			if err == nil {
				prInfo = pr
			}
		}
	}

	if prInfo == nil && review.PRNumber > 0 && review.RepoURL != "" {
		prURL := utils.BuildPRURL(review.RepoURL, providerName, review.PRNumber)
		if prURL != "" {
			prInfo = &provider.PullRequest{
				Number: review.PRNumber,
				URL:    prURL,
			}
		}
	}

	// Create execution context
	execCtx := &runner.RuleExecutionContext{
		BuildCtx:     buildCtx,
		Review:       review,
		ReviewRule:   reviewRule,
		Rule:         ruleConfig,
		RuleIndex:    reviewRule.RuleIndex,
		Provider:     prov,
		ProviderName: providerName,
		PRInfo:       prInfo,
		OutputDir:    "", // Default output dir
	}

	// Execute the rule
	_, err = h.runner.ExecuteSingleRule(ctx, execCtx)
	if err != nil {
		logger.Error("Rule retry execution failed",
			zap.String("review_id", reviewID),
			zap.String("rule_id", ruleID),
			zap.Error(err),
		)
		// ExecuteSingleRule already handles the error, but we log it here
	}

	// Update review status based on all rules
	h.runner.UpdateReviewStatusAfterRuleExecution(review)

	logger.Info("Rule retry execution completed",
		zap.String("review_id", reviewID),
		zap.String("rule_id", ruleID),
		zap.String("rule_status", string(reviewRule.Status)),
	)
}
