// Package executor handles review rule execution.
// This file contains multi-run execution and result merging logic.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/engine/task"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
)

// executeMultiRun executes multiple review runs and merges the results
func (e *Executor) executeMultiRun(ctx context.Context, rule *dsl.ReviewRuleConfig, buildCtx *prompt.BuildContext, promptText string, agent base.Agent, reviewRule *model.ReviewRule) (*prompt.ReviewResult, error) {
	multiRun := rule.MultiRun
	runs := multiRun.Runs
	if runs <= 0 {
		runs = DefaultMultiRunCount
	}

	// Fast path: if runs == 1, execute single run and return directly without merge
	if runs == 1 {
		logger.Info("Executing single run (multi_run enabled but runs=1, skipping merge)",
			zap.String("rule_id", rule.ID),
		)

		// Determine model to use
		modelName := ""
		if len(multiRun.Models) > 0 {
			modelName = multiRun.Models[0]
		}
		if modelName == "" {
			modelName = "default"
		}

		// Build review request
		req := &base.ReviewRequest{
			RequestID:    idgen.NewRequestID(),
			RuleID:       rule.ID,
			Model:        modelName,
			RepoPath:     buildCtx.RepoPath,
			RepoURL:      buildCtx.RepoURL,
			Ref:          buildCtx.Ref,
			CommitSHA:    buildCtx.CommitSHA,
			PRNumber:     buildCtx.PRNumber,
			PRTitle:      buildCtx.PRTitle,
			PRBody:       buildCtx.PRDescription,
			ChangedFiles: buildCtx.ChangedFiles,
		}

		// Execute with retry logic
		maxRetries := e.cfg.Review.MaxRetries
		if maxRetries <= 0 {
			maxRetries = DefaultMaxRetries
		}
		retryDelay := time.Duration(e.cfg.Review.RetryDelay) * time.Second
		if retryDelay <= 0 {
			retryDelay = DefaultRetryDelay
		}

		var agentResult *base.ReviewResult
		var lastErr error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				logger.Info("Retrying single run execution",
					zap.String("rule_id", rule.ID),
					zap.Int("attempt", attempt),
					zap.Duration("delay", retryDelay),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryDelay):
				}

				retryDelay = retryDelay * 2
				if retryDelay > MaxRetryDelay {
					retryDelay = MaxRetryDelay
				}
			}

			agentResult, lastErr = agent.ExecuteWithPrompt(ctx, req, promptText)
			if lastErr == nil {
				break
			}

			logger.Warn("Single run execution failed",
				zap.String("review_id", reviewRule.ReviewID),
				zap.String("rule_id", rule.ID),
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
			)
		}

		if lastErr != nil {
			return nil, errors.Wrap(errors.ErrCodeAgentExecution,
				fmt.Sprintf("single run failed after %d retries", maxRetries), lastErr)
		}

		// Convert agent result to prompt.ReviewResult
		// Data contains the complete AI response (JSON Schema mode)
		// Text contains the raw text output (Markdown mode)
		result := prompt.NewReviewResult(rule.ID)
		result.Data = agentResult.Data
		result.Text = agentResult.Text

		return result, nil
	}

	models := multiRun.Models
	if len(models) == 0 {
		// Use empty string to indicate using agent's default model
		models = make([]string, runs)
	} else if len(models) < runs {
		// Cycle through models if fewer models than runs
		extendedModels := make([]string, runs)
		for i := 0; i < runs; i++ {
			extendedModels[i] = models[i%len(models)]
		}
		models = extendedModels
	}

	logger.Info("Starting multi-run review",
		zap.String("rule_id", rule.ID),
		zap.Int("runs", runs),
		zap.Strings("models", models),
	)

	// Load existing ReviewRuleRun records for recovery (阶段 2)
	existingRunsMap := make(map[int]*model.ReviewRuleRun)
	if reviewRule != nil {
		var err error
		existingRunsMap, err = e.LoadExistingReviewRuleRuns(reviewRule.ID)
		if err != nil {
			logger.Warn("Failed to load existing review rule runs, will create new ones",
				zap.String("review_id", reviewRule.ReviewID),
				zap.Uint("review_rule_id", reviewRule.ID),
				zap.Error(err),
			)
		}
	}

	// Collect all review results
	results := make([]task.RunResult, 0, runs)

	// Execute all runs sequentially
	for i := 0; i < runs; i++ {
		modelName := models[i]
		if modelName == "" {
			modelName = "default"
		}

		// Check if run already exists (阶段 2)
		var ruleRun *model.ReviewRuleRun
		if existingRun, exists := existingRunsMap[i]; exists {
			ruleRun = existingRun
			// If run is completed, skip execution but collect result for merging
			if ruleRun.Status == model.RunStatusCompleted {
				logger.Info("Skipping completed run during recovery",
					zap.Uint("review_rule_id", reviewRule.ID),
					zap.String("rule_id", rule.ID),
					zap.Int("run_index", i),
					zap.String("model", modelName),
					zap.String("run_status", string(ruleRun.Status)),
				)
				// Collect completed run result
				// Note: Data is not available from ReviewRuleRun, it's stored in ReviewResult
				// For recovery scenarios, we skip the completed run and don't include it in merge
				// TODO: Load data from ReviewResult if needed for merge
				results = append(results, task.RunResult{
					Index:    i + 1,
					Model:    modelName,
					Data:     nil, // Data not available during recovery
					Text:     "",
					Duration: time.Duration(ruleRun.Duration) * time.Millisecond,
					Err:      nil,
				})
				continue
			}
			// If run failed, retry it
			if ruleRun.Status == model.RunStatusFailed {
				logger.Info("Retrying failed run",
					zap.String("rule_id", rule.ID),
					zap.Int("run_index", i),
				)
				// Reset status to running
				ruleRun.Status = model.RunStatusRunning
				now := time.Now()
				ruleRun.StartedAt = &now
				e.store.Review().UpdateRun(ruleRun)
			}
		} else {
			// Create new ReviewRuleRun record
			if reviewRule != nil {
				now := time.Now()
				ruleRun = &model.ReviewRuleRun{
					ReviewRuleID: reviewRule.ID,
					RunIndex:     i,
					Model:        modelName,
					Agent:        agent.Name(),
					Status:       model.RunStatusRunning,
					StartedAt:    &now,
				}
				if err := e.store.Review().CreateRun(ruleRun); err != nil {
					logger.Warn("Failed to create review rule run",
						zap.String("review_id", reviewRule.ReviewID),
						zap.Error(err),
					)
				} else {
					// Update review rule current run index
					reviewRule.CurrentRunIndex = i
					e.store.Review().UpdateRule(reviewRule)
				}
			}
		}

		logger.Info("Executing multi-run review",
			zap.String("rule_id", rule.ID),
			zap.Int("run", i+1),
			zap.Int("total_runs", runs),
			zap.String("model", modelName),
		)

		startTime := time.Now()

		// Build review request with model override
		req := &base.ReviewRequest{
			RequestID:    idgen.NewRequestID(),
			RuleID:       rule.ID,
			Model:        modelName, // Set model for this run
			RepoPath:     buildCtx.RepoPath,
			RepoURL:      buildCtx.RepoURL,
			Ref:          buildCtx.Ref,
			CommitSHA:    buildCtx.CommitSHA,
			PRNumber:     buildCtx.PRNumber,
			PRTitle:      buildCtx.PRTitle,
			PRBody:       buildCtx.PRDescription,
			ChangedFiles: buildCtx.ChangedFiles,
		}

		// Execute with retry logic
		maxRetries := e.cfg.Review.MaxRetries
		if maxRetries <= 0 {
			maxRetries = DefaultMaxRetries
		}
		retryDelay := time.Duration(e.cfg.Review.RetryDelay) * time.Second
		if retryDelay <= 0 {
			retryDelay = DefaultRetryDelay
		}

		var agentResult *base.ReviewResult
		var lastErr error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				logger.Info("Retrying multi-run review execution",
					zap.String("rule_id", rule.ID),
					zap.Int("run", i+1),
					zap.Int("attempt", attempt),
					zap.Duration("delay", retryDelay),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryDelay):
				}

				retryDelay = retryDelay * 2
				if retryDelay > MaxRetryDelay {
					retryDelay = MaxRetryDelay
				}
			}

			agentResult, lastErr = agent.ExecuteWithPrompt(ctx, req, promptText)
			if lastErr == nil {
				break
			}

			logger.Warn("Multi-run review execution failed",
				zap.String("review_id", reviewRule.ReviewID),
				zap.String("rule_id", rule.ID),
				zap.Int("run", i+1),
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
			)
		}

		duration := time.Since(startTime)

		// Update ReviewRuleRun record
		if ruleRun != nil {
			now := time.Now()
			ruleRun.CompletedAt = &now
			ruleRun.Duration = duration.Milliseconds()

			if lastErr != nil {
				ruleRun.Status = model.RunStatusFailed
				ruleRun.ErrorMessage = lastErr.Error()
			} else {
				ruleRun.Status = model.RunStatusCompleted
				// FindingsCount: try to get from Data if available
				if findings, ok := agentResult.Data["findings"].([]interface{}); ok {
					ruleRun.FindingsCount = len(findings)
				}
			}

			if err := e.store.Review().UpdateRun(ruleRun); err != nil {
				logger.Warn("Failed to update review rule run",
					zap.String("review_id", reviewRule.ReviewID),
					zap.Error(err),
				)
			}
		}

		if lastErr != nil {
			logger.Error("Multi-run review execution failed after retries",
				zap.String("review_id", reviewRule.ReviewID),
				zap.String("rule_id", rule.ID),
				zap.Int("run", i+1),
				zap.Error(lastErr),
			)
			// Continue with other runs even if one fails
			results = append(results, task.RunResult{
				Index:    i + 1,
				Model:    modelName,
				Data:     nil,
				Text:     "",
				Duration: duration,
				Err:      lastErr,
			})
			continue
		}

		logger.Info("Multi-run review completed",
			zap.String("rule_id", rule.ID),
			zap.Int("run", i+1),
			zap.String("model", modelName),
			zap.Duration("duration", duration),
			zap.Int("text_length", len(agentResult.Text)),
		)

		results = append(results, task.RunResult{
			Index:    i + 1,
			Model:    modelName,
			Data:     agentResult.Data,
			Text:     agentResult.Text,
			Duration: duration,
			Err:      nil,
		})
	}

	// Check if we have any successful results
	successfulResults := make([]task.RunResult, 0)
	for _, r := range results {
		if r.Err == nil && (len(r.Data) > 0 || r.Text != "") {
			successfulResults = append(successfulResults, r)
		}
	}

	if len(successfulResults) == 0 {
		return nil, errors.New(errors.ErrCodeAgentExecution,
			fmt.Sprintf("all %d multi-run reviews failed for rule %s", runs, rule.ID))
	}

	logger.Info("Multi-run reviews completed",
		zap.String("rule_id", rule.ID),
		zap.Int("total_runs", runs),
		zap.Int("successful_runs", len(successfulResults)),
		zap.Int("failed_runs", len(results)-len(successfulResults)),
	)

	// Merge results using LLM
	mergedText, err := e.mergeReviewResults(ctx, rule, agent, successfulResults)
	if err != nil {
		logger.Error("Failed to merge review results",
			zap.String("review_id", reviewRule.ReviewID),
			zap.String("rule_id", rule.ID),
			zap.Error(err),
		)
		// Fallback: use the first successful result
		mergedText = successfulResults[0].Text
	}

	// Create final result with merged text
	result := prompt.NewReviewResult(rule.ID)
	// Store merged text as the complete AI response
	result.Text = mergedText
	// Try to parse merged text as JSON for Data
	if mergedText != "" {
		var parsed map[string]interface{}
		if parseErr := json.Unmarshal([]byte(mergedText), &parsed); parseErr == nil {
			result.Data = parsed
		}
	}

	return result, nil
}

