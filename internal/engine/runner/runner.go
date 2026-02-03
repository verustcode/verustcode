// Package runner provides the ReviewRunner which handles review execution logic.
package runner

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/engine/executor"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/output"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// ReviewRequest contains information needed to run a review.
type ReviewRequest struct {
	RepoPath          string
	RepoURL           string
	Ref               string
	CommitSHA         string
	PRNumber          int
	PRTitle           string
	PRDescription     string
	BaseCommitSHA     string
	ChangedFiles      []string
	ReviewRulesConfig *dsl.ReviewRulesConfig
	OutputDir         string
}

// RuleExecutionContext contains all context needed to execute a single rule.
type RuleExecutionContext struct {
	BuildCtx     *prompt.BuildContext
	Review       *model.Review
	ReviewRule   *model.ReviewRule
	Rule         *dsl.ReviewRuleConfig
	RuleIndex    int
	Provider     provider.Provider
	ProviderName string
	PRInfo       *provider.PullRequest
	OutputDir    string
}

// Runner handles review execution logic including running all rules for a review.
type Runner struct {
	cfg            *config.Config
	configProvider config.ConfigProvider
	store          store.Store
	executor       *executor.Executor
	promptBuilder  *prompt.Builder
}

// NewRunner creates a new Runner instance.
func NewRunner(cfg *config.Config, s store.Store, exec *executor.Executor, promptBuilder *prompt.Builder) *Runner {
	return &Runner{
		cfg:            cfg,
		configProvider: config.NewDBConfigProvider(s),
		store:          s,
		executor:       exec,
		promptBuilder:  promptBuilder,
	}
}

// getReviewConfig retrieves review configuration from database with fallback to cached config.
func (r *Runner) getReviewConfig() *config.ReviewConfig {
	if r.configProvider != nil {
		reviewCfg, err := r.configProvider.GetReviewConfig()
		if err == nil && reviewCfg != nil {
			return reviewCfg
		}
	}
	// Fallback to cached config
	if r.cfg != nil {
		return &r.cfg.Review
	}
	return nil
}

