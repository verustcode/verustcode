// Package executor handles review rule execution.
// It provides the core logic for executing single and multi-run reviews.
package executor

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/engine/utils"
	"github.com/verustcode/verustcode/internal/llm"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Default execution configuration values
const (
	DefaultMaxRetries    = 3
	DefaultRetryDelay    = 5 * time.Second
	MaxRetryDelay        = 5 * time.Minute
	DefaultMultiRunCount = 2
)

// Executor handles the execution of review rules.
// It encapsulates the logic for single-run and multi-run reviews.
type Executor struct {
	cfg            *config.Config
	configProvider config.ConfigProvider
	agents         map[string]base.Agent
	promptBuilder  *prompt.Builder
	store          store.Store
}

// NewExecutor creates a new Executor instance.
func NewExecutor(cfg *config.Config, agents map[string]base.Agent, promptBuilder *prompt.Builder, s store.Store) *Executor {
	return &Executor{
		cfg:            cfg,
		configProvider: config.NewDBConfigProvider(s),
		agents:         agents,
		promptBuilder:  promptBuilder,
		store:          s,
	}
}

// getReviewConfig retrieves review configuration from database with fallback to cached config.
func (e *Executor) getReviewConfig() *config.ReviewConfig {
	if e.configProvider != nil {
		reviewCfg, err := e.configProvider.GetReviewConfig()
		if err == nil && reviewCfg != nil {
			return reviewCfg
		}
	}
	// Fallback to cached config
	if e.cfg != nil {
		return &e.cfg.Review
	}
	return nil
}

