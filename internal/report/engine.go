// Package report provides report generation functionality.
package report

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine/utils"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/notification"
	"github.com/verustcode/verustcode/internal/report/recovery"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Engine orchestrates the report generation process
type Engine struct {
	configProvider config.ConfigProvider
	providers      map[string]provider.Provider
	providerMu     sync.RWMutex // Protects providers for hot-reload
	agents         map[string]base.Agent
	store          store.Store

	repoManager RepositoryManager
	structGen   *StructureGenerator
	sectionGen  *SectionGenerator
	summaryGen  *SummaryGenerator
	recovery    *recovery.Service

	// Task management
	taskQueue chan *ReportTask
	workers   int
	wg        sync.WaitGroup

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// ReportTask represents a report generation task
type ReportTask struct {
	Report   *model.Report
	Callback func(*model.Report, error)
}

// NewEngine creates a new report engine
func NewEngine(cfg *config.Config, providers map[string]provider.Provider, agents map[string]base.Agent, s store.Store) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	// Get cursor agent as default for report generation
	// Agent selection is determined by DSL config, not config.yaml
	agent := agents["cursor"]

	// Create config provider for real-time database reads
	configProvider := config.NewDBConfigProvider(s)

	e := &Engine{
		configProvider: configProvider,
		providers:      providers,
		agents:         agents,
		store:          s,
		repoManager:    NewRepositoryManager(),
		structGen:      NewStructureGenerator(agent, configProvider),
		sectionGen:     NewSectionGenerator(agent, configProvider),
		summaryGen:     NewSummaryGenerator(agent, configProvider),
		taskQueue:      make(chan *ReportTask, 100),
		workers:        getReportWorkers(configProvider),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize recovery service
	e.recovery = recovery.NewService(cfg, s, e)

	return e
}

// getReportWorkers retrieves MaxConcurrent from config provider with fallback
func getReportWorkers(provider config.ConfigProvider) int {
	reportCfg, err := provider.GetReportConfig()
	if err != nil || reportCfg == nil {
		return 3 // Default workers
	}
	if reportCfg.MaxConcurrent <= 0 {
		return 3
	}
	return reportCfg.MaxConcurrent
}

// Start starts the report engine workers
func (e *Engine) Start() {
	logger.Info("Starting report engine", zap.Int("workers", e.workers))

	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}

	// Recover pending/processing reports to memory queue
	// This must be done after workers start to ensure tasks are processed
	e.recovery.RecoverToQueue(e.ctx)
}

// Stop stops the report engine
func (e *Engine) Stop() {
	logger.Info("Stopping report engine")
	e.cancel()
	close(e.taskQueue)
	e.wg.Wait()
	logger.Info("Report engine stopped")
}

// worker processes report tasks
func (e *Engine) worker(id int) {
	defer e.wg.Done()
	logger.Info("Report worker started", zap.Int("worker_id", id))

	for task := range e.taskQueue {
		select {
		case <-e.ctx.Done():
			return
		default:
			e.processTask(task)
		}
	}
}

// Submit submits a report for generation
func (e *Engine) Submit(report *model.Report, callback func(*model.Report, error)) error {
	if report == nil {
		return fmt.Errorf("report cannot be nil")
	}

	task := &ReportTask{
		Report:   report,
		Callback: callback,
	}

	select {
	case e.taskQueue <- task:
		logger.Info("Report submitted to queue",
			zap.String("report_id", report.ID),
			zap.String("report_type", report.ReportType),
		)
		return nil
	default:
		return fmt.Errorf("report queue is full")
	}
}

// Enqueue enqueues a report for processing (implements recovery.TaskEnqueuer interface)
func (e *Engine) Enqueue(report *model.Report, callback func(*model.Report, error)) bool {
	if report == nil {
		return false
	}

	task := &ReportTask{
		Report:   report,
		Callback: callback,
	}

	select {
	case e.taskQueue <- task:
		return true
	default:
		return false
	}
}

// processTask processes a single report task
func (e *Engine) processTask(task *ReportTask) {
	report := task.Report
	var err error

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during report generation: %v", r)
			reportID := "unknown"
			if report != nil {
				reportID = report.ID
			}
			logger.Error("Report generation panic",
				zap.String("report_id", reportID),
				zap.Any("panic", r),
			)
		}
		if task.Callback != nil {
			task.Callback(report, err)
		}
	}()

	// Validate report before processing
	if report == nil {
		err = fmt.Errorf("report is nil")
		logger.Error("Cannot process nil report")
		return
	}

	logger.Info("Processing report",
		zap.String("report_id", report.ID),
		zap.String("report_type", report.ReportType),
	)

	err = e.Run(e.ctx, report)
	if err != nil {
		logger.Error("Report generation failed",
			zap.String("report_id", report.ID),
			zap.Error(err),
		)
	}
}

