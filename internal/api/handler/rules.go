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

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// reviewsDir is the directory containing review rule files
var reviewsDir = config.ReviewsDir

// RulesHandler handles rule-related HTTP requests
type RulesHandler struct {
	rulesMu sync.Mutex // protects rule files read-compare-write operations
}

// NewRulesHandler creates a new rules handler
func NewRulesHandler() *RulesHandler {
	return &RulesHandler{}
}

// RuleFileInfo represents a rule file info
type RuleFileInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	ModifiedAt string `json:"modified_at"`
}

// SaveRuleRequest represents the save rule request body
type SaveRuleRequest struct {
	Content string `json:"content" binding:"required"`
	Hash    string `json:"hash" binding:"required"` // SHA256 hash for optimistic locking
}

// ValidateRuleRequest represents the validate rule request body
type ValidateRuleRequest struct {
	Content string `json:"content" binding:"required"`
}

// CreateRuleFileRequest represents the request to create a new rule file
type CreateRuleFileRequest struct {
	Name     string `json:"name" binding:"required"`
	CopyFrom string `json:"copy_from"` // optional: file to copy content from (default: default.example.yaml)
}

// ListRules handles GET /api/v1/admin/rules
// Returns all review files from config/reviews/ directory
func (h *RulesHandler) ListRules(c *gin.Context) {
	var files []RuleFileInfo

	// List files from config/reviews/ directory
	reviewFiles, err := dsl.ListReviewFiles()
	if err != nil {
		logger.Warn("Failed to list review files", zap.Error(err))
	}

	for _, fileName := range reviewFiles {
		filePath := filepath.Join(reviewsDir, fileName)
		info, err := os.Stat(filePath)
		if err == nil {
			files = append(files, RuleFileInfo{
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

// GetRule handles GET /api/v1/admin/rules/:name
func (h *RulesHandler) GetRule(c *gin.Context) {
	name := c.Param("name")

	// Validate filename and build safe path (from config/reviews/ directory)
	filePath, ok := safeJoinPath(reviewsDir, name)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid rule file name",
		})
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    errors.ErrCodeNotFound,
				"message": "Rule file not found",
			})
			return
		}
		logger.Error("Failed to read rule file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to read rule file",
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

// SaveRule handles PUT /api/v1/admin/rules/:name
// Supports optimistic locking via hash parameter
func (h *RulesHandler) SaveRule(c *gin.Context) {
	name := c.Param("name")

	// Validate filename and build safe path (from config/reviews/ directory)
	filePath, ok := safeJoinPath(reviewsDir, name)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid rule file name",
		})
		return
	}

	var req SaveRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: content and hash are required",
		})
		return
	}

	// Validate YAML syntax by parsing it
	var ruleConfig dsl.ReviewRulesConfig
	if err := yaml.Unmarshal([]byte(req.Content), &ruleConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": fmt.Sprintf("Invalid YAML: %v", err),
		})
		return
	}

	// Lock for read-compare-write operation
	h.rulesMu.Lock()
	defer h.rulesMu.Unlock()

	// Read current content and verify hash for optimistic locking
	currentContent, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		logger.Error("Failed to read current rule file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to read current rule file",
		})
		return
	}

	// If file exists, verify hash matches
	if err == nil {
		currentHash := computeContentHash(currentContent)
		if currentHash != req.Hash {
			logger.Warn("Rule file modified by another user",
				zap.String("name", name),
				zap.String("expected_hash", req.Hash),
				zap.String("current_hash", currentHash))
			c.JSON(http.StatusConflict, gin.H{
				"code":    errors.ErrCodeConflict,
				"message": "Rule file has been modified by another user. Please refresh and try again.",
			})
			return
		}
	}

	// Write to file (filePath already validated above)
	if err := os.WriteFile(filePath, []byte(req.Content), 0644); err != nil {
		logger.Error("Failed to write rule file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to save rule file",
		})
		return
	}

	// Compute new hash for response
	newHash := computeContentHash([]byte(req.Content))

	logger.Info("Rule file saved", zap.String("name", name))

	c.JSON(http.StatusOK, gin.H{
		"message": "Rule file saved successfully",
		"hash":    newHash,
	})
}

// ValidateRule handles POST /api/v1/admin/rules/validate
func (h *RulesHandler) ValidateRule(c *gin.Context) {
	var req ValidateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body",
		})
		return
	}

	// Try to parse the YAML
	var ruleConfig dsl.ReviewRulesConfig
	if err := yaml.Unmarshal([]byte(req.Content), &ruleConfig); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"errors": []string{fmt.Sprintf("YAML syntax error: %v", err)},
		})
		return
	}

	// Basic validation
	var validationErrors []string
	if len(ruleConfig.Rules) == 0 {
		validationErrors = append(validationErrors, "At least one rule is required")
	}

	for i, rule := range ruleConfig.Rules {
		if rule.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("Rule %d: id is required", i+1))
		}
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"errors": validationErrors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
	})
}

// CreateRuleFile handles POST /api/v1/admin/rules
// Creates a new review rule file by copying from an existing file
func (h *RulesHandler) CreateRuleFile(c *gin.Context) {
	var req CreateRuleFileRequest
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
	filePath, ok := safeJoinPath(reviewsDir, req.Name)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid rule file name",
		})
		return
	}

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    errors.ErrCodeConflict,
			"message": "Rule file already exists",
		})
		return
	}

	// Determine content: copy from specified file or use default example
	var content []byte
	var err error

	copyFrom := req.CopyFrom
	if copyFrom == "" {
		copyFrom = "default.example.yaml" // default template
	}

	// Try to read the source file
	sourcePath, ok := safeJoinPath(reviewsDir, copyFrom)
	if ok {
		content, err = os.ReadFile(sourcePath)
		if err != nil {
			logger.Warn("Failed to read source file for copy, using empty template",
				zap.String("copy_from", copyFrom),
				zap.Error(err),
			)
			// Use a minimal default template if source file not found
			content = []byte(`# Review Rules Configuration
version: "1.0"

rule_base:
  agent:
    type: cursor

  output:
    format: markdown
    channels:
      - type: file
      - type: comment
        overwrite: true

rules:
  - id: code-quality
    description: |
      Conducts code quality review.

    goals:
      areas:
        - business-logic
        - error-handling
        - readability

    constraints:
      scope_control:
        - Review only code changed in this PR
`)
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid source file name",
		})
		return
	}

	// Validate YAML syntax
	var ruleConfig dsl.ReviewRulesConfig
	if err := yaml.Unmarshal(content, &ruleConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": fmt.Sprintf("Invalid YAML in source file: %v", err),
		})
		return
	}

	// Write file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		logger.Error("Failed to create rule file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to create rule file",
		})
		return
	}

	// Compute hash
	hash := computeContentHash(content)

	logger.Info("Created rule file",
		zap.String("name", req.Name),
		zap.String("copy_from", copyFrom),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Rule file created successfully",
		"name":    req.Name,
		"hash":    hash,
	})
}

// ListReviewFiles handles GET /api/v1/admin/review-files
// Returns all available review configuration files
func (h *RulesHandler) ListReviewFiles(c *gin.Context) {
	files, err := dsl.ListReviewFiles()
	if err != nil {
		logger.Error("Failed to list review files", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to list review files",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
	})
}