// RunReviewWithTracking executes all review rules with status tracking.
// This is the main entry point for executing a complete review.
func (r *Runner) RunReviewWithTracking(ctx context.Context, req *ReviewRequest, review *model.Review, prov provider.Provider) (*prompt.ReviewResult, error) {
	rulesConfig := req.ReviewRulesConfig
	if rulesConfig == nil || len(rulesConfig.Rules) == 0 {
		logger.Warn("No rules found in DSL config",
			zap.String("review_id", review.ID),
		)
		return nil, nil
	}

	// Load existing review rules to resume if any
	existingRules, err := r.loadExistingReviewRules(review.ID, rulesConfig.Rules)
	if err != nil {
		logger.Warn("Failed to load existing review rules",
			zap.String("review_id", review.ID),
			zap.Error(err),
		)
	}

	// Track overall result for returning last result
	var lastResult *prompt.ReviewResult
	var hasFailedRule bool

	// Get PR info for output publishing
	var prInfo *provider.PullRequest
	if req.PRNumber > 0 && prov != nil {
		// Extract owner and repo from RepoURL
		owner, repoName := extractOwnerRepo(req.RepoURL)
		if owner != "" && repoName != "" {
			pr, err := prov.GetPullRequest(ctx, owner, repoName, req.PRNumber)
			if err == nil {
				prInfo = pr
			}
		}
	}

	// Execute each rule
	for i, rule := range rulesConfig.Rules {
		// Check if rule already exists in database (for resume)
		var reviewRule *model.ReviewRule
		wasAlreadyCompleted := false

		if existing, ok := existingRules[rule.ID]; ok {
			reviewRule = existing
			wasAlreadyCompleted = existing.Status == model.RuleStatusCompleted

			// If rule is already completed, skip execution
			if wasAlreadyCompleted && !r.shouldRerunCompletedRule(rule) {
				logger.Info("Rule already completed, skipping",
					zap.String("rule_id", rule.ID),
					zap.String("review_id", review.ID),
				)
				continue
			}

			// Update status to running for resume
			reviewRule.Status = model.RuleStatusRunning
			if err := r.store.Review().UpdateRule(reviewRule); err != nil {
				logger.Warn("Failed to update review rule status",
					zap.String("review_id", review.ID),
					zap.String("rule_id", rule.ID),
					zap.Error(err),
				)
			}
		} else {
			// Create new review rule record
			reviewRule = &model.ReviewRule{
				ReviewID:  review.ID,
				RuleID:    rule.ID,
				Status:    model.RuleStatusRunning,
				StartedAt: timePtr(time.Now()),
			}
			if err := r.store.Review().CreateRule(reviewRule); err != nil {
				logger.Warn("Failed to create review rule record",
					zap.String("review_id", review.ID),
					zap.String("rule_id", rule.ID),
					zap.Error(err),
				)
			}
		}

		// Get output language configuration from database
		outputLanguage := ""
		reviewCfg := r.getReviewConfig()
		if reviewCfg != nil {
			if langCfg, err := reviewCfg.GetOutputLanguage(); err == nil {
				outputLanguage = langCfg.PromptInstruction()
			}
		}

		// Build context for rule execution
		buildCtx := prompt.BuildContext{
			RepoPath:       req.RepoPath,
			RepoURL:        req.RepoURL,
			Ref:            req.Ref,
			CommitSHA:      req.CommitSHA,
			PRNumber:       req.PRNumber,
			PRTitle:        req.PRTitle,
			PRDescription:  req.PRDescription,
			BaseCommitSHA:  req.BaseCommitSHA,
			ChangedFiles:   req.ChangedFiles,
			OutputLanguage: outputLanguage,
		}

		// Load base rule if using inheritance
		ruleBuildCtx := buildCtx
		if rule.HistoryCompare != nil && rule.HistoryCompare.Enabled && review.PRURL != "" {
			previousResult, found, err := r.store.Review().FindPreviousReviewResult(review.PRURL, rule.ID, review.ID)
			if err == nil && found {
				ruleBuildCtx.PreviousReviewForComparison = previousResult
				logger.Info("Injecting previous review result for comparison",
					zap.String("review_id", review.ID),
					zap.String("rule_id", rule.ID),
					zap.Int("previous_result_length", len(previousResult)),
				)
			}
		}

		// Execute rule using executor
		result, err := r.executor.ExecuteRule(ctx, &rule, &ruleBuildCtx, reviewRule, i)
		if err != nil {
			logger.Error("Review rule execution failed",
				zap.String("review_id", review.ID),
				zap.String("rule_id", rule.ID),
				zap.Error(err),
			)
			result = prompt.NewReviewResult(rule.ID)
			result.Error = err.Error()
			hasFailedRule = true
		}

		lastResult = result

		// Save complete AI response as JSON to ReviewResult
		if len(result.Data) > 0 {
			reviewResult := &model.ReviewResult{
				ReviewRuleID: reviewRule.ID,
				Data:         model.JSONMap(result.Data),
			}
			if err := r.store.Review().CreateResult(reviewResult); err != nil {
				logger.Warn("Failed to save review result",
					zap.String("review_id", review.ID),
					zap.String("rule_id", rule.ID),
					zap.Error(err),
				)
			}
		}

		// Publish result (skip if rule was already completed before this execution)
		shouldPublish := true
		if wasAlreadyCompleted && reviewRule.Status == model.RuleStatusCompleted {
			logger.Info("Rule already completed, skipping output publish",
				zap.String("rule_id", rule.ID),
				zap.String("review_id", review.ID),
			)
			shouldPublish = false
		}

		if shouldPublish {
			r.publishRuleResult(ctx, result, &rule, review, &buildCtx, prov, prInfo, req.OutputDir)
		}
	}

	// Check if all rules are completed and update review status
	r.UpdateReviewStatusAfterRuleExecution(review)

	if hasFailedRule {
		return lastResult, errors.New("one or more rules failed")
	}
	return lastResult, nil
}

