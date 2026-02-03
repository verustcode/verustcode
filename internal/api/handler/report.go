// Package handler provides HTTP handlers for the API.
package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine/utils"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/report"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
)

// ReportHandler handles report-related HTTP requests
type ReportHandler struct {
	engine *report.Engine
	cfg    *config.Config
	store  store.Store
}

// NewReportHandler creates a new report handler
func NewReportHandler(e *report.Engine, cfg *config.Config, s store.Store) *ReportHandler {
	return &ReportHandler{engine: e, cfg: cfg, store: s}
}

// CreateReportRequest represents the request body for creating a report
type CreateReportRequest struct {
	RepoURL    string `json:"repo_url" binding:"required"`    // Repository URL
	Ref        string `json:"ref" binding:"required"`         // Branch, tag, or commit
	ReportType string `json:"report_type" binding:"required"` // Report type: wiki, security, etc.
	Title      string `json:"title"`                          // Optional custom title
}

// CreateReport handles POST /api/v1/reports
func (h *ReportHandler) CreateReport(c *gin.Context) {
	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate report type (dynamically load from filesystem, no cache)
	if _, err := report.ScanReportType(req.ReportType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report type: " + req.ReportType,
		})
		return
	}

	// Create report record
	rpt := &model.Report{
		ID:         idgen.NewReportID(),
		RepoURL:    req.RepoURL,
		Ref:        req.Ref,
		ReportType: req.ReportType,
		Title:      req.Title,
		Status:     model.ReportStatusPending,
	}

	if err := h.store.Report().Create(rpt); err != nil {
		logger.Error("Failed to create report", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to create report",
		})
		return
	}

	// Auto-create repository config if not exists
	// This ensures the repo appears in the Repositories list
	if _, err := h.store.RepositoryConfig().EnsureConfig(req.RepoURL); err != nil {
		// Log warning but don't fail the report creation - this is a non-critical operation
		logger.Warn("Failed to ensure repository config for report",
			zap.String("repo_url", req.RepoURL),
			zap.Error(err),
		)
	}

	// Submit to engine
	err := h.engine.Submit(rpt, h.onReportComplete)
	if err != nil {
		h.store.Report().UpdateStatusWithError(rpt.ID, model.ReportStatusFailed, err.Error())

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": err.Error(),
		})
		return
	}

	logger.Info("Report created and submitted",
		zap.String("report_id", rpt.ID),
		zap.String("report_type", req.ReportType),
		zap.String("repo_url", req.RepoURL),
	)

	c.JSON(http.StatusAccepted, gin.H{
		"id":          rpt.ID,
		"status":      rpt.Status,
		"report_type": rpt.ReportType,
		"repo_url":    rpt.RepoURL,
		"ref":         rpt.Ref,
		"created_at":  rpt.CreatedAt,
	})
}

// onReportComplete is called when a report completes
func (h *ReportHandler) onReportComplete(rpt *model.Report, err error) {
	if err != nil {
		// Extract raw response information from error message if available
		errorMsg := err.Error()
		rawResponseInfo := extractRawResponseInfo(errorMsg)

		// Build detailed log entry
		logFields := []zap.Field{
			zap.String("report_id", rpt.ID),
			zap.String("report_type", rpt.ReportType),
			zap.String("repo_url", rpt.RepoURL),
			zap.String("ref", rpt.Ref),
			zap.String("status", string(rpt.Status)),
			zap.String("error_message", rpt.ErrorMessage),
			zap.Error(err),
		}

		// Add raw response information if found
		if rawResponseInfo != "" {
			logFields = append(logFields, zap.String("raw_response_info", rawResponseInfo))
		}

		// Add progress information if available
		if rpt.TotalSections > 0 {
			logFields = append(logFields,
				zap.Int("total_sections", rpt.TotalSections),
				zap.Int("current_section", rpt.CurrentSection),
			)
		}

		// Add timing information if available
		if rpt.StartedAt != nil {
			logFields = append(logFields, zap.Time("started_at", *rpt.StartedAt))
		}
		if rpt.Duration > 0 {
			logFields = append(logFields, zap.Int64("duration_ms", rpt.Duration))
		}

		logger.Error("Report generation failed",
			logFields...,
		)
	}
}