// Run executes the full report generation pipeline
func (e *Engine) Run(ctx context.Context, report *model.Report) error {
	startTime := time.Now()

	// Update status to analyzing
	report.Status = model.ReportStatusAnalyzing
	report.StartedAt = &startTime
	if err := e.store.Report().Save(report); err != nil {
		return fmt.Errorf("failed to update report status: %w", err)
	}

	// Phase 0: Prepare repository
	repoPath, err := e.prepareRepository(ctx, report)
	if err != nil {
		return e.handleError(report, err, "repository preparation failed")
	}
	report.RepoPath = repoPath
	e.store.Report().Save(report)

	// Phase 1: Generate structure
	structure, err := e.runPhase1(ctx, report, repoPath)
	if err != nil {
		return e.handleError(report, err, "structure generation failed")
	}

	// Phase 2: Generate sections
	if err := e.runPhase2(ctx, report, structure, repoPath); err != nil {
		return e.handleError(report, err, "section generation failed")
	}

	// Phase 3: Merge, generate summary, and finalize
	if err := e.runPhase3(ctx, report, structure, repoPath); err != nil {
		return e.handleError(report, err, "report finalization failed")
	}

	// Mark as completed
	completedAt := time.Now()
	report.Status = model.ReportStatusCompleted
	report.CompletedAt = &completedAt
	report.Duration = completedAt.Sub(startTime).Milliseconds()
	if err := e.store.Report().Save(report); err != nil {
		logger.Error("Failed to save completed report", zap.Error(err))
	}

	// Send notification for report completion
	extra := map[string]interface{}{
		"report_type": report.ReportType,
		"ref":         report.Ref,
		"title":       report.Title,
		"duration_ms": report.Duration,
	}

	if notifyErr := notification.NotifyReportCompleted(
		ctx,
		report.ID,
		report.RepoURL,
		extra,
	); notifyErr != nil {
		logger.Warn("Failed to send report completion notification",
			zap.String("report_id", report.ID),
			zap.Error(notifyErr),
		)
	}

	logger.Info("Report generation completed",
		zap.String("report_id", report.ID),
		zap.Int64("duration_ms", report.Duration),
	)

	return nil
}

