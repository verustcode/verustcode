// Package handler provides HTTP handlers for the API.
package handler

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// FindingsHandler handles findings-related HTTP requests
type FindingsHandler struct {
	store store.Store
}

// NewFindingsHandler creates a new findings handler
func NewFindingsHandler(s store.Store) *FindingsHandler {
	return &FindingsHandler{store: s}
}

// FindingItem represents a single finding item for the API response
type FindingItem struct {
	ReviewID    string    `json:"review_id"`
	RepoURL     string    `json:"repo_url"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// FindingsListResponse represents the response for the findings list endpoint
type FindingsListResponse struct {
	Data     []FindingItem `json:"data"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// Severity order for sorting (lower index = higher priority)
var severityOrder = map[string]int{
	"critical": 0,
	"high":     1,
	"medium":   2,
	"low":      3,
	"info":     4,
}

// ListFindings handles GET /api/v1/admin/findings
// Query parameters:
//   - page: page number (default: 1)
//   - page_size: items per page (default: 20, max: 100)
//   - repo_url: filter by repository URL (optional)
//   - severity: filter by severity level (optional)
//   - category: filter by category (optional)
//   - sort_by: sort field - "severity" or "category" (default: severity)
//   - sort_order: sort order - "asc" or "desc" (default: desc for severity priority)
func (h *FindingsHandler) ListFindings(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Parse filter parameters
	repoURL := c.Query("repo_url")
	severityFilter := strings.ToLower(c.Query("severity"))
	categoryFilter := c.Query("category")

	// Parse sort parameters
	sortBy := c.DefaultQuery("sort_by", "severity")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// Validate sort_by parameter
	if sortBy != "severity" && sortBy != "category" {
		sortBy = "severity"
	}

	// Validate sort_order parameter
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Fetch all findings with repo info
	results, err := h.store.Review().GetAllFindingsWithRepoInfo(repoURL)
	if err != nil {
		logger.Error("Failed to fetch findings",
			zap.Error(err),
			zap.String("repo_url", repoURL),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to fetch findings data",
		})
		return
	}

	// Extract and flatten all findings
	var findings []FindingItem
	for _, result := range results {
		extractedFindings := h.extractFindingsFromData(result.ReviewID, result.RepoURL, result.Data, result.CreatedAt)
		findings = append(findings, extractedFindings...)
	}

	// Apply filters
	var filteredFindings []FindingItem
	for _, f := range findings {
		// Apply severity filter
		if severityFilter != "" && strings.ToLower(f.Severity) != severityFilter {
			continue
		}
		// Apply category filter
		if categoryFilter != "" && f.Category != categoryFilter {
			continue
		}
		filteredFindings = append(filteredFindings, f)
	}

	// Sort findings
	h.sortFindings(filteredFindings, sortBy, sortOrder)

	// Calculate pagination
	total := len(filteredFindings)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		start = 0
		end = 0
	}
	if end > total {
		end = total
	}

	// Get page slice
	var pageData []FindingItem
	if start < total {
		pageData = filteredFindings[start:end]
	} else {
		pageData = []FindingItem{}
	}

	c.JSON(http.StatusOK, FindingsListResponse{
		Data:     pageData,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// extractFindingsFromData extracts findings from the review result data
func (h *FindingsHandler) extractFindingsFromData(reviewID, repoURL string, data map[string]interface{}, createdAt time.Time) []FindingItem {
	var findings []FindingItem

	// Try to extract findings array from data
	findingsRaw, ok := data["findings"]
	if !ok {
		return findings
	}

	// Handle different types of findings
	var findingsList []map[string]interface{}
	switch f := findingsRaw.(type) {
	case []interface{}:
		for _, item := range f {
			if m, ok := item.(map[string]interface{}); ok {
				findingsList = append(findingsList, m)
			}
		}
	case []map[string]interface{}:
		findingsList = f
	}

	// Convert to FindingItem
	for _, finding := range findingsList {
		item := FindingItem{
			ReviewID:  reviewID,
			RepoURL:   repoURL,
			CreatedAt: createdAt,
		}

		// Extract severity
		if severity, ok := finding["severity"].(string); ok {
			item.Severity = severity
		} else {
			item.Severity = "info"
		}

		// Extract category
		if category, ok := finding["category"].(string); ok {
			item.Category = category
		}

		// Extract description
		if description, ok := finding["description"].(string); ok {
			item.Description = description
		}

		findings = append(findings, item)
	}

	return findings
}

// sortFindings sorts findings by the specified field and order
func (h *FindingsHandler) sortFindings(findings []FindingItem, sortBy, sortOrder string) {
	sort.Slice(findings, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "severity":
			// Sort by severity order (critical > high > medium > low > info)
			orderI := severityOrder[strings.ToLower(findings[i].Severity)]
			orderJ := severityOrder[strings.ToLower(findings[j].Severity)]
			// If severity not in map, treat as lowest priority
			if _, ok := severityOrder[strings.ToLower(findings[i].Severity)]; !ok {
				orderI = 5
			}
			if _, ok := severityOrder[strings.ToLower(findings[j].Severity)]; !ok {
				orderJ = 5
			}
			less = orderI < orderJ
		case "category":
			less = findings[i].Category < findings[j].Category
		default:
			// Default to severity order
			orderI := severityOrder[strings.ToLower(findings[i].Severity)]
			orderJ := severityOrder[strings.ToLower(findings[j].Severity)]
			less = orderI < orderJ
		}

		// For descending order, invert the comparison
		// Note: For severity, "desc" means most severe first (critical first)
		// which is actually ascending order in our severityOrder map
		if sortOrder == "desc" {
			// For severity, desc means critical first, so we don't invert
			if sortBy == "severity" {
				return less
			}
			return !less
		}

		// For severity with asc, we want least severe first (info first)
		if sortBy == "severity" {
			return !less
		}
		return less
	})
}