// ExecuteRule executes a single review rule with retry logic.
// reviewRule is optional - if provided, will be updated with execution status.
// ruleIndex is the index of the rule in the rules list (0-based).
func (e *Executor) ExecuteRule(ctx context.Context, rule *dsl.ReviewRuleConfig, buildCtx *prompt.BuildContext, reviewRule *model.ReviewRule, ruleIndex int) (*prompt.ReviewResult, error) {
	// Create or update review rule record if reviewRule is provided
	if reviewRule != nil {
		now := time.Now()

		// Save rule config
		ruleConfigJSON, err := utils.RuleConfigToJSONMap(rule)
		if err == nil {
			reviewRule.RuleConfig = ruleConfigJSON
		}

		// Update status to running
		reviewRule.Status = model.RuleStatusRunning
		reviewRule.StartedAt = &now

		// Set multi-run config (automatically enabled when runs >= 2)
		if rule.MultiRun != nil && rule.MultiRun.Runs >= 2 {
			reviewRule.MultiRunEnabled = true
			reviewRule.MultiRunRuns = rule.MultiRun.Runs
		} else {
			reviewRule.MultiRunEnabled = false
			reviewRule.MultiRunRuns = 1
		}

		if err := e.store.Review().UpdateRule(reviewRule); err != nil {
			logger.Warn("Failed to update review rule status",
				zap.String("review_id", reviewRule.ReviewID),
				zap.Error(err),
			)
		}
	}

	// Build prompt spec from DSL
	spec := e.promptBuilder.Build(rule, buildCtx)

	// Render prompt
	renderer := prompt.NewRenderer()
	promptText, err := renderer.Render(spec)
	if err != nil {
		renderErr := errors.Wrap(errors.ErrCodeInternal, "failed to render prompt", err)
		if reviewRule != nil {
			e.UpdateReviewRuleAfterExecution(reviewRule, nil, renderErr, nil)
		}
		return nil, renderErr
	}

	// Add format instructions based on output schema configuration
	formatInstructions := BuildFormatInstructions(rule, buildCtx)
	if formatInstructions != "" {
		promptText = promptText + formatInstructions
	}

	// Save complete prompt to reviewRule if available
	if reviewRule != nil {
		reviewRule.Prompt = promptText
		if err := e.store.Review().UpdateRule(reviewRule); err != nil {
			logger.Warn("Failed to save prompt to review rule",
				zap.String("review_id", reviewRule.ReviewID),
				zap.Error(err),
			)
		}
	}

	// Log complete prompt for debugging (includes format instructions/schema)
	logger.Debug("Rendered prompt",
		zap.String("rule_id", rule.ID),
		zap.String("prompt", promptText),
	)

	// Log prompt info
	reviewID := ""
	if reviewRule != nil {
		reviewID = reviewRule.ReviewID
	}
	logger.Info("Rendered prompt for rule",
		zap.String("review_id", reviewID),
		zap.String("rule_id", rule.ID),
		zap.Int("prompt_length", len(promptText)),
	)

	// Get agent (确认继承逻辑)
	agentName := rule.Agent.GetType()
	if agentName == "" {
		agentErr := errors.New(errors.ErrCodeAgentNotFound,
			"agent is required in review rule. Please specify agent in the rule configuration or rule_base section")
		if reviewRule != nil {
			e.UpdateReviewRuleAfterExecution(reviewRule, nil, agentErr, nil)
		}
		return nil, agentErr
	}

	logger.Info("Using agent for rule",
		zap.String("review_id", reviewID),
		zap.String("rule_id", rule.ID),
		zap.String("agent", agentName),
	)

	agent, ok := e.agents[agentName]
	if !ok {
		agentErr := errors.New(errors.ErrCodeAgentNotFound,
			fmt.Sprintf("agent %s not available. Please ensure the agent is configured in agents section", agentName))
		if reviewRule != nil {
			e.UpdateReviewRuleAfterExecution(reviewRule, nil, agentErr, nil)
		}
		return nil, agentErr
	}

	// Get metadata config for appending to summary (from database)
	var metadataConfig *config.OutputMetadataConfig
	reviewCfg := e.getReviewConfig()
	if reviewCfg != nil {
		metadataConfig = &reviewCfg.OutputMetadata
	}

	// Check if multi-run is enabled (automatically enabled when runs >= 2)
	if rule.MultiRun != nil && rule.MultiRun.Runs >= 2 {
		result, err := e.executeMultiRun(ctx, rule, buildCtx, promptText, agent, reviewRule)
		if reviewRule != nil {
			e.UpdateReviewRuleAfterExecution(reviewRule, result, err, metadataConfig)
		}
		return result, err
	}

	// Single execution path (original logic)
	result, err := e.executeSingleRun(ctx, rule, buildCtx, promptText, agent, reviewRule)
	if reviewRule != nil {
		e.UpdateReviewRuleAfterExecution(reviewRule, result, err, metadataConfig)
	}
	return result, err
}

