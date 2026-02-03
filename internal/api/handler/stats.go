// Package handler provides HTTP handlers for the API.
package handler

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// StatsHandler handles statistics-related HTTP requests
type StatsHandler struct {
	store store.Store
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(s store.Store) *StatsHandler {
	return &StatsHandler{store: s}
}

// GetRepoStats handles GET /api/v1/admin/stats/repo
// Query parameters:
//   - repo_url: optional, filter by repository URL (empty = all repos)
//   - time_range: required, one of "1m" (1 month), "3m" (3 months), "6m" (6 months), "1y" (1 year)
func (h *StatsHandler) GetRepoStats(c *gin.Context) {
	repoURL := c.Query("repo_url")
	timeRange := c.Query("time_range")

	// Validate time_range parameter
	if timeRange == "" {
		timeRange = "3m" // default to 3 months
	}

	var startTime time.Time
	now := time.Now()

	switch timeRange {
	case "1m":
		startTime = now.AddDate(0, -1, 0)
	case "3m":
		startTime = now.AddDate(0, -3, 0)
	case "6m":
		startTime = now.AddDate(0, -6, 0)
	case "1y":
		startTime = now.AddDate(-1, 0, 0)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid time_range parameter. Must be one of: 1m, 3m, 6m, 1y",
		})
		return
	}

	// Fetch all reviews in the time range
	reviews, err := h.store.Review().ListCompletedByRepoAndDateRange(repoURL, startTime)
	if err != nil {
		logger.Error("Failed to fetch reviews for statistics",
			zap.Error(err),
			zap.String("repo_url", repoURL),
			zap.String("time_range", timeRange),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to fetch statistics data",
		})
		return
	}

	// Collect review IDs for fetching ReviewResults
	reviewIDs := make([]string, len(reviews))
	for i, r := range reviews {
		reviewIDs[i] = r.ID
	}

	// Fetch all review results for issue statistics
	reviewResults, err := h.store.Review().GetReviewResultsByReviewIDs(reviewIDs)
	if err != nil {
		logger.Warn("Failed to fetch review results for issue statistics",
			zap.Error(err),
		)
		// Continue without issue stats if this fails
	}

	// Calculate statistics
	stats := h.calculateStats(reviews, startTime, now)

	// Calculate issue statistics from review results
	stats.IssueSeverityStats, stats.IssueCategoryStats = h.calculateIssueStats(reviewResults)

	c.JSON(http.StatusOK, stats)
}

// calculateStats computes all statistics from the review data
func (h *StatsHandler) calculateStats(reviews []model.Review, startTime, endTime time.Time) model.RepoStats {
	// Group reviews by week
	weeklyData := make(map[string][]model.Review)

	for _, review := range reviews {
		week := getISOWeek(review.CreatedAt)
		weeklyData[week] = append(weeklyData[week], review)
	}

	// Get all weeks in the time range
	allWeeks := generateWeekRange(startTime, endTime)

	// Calculate statistics for each week
	deliveryTimeStats := h.calculateDeliveryTimeStats(weeklyData, allWeeks)
	codeChangeStats := h.calculateCodeChangeStats(weeklyData, allWeeks)
	fileChangeStats := h.calculateFileChangeStats(weeklyData, allWeeks)
	mrCountStats := h.calculateMRCountStats(weeklyData, allWeeks)
	revisionStats := h.calculateRevisionStats(weeklyData, allWeeks)

	return model.RepoStats{
		DeliveryTimeStats: deliveryTimeStats,
		CodeChangeStats:   codeChangeStats,
		FileChangeStats:   fileChangeStats,
		MRCountStats:      mrCountStats,
		RevisionStats:     revisionStats,
	}
}

