// Package handler provides HTTP handlers for the API.
package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/report"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// reportsDir is the directory containing report configuration files
var reportsDir = report.DefaultReportsConfigDir

// ReportTypesHandler handles report type configuration related HTTP requests
type ReportTypesHandler struct {
	reportTypesMu sync.Mutex // protects report type files read-compare-write operations
}

// NewReportTypesHandler creates a new report types handler
func NewReportTypesHandler() *ReportTypesHandler {
	return &ReportTypesHandler{}
}

// ReportTypeFileInfo represents a report type file info
type ReportTypeFileInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	ModifiedAt string `json:"modified_at"`
}

// SaveReportTypeRequest represents the save report type request body
type SaveReportTypeRequest struct {
	Content string `json:"content" binding:"required"`
	Hash    string `json:"hash" binding:"required"` // SHA256 hash for optimistic locking
}

// ValidateReportTypeRequest represents the validate report type request body
type ValidateReportTypeRequest struct {
	Content string `json:"content" binding:"required"`
}

// CreateReportTypeFileRequest represents the request to create a new report type file
type CreateReportTypeFileRequest struct {
	Name     string `json:"name" binding:"required"`
	CopyFrom string `json:"copy_from"` // optional: file to copy content from (default: wiki_simple.yaml)
}

// ListReportTypes handles GET /api/v1/admin/report-types
// Returns all report configuration files from config/reports/ directory
func (h *ReportTypesHandler) ListReportTypes(c *gin.Context) {
	var files []ReportTypeFileInfo

	// List files from config/reports/ directory
	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		logger.Warn("Failed to read reports directory", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"files": []ReportTypeFileInfo{},
		})
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		// Only include YAML files
		if !strings.HasSuffix(fileName, ".yaml") && !strings.HasSuffix(fileName, ".yml") {
			continue
		}

		filePath := filepath.Join(reportsDir, fileName)
		info, err := os.Stat(filePath)
		if err == nil {
			files = append(files, ReportTypeFileInfo{
				Name:       fileName,
				Path:       filePath,
				ModifiedAt: info.ModTime().Format(time.RFC3339),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
	})
}

// GetReportType handles GET /api/v1/admin/report-types/:name
func (h *ReportTypesHandler) GetReportType(c *gin.Context) {
	name := c.Param("name")

	// Validate filename and build safe path (from config/reports/ directory)
	filePath, ok := safeJoinPath(reportsDir, name)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report type file name",
		})
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    errors.ErrCodeNotFound,
				"message": "Report type file not found",
			})
			return
		}
		logger.Error("Failed to read report type file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to read report type file",
		})
		return
	}

	// Compute hash of content for optimistic locking
	contentHash := computeContentHash(content)

	c.JSON(http.StatusOK, gin.H{
		"content": string(content),
		"hash":    contentHash,
	})
}

// SaveReportType handles PUT /api/v1/admin/report-types/:name
// Supports optimistic locking via hash parameter
func (h *ReportTypesHandler) SaveReportType(c *gin.Context) {
	name := c.Param("name")

	// Validate filename and build safe path (from config/reports/ directory)
	filePath, ok := safeJoinPath(reportsDir, name)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report type file name",
		})
		return
	}

	var req SaveReportTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: content and hash are required",
		})
		return
	}

	// Validate YAML syntax by parsing it
	var reportConfig dsl.ReportConfig
	if err := yaml.Unmarshal([]byte(req.Content), &reportConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": fmt.Sprintf("Invalid YAML: %v", err),
		})
		return
	}

	// Validate report config structure
	loader := dsl.NewReportLoader()
	if _, err := loader.LoadFromBytes([]byte(req.Content)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": fmt.Sprintf("Invalid report configuration: %v", err),
		})
		return
	}

	// Lock for read-compare-write operation
	h.reportTypesMu.Lock()
	defer h.reportTypesMu.Unlock()

	// Read current content and verify hash for optimistic locking
	currentContent, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		logger.Error("Failed to read current report type file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to read current report type file",
		})
		return
	}

	// If file exists, verify hash matches
	if err == nil {
		currentHash := computeContentHash(currentContent)
		if currentHash != req.Hash {
			logger.Warn("Report type file modified by another user",
				zap.String("name", name),
				zap.String("expected_hash", req.Hash),
				zap.String("current_hash", currentHash))
			c.JSON(http.StatusConflict, gin.H{
				"code":    errors.ErrCodeConflict,
				"message": "Report type file has been modified by another user. Please refresh and try again.",
			})
			return
		}
	}

	// Write to file (filePath already validated above)
	if err := os.WriteFile(filePath, []byte(req.Content), 0644); err != nil {
		logger.Error("Failed to write report type file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to save report type file",
		})
		return
	}

	// Compute new hash for response
	newHash := computeContentHash([]byte(req.Content))

	logger.Info("Report type file saved", zap.String("name", name))

	c.JSON(http.StatusOK, gin.H{
		"message": "Report type file saved successfully",
		"hash":    newHash,
	})
}