// extractRawResponseInfo extracts raw response information from error message
// Looks for patterns like "(raw response length: 123)" or "(raw response preview: ...)"
// Returns a summary of raw response information found in the error message
func extractRawResponseInfo(errorMsg string) string {
	var info []string

	// Look for raw response length pattern: "(raw response length: 123)"
	if idx := strings.Index(errorMsg, "(raw response length:"); idx >= 0 {
		start := idx + len("(raw response length:")
		end := strings.Index(errorMsg[start:], ")")
		if end > 0 {
			length := strings.TrimSpace(errorMsg[start : start+end])
			info = append(info, "length: "+length)
		}
	}

	// Look for raw response preview pattern: "(raw response preview: \"...\")"
	if idx := strings.Index(errorMsg, "(raw response preview:"); idx >= 0 {
		start := idx + len("(raw response preview:")
		// Find the opening quote
		quoteStart := strings.Index(errorMsg[start:], "\"")
		if quoteStart >= 0 {
			contentStart := start + quoteStart + 1
			// Find the closing quote
			quoteEnd := strings.Index(errorMsg[contentStart:], "\"")
			if quoteEnd > 0 {
				preview := errorMsg[contentStart : contentStart+quoteEnd]
				// Limit preview length for log readability
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				info = append(info, "preview: "+preview)
			}
		}
	}

	// Look for extracted JSON preview pattern: "(extracted JSON preview: \"...\")"
	if idx := strings.Index(errorMsg, "(extracted JSON preview:"); idx >= 0 {
		start := idx + len("(extracted JSON preview:")
		quoteStart := strings.Index(errorMsg[start:], "\"")
		if quoteStart >= 0 {
			contentStart := start + quoteStart + 1
			quoteEnd := strings.Index(errorMsg[contentStart:], "\"")
			if quoteEnd > 0 {
				preview := errorMsg[contentStart : contentStart+quoteEnd]
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				info = append(info, "json_preview: "+preview)
			}
		}
	}

	if len(info) > 0 {
		return strings.Join(info, "; ")
	}
	return ""
}

// GetReport handles GET /api/v1/reports/:id
func (h *ReportHandler) GetReport(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report ID",
		})
		return
	}

	rpt, err := h.store.Report().GetByIDWithSections(id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Report not found",
		})
		return
	} else if err != nil {
		logger.Error("Database error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Database error",
		})
		return
	}

	c.JSON(http.StatusOK, rpt)
}

// ListReports handles GET /api/v1/reports
func (h *ReportHandler) ListReports(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	reportType := c.Query("report_type")

	if page < 1 {
		page = 1
	}
	// Allow small page sizes for dashboard widgets (min 1)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	reports, total, err := h.store.Report().List(status, reportType, page, pageSize)
	if err != nil {
		logger.Error("Database error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Database error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      reports,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CancelReport handles POST /api/v1/reports/:id/cancel
func (h *ReportHandler) CancelReport(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report ID",
		})
		return
	}

	rowsAffected, err := h.store.Report().CancelByID(id)
	if err != nil {
		logger.Error("Failed to cancel report", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Database error",
		})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Report not found or cannot be cancelled",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Report cancelled",
	})
}

// RetryReport handles POST /api/v1/reports/:id/retry
func (h *ReportHandler) RetryReport(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report ID",
		})
		return
	}

	rpt, err := h.store.Report().GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Report not found",
		})
		return
	}

	if rpt.Status != model.ReportStatusFailed && rpt.Status != model.ReportStatusCancelled {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Only failed or cancelled reports can be retried",
		})
		return
	}

	// Update status and retry count
	rpt.Status = model.ReportStatusPending
	rpt.RetryCount++
	rpt.ErrorMessage = ""
	if err := h.store.Report().Save(rpt); err != nil {
		logger.Error("Failed to update report for retry", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Database error",
		})
		return
	}

	// Submit for retry
	err = h.engine.Submit(rpt, h.onReportComplete)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": err.Error(),
		})
		return
	}

	logger.Info("Report retry initiated",
		zap.String("report_id", id),
		zap.Int("retry_count", rpt.RetryCount),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Report retry initiated",
	})
}