// calculateDeliveryTimeStats calculates P50, P90, P95 delivery times per week
// Delivery time is calculated as MergedAt - BranchCreatedAt (MR lifecycle time)
func (h *StatsHandler) calculateDeliveryTimeStats(weeklyData map[string][]model.Review, allWeeks []string) []model.WeeklyDeliveryTime {
	stats := make([]model.WeeklyDeliveryTime, 0, len(allWeeks))

	for _, week := range allWeeks {
		reviews := weeklyData[week]

		// Calculate delivery times in hours
		// Delivery time = MergedAt - BranchCreatedAt (MR lifecycle time)
		var deliveryTimes []float64
		for _, review := range reviews {
			// Use MergedAt - BranchCreatedAt for MR lifecycle time
			if review.MergedAt != nil && review.BranchCreatedAt != nil {
				duration := review.MergedAt.Sub(*review.BranchCreatedAt)
				hours := duration.Hours()
				if hours > 0 {
					deliveryTimes = append(deliveryTimes, hours)
				}
			}
		}

		stat := model.WeeklyDeliveryTime{
			Week: week,
			P50:  0,
			P90:  0,
			P95:  0,
		}

		if len(deliveryTimes) > 0 {
			sort.Float64s(deliveryTimes)
			stat.P50 = percentile(deliveryTimes, 0.50)
			stat.P90 = percentile(deliveryTimes, 0.90)
			stat.P95 = percentile(deliveryTimes, 0.95)
		}

		stats = append(stats, stat)
	}

	return stats
}

// calculateCodeChangeStats calculates code changes per week
func (h *StatsHandler) calculateCodeChangeStats(weeklyData map[string][]model.Review, allWeeks []string) []model.WeeklyCodeChange {
	stats := make([]model.WeeklyCodeChange, 0, len(allWeeks))

	for _, week := range allWeeks {
		reviews := weeklyData[week]

		var linesAdded, linesDeleted int
		for _, review := range reviews {
			linesAdded += review.LinesAdded
			linesDeleted += review.LinesDeleted
		}

		stats = append(stats, model.WeeklyCodeChange{
			Week:         week,
			LinesAdded:   linesAdded,
			LinesDeleted: linesDeleted,
			NetChange:    linesAdded - linesDeleted,
		})
	}

	return stats
}

// calculateFileChangeStats calculates file changes per week
func (h *StatsHandler) calculateFileChangeStats(weeklyData map[string][]model.Review, allWeeks []string) []model.WeeklyFileChange {
	stats := make([]model.WeeklyFileChange, 0, len(allWeeks))

	for _, week := range allWeeks {
		reviews := weeklyData[week]

		var filesChanged int
		for _, review := range reviews {
			filesChanged += review.FilesChanged
		}

		stats = append(stats, model.WeeklyFileChange{
			Week:         week,
			FilesChanged: filesChanged,
		})
	}

	return stats
}

// calculateMRCountStats calculates MR count per week
func (h *StatsHandler) calculateMRCountStats(weeklyData map[string][]model.Review, allWeeks []string) []model.WeeklyMRCount {
	stats := make([]model.WeeklyMRCount, 0, len(allWeeks))

	for _, week := range allWeeks {
		reviews := weeklyData[week]

		stats = append(stats, model.WeeklyMRCount{
			Week:  week,
			Count: len(reviews),
		})
	}

	return stats
}

// Revision layer boundaries (hardcoded in backend)
const (
	revisionLayer1Min = 1
	revisionLayer1Max = 2
	revisionLayer2Min = 3
	revisionLayer2Max = 4
	revisionLayer3Min = 5
)

// Revision layer labels
const (
	revisionLayer1Label = "1-2次"
	revisionLayer2Label = "3-4次"
	revisionLayer3Label = "5+次"
)