// ValidateReportType handles POST /api/v1/admin/report-types/validate
func (h *ReportTypesHandler) ValidateReportType(c *gin.Context) {
	var req ValidateReportTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body",
		})
		return
	}

	// Try to parse the YAML
	var reportConfig dsl.ReportConfig
	if err := yaml.Unmarshal([]byte(req.Content), &reportConfig); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"errors": []string{fmt.Sprintf("YAML syntax error: %v", err)},
		})
		return
	}

	// Validate using report loader
	loader := dsl.NewReportLoader()
	if _, err := loader.LoadFromBytes([]byte(req.Content)); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"errors": []string{fmt.Sprintf("Validation error: %v", err)},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
	})
}

// CreateReportType handles POST /api/v1/admin/report-types
// Creates a new report type file by copying from an existing file or using default template
func (h *ReportTypesHandler) CreateReportType(c *gin.Context) {
	var req CreateReportTypeFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Ensure filename ends with .yaml
	if !strings.HasSuffix(req.Name, ".yaml") && !strings.HasSuffix(req.Name, ".yml") {
		req.Name = req.Name + ".yaml"
	}

	// Validate filename
	filePath, ok := safeJoinPath(reportsDir, req.Name)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report type file name",
		})
		return
	}

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    errors.ErrCodeConflict,
			"message": "Report type file already exists",
		})
		return
	}

	// Determine content: copy from specified file or use default template
	var content []byte
	var err error

	copyFrom := req.CopyFrom
	if copyFrom == "" {
		copyFrom = "wiki_simple.yaml" // default template
	}

	// Try to read the source file
	sourcePath, ok := safeJoinPath(reportsDir, copyFrom)
	if ok {
		content, err = os.ReadFile(sourcePath)
		if err != nil {
			logger.Warn("Failed to read source file for copy, using default template",
				zap.String("copy_from", copyFrom),
				zap.Error(err),
			)
			// Use default template from DefaultReportConfig
			defaultConfig := dsl.DefaultReportConfig()
			// Set basic fields
			defaultConfig.ID = strings.TrimSuffix(req.Name, filepath.Ext(req.Name))
			defaultConfig.Name = "New Report Type"
			defaultConfig.Description = "Description of this report type"

			// Marshal to YAML
			content, err = yaml.Marshal(defaultConfig)
			if err != nil {
				logger.Error("Failed to marshal default config", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    errors.ErrCodeInternal,
					"message": "Failed to generate default template",
				})
				return
			}
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid source file name",
		})
		return
	}

	// Validate YAML syntax
	var reportConfig dsl.ReportConfig
	if err := yaml.Unmarshal(content, &reportConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": fmt.Sprintf("Invalid YAML in source file: %v", err),
		})
		return
	}

	// Write file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		logger.Error("Failed to create report type file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to create report type file",
		})
		return
	}

	// Compute hash
	hash := computeContentHash(content)

	logger.Info("Created report type file",
		zap.String("name", req.Name),
		zap.String("copy_from", copyFrom),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Report type file created successfully",
		"name":    req.Name,
		"hash":    hash,
	})
}