// executeSingleRun executes a single review run
func (e *Executor) executeSingleRun(ctx context.Context, rule *dsl.ReviewRuleConfig, buildCtx *prompt.BuildContext, promptText string, agent base.Agent, reviewRule *model.ReviewRule) (*prompt.ReviewResult, error) {
	// Get review ID if available
	reviewID := ""
	if reviewRule != nil {
		reviewID = reviewRule.ReviewID
	}

	// Build review request
	req := &base.ReviewRequest{
		RequestID:    idgen.NewRequestID(),
		RuleID:       rule.ID,
		ReviewID:     reviewID,
		RepoPath:     buildCtx.RepoPath,
		RepoURL:      buildCtx.RepoURL,
		Ref:          buildCtx.Ref,
		CommitSHA:    buildCtx.CommitSHA,
		PRNumber:     buildCtx.PRNumber,
		PRTitle:      buildCtx.PRTitle,
		PRBody:       buildCtx.PRDescription,
		ChangedFiles: buildCtx.ChangedFiles,
	}

	// Get retry configuration from database
	singleRunReviewCfg := e.getReviewConfig()
	maxRetries := DefaultMaxRetries
	retryDelay := DefaultRetryDelay
	if singleRunReviewCfg != nil {
		if singleRunReviewCfg.MaxRetries > 0 {
			maxRetries = singleRunReviewCfg.MaxRetries
		}
		if singleRunReviewCfg.RetryDelay > 0 {
			retryDelay = time.Duration(singleRunReviewCfg.RetryDelay) * time.Second
		}
	}

	// Execute agent with the prompt (with retry logic)
	var agentResult *base.ReviewResult
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying review rule execution",
				zap.String("rule_id", rule.ID),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Duration("delay", retryDelay),
			)

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}

			// Exponential backoff (capped at MaxRetryDelay)
			retryDelay = retryDelay * 2
			if retryDelay > MaxRetryDelay {
				retryDelay = MaxRetryDelay
			}
		}

		agentResult, lastErr = agent.ExecuteWithPrompt(ctx, req, promptText)
		if lastErr == nil {
			break
		}

		logger.Warn("Review rule execution failed",
			zap.String("review_id", reviewID),
			zap.String("rule_id", rule.ID),
			zap.Int("attempt", attempt),
			zap.Error(lastErr),
		)

		// Check if error is retryable; stop retry if not
		if !llm.IsRetryable(lastErr) {
			logger.Warn("Non-retryable error, stopping retry",
				zap.String("review_id", reviewID),
				zap.String("rule_id", rule.ID),
				zap.Error(lastErr),
			)
			break
		}
	}

	if lastErr != nil {
		return nil, errors.Wrap(errors.ErrCodeAgentExecution,
			fmt.Sprintf("review rule %s failed after %d retries", rule.ID, maxRetries), lastErr)
	}

	// Convert agent result to prompt.ReviewResult
	// Data contains the complete AI response (JSON Schema mode)
	// Text contains the raw text output (Markdown mode)
	result := prompt.NewReviewResult(rule.ID)
	result.Text = agentResult.Text
	result.AgentName = agentResult.AgentName
	result.ModelName = agentResult.ModelName

	// If agent returned Data directly, use it
	// Otherwise, try to extract JSON from Text (AI returns JSON in markdown code block)
	if len(agentResult.Data) > 0 {
		result.Data = agentResult.Data
	} else if agentResult.Text != "" {
		var parsed map[string]any
		if err := llm.ParseResponseJSON(agentResult.Text, &parsed); err == nil {
			result.Data = parsed
		}
	}

	return result, nil
}

// UpdateReviewRuleAfterExecution updates review rule record after execution.
// Results are stored in ReviewResult.Data, not in ReviewRule.Summary.
func (e *Executor) UpdateReviewRuleAfterExecution(reviewRule *model.ReviewRule, result *prompt.ReviewResult, err error, _ *config.OutputMetadataConfig) {
	now := time.Now()

	if err != nil {
		reviewRule.Status = model.RuleStatusFailed
		reviewRule.ErrorMessage = err.Error()
	} else {
		reviewRule.Status = model.RuleStatusCompleted

		// FindingsCount: try to get from Data if available
		if findings, ok := result.Data["findings"].([]interface{}); ok {
			reviewRule.FindingsCount = len(findings)
		}
	}

	reviewRule.CompletedAt = &now
	if reviewRule.StartedAt != nil {
		reviewRule.Duration = now.Sub(*reviewRule.StartedAt).Milliseconds()
	}

	if dbErr := e.store.Review().UpdateRule(reviewRule); dbErr != nil {
		logger.Warn("Failed to update review rule after execution",
			zap.String("review_id", reviewRule.ReviewID),
			zap.Error(dbErr),
		)
	}
}

// LoadExistingReviewRuleRuns loads existing ReviewRuleRun records for a review rule.
func (e *Executor) LoadExistingReviewRuleRuns(reviewRuleID uint) (map[int]*model.ReviewRuleRun, error) {
	runs, err := e.store.Review().GetRunsByRuleID(reviewRuleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing review rule runs: %w", err)
	}
	runsMap := make(map[int]*model.ReviewRuleRun)
	for i := range runs {
		runsMap[runs[i].RunIndex] = &runs[i]
	}
	return runsMap, nil
}