// ExportReport handles GET /api/v1/reports/:id/export
func (h *ReportHandler) ExportReport(c *gin.Context) {
	id := c.Param("id")
	format := c.DefaultQuery("format", "markdown")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report ID",
		})
		return
	}

	rpt, err := h.store.Report().GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Report not found",
		})
		return
	}

	if rpt.Status != model.ReportStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Report is not completed yet",
		})
		return
	}

	if rpt.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Report has no content",
		})
		return
	}

	switch format {
	case "markdown", "md":
		// Return as markdown file
		filename := rpt.ID + ".md"
		if rpt.Title != "" {
			filename = sanitizeFilename(rpt.Title) + ".md"
		}
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "text/markdown; charset=utf-8")
		c.String(http.StatusOK, rpt.Content)

	case "html":
		// Return as self-contained HTML file
		// Fetch sections for HTML export
		sections, err := h.store.Report().GetSectionsByReportID(id)
		if err != nil {
			logger.Error("Failed to fetch report sections for HTML export",
				zap.String("report_id", id),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    errors.ErrCodeInternal,
				"message": "Failed to fetch report sections",
			})
			return
		}

		// Generate HTML
		exporter := report.NewExporter()
		htmlContent, err := exporter.ExportToHTML(rpt, sections)
		if err != nil {
			logger.Error("Failed to export report to HTML",
				zap.String("report_id", id),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    errors.ErrCodeInternal,
				"message": "Failed to generate HTML export",
			})
			return
		}

		filename := rpt.ID + ".html"
		if rpt.Title != "" {
			filename = sanitizeFilename(rpt.Title) + ".html"
		}
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, htmlContent)

	case "json":
		// Return report data as JSON
		c.JSON(http.StatusOK, gin.H{
			"id":          rpt.ID,
			"title":       rpt.Title,
			"report_type": rpt.ReportType,
			"repo_url":    rpt.RepoURL,
			"ref":         rpt.Ref,
			"content":     rpt.Content,
			"created_at":  rpt.CreatedAt,
		})

	case "pdf":
		// Return as PDF file
		pdfStartTime := time.Now()
		logger.Info("[API] PDF export request received",
			zap.String("report_id", id),
			zap.String("report_title", rpt.Title),
		)

		// Fetch sections for PDF export
		logger.Debug("[API] Fetching report sections for PDF export",
			zap.String("report_id", id),
		)
		sectionsStartTime := time.Now()
		sections, err := h.store.Report().GetSectionsByReportID(id)
		if err != nil {
			logger.Error("[API] Failed to fetch report sections for PDF export",
				zap.String("report_id", id),
				zap.Error(err),
				zap.Duration("duration", time.Since(sectionsStartTime)),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    errors.ErrCodeInternal,
				"message": "Failed to fetch report sections",
			})
			return
		}
		logger.Debug("[API] Report sections fetched successfully",
			zap.String("report_id", id),
			zap.Int("sections_count", len(sections)),
			zap.Duration("duration", time.Since(sectionsStartTime)),
		)

		// Generate PDF (this may take a few seconds)
		logger.Info("[API] Starting PDF generation",
			zap.String("report_id", id),
			zap.Int("sections_count", len(sections)),
		)
		exportStartTime := time.Now()
		exporter := report.NewExporter()
		pdfData, err := exporter.ExportToPDF(rpt, sections)
		if err != nil {
			logger.Error("[API] Failed to export report to PDF",
				zap.String("report_id", id),
				zap.Error(err),
				zap.Duration("export_duration", time.Since(exportStartTime)),
				zap.Duration("total_duration", time.Since(pdfStartTime)),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    errors.ErrCodeInternal,
				"message": "Failed to generate PDF export: " + err.Error(),
			})
			return
		}

		filename := rpt.ID + ".pdf"
		if rpt.Title != "" {
			filename = sanitizeFilename(rpt.Title) + ".pdf"
		}

		logger.Info("[API] PDF export completed successfully",
			zap.String("report_id", id),
			zap.String("filename", filename),
			zap.Int("pdf_size_bytes", len(pdfData)),
			zap.Duration("export_duration", time.Since(exportStartTime)),
			zap.Duration("total_duration", time.Since(pdfStartTime)),
		)

		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Length", fmt.Sprintf("%d", len(pdfData)))
		c.Data(http.StatusOK, "application/pdf", pdfData)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Unsupported export format: " + format,
		})
	}
}