// ExecuteSingleRule executes a single rule with the given context.
// This method can be used by both RunReviewWithTracking and RetryRule.
func (r *Runner) ExecuteSingleRule(ctx context.Context, execCtx *RuleExecutionContext) (*prompt.ReviewResult, error) {
	rule := execCtx.Rule
	reviewRule := execCtx.ReviewRule
	review := execCtx.Review
	buildCtx := execCtx.BuildCtx

	logger.Info("Executing single rule",
		zap.String("review_id", review.ID),
		zap.String("rule_id", rule.ID),
		zap.Int("rule_index", execCtx.RuleIndex),
	)

	// If rule is being retried, delete old ReviewResult records
	if reviewRule.Status == model.RuleStatusFailed || reviewRule.Status == model.RuleStatusRunning {
		// Delete all old ReviewResult records for this rule
		if err := r.store.Review().DeleteReviewResultsByRuleID(reviewRule.ID); err != nil {
			logger.Warn("Failed to delete old review results",
				zap.String("review_id", review.ID),
				zap.Uint("review_rule_id", reviewRule.ID),
				zap.Error(err),
			)
		} else {
			logger.Info("Deleted old review results for retry",
				zap.String("review_id", review.ID),
				zap.Uint("review_rule_id", reviewRule.ID),
			)
		}

		// Delete old ReviewRuleRun records
		if err := r.store.Review().DeleteReviewRuleRunsByRuleID(reviewRule.ID); err != nil {
			logger.Warn("Failed to delete old review rule runs",
				zap.String("review_id", review.ID),
				zap.Uint("review_rule_id", reviewRule.ID),
				zap.Error(err),
			)
		}
	}

	// Execute rule using executor
	result, err := r.executor.ExecuteRule(ctx, rule, buildCtx, reviewRule, execCtx.RuleIndex)
	if err != nil {
		logger.Error("Review rule execution failed",
			zap.String("review_id", review.ID),
			zap.String("rule_id", rule.ID),
			zap.Error(err),
		)
		result = prompt.NewReviewResult(rule.ID)
		result.Error = err.Error()
	}

	// Save complete AI response as JSON to ReviewResult
	if len(result.Data) > 0 {
		reviewResult := &model.ReviewResult{
			ReviewRuleID: reviewRule.ID,
			Data:         model.JSONMap(result.Data),
		}
		if err := r.store.Review().CreateResult(reviewResult); err != nil {
			logger.Warn("Failed to save review result",
				zap.String("review_id", review.ID),
				zap.String("rule_id", rule.ID),
				zap.Error(err),
			)
		}
	}

	// Publish result
	r.publishRuleResult(ctx, result, rule, review, buildCtx, execCtx.Provider, execCtx.PRInfo, execCtx.OutputDir)

	return result, nil
}

// UpdateReviewStatusAfterRuleExecution checks all rules and updates review status accordingly.
// This is called after a single rule execution completes (including retry).
func (r *Runner) UpdateReviewStatusAfterRuleExecution(review *model.Review) {
	// Reload review from database to get the latest status
	// This is important because the passed review object may be stale (e.g., during retry)
	currentReview, err := r.store.Review().GetByID(review.ID)
	if err != nil {
		logger.Warn("Failed to load review for status update",
			zap.String("review_id", review.ID),
			zap.Error(err),
		)
		return
	}

	allRules, err := r.store.Review().GetRulesByReviewID(review.ID)
	if err != nil {
		logger.Warn("Failed to load all rules for status update",
			zap.String("review_id", review.ID),
			zap.Error(err),
		)
		return
	}

	allCompleted := true
	hasFailed := false
	hasRunning := false
	for _, rl := range allRules {
		if rl.Status != model.RuleStatusCompleted {
			allCompleted = false
		}
		if rl.Status == model.RuleStatusFailed {
			hasFailed = true
		}
		if rl.Status == model.RuleStatusRunning {
			hasRunning = true
		}
	}

	// If any rule is still running, don't update review status yet
	if hasRunning {
		logger.Debug("Some rules still running, skipping review status update",
			zap.String("review_id", review.ID),
		)
		return
	}

	if allCompleted {
		// All rules completed, update review status
		r.store.Review().UpdateStatusWithErrorAndCompletedAt(review.ID, model.ReviewStatusCompleted, "")

		// Calculate and update duration
		if currentReview.StartedAt != nil {
			duration := time.Since(*currentReview.StartedAt).Milliseconds()
			r.store.Review().UpdateMetadata(review.ID, map[string]interface{}{
				"duration": duration,
			})
		}

		logger.Info("Review completed after rule execution",
			zap.String("review_id", review.ID),
			zap.Int("total_rules", len(allRules)),
		)
	} else if hasFailed {
		// Some rules failed, mark review as failed
		r.store.Review().UpdateStatusWithErrorAndCompletedAt(review.ID, model.ReviewStatusFailed, "one or more rules failed")
		logger.Info("Review marked as failed after rule execution",
			zap.String("review_id", review.ID),
		)
	}
}

