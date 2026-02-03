// Package engine provides the core review engine for VerustCode.
// It orchestrates the code review process including repository cloning,
// DSL-driven multi-reviewer execution, result aggregation, and output.
package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/engine/agent"
	"github.com/verustcode/verustcode/internal/engine/executor"
	providermgr "github.com/verustcode/verustcode/internal/engine/provider"
	"github.com/verustcode/verustcode/internal/engine/recovery"
	"github.com/verustcode/verustcode/internal/engine/retry"
	"github.com/verustcode/verustcode/internal/engine/runner"
	"github.com/verustcode/verustcode/internal/engine/task"
	"github.com/verustcode/verustcode/internal/engine/utils"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/git/workspace"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/notification"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
	"github.com/verustcode/verustcode/pkg/telemetry"
)

// Re-export types from task package for API compatibility
type (
	// Task represents a review task
	Task = task.Task

	// ReviewRequest represents a request to run a DSL-driven review
	ReviewRequest = task.ReviewRequest

	// PRInfo contains additional PR information from webhook
	PRInfo = task.PRInfo
)

// Engine manages code review tasks and orchestrates all sub-modules.
// It serves as the main orchestration layer, delegating specific responsibilities
// to specialized modules: ProviderManager, AgentManager, RecoveryService,
// ReviewRunner, and RetryHandler.
type Engine struct {
	cfg            *config.Config
	store          store.Store
	configProvider config.ConfigProvider

	// Sub-modules
	providerMgr  *providermgr.Manager
	agentMgr     *agent.Manager
	recovery     *recovery.Service
	runner       *runner.Runner
	retryHandler *retry.Handler

	// DSL components
	dslLoader     *dsl.Loader
	promptBuilder *prompt.Builder

	// Executor for review rule execution
	executor *executor.Executor

	// Memory-based task queue management
	repoQueue  *RepoTaskQueue // Per-repo task queue for serialization
	dispatcher *Dispatcher    // Event-driven task dispatcher
	workers    int

	// Callbacks (for server mode)
	onComplete func(task *Task, result *prompt.ReviewResult)
	onError    func(task *Task, err error)

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEngine creates a new review engine and initializes all sub-modules.
func NewEngine(cfg *config.Config, s store.Store) (*Engine, error) {

	ctx, cancel := context.WithCancel(context.Background())

	// Create the repo task queue
	repoQueue := NewRepoTaskQueue(ctx)

	// Create config provider for real-time database access
	configProvider := config.NewDBConfigProvider(s)

	e := &Engine{
		cfg:            cfg,
		store:          s,
		configProvider: configProvider,
		dslLoader:      dsl.NewLoader(),
		promptBuilder:  prompt.NewBuilder(),
		repoQueue:      repoQueue,
		workers:        cfg.Review.MaxConcurrent,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize provider manager - now uses store for real-time DB access
	e.providerMgr = providermgr.NewManager(s)
	if err := e.providerMgr.Initialize(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Initialize agent manager

	e.agentMgr = agent.NewManager(cfg, s)
	if err := e.agentMgr.Initialize(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize agents: %w", err)
	}

	// Initialize executor with agents
	e.executor = executor.NewExecutor(cfg, e.agentMgr.All(), e.promptBuilder, s)

	// Initialize runner
	e.runner = runner.NewRunner(cfg, s, e.executor, e.promptBuilder)

	// Initialize recovery service
	e.recovery = recovery.NewService(cfg, s, e.providerMgr, repoQueue)

	// Initialize retry handler

	e.retryHandler = retry.NewHandler(
		cfg,
		s,
		e.providerMgr,
		repoQueue,
		e.recovery,
		e.runner,
		ctx,
	)

	return e, nil
}

// runReviewWithTracking executes a review with database tracking (for server mode).
// Delegates to ReviewRunner.
func (e *Engine) runReviewWithTracking(ctx context.Context, req *ReviewRequest, review *model.Review) (*prompt.ReviewResult, error) {
	// Determine provider for comment channel
	var prov provider.Provider
	if req.RepoURL != "" {
		providerName := e.detectProviderFromURL(req.RepoURL)
		if providerName != "" {
			prov, _ = e.GetProvider(providerName)
		}
	}

	// Convert task.ReviewRequest to runner.ReviewRequest
	runnerReq := &runner.ReviewRequest{
		RepoPath:          req.RepoPath,
		RepoURL:           req.RepoURL,
		Ref:               req.Ref,
		CommitSHA:         req.CommitSHA,
		PRNumber:          req.PRNumber,
		PRTitle:           req.PRTitle,
		PRDescription:     req.PRDescription,
		BaseCommitSHA:     req.BaseCommitSHA,
		ChangedFiles:      req.ChangedFiles,
		ReviewRulesConfig: req.ReviewRulesConfig,
		OutputDir:         req.OutputDir,
	}

	return e.runner.RunReviewWithTracking(ctx, runnerReq, review, prov)
}

// executeSingleRule executes a single rule with the given context.
// Delegates to ReviewRunner.
func (e *Engine) executeSingleRule(ctx context.Context, execCtx *runner.RuleExecutionContext) (*prompt.ReviewResult, error) {
	return e.runner.ExecuteSingleRule(ctx, execCtx)
}

// updateReviewStatusAfterRuleExecution checks all rules and updates review status accordingly.
// Delegates to ReviewRunner.
func (e *Engine) updateReviewStatusAfterRuleExecution(review *model.Review) {
	e.runner.UpdateReviewStatusAfterRuleExecution(review)
}

// LoadDSLConfig loads a DSL configuration file
func (e *Engine) LoadDSLConfig(path string) (*dsl.ReviewRulesConfig, error) {
	return e.dslLoader.Load(path)
}

// loadDSLConfigForRepo loads review configuration for a specific repository
// Priority: database configured review file > default.yaml
// Note: This is used for pre-loading before clone. Use loadDSLConfigWithPriority after clone.
func (e *Engine) loadDSLConfigForRepo(repoURL string) (*dsl.ReviewRulesConfig, error) {
	// Query database for repository-specific review config using repo_url
	repoConfig, err := e.store.RepositoryConfig().GetByRepoURL(repoURL)

	if err == nil && repoConfig.ReviewFile != "" {
		// Found repository-specific config, load the configured review file
		logger.Info("Using repository-specific review config",
			zap.String("repo_url", repoURL),
			zap.String("review_file", repoConfig.ReviewFile),
		)

		reviewFilePath := filepath.Join(config.ReviewsDir, repoConfig.ReviewFile)
		cfg, loadErr := e.dslLoader.Load(reviewFilePath)
		if loadErr != nil {
			logger.Warn("Failed to load configured review file, falling back to default",
				zap.String("review_file", repoConfig.ReviewFile),
				zap.Error(loadErr),
			)
			// Fall through to load default
		} else {
			return cfg, nil
		}
	} else if err != nil && err != gorm.ErrRecordNotFound {
		logger.Warn("Failed to query repository review config",
			zap.String("repo_url", repoURL),
			zap.Error(err),
		)
	}

	// Load default review configuration
	logger.Debug("Using default review config",
		zap.String("repo_url", repoURL),
	)
	return e.dslLoader.LoadDefaultReviewConfig()
}

// loadDSLConfigWithPriority loads review configuration with priority order:
// 1. .verust-review.yaml at repository root (highest priority)
// 2. Database configured review file for this repository
// 3. config/reviews/default.yaml (fallback)
// This method should be called after repository is cloned.
func (e *Engine) loadDSLConfigWithPriority(repoPath, repoURL string) (*dsl.ReviewRulesConfig, error) {
	// Priority 1: Check for .verust-review.yaml at repository root
	if repoPath != "" {
		rootConfig, err := e.dslLoader.LoadFromRepoRoot(repoPath)
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

	// Priority 2 & 3: Database configured review file > default.yaml
	return e.loadDSLConfigForRepo(repoURL)
}

// Start starts the engine workers (for server mode).
func (e *Engine) Start() {
	logger.Info("Starting review engine", zap.Int("workers", e.workers))

	// Create dispatcher with process function
	dispatcherConfig := &DispatcherConfig{
		MaxWorkers: e.workers,
		QueueSize:  e.workers * 10,
	}
	e.dispatcher = NewDispatcher(e.ctx, e.repoQueue, dispatcherConfig, e.processTask)

	// Start the dispatcher (starts workers)
	e.dispatcher.Start()

	// Recover pending reviews to memory queue
	// This must be done after dispatcher starts to ensure tasks are processed
	e.recovery.RecoverToQueue(e.ctx)

	logger.Info("Review engine started",
		zap.Int("workers", e.workers),
	)
}

// Stop stops the engine gracefully
func (e *Engine) Stop() {

	logger.Info("Stopping review engine")

	// Stop the dispatcher (this stops workers and waits for completion)
	if e.dispatcher != nil {
		e.dispatcher.Stop()
	}

	// Stop the queue
	if e.repoQueue != nil {
		e.repoQueue.Stop()
	}

	// Cancel engine context
	e.cancel()

	logger.Info("Review engine stopped")

}

// SetCallbacks sets the completion and error callbacks (for server mode)
func (e *Engine) SetCallbacks(onComplete func(*Task, *prompt.ReviewResult), onError func(*Task, error)) {
	e.onComplete = onComplete
	e.onError = onError
}

// Submit submits a review task (for server mode)
// prInfo is optional and contains PR information from webhook payload
// outputDir is optional and specifies the output directory for file channels
func (e *Engine) Submit(review *model.Review, prInfo *PRInfo, outputDir ...string) (*Task, error) {
	// Detect provider from RepoURL
	providerName := e.detectProviderFromURL(review.RepoURL)
	if providerName == "" {
		return nil, errors.New(errors.ErrCodeGitNotFound, "could not detect provider from repository URL")
	}

	// Validate provider (using thread-safe access)
	prov, ok := e.GetProvider(providerName)
	if !ok {
		return nil, errors.New(errors.ErrCodeGitNotFound, fmt.Sprintf("provider %s not configured", providerName))
	}

	// Extract owner and repo from URL for config lookup
	owner, repoName, err := prov.ParseRepoPath(review.RepoURL)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeValidation, "failed to parse repository URL", err)
	}

	// Load DSL config based on repository configuration
	// Note: This is a preliminary load. The actual config will be reloaded in processTask
	// after cloning, where we can check for .verust-review.yaml in the repo root.
	dslConfig, err := e.loadDSLConfigForRepo(review.RepoURL)
	if err != nil {
		return nil, err
	}

	// Update review source info
	updates := map[string]interface{}{
		"source": review.Source,
	}
	if review.Source == "" {
		updates["source"] = "webhook"
	}
	// Build PR URL if PR number is provided but PRURL is not set
	if review.PRNumber > 0 && review.PRURL == "" {
		prURL := utils.BuildPRURL(review.RepoURL, providerName, review.PRNumber)
		if prURL != "" {
			updates["pr_url"] = prURL
			review.PRURL = prURL
		}
	}
	if err := e.store.Review().UpdateMetadata(review.ID, updates); err != nil {
		logger.Error("Failed to update review metadata", zap.Error(err))
		return nil, errors.Wrap(errors.ErrCodeDBQuery, "failed to update review", err)
	}

	// Pre-create all ReviewRule records with pending status
	// This allows frontend to display all rules immediately, showing pending status for rules not yet executed
	for i, rule := range dslConfig.Rules {
		ruleConfigJSON, _ := utils.RuleConfigToJSONMap(&rule)
		reviewRule := &model.ReviewRule{
			ReviewID:   review.ID,
			RuleIndex:  i,
			RuleID:     rule.ID,
			RuleConfig: ruleConfigJSON,
			Status:     model.RuleStatusPending,
		}

		// Set multi-run config (automatically enabled when runs >= 2)
		if rule.MultiRun != nil && rule.MultiRun.Runs >= 2 {
			reviewRule.MultiRunEnabled = true
			reviewRule.MultiRunRuns = rule.MultiRun.Runs
		} else {
			reviewRule.MultiRunEnabled = false
			reviewRule.MultiRunRuns = 1
		}

		if err := e.store.Review().CreateRule(reviewRule); err != nil {
			logger.Warn("Failed to pre-create review rule",
				zap.String("review_id", review.ID),
				zap.String("rule_id", rule.ID),
				zap.Int("rule_index", i),
				zap.Error(err),
			)
		}
	}

	logger.Info("Pre-created all review rules",
		zap.String("review_id", review.ID),
		zap.Int("rule_count", len(dslConfig.Rules)),
	)

	// Create task (task is identified by Review.ID, no separate task ID)
	task := &Task{
		Review:            review,
		ProviderName:      providerName,
		CreatedAt:         time.Now(),
		ReviewRulesConfig: dslConfig,
		Request: &base.ReviewRequest{
			RequestID: idgen.NewRequestID(),
			RepoURL:   review.RepoURL,
			Owner:     owner,
			RepoName:  repoName,
			Ref:       review.Ref,
			CommitSHA: review.CommitSHA,
			PRNumber:  review.PRNumber,
			// Include PR information if available from webhook
			PRTitle:      "",
			PRBody:       "",
			ChangedFiles: []string{},
		},
	}

	// Populate PR information if provided
	if prInfo != nil {
		task.Request.PRTitle = prInfo.Title
		task.Request.PRBody = prInfo.Description
		task.BaseCommitSHA = prInfo.BaseSHA
		if prInfo.ChangedFiles != nil {
			task.Request.ChangedFiles = prInfo.ChangedFiles
		}
	}

	// Set output directory if provided
	if len(outputDir) > 0 && outputDir[0] != "" {
		task.OutputDir = outputDir[0]
	}

	// Submit to memory queue
	// The queue handles repo-level serialization automatically
	if !e.repoQueue.Enqueue(task) {
		// Task already exists in queue (duplicate submission)
		logger.Warn("Task already in queue (duplicate submission)",
			zap.String("review_id", review.ID),
			zap.String("repo_url", review.RepoURL),
		)
		// Return success since the task is already queued
		return task, nil
	}

	logger.Info("Task submitted to memory queue",
		zap.String("review_id", review.ID),
		zap.String("repo_url", review.RepoURL),
		zap.String("ref", review.Ref),
		zap.Int("queue_pending", e.repoQueue.GetPendingCount()),
	)

	return task, nil
}

// processTask processes a single review task (for server mode)
func (e *Engine) processTask(task *Task) {
	ctx, span := telemetry.StartSpan(e.ctx, "engine.processTask",
		trace.WithAttributes(
			attribute.String("review.id", task.Review.ID),
			attribute.String("repo.url", task.Review.RepoURL),
			attribute.String("repo.ref", task.Request.Ref),
			attribute.String("provider", task.ProviderName),
		),
	)
	defer span.End()

	metrics := telemetry.GetMetrics()
	metrics.RecordReviewStarted(ctx, "", task.ProviderName)
	startTime := time.Now()

	logger.Info("Processing task",
		zap.String("review_id", task.Review.ID),
		zap.String("repo_url", task.Review.RepoURL),
	)

	// Check review status to ensure it's not being processed by another worker (阶段 7: 并发安全机制)
	currentReview, err := e.store.Review().GetByID(task.Review.ID)
	if err != nil {
		logger.Error("Failed to load review for processing",
			zap.String("review_id", task.Review.ID),
			zap.Error(err),
		)
		return
	}

	// If review status is not running or pending, it's already been processed
	if currentReview.Status != model.ReviewStatusRunning && currentReview.Status != model.ReviewStatusPending {
		logger.Info("Review already processed, skipping",
			zap.String("review_id", task.Review.ID),
			zap.String("status", string(currentReview.Status)),
		)
		return
	}

	// Note: Repo-level serialization is now handled by RepoTaskQueue
	// The queue ensures only one task per repo is dispatched at a time

	// Update status to running atomically (only one worker can succeed)
	// Also set started_at timestamp
	now := time.Now()
	updated, err := e.store.Review().UpdateStatusToRunningIfPending(currentReview.ID, now)
	if err != nil {
		logger.Error("Failed to update review status",
			zap.String("review_id", task.Review.ID),
			zap.Error(err),
		)
		return
	}

	if !updated {
		// Status already updated by another worker, skip processing
		logger.Info("Review status already updated by another worker, skipping",
			zap.String("review_id", task.Review.ID),
		)
		return
	}

	// Update task.Review with latest state
	task.Review = currentReview
	task.Review.Status = model.ReviewStatusRunning
	task.Review.StartedAt = &now

	// Get provider and its config for authentication (using thread-safe access)
	prov, _ := e.GetProvider(task.ProviderName)
	provConfig, _ := e.GetProviderConfig(task.ProviderName)

	var repoPath string

	// Check if this is a PR review - use PR cloning logic (core competitive advantage)
	if task.Request.PRNumber > 0 {
		logger.Info("Processing PR review, using PR cloning logic",
			zap.String("review_id", task.Review.ID),
			zap.String("repo_url", task.Review.RepoURL),
			zap.Int("pr_number", task.Request.PRNumber),
			zap.String("commit_sha", task.Request.CommitSHA),
		)

		// Build PR repository request with authentication from provider config
		prRequest := &workspace.PRRepositoryRequest{
			Provider:  prov,
			Owner:     task.Request.Owner,
			Repo:      task.Request.RepoName,
			PRNumber:  task.Request.PRNumber,
			HeadSHA:   task.Request.CommitSHA,
			Workspace: e.GetWorkspace(),
		}

		// Pass authentication information from provider config
		if provConfig != nil {
			prRequest.Token = provConfig.Token
			prRequest.InsecureSkipVerify = provConfig.InsecureSkipVerify
			logger.Debug("Using authentication from provider config",
				zap.String("review_id", task.Review.ID),
				zap.String("provider", task.ProviderName),
				zap.String("token", workspace.MaskToken(provConfig.Token)),
				zap.Bool("insecure_skip_verify", provConfig.InsecureSkipVerify),
			)
		} else {
			logger.Warn("No provider config found, proceeding without authentication",
				zap.String("review_id", task.Review.ID),
				zap.String("provider", task.ProviderName),
			)
		}

		// Use workspace.PRRepositoryManager.EnsurePRRepository - same as CLI flow
		// This uses provider.ClonePR with refs/pull/<pr>/head, supporting fork PRs
		prManager := workspace.NewPRRepositoryManager()
		repoPath, err = prManager.EnsurePRRepository(ctx, prRequest)
		if err != nil {
			logger.Error("Failed to ensure PR repository",
				zap.String("review_id", task.Review.ID),
				zap.String("owner", task.Request.Owner),
				zap.String("repo", task.Request.RepoName),
				zap.Int("pr_number", task.Request.PRNumber),
				zap.Error(err),
			)
			telemetry.SetSpanError(span, err)
			metrics.RecordReviewCompleted(ctx, "failed", time.Since(startTime).Seconds())
			e.handleError(task, errors.Wrap(errors.ErrCodeGitClone, "failed to clone PR repository", err))
			return
		}

		logger.Info("PR repository ensured successfully",
			zap.String("review_id", task.Review.ID),
			zap.String("repo_path", repoPath),
			zap.Int("pr_number", task.Request.PRNumber),
		)
	} else {
		// Non-PR scenario (e.g., push event) - use regular branch clone
		logger.Info("Processing non-PR review, using branch clone",
			zap.String("review_id", task.Review.ID),
			zap.String("repo_url", task.Review.RepoURL),
			zap.String("ref", task.Request.Ref),
		)

		// Create workspace directory
		workDir := filepath.Join(e.GetWorkspace(), task.Review.ID)
		if err := os.MkdirAll(workDir, 0755); err != nil {
			telemetry.SetSpanError(span, err)
			metrics.RecordReviewCompleted(ctx, "failed", time.Since(startTime).Seconds())
			e.handleError(task, errors.Wrap(errors.ErrCodeInternal, "failed to create workspace", err))
			return
		}
		defer utils.CleanupWorkspace(workDir)

		// Clone repository branch
		cloneOpts := &provider.CloneOptions{
			Branch: task.Request.Ref,
			Depth:  1,
		}
		repoPath = filepath.Join(workDir, "repo")
		if err := prov.Clone(ctx, task.Request.Owner, task.Request.RepoName, repoPath, cloneOpts); err != nil {
			telemetry.SetSpanError(span, err)
			metrics.RecordReviewCompleted(ctx, "failed", time.Since(startTime).Seconds())
			e.handleError(task, errors.Wrap(errors.ErrCodeGitClone, "failed to clone repository", err))
			return
		}
	}

	// Update review with repo path
	if err := e.store.Review().UpdateRepoPath(task.Review.ID, repoPath); err != nil {
		logger.Warn("Failed to update review with repo path", zap.Error(err))
	}

	// Load review configuration with priority:
	// 1. .verust-review.yaml at repository root (highest priority)
	// 2. Database configured review file for this repository
	// 3. config/reviews/default.yaml (fallback)
	dslConfig, err := e.loadDSLConfigWithPriority(repoPath, task.Review.RepoURL)
	if err != nil {
		logger.Error("Failed to load review configuration",
			zap.String("review_id", task.Review.ID),
			zap.String("repo_path", repoPath),
			zap.Error(err),
		)
		metrics.RecordReviewCompleted(ctx, "failed", time.Since(startTime).Seconds())
		e.handleError(task, errors.Wrap(errors.ErrCodeConfigInvalid, "failed to load review configuration", err))
		return
	}
	task.ReviewRulesConfig = dslConfig
	logger.Info("Loaded review configuration",
		zap.String("review_id", task.Review.ID),
		zap.Int("rules", len(dslConfig.Rules)),
	)

	// Build review request with complete PR information
	// Use PR information from webhook payload if available, otherwise from Task.Request
	req := &ReviewRequest{
		RepoPath:          repoPath,
		RepoURL:           task.Request.RepoURL,
		Ref:               task.Request.Ref,
		CommitSHA:         task.Request.CommitSHA,
		PRNumber:          task.Request.PRNumber,
		PRTitle:           task.Request.PRTitle,
		PRDescription:     task.Request.PRBody,
		BaseCommitSHA:     task.BaseCommitSHA,
		ChangedFiles:      task.Request.ChangedFiles,
		ReviewRulesConfig: task.ReviewRulesConfig,
		OutputDir:         task.OutputDir,
	}

	// Track PR author for later update
	var prAuthor string

	// If PR information is available but BaseCommitSHA is missing, try to get it from provider
	// This handles cases where webhook didn't include base_sha
	if task.Request.PRNumber > 0 && req.BaseCommitSHA == "" {
		logger.Info("BaseCommitSHA not available, fetching PR details from provider",
			zap.String("review_id", task.Review.ID),
			zap.String("owner", task.Request.Owner),
			zap.String("repo", task.Request.RepoName),
			zap.Int("pr_number", task.Request.PRNumber),
		)

		pr, err := prov.GetPullRequest(ctx, task.Request.Owner, task.Request.RepoName, task.Request.PRNumber)
		if err != nil {
			logger.Warn("Failed to get PR details from provider, continuing without base_sha",
				zap.String("review_id", task.Review.ID),
				zap.Error(err),
			)
		} else {
			req.BaseCommitSHA = pr.BaseSHA
			prAuthor = pr.Author
			// Also update PR title and description if not already set
			if req.PRTitle == "" {
				req.PRTitle = pr.Title
			}
			if req.PRDescription == "" {
				req.PRDescription = pr.Description
			}
			logger.Info("Retrieved PR details from provider",
				zap.String("review_id", task.Review.ID),
				zap.String("base_sha", req.BaseCommitSHA),
				zap.String("author", prAuthor),
			)
		}
	} else if task.Request.PRNumber > 0 {
		// BaseCommitSHA is available, but we still need to fetch PR author
		pr, err := prov.GetPullRequest(ctx, task.Request.Owner, task.Request.RepoName, task.Request.PRNumber)
		if err != nil {
			logger.Warn("Failed to get PR author from provider",
				zap.String("review_id", task.Review.ID),
				zap.Error(err),
			)
		} else {
			prAuthor = pr.Author
		}
	}

	// Get branch metadata and diff statistics from git
	var branchCreatedAt *time.Time
	var linesAdded, linesDeleted, filesChanged, commitCount int

	if req.BaseCommitSHA != "" && req.CommitSHA != "" {
		// Get branch creation time (first commit in the range)
		if timestamp := utils.GetBranchCreatedAt(ctx, repoPath, req.BaseCommitSHA, req.CommitSHA); timestamp != nil {
			t := time.Unix(*timestamp, 0)
			branchCreatedAt = &t
		}

		// Get diff statistics
		if diffStats := utils.GetDiffStats(ctx, repoPath, req.BaseCommitSHA, req.CommitSHA); diffStats != nil {
			linesAdded = diffStats.LinesAdded
			linesDeleted = diffStats.LinesDeleted
			filesChanged = diffStats.FilesChanged
		}

		// Get commit count in the range
		commits := utils.GetCommitsInRange(ctx, repoPath, req.BaseCommitSHA, req.CommitSHA)
		commitCount = len(commits)
	}

	// Update review status to running and save metadata
	now = time.Now()
	updateFields := map[string]interface{}{
		"status":        model.ReviewStatusRunning,
		"started_at":    now,
		"lines_added":   linesAdded,
		"lines_deleted": linesDeleted,
		"files_changed": filesChanged,
		"commit_count":  commitCount,
	}
	if prAuthor != "" {
		updateFields["author"] = prAuthor
	}
	if branchCreatedAt != nil {
		updateFields["branch_created_at"] = branchCreatedAt
	}

	if err := e.store.Review().UpdateMetadata(task.Review.ID, updateFields); err != nil {
		logger.Warn("Failed to update review status and metadata", zap.Error(err))
	}

	// Execute review with tracking
	result, err := e.runReviewWithTracking(ctx, req, task.Review)
	if err != nil {
		telemetry.SetSpanError(span, err)
		metrics.RecordReviewCompleted(ctx, "failed", time.Since(startTime).Seconds())

		// Update review status to failed
		e.store.Review().UpdateStatusWithErrorAndCompletedAt(task.Review.ID, model.ReviewStatusFailed, err.Error())

		e.handleError(task, err)
		return
	}

	// Update review metadata (duration, completed_at)
	// Note: status is already set by runner.UpdateReviewStatusAfterRuleExecution
	completedAt := time.Now()
	duration := completedAt.Sub(now).Milliseconds()
	e.store.Review().UpdateMetadata(task.Review.ID, map[string]interface{}{
		"completed_at":  completedAt,
		"duration":      duration,
		"error_message": "", // Clear previous error message on successful completion
	})

	metrics.RecordReviewCompleted(ctx, "completed", time.Since(startTime).Seconds())
	telemetry.SetSpanOK(span)

	// Send notification for review completion
	extra := map[string]interface{}{
		"ref":         task.Review.Ref,
		"commit_sha":  task.Review.CommitSHA,
		"source":      task.Review.Source,
		"duration_ms": duration,
	}
	if task.Review.PRNumber > 0 {
		extra["pr_number"] = task.Review.PRNumber
	}

	if notifyErr := notification.NotifyReviewCompleted(
		ctx,
		task.Review.ID,
		task.Review.RepoURL,
		extra,
	); notifyErr != nil {
		logger.Warn("Failed to send review completion notification",
			zap.String("review_id", task.Review.ID),
			zap.Error(notifyErr),
		)
	}

	if e.onComplete != nil {
		e.onComplete(task, result)
	}

	logger.Info("Task completed",
		zap.String("review_id", task.Review.ID),
		zap.Duration("duration", time.Since(startTime)),
	)
}

// handleError handles task errors
func (e *Engine) handleError(task *Task, err error) {
	logger.Error("Task failed",
		zap.String("review_id", task.Review.ID),
		zap.Error(err),
	)

	// Send notification for review failure
	extra := map[string]interface{}{
		"ref":          task.Review.Ref,
		"commit_sha":   task.Review.CommitSHA,
		"source":       task.Review.Source,
		"triggered_by": task.Review.TriggeredBy,
	}
	if task.Review.PRNumber > 0 {
		extra["pr_number"] = task.Review.PRNumber
	}

	if notifyErr := notification.NotifyReviewFailed(
		e.ctx,
		task.Review.ID,
		task.Review.RepoURL,
		err.Error(),
		extra,
	); notifyErr != nil {
		logger.Warn("Failed to send review failure notification",
			zap.String("review_id", task.Review.ID),
			zap.Error(notifyErr),
		)
	}

	if e.onError != nil {
		e.onError(task, err)
	}
}

// GetProvider returns a provider by name.
// Thread-safe: delegates to ProviderManager.
func (e *Engine) GetProvider(name string) (provider.Provider, bool) {
	if e.providerMgr == nil {
		return nil, false
	}

	result, ok := e.providerMgr.GetWithOK(name)

	return result, ok
}

// GetProviderConfig returns the provider configuration by name.
// Thread-safe: delegates to ProviderManager.
func (e *Engine) GetProviderConfig(name string) (*config.ProviderConfig, bool) {
	if e.providerMgr == nil {
		return nil, false
	}
	return e.providerMgr.GetConfig(name)
}

// GetAgent returns an agent by name.
// Delegates to AgentManager.
func (e *Engine) GetAgent(name string) (base.Agent, bool) {
	return e.agentMgr.GetWithOK(name)
}

// ListAgents returns all available agents.
// Delegates to AgentManager.
func (e *Engine) ListAgents() []string {
	return e.agentMgr.List()
}

// GetWorkspace returns the workspace directory path from configuration.
// Reads from database in real-time to ensure latest configuration is used.
func (e *Engine) GetWorkspace() string {
	reviewCfg, err := e.configProvider.GetReviewConfig()
	if err != nil {
		logger.Warn("Failed to get review config from database, using cached value",
			zap.Error(err))
		return e.cfg.Review.Workspace
	}
	if reviewCfg != nil && reviewCfg.Workspace != "" {
		return reviewCfg.Workspace
	}
	return e.cfg.Review.Workspace
}

// ListProviders returns all available providers.
// Thread-safe: delegates to ProviderManager.
func (e *Engine) ListProviders() []string {
	return e.providerMgr.List()
}

// Config returns the engine configuration
func (e *Engine) Config() *config.Config {
	return e.cfg
}

// detectProviderFromURL detects the Git provider from a repository URL.
// Delegates to ProviderManager.
func (e *Engine) detectProviderFromURL(repoURL string) string {
	return e.providerMgr.DetectFromURL(repoURL)
}

// loadExistingReviewRules loads existing ReviewRule records for a review
func (e *Engine) loadExistingReviewRules(reviewID string) (map[int]*model.ReviewRule, error) {
	rules, err := e.store.Review().GetRulesByReviewID(reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing review rules: %w", err)
	}
	rulesMap := make(map[int]*model.ReviewRule)
	for i := range rules {
		rulesMap[rules[i].RuleIndex] = &rules[i]
	}
	return rulesMap, nil
}

// checkAndFixReviewState checks and fixes review state inconsistencies
func (e *Engine) checkAndFixReviewState(review *model.Review) error {
	// Load all rules
	rules, err := e.store.Review().GetRulesByReviewID(review.ID)
	if err != nil {
		return fmt.Errorf("failed to load review rules: %w", err)
	}

	// Check if there are running rules, but review status is not running
	hasRunning := false
	for _, r := range rules {
		if r.Status == model.RuleStatusRunning {
			hasRunning = true
			break
		}
	}

	if hasRunning && review.Status != model.ReviewStatusRunning {
		logger.Warn("Review state inconsistency detected, fixing",
			zap.String("review_id", review.ID),
			zap.String("review_status", string(review.Status)),
		)
		e.store.Review().UpdateStatus(review.ID, model.ReviewStatusRunning)
	}

	return nil
}

// Retry retries a failed review by resetting its state and re-enqueuing.
// Delegates to RetryHandler.
func (e *Engine) Retry(reviewID string) error {
	return e.retryHandler.Retry(reviewID)
}

// RetryRule retries a single failed rule within a review.
// Delegates to RetryHandler.
func (e *Engine) RetryRule(reviewID string, ruleID string) error {
	return e.retryHandler.RetryRule(reviewID, ruleID)
}

// GetQueueStats returns the current queue statistics
// Useful for monitoring and debugging
func (e *Engine) GetQueueStats() QueueStats {
	if e.repoQueue == nil {
		return QueueStats{}
	}
	return e.repoQueue.GetStats()
}

// GetQueuePendingCount returns the number of pending tasks in the queue
func (e *Engine) GetQueuePendingCount() int {
	if e.repoQueue == nil {
		return 0
	}
	return e.repoQueue.GetPendingCount()
}

// GetQueueRunningCount returns the number of repos with running tasks
func (e *Engine) GetQueueRunningCount() int {
	if e.repoQueue == nil {
		return 0
	}
	return e.repoQueue.GetRunningCount()
}