// GetReportTypes handles GET /api/v1/report-types
// It scans the config/reports directory on each request for real-time updates.
func (h *ReportHandler) GetReportTypes(c *gin.Context) {
	// Scan config/reports directory for real-time report type discovery
	types, err := report.ScanReportTypesFromDir("config/reports")
	if err != nil {
		// Log error and fallback to cached types
		logger.Error("Failed to scan report types from config/reports, using cached types",
			zap.Error(err),
		)
		types = report.ListReportTypes()
	}

	result := make([]gin.H, len(types))
	for i, t := range types {
		// Sections are now dynamically generated by AI, so we show focus topics count instead
		focusCount := 0
		if t.Config != nil && len(t.Config.Structure.Goals.Topics) > 0 {
			focusCount = len(t.Config.Structure.Goals.Topics)
		}
		result[i] = gin.H{
			"id":          t.ID,
			"name":        t.Name,
			"description": t.Description,
			"focus_areas": focusCount,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// GetReportProgress handles GET /api/v1/reports/:id/progress
func (h *ReportHandler) GetReportProgress(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report ID",
		})
		return
	}

	progress, err := h.engine.GetProgress(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Report not found",
		})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetRepositories handles GET /api/v1/reports/repositories
// Returns a list of unique repository URLs from all reports
func (h *ReportHandler) GetRepositories(c *gin.Context) {
	repos, err := h.store.Report().GetDistinctRepositories()
	if err != nil {
		logger.Error("Database error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Database error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": repos,
	})
}

// GetBranches handles GET /api/v1/reports/branches?repo_url=xxx
// Returns a list of branches for the specified repository
func (h *ReportHandler) GetBranches(c *gin.Context) {
	repoURL := c.Query("repo_url")
	if repoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "repo_url query parameter is required",
		})
		return
	}

	// Get providers from database to ensure latest configuration is used
	svc := config.NewSettingsService(h.store)
	providers, err := svc.GetGitProviders()
	if err != nil {
		logger.Error("Failed to get git providers from database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to load git providers configuration",
		})
		return
	}

	// 使用公共方法获取或创建 provider（带回退机制）
	p, err := utils.GetOrCreateProviderForURL(repoURL, providers)
	if err != nil {
		logger.Error("Failed to get provider for URL",
			zap.String("repo_url", repoURL),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "unsupported repository provider",
		})
		return
	}

	// Parse owner and repo from URL
	owner, repo, err := p.ParseRepoPath(repoURL)
	if err != nil {
		logger.Error("Failed to parse repo path",
			zap.String("repo_url", repoURL),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "invalid repository URL format",
		})
		return
	}

	// List branches
	ctx := context.Background()
	branches, err := p.ListBranches(ctx, owner, repo)
	if err != nil {

		// Provide user-friendly error messages based on error type
		var errMsg string
		var statusCode int
		errStr := err.Error()

		if strings.Contains(errStr, "rate limit exceeded") {
			errMsg = "GitHub API rate limit exceeded. Please configure GITHUB_TOKEN environment variable for higher rate limits (60 req/hour anonymous vs 5000 req/hour authenticated)"
			statusCode = http.StatusTooManyRequests
		} else if strings.Contains(errStr, "Bad credentials") {
			errMsg = "Invalid GitHub token. For public repositories, the system will automatically retry with anonymous access"
			statusCode = http.StatusUnauthorized
		} else if strings.Contains(errStr, "Not Found") || strings.Contains(errStr, "404") {
			errMsg = "Repository not found. Please check the repository URL and ensure it exists"
			statusCode = http.StatusNotFound
		} else {
			errMsg = "Failed to list branches. Please check the repository URL and your network connection"
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": errMsg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": branches,
	})
}

// sanitizeFilename removes unsafe characters from filename
func sanitizeFilename(name string) string {
	// Replace unsafe characters
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range unsafe {
		result = replaceAll(result, char, "_")
	}
	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

// replaceAll is a simple string replace helper
func replaceAll(s, old, new string) string {
	for {
		idx := -1
		for i := 0; i <= len(s)-len(old); i++ {
			if s[i:i+len(old)] == old {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}