// publishRuleResult publishes the result of a single rule execution.
func (r *Runner) publishRuleResult(ctx context.Context, result *prompt.ReviewResult, rule *dsl.ReviewRuleConfig, review *model.Review, buildCtx *prompt.BuildContext, prov provider.Provider, prInfo *provider.PullRequest, outputDir string) {
	var outputCfg *dsl.OutputConfig
	if rule.Output != nil && len(rule.Output.Channels) > 0 {
		outputCfg = rule.Output
	}

	if outputCfg == nil || len(outputCfg.Channels) == 0 {
		logger.Warn("No output channels configured for rule, skipping publish",
			zap.String("rule_id", rule.ID),
			zap.String("review_id", review.ID),
		)
		return
	}

	publisher, err := output.NewPublisherFromConfig(outputCfg, r.store)
	if err != nil {
		logger.Error("Failed to create publisher from config",
			zap.String("rule_id", rule.ID),
			zap.Error(err),
		)
		return
	}

	// Get metadata config from database
	var metadataConfig *config.OutputMetadataConfig
	reviewCfg := r.getReviewConfig()
	if reviewCfg != nil {
		metadataConfig = &reviewCfg.OutputMetadata
	}

	publishOpts := &output.PublishOptions{
		ReviewID:       review.ID,
		RepoURL:        buildCtx.RepoURL,
		Ref:            buildCtx.Ref,
		PRNumber:       buildCtx.PRNumber,
		PRInfo:         prInfo,
		OutputDir:      outputDir,
		RepoPath:       buildCtx.RepoPath,
		Overwrite:      true,
		Provider:       prov,
		MetadataConfig: metadataConfig,
		AgentName:      result.AgentName,
		ModelName:      result.ModelName,
	}

	if err := publisher.Publish(ctx, result, publishOpts); err != nil {
		logger.Error("Failed to publish results for rule",
			zap.String("review_id", review.ID),
			zap.String("rule_id", rule.ID),
			zap.Error(err),
		)
	}
}

// loadExistingReviewRules loads existing ReviewRule records for a review.
func (r *Runner) loadExistingReviewRules(reviewID string, rules []dsl.ReviewRuleConfig) (map[string]*model.ReviewRule, error) {
	existingRules := make(map[string]*model.ReviewRule)

	dbRules, err := r.store.Review().GetRulesByReviewID(reviewID)
	if err != nil {
		return existingRules, err
	}

	for i := range dbRules {
		existingRules[dbRules[i].RuleID] = &dbRules[i]
	}

	return existingRules, nil
}

// shouldRerunCompletedRule determines if a completed rule should be re-run.
// Currently always returns false, but can be extended for force-rerun scenarios.
func (r *Runner) shouldRerunCompletedRule(rule dsl.ReviewRuleConfig) bool {
	return false
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}

// extractOwnerRepo extracts owner and repository name from a repository URL.
func extractOwnerRepo(repoURL string) (owner, repo string) {
	// Parse various Git URL formats
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// etc.

	// This is a simplified implementation; a more robust one would use regex
	// For now, we return empty strings and let the caller handle it
	// The actual parsing is typically done elsewhere in the codebase

	return "", ""
}
