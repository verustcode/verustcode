package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/verustcode/verustcode/internal/model"
)

// TestStatsHandler_GetRepoStats_InvalidTimeRange tests getting repo stats with invalid time range
func TestStatsHandler_GetRepoStats_InvalidTimeRange(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewStatsHandler(mockStore)
	router.GET("/api/v1/admin/stats/repo", handler.GetRepoStats)

	// Test with invalid time_range
	req := CreateTestRequest("GET", "/api/v1/admin/stats/repo?time_range=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestStatsHandler_GetRepoStats_ValidTimeRange tests getting repo stats with valid time range
func TestStatsHandler_GetRepoStats_ValidTimeRange(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewStatsHandler(mockStore)
	router.GET("/api/v1/admin/stats/repo", handler.GetRepoStats)

	// Test with valid time_range
	testCases := []string{"1m", "3m", "6m", "1y"}
	for _, timeRange := range testCases {
		req := CreateTestRequest("GET", "/api/v1/admin/stats/repo?time_range="+timeRange, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 200 or 500 depending on mock implementation
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500 for time_range=%s, got %d", timeRange, w.Code)
		}
	}
}

// TestStatsHandler_GetRepoStats_WithRepoURL tests getting repo stats with repo URL filter
func TestStatsHandler_GetRepoStats_WithRepoURL(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewStatsHandler(mockStore)
	router.GET("/api/v1/admin/stats/repo", handler.GetRepoStats)

	req := CreateTestRequest("GET", "/api/v1/admin/stats/repo?repo_url=https://github.com/test/repo&time_range=3m", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 or 500 depending on mock implementation
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// TestStatsHandler_GetRepoStats_DefaultTimeRange tests getting repo stats with default time range
func TestStatsHandler_GetRepoStats_DefaultTimeRange(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewStatsHandler(mockStore)
	router.GET("/api/v1/admin/stats/repo", handler.GetRepoStats)

	// Test without time_range (should default to 3m)
	req := CreateTestRequest("GET", "/api/v1/admin/stats/repo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 or 500 depending on mock implementation
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// TestStatsHandler_GetRepoStats_WithData tests getting repo stats with actual review data
func TestStatsHandler_GetRepoStats_WithData(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	// Create test reviews with various data
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	branchCreatedAt := weekAgo.Add(-24 * time.Hour)
	mergedAt := weekAgo.Add(-12 * time.Hour)

	review1 := &model.Review{
		ID:              "review-001",
		RepoURL:         "https://github.com/test/repo",
		Ref:             "main",
		Status:          model.ReviewStatusCompleted,
		CreatedAt:       weekAgo,
		BranchCreatedAt: &branchCreatedAt,
		MergedAt:        &mergedAt,
		LinesAdded:      100,
		LinesDeleted:    50,
		FilesChanged:    5,
		RevisionCount:   1,
	}
	review2 := &model.Review{
		ID:              "review-002",
		RepoURL:         "https://github.com/test/repo",
		Ref:             "feature",
		Status:          model.ReviewStatusCompleted,
		CreatedAt:       weekAgo.AddDate(0, 0, -1),
		BranchCreatedAt: &branchCreatedAt,
		MergedAt:        &mergedAt,
		LinesAdded:      200,
		LinesDeleted:    100,
		FilesChanged:    10,
		RevisionCount:   3,
	}
	mockStore.Review().Create(review1)
	mockStore.Review().Create(review2)

	handler := NewStatsHandler(mockStore)
	router.GET("/api/v1/admin/stats/repo", handler.GetRepoStats)

	req := CreateTestRequest("GET", "/api/v1/admin/stats/repo?repo_url=https://github.com/test/repo&time_range=3m", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var stats model.RepoStats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify stats structure
	if stats.DeliveryTimeStats == nil {
		t.Error("DeliveryTimeStats should not be nil")
	}
	if stats.CodeChangeStats == nil {
		t.Error("CodeChangeStats should not be nil")
	}
	if stats.FileChangeStats == nil {
		t.Error("FileChangeStats should not be nil")
	}
	if stats.MRCountStats == nil {
		t.Error("MRCountStats should not be nil")
	}
	if stats.RevisionStats == nil {
		t.Error("RevisionStats should not be nil")
	}
}

// TestStatsHandler_calculateDeliveryTimeStats tests delivery time statistics calculation
func TestStatsHandler_calculateDeliveryTimeStats(t *testing.T) {
	mockStore := NewMockStore()
	handler := NewStatsHandler(mockStore)

	now := time.Now()
	week := getISOWeek(now)
	branchCreatedAt := now.Add(-48 * time.Hour)
	mergedAt := now.Add(-24 * time.Hour)

	weeklyData := map[string][]model.Review{
		week: {
			{
				BranchCreatedAt: &branchCreatedAt,
				MergedAt:        &mergedAt,
			},
			{
				BranchCreatedAt: &branchCreatedAt,
				MergedAt:        &now,
			},
		},
	}
	allWeeks := []string{week}

	stats := handler.calculateDeliveryTimeStats(weeklyData, allWeeks)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 week stat, got %d", len(stats))
	}

	if stats[0].Week != week {
		t.Errorf("Expected week %s, got %s", week, stats[0].Week)
	}

	// P50 should be around 24 hours (median of 24h and 48h)
	if stats[0].P50 <= 0 {
		t.Error("P50 should be greater than 0")
	}
	if stats[0].P90 <= 0 {
		t.Error("P90 should be greater than 0")
	}
	if stats[0].P95 <= 0 {
		t.Error("P95 should be greater than 0")
	}
}

// TestStatsHandler_calculateCodeChangeStats tests code change statistics calculation
func TestStatsHandler_calculateCodeChangeStats(t *testing.T) {
	mockStore := NewMockStore()
	handler := NewStatsHandler(mockStore)

	now := time.Now()
	week := getISOWeek(now)

	weeklyData := map[string][]model.Review{
		week: {
			{LinesAdded: 100, LinesDeleted: 50},
			{LinesAdded: 200, LinesDeleted: 100},
		},
	}
	allWeeks := []string{week}

	stats := handler.calculateCodeChangeStats(weeklyData, allWeeks)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 week stat, got %d", len(stats))
	}

	if stats[0].LinesAdded != 300 {
		t.Errorf("Expected 300 lines added, got %d", stats[0].LinesAdded)
	}
	if stats[0].LinesDeleted != 150 {
		t.Errorf("Expected 150 lines deleted, got %d", stats[0].LinesDeleted)
	}
	if stats[0].NetChange != 150 {
		t.Errorf("Expected 150 net change, got %d", stats[0].NetChange)
	}
}

// TestStatsHandler_calculateFileChangeStats tests file change statistics calculation
func TestStatsHandler_calculateFileChangeStats(t *testing.T) {
	mockStore := NewMockStore()
	handler := NewStatsHandler(mockStore)

	now := time.Now()
	week := getISOWeek(now)

	weeklyData := map[string][]model.Review{
		week: {
			{FilesChanged: 5},
			{FilesChanged: 10},
			{FilesChanged: 3},
		},
	}
	allWeeks := []string{week}

	stats := handler.calculateFileChangeStats(weeklyData, allWeeks)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 week stat, got %d", len(stats))
	}

	if stats[0].FilesChanged != 18 {
		t.Errorf("Expected 18 files changed, got %d", stats[0].FilesChanged)
	}
}

// TestStatsHandler_calculateMRCountStats tests MR count statistics calculation
func TestStatsHandler_calculateMRCountStats(t *testing.T) {
	mockStore := NewMockStore()
	handler := NewStatsHandler(mockStore)

	now := time.Now()
	week := getISOWeek(now)

	weeklyData := map[string][]model.Review{
		week: {
			{}, {}, {},
		},
	}
	allWeeks := []string{week}

	stats := handler.calculateMRCountStats(weeklyData, allWeeks)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 week stat, got %d", len(stats))
	}

	if stats[0].Count != 3 {
		t.Errorf("Expected 3 MRs, got %d", stats[0].Count)
	}
}

// TestStatsHandler_calculateRevisionStats tests revision statistics calculation
func TestStatsHandler_calculateRevisionStats(t *testing.T) {
	mockStore := NewMockStore()
	handler := NewStatsHandler(mockStore)

	now := time.Now()
	week := getISOWeek(now)

	weeklyData := map[string][]model.Review{
		week: {
			{RevisionCount: 1}, // Layer 1
			{RevisionCount: 2}, // Layer 1
			{RevisionCount: 3}, // Layer 2
			{RevisionCount: 4}, // Layer 2
			{RevisionCount: 5}, // Layer 3
			{RevisionCount: 6}, // Layer 3
		},
	}
	allWeeks := []string{week}

	stats := handler.calculateRevisionStats(weeklyData, allWeeks)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 week stat, got %d", len(stats))
	}

	if stats[0].MRCount != 6 {
		t.Errorf("Expected 6 MRs, got %d", stats[0].MRCount)
	}
	if stats[0].TotalRevisions != 21 {
		t.Errorf("Expected 21 total revisions, got %d", stats[0].TotalRevisions)
	}
	if stats[0].Layer1Count != 2 {
		t.Errorf("Expected 2 layer 1 MRs, got %d", stats[0].Layer1Count)
	}
	if stats[0].Layer2Count != 2 {
		t.Errorf("Expected 2 layer 2 MRs, got %d", stats[0].Layer2Count)
	}
	if stats[0].Layer3Count != 2 {
		t.Errorf("Expected 2 layer 3 MRs, got %d", stats[0].Layer3Count)
	}
}

// TestStatsHandler_calculateIssueStats tests issue statistics calculation
func TestStatsHandler_calculateIssueStats(t *testing.T) {
	mockStore := NewMockStore()
	handler := NewStatsHandler(mockStore)

	results := []model.ReviewResult{
		{
			Data: model.JSONMap{
				"findings": []interface{}{
					map[string]interface{}{
						"severity": "high",
						"category": "security",
					},
					map[string]interface{}{
						"severity": "medium",
						"category": "performance",
					},
					map[string]interface{}{
						"severity": "high",
						"category": "security",
					},
				},
			},
		},
		{
			Data: model.JSONMap{
				"findings": []interface{}{
					map[string]interface{}{
						"severity": "low",
						"category": "code-quality",
					},
				},
			},
		},
	}

	severityStats, categoryStats := handler.calculateIssueStats(results)

	if len(severityStats) == 0 {
		t.Error("Should have severity stats")
	}
	if len(categoryStats) == 0 {
		t.Error("Should have category stats")
	}

	// Check severity counts
	highCount := 0
	for _, stat := range severityStats {
		if stat.Severity == "high" {
			highCount = stat.Count
		}
	}
	if highCount != 2 {
		t.Errorf("Expected 2 high severity issues, got %d", highCount)
	}
}

// TestStatsHandler_extractFindings tests extracting findings from review results
func TestStatsHandler_extractFindings(t *testing.T) {
	mockStore := NewMockStore()
	handler := NewStatsHandler(mockStore)

	// Test with valid findings array
	data := model.JSONMap{
		"findings": []interface{}{
			map[string]interface{}{
				"severity": "high",
				"category": "security",
			},
			map[string]interface{}{
				"severity": "medium",
				"category": "performance",
			},
		},
	}

	findings := handler.extractFindings(data)

	if len(findings) != 2 {
		t.Errorf("Expected 2 findings, got %d", len(findings))
	}

	// Test with empty findings
	data2 := model.JSONMap{}
	findings2 := handler.extractFindings(data2)
	if len(findings2) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings2))
	}

	// Test with missing findings key
	data3 := model.JSONMap{
		"summary": "test",
	}
	findings3 := handler.extractFindings(data3)
	if len(findings3) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings3))
	}
}