// prepareRepository prepares the repository for analysis
func (e *Engine) prepareRepository(ctx context.Context, report *model.Report) (string, error) {
	logger.Info("Preparing repository",
		zap.String("repo_url", report.RepoURL),
		zap.String("ref", report.Ref),
	)

	// Parse repository URL to get provider, owner, repo
	prov, owner, repo, err := e.parseRepoURL(report.RepoURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Get workspace directory from database config (real-time read)
	workspaceDir := "./report_workspace" // Default
	if reportCfg, err := e.configProvider.GetReportConfig(); err == nil && reportCfg != nil {
		if reportCfg.Workspace != "" {
			workspaceDir = reportCfg.Workspace
		}
	}

	// Clone/update repository
	req := &RepositoryRequest{
		Provider:  prov,
		Owner:     owner,
		Repo:      repo,
		Ref:       report.Ref,
		Workspace: workspaceDir,
	}

	return e.repoManager.EnsureRepository(ctx, req)
}

// parseRepoURL parses a repository URL and returns provider, owner, repo
// Thread-safe: uses read lock to protect concurrent access during hot-reload
// Priority:
// 1. Use configured providers first (may have token for private repos)
// 2. Fall back to anonymous provider for public repos if no configured provider matches
func (e *Engine) parseRepoURL(repoURL string) (provider.Provider, string, string, error) {
	// 获取配置的 providers 列表
	providers, err := e.configProvider.GetGitProviders()
	if err != nil {
		logger.Warn("Failed to get git providers from config",
			zap.Error(err),
		)
		// 即使获取配置失败，也尝试使用空列表继续（会回退到匿名 provider）
		providers = []config.ProviderConfig{}
	}

	// 使用公共方法获取或创建 provider
	prov, err := utils.GetOrCreateProviderForURL(repoURL, providers)
	if err != nil {
		return nil, "", "", err
	}

	// 解析 owner 和 repo
	owner, repo, err := prov.ParseRepoPath(repoURL)
	if err != nil || owner == "" || repo == "" {
		return nil, "", "", fmt.Errorf("failed to parse repository path: %w", err)
	}

	return prov, owner, repo, nil
}

// runPhase1 executes Phase 1: Structure Generation
func (e *Engine) runPhase1(ctx context.Context, report *model.Report, repoPath string) (*ReportStructure, error) {
	logger.Info("Starting Phase 1: Structure Generation",
		zap.String("report_id", report.ID),
	)

	// Generate structure
	structure, err := e.structGen.Generate(ctx, report, repoPath)
	if err != nil {
		return nil, err
	}

	// Store structure in report
	report.Structure = StructureToJSONMap(structure)
	report.Title = structure.Title
	// TotalSections counts only leaf sections (those that will have content generated)
	report.TotalSections = CountLeafSections(structure)

	// Create section records (includes both parent and child sections)
	sections := StructureToModel(structure, report.ID)
	for i := range sections {
		if err := e.store.Report().CreateSection(&sections[i]); err != nil {
			logger.Error("Failed to create section record", zap.Error(err))
		}
	}

	// Update report status
	report.Status = model.ReportStatusGenerating
	if err := e.store.Report().Save(report); err != nil {
		logger.Error("Failed to save report after phase 1", zap.Error(err))
	}

	logger.Info("Phase 1 completed",
		zap.String("report_id", report.ID),
		zap.String("title", structure.Title),
		zap.Int("total_sections", len(sections)),
		zap.Int("leaf_sections", report.TotalSections),
	)

	return structure, nil
}

// runPhase2 executes Phase 2: Section Content Generation
// Only generates content for leaf sections (is_leaf = true)
func (e *Engine) runPhase2(ctx context.Context, report *model.Report, structure *ReportStructure, repoPath string) error {
	logger.Info("Starting Phase 2: Section Content Generation",
		zap.String("report_id", report.ID),
		zap.Int("total_sections", CountLeafSections(structure)),
	)

	// Load only leaf section records (sections that need content generation)
	// Parent sections (is_leaf = false) are structural containers and don't need content
	sections, err := e.store.Report().GetLeafSectionsByReportID(report.ID)
	if err != nil {
		return fmt.Errorf("failed to load leaf sections: %w", err)
	}

	logger.Info("Loaded leaf sections for content generation",
		zap.String("report_id", report.ID),
		zap.Int("leaf_sections", len(sections)),
	)

	// Generate each leaf section
	leafIndex := 0
	for i := range sections {
		section := &sections[i]

		// Skip already completed sections (for resume support)
		if section.Status == model.SectionStatusCompleted {
			logger.Info("Skipping completed section",
				zap.String("section_id", section.SectionID),
			)
			leafIndex++
			continue
		}

		// Update progress (based on leaf index, not global index)
		report.CurrentSection = leafIndex
		e.store.Report().Save(report)

		// Update section status
		startTime := time.Now()
		section.Status = model.SectionStatusRunning
		section.StartedAt = &startTime
		e.store.Report().UpdateSection(section)

		// Generate content and summary
		result, err := e.sectionGen.GenerateSection(ctx, report, section, structure, repoPath)
		if err != nil {
			logger.Error("Failed to generate section",
				zap.String("section_id", section.SectionID),
				zap.Error(err),
			)
			section.Status = model.SectionStatusFailed
			section.ErrorMessage = err.Error()
			e.store.Report().UpdateSection(section)
			leafIndex++
			continue // Continue with other sections
		}

		// Save content and summary
		completedAt := time.Now()
		section.Content = result.Content
		section.Summary = result.Summary
		section.Status = model.SectionStatusCompleted
		section.CompletedAt = &completedAt
		section.Duration = completedAt.Sub(startTime).Milliseconds()
		e.store.Report().UpdateSection(section)

		logger.Info("Section generated",
			zap.String("section_id", section.SectionID),
			zap.Int("content_length", len(result.Content)),
			zap.Int("summary_length", len(result.Summary)),
			zap.Int64("duration_ms", section.Duration),
		)

		leafIndex++
	}

	logger.Info("Phase 2 completed",
		zap.String("report_id", report.ID),
		zap.Int("generated_sections", leafIndex),
	)

	return nil
}

// runPhase3 executes Phase 3: Merge, Summary Generation, and Finalize
func (e *Engine) runPhase3(ctx context.Context, report *model.Report, structure *ReportStructure, repoPath string) error {
	logger.Info("Starting Phase 3: Merge, Summary Generation, and Finalize",
		zap.String("report_id", report.ID),
	)

	// Load all sections
	sections, err := e.store.Report().GetSectionsByReportID(report.ID)
	if err != nil {
		return fmt.Errorf("failed to load sections: %w", err)
	}

	// Phase 3a: Merge sections into final content
	report.Content = MergeSections(report, sections)

	// Phase 3b: Generate overall report summary from section summaries
	summary, err := e.summaryGen.GenerateSummary(ctx, report, sections, structure, repoPath)
	if err != nil {
		logger.Warn("Failed to generate report summary, continuing without summary",
			zap.String("report_id", report.ID),
			zap.Error(err),
		)
		// Summary generation is not critical, continue without it
	} else {
		report.Summary = summary
		logger.Info("Report summary generated",
			zap.String("report_id", report.ID),
			zap.Int("summary_length", len(summary)),
		)
	}

	if err := e.store.Report().Save(report); err != nil {
		return fmt.Errorf("failed to save merged content and summary: %w", err)
	}

	logger.Info("Phase 3 completed",
		zap.String("report_id", report.ID),
		zap.Int("content_length", len(report.Content)),
		zap.Int("summary_length", len(report.Summary)),
	)

	return nil
}

// handleError handles errors during report generation
func (e *Engine) handleError(report *model.Report, err error, context string) error {
	report.Status = model.ReportStatusFailed
	report.ErrorMessage = fmt.Sprintf("%s: %v", context, err)

	if saveErr := e.store.Report().Save(report); saveErr != nil {
		logger.Error("Failed to save error status", zap.Error(saveErr))
	}

	// Send notification for report failure
	extra := map[string]interface{}{
		"report_type": report.ReportType,
		"ref":         report.Ref,
		"context":     context,
	}

	if notifyErr := notification.NotifyReportFailed(
		e.ctx,
		report.ID,
		report.RepoURL,
		err.Error(),
		extra,
	); notifyErr != nil {
		logger.Warn("Failed to send report failure notification",
			zap.String("report_id", report.ID),
			zap.Error(notifyErr),
		)
	}

	return fmt.Errorf("%s: %w", context, err)
}

// Resume resumes a failed or interrupted report
func (e *Engine) Resume(ctx context.Context, reportID string) error {
	report, err := e.store.Report().GetByID(reportID)
	if err != nil {
		return fmt.Errorf("report not found: %w", err)
	}

	if report.Status == model.ReportStatusCompleted {
		return fmt.Errorf("report already completed")
	}

	// Determine where to resume
	if len(report.Structure) == 0 {
		// Need to restart from Phase 1
		return e.Run(ctx, report)
	}

	// Parse structure
	var structure ReportStructure
	structData, _ := json.Marshal(report.Structure)
	if err := json.Unmarshal(structData, &structure); err != nil {
		// Structure corrupted, restart
		return e.Run(ctx, report)
	}

	// Resume from Phase 2
	repoPath := report.RepoPath
	if repoPath == "" {
		var err error
		repoPath, err = e.prepareRepository(ctx, report)
		if err != nil {
			return e.handleError(report, err, "repository preparation failed")
		}
		report.RepoPath = repoPath
		e.store.Report().Save(report)
	}

	// Update status
	report.Status = model.ReportStatusGenerating
	e.store.Report().Save(report)

	if err := e.runPhase2(ctx, report, &structure, repoPath); err != nil {
		return err
	}

	return e.runPhase3(ctx, report, &structure, repoPath)
}

// GetProgress returns the current progress of a report
func (e *Engine) GetProgress(reportID string) (*ReportProgress, error) {
	report, err := e.store.Report().GetByID(reportID)
	if err != nil {
		return nil, err
	}

	sections, err := e.store.Report().GetSectionsByReportID(reportID)
	if err != nil {
		// Just log warning and return empty sections if failed
		logger.Warn("Failed to load sections for progress", zap.Error(err))
		sections = []model.ReportSection{}
	}

	progress := &ReportProgress{
		ReportID:       report.ID,
		Status:         string(report.Status),
		TotalSections:  report.TotalSections,
		CurrentSection: report.CurrentSection,
		Sections:       make([]SectionProgress, len(sections)),
	}

	for i, s := range sections {
		progress.Sections[i] = SectionProgress{
			SectionID: s.SectionID,
			Title:     s.Title,
			Status:    string(s.Status),
		}
	}

	return progress, nil
}

// ReportProgress represents the current progress of a report
type ReportProgress struct {
	ReportID       string            `json:"report_id"`
	Status         string            `json:"status"`
	TotalSections  int               `json:"total_sections"`
	CurrentSection int               `json:"current_section"`
	Sections       []SectionProgress `json:"sections"`
}

// SectionProgress represents the progress of a section
type SectionProgress struct {
	SectionID string `json:"section_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
}