// calculateRevisionStats calculates revision statistics per week
// RevisionCount represents how many times an MR was updated (opened=1, each sync +1)
// Also calculates layer distribution for revision count analysis
func (h *StatsHandler) calculateRevisionStats(weeklyData map[string][]model.Review, allWeeks []string) []model.WeeklyRevision {
	stats := make([]model.WeeklyRevision, 0, len(allWeeks))

	for _, week := range allWeeks {
		reviews := weeklyData[week]

		var totalRevisions int
		var layer1, layer2, layer3 int

		for _, review := range reviews {
			// Use RevisionCount field which tracks MR update count
			totalRevisions += review.RevisionCount

			// Count layer distribution
			switch {
			case review.RevisionCount >= revisionLayer1Min && review.RevisionCount <= revisionLayer1Max:
				layer1++
			case review.RevisionCount >= revisionLayer2Min && review.RevisionCount <= revisionLayer2Max:
				layer2++
			case review.RevisionCount >= revisionLayer3Min:
				layer3++
			}
		}

		avgRevisions := 0.0
		if len(reviews) > 0 {
			avgRevisions = float64(totalRevisions) / float64(len(reviews))
		}

		stats = append(stats, model.WeeklyRevision{
			Week:           week,
			AvgRevisions:   math.Round(avgRevisions*100) / 100, // round to 2 decimal places
			TotalRevisions: totalRevisions,
			MRCount:        len(reviews),
			Layer1Count:    layer1,
			Layer2Count:    layer2,
			Layer3Count:    layer3,
			Layer1Label:    revisionLayer1Label,
			Layer2Label:    revisionLayer2Label,
			Layer3Label:    revisionLayer3Label,
		})
	}

	return stats
}

// getISOWeek returns the ISO week string for a given time (format: "2024-W01")
func getISOWeek(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// generateWeekRange generates all ISO week strings between start and end times
func generateWeekRange(start, end time.Time) []string {
	weeks := make([]string, 0)
	weekMap := make(map[string]bool)

	// Start from the beginning of the start week
	current := start
	for current.Before(end) || current.Equal(end) {
		week := getISOWeek(current)
		if !weekMap[week] {
			weeks = append(weeks, week)
			weekMap[week] = true
		}
		current = current.AddDate(0, 0, 7) // move to next week
	}

	return weeks
}

// percentile calculates the percentile value from a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	if len(sorted) == 1 {
		return sorted[0]
	}

	// Calculate index
	index := p * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// calculateIssueStats extracts findings from review results and calculates statistics
// by severity and category dimensions
func (h *StatsHandler) calculateIssueStats(results []model.ReviewResult) ([]model.IssueSeverityStats, []model.IssueCategoryStats) {
	severityCount := make(map[string]int)
	categoryCount := make(map[string]int)

	// Define valid severity levels for ordering
	severityOrder := []string{"critical", "high", "medium", "low", "info"}

	// Iterate through all review results and extract findings
	for _, result := range results {
		// Extract findings from result.Data
		findings := h.extractFindings(result.Data)
		for _, finding := range findings {
			// Count by severity
			if severity, ok := finding["severity"].(string); ok && severity != "" {
				severityCount[severity]++
			}
			// Count by category
			if category, ok := finding["category"].(string); ok && category != "" {
				categoryCount[category]++
			}
		}
	}

	// Build severity stats in predefined order
	severityStats := make([]model.IssueSeverityStats, 0)
	for _, sev := range severityOrder {
		if count, exists := severityCount[sev]; exists && count > 0 {
			severityStats = append(severityStats, model.IssueSeverityStats{
				Severity: sev,
				Count:    count,
			})
		}
	}

	// Build category stats sorted by count (descending)
	categoryStats := make([]model.IssueCategoryStats, 0, len(categoryCount))
	for cat, count := range categoryCount {
		categoryStats = append(categoryStats, model.IssueCategoryStats{
			Category: cat,
			Count:    count,
		})
	}
	// Sort by count descending
	sort.Slice(categoryStats, func(i, j int) bool {
		return categoryStats[i].Count > categoryStats[j].Count
	})

	return severityStats, categoryStats
}

// extractFindings extracts the findings array from review result data
// The data structure follows the JSON schema: { "summary": "...", "findings": [...] }
func (h *StatsHandler) extractFindings(data model.JSONMap) []map[string]interface{} {
	findings := make([]map[string]interface{}, 0)

	// Try to extract findings array from data
	findingsRaw, ok := data["findings"]
	if !ok {
		return findings
	}

	// Handle different types of findings
	switch f := findingsRaw.(type) {
	case []interface{}:
		for _, item := range f {
			if m, ok := item.(map[string]interface{}); ok {
				findings = append(findings, m)
			}
		}
	case []map[string]interface{}:
		findings = f
	}

	return findings
}