// mergeReviewResults merges multiple review results using LLM
func (e *Executor) mergeReviewResults(ctx context.Context, rule *dsl.ReviewRuleConfig, agent base.Agent, results []task.RunResult) (string, error) {
	if len(results) == 0 {
		return "", errors.New(errors.ErrCodeInternal, "no results to merge")
	}

	if len(results) == 1 {
		// No need to merge if only one result
		return results[0].Text, nil
	}

	// Build merge prompt using XML format to avoid conflicts with markdown review results
	var mergePrompt strings.Builder
	mergePrompt.WriteString("You are tasked with merging multiple code review results. Please analyze the following review results and produce a merged, deduplicated review.\n\n")

	mergePrompt.WriteString("<review_results>\n")
	for _, r := range results {
		mergePrompt.WriteString(fmt.Sprintf("<review_result index=\"%d\" model=\"%s\">\n", r.Index, r.Model))
		// Use Text for merge prompt, or convert Data to JSON string
		content := r.Text
		if content == "" && len(r.Data) > 0 {
			if jsonBytes, err := json.Marshal(r.Data); err == nil {
				content = string(jsonBytes)
			}
		}
		mergePrompt.WriteString(content)
		mergePrompt.WriteString("\n</review_result>\n\n")
	}
	mergePrompt.WriteString("</review_results>\n\n")

	mergePrompt.WriteString("<instructions>\n")
	mergePrompt.WriteString("Please perform the following tasks:\n")
	mergePrompt.WriteString("1. Merge all review results\n")
	mergePrompt.WriteString("2. Remove duplicate or similar issues\n")
	mergePrompt.WriteString("3. Maintain the original markdown format in your output\n")
	mergePrompt.WriteString("4. Ensure the output is friendly and well-structured\n")
	mergePrompt.WriteString("5. Preserve all important issues and suggestions\n")
	mergePrompt.WriteString("</instructions>\n\n")

	mergePrompt.WriteString("<output_format>\n")
	mergePrompt.WriteString("Output the merged review result in markdown format below:\n")
	mergePrompt.WriteString("</output_format>\n")

	// Determine merge model
	mergeModel := rule.MultiRun.MergeModel
	if mergeModel == "" {
		// Use agent's default model (empty string means use default)
		mergeModel = ""
	}

	logger.Info("Merging review results with LLM",
		zap.String("rule_id", rule.ID),
		zap.Int("result_count", len(results)),
		zap.String("merge_model", mergeModel),
	)

	// Log merge prompt for debugging (print separately for clarity)
	mergePromptText := mergePrompt.String()
	logger.Info("Merge review prompt",
		zap.String("rule_id", rule.ID),
		zap.Int("prompt_length", len(mergePromptText)),
	)
	// Log prompt content for debugging
	logger.Debug("Merge review prompt content",
		zap.String("rule_id", rule.ID),
		zap.String("prompt", mergePromptText),
	)

	// Get LLM client from agent
	// We need to access the underlying LLM client
	// For now, we'll use the agent to execute the merge prompt
	mergeReq := &base.ReviewRequest{
		RequestID: idgen.NewRequestID(),
		RuleID:    rule.ID + "-merge",
		Model:     mergeModel,
		RepoPath:  "", // Not needed for merging
	}

	startTime := time.Now()
	mergeResult, err := agent.ExecuteWithPrompt(ctx, mergeReq, mergePromptText)
	if err != nil {
		return "", errors.Wrap(errors.ErrCodeAgentExecution, "failed to merge review results", err)
	}

	duration := time.Since(startTime)
	logger.Info("Review results merged successfully",
		zap.String("rule_id", rule.ID),
		zap.Duration("duration", duration),
		zap.Int("merged_text_length", len(mergeResult.Text)),
	)

	return mergeResult.Text, nil
}