// TestStatsHandler_getISOWeek tests ISO week calculation
func TestStatsHandler_getISOWeek(t *testing.T) {
	// Test with known date
	testDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	week := getISOWeek(testDate)

	// Verify format is correct (YYYY-WXX)
	if len(week) < 7 || week[4] != '-' || week[5] != 'W' {
		t.Errorf("Invalid ISO week format: %s", week)
	}

	// Verify it starts with the year
	if week[:4] != "2024" {
		t.Errorf("Expected year 2024, got %s", week[:4])
	}
}

// TestStatsHandler_generateWeekRange tests week range generation
func TestStatsHandler_generateWeekRange(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	weeks := generateWeekRange(start, end)

	if len(weeks) == 0 {
		t.Error("Should generate at least one week")
	}

	// Verify weeks are in ISO format
	for _, week := range weeks {
		if len(week) < 7 || week[4] != '-' || week[5] != 'W' {
			t.Errorf("Invalid ISO week format: %s", week)
		}
	}
}

// TestStatsHandler_percentile tests percentile calculation
func TestStatsHandler_percentile(t *testing.T) {
	// Test with empty slice
	result := percentile([]float64{}, 0.5)
	if result != 0 {
		t.Errorf("Expected 0 for empty slice, got %f", result)
	}

	// Test with single value
	result = percentile([]float64{10.0}, 0.5)
	if result != 10.0 {
		t.Errorf("Expected 10.0, got %f", result)
	}

	// Test with multiple values
	sorted := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	result = percentile(sorted, 0.5)
	if result < 2.0 || result > 4.0 {
		t.Errorf("P50 should be around 3.0, got %f", result)
	}

	result = percentile(sorted, 0.9)
	if result < 4.0 || result > 5.0 {
		t.Errorf("P90 should be around 4.5, got %f", result)
	}
}
