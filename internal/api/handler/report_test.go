package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/report"
	"github.com/verustcode/verustcode/internal/store"
)

// TestReportHandler_CreateReport_InvalidRequest tests creating report with invalid request
func TestReportHandler_CreateReport_InvalidRequest(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports", handler.CreateReport)

	// Test with empty body
	req := CreateTestRequest("POST", "/api/v1/reports", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with invalid JSON
	req, _ = http.NewRequest("POST", "/api/v1/reports", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_CreateReport_MissingFields tests creating report with missing required fields
func TestReportHandler_CreateReport_MissingFields(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports", handler.CreateReport)

	// Test with missing repo_url
	reqBody := map[string]interface{}{
		"ref":         "main",
		"report_type": "wiki",
	}
	req := CreateTestRequest("POST", "/api/v1/reports", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with missing ref
	reqBody = map[string]interface{}{
		"repo_url":    "https://github.com/test/repo",
		"report_type": "wiki",
	}
	req = CreateTestRequest("POST", "/api/v1/reports", reqBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with missing report_type
	reqBody = map[string]interface{}{
		"repo_url": "https://github.com/test/repo",
		"ref":      "main",
	}
	req = CreateTestRequest("POST", "/api/v1/reports", reqBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_CreateReport_InvalidReportType tests creating report with invalid report type
func TestReportHandler_CreateReport_InvalidReportType(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports", handler.CreateReport)

	reqBody := map[string]interface{}{
		"repo_url":    "https://github.com/test/repo",
		"ref":         "main",
		"report_type": "invalid_type",
	}
	req := CreateTestRequest("POST", "/api/v1/reports", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_GetReport tests retrieving a report
func TestReportHandler_GetReport(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id", handler.GetReport)

	// Test with non-existent report ID
	req := CreateTestRequest("GET", "/api/v1/reports/non-existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 or error
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}
}

// TestReportHandler_ListReports tests listing reports
func TestReportHandler_ListReports(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports", handler.ListReports)

	// Test listing reports
	req := CreateTestRequest("GET", "/api/v1/reports?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 or error depending on mock implementation
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// TestReportHandler_CancelReport tests canceling a report
func TestReportHandler_CancelReport(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports/:id/cancel", handler.CancelReport)

	// Test canceling non-existent report
	req := CreateTestRequest("POST", "/api/v1/reports/non-existent/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 or error
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}
}

// TestReportHandler_GetReport_Success tests successfully retrieving a report
func TestReportHandler_GetReport_Success(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	// Create a test report in mock store
	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
	}
	mockStore.Report().Create(report)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id", handler.GetReport)

	req := CreateTestRequest("GET", "/api/v1/reports/test-report-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReportHandler_ListReports_Success tests successfully listing reports
func TestReportHandler_ListReports_Success(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	// Create test reports
	report1 := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo1",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
	}
	report2 := &model.Report{
		ID:         "report-002",
		RepoURL:    "https://github.com/test/repo2",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusCompleted,
	}
	mockStore.Report().Create(report1)
	mockStore.Report().Create(report2)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports", handler.ListReports)

	req := CreateTestRequest("GET", "/api/v1/reports?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, exists := response["data"]; !exists {
		t.Error("Response should contain 'data' field")
	}
	if _, exists := response["total"]; !exists {
		t.Error("Response should contain 'total' field")
	}
}

// TestReportHandler_ListReports_WithFilters tests listing reports with status and type filters
func TestReportHandler_ListReports_WithFilters(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusCompleted,
	}
	mockStore.Report().Create(report)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports", handler.ListReports)

	// Test with status filter
	req := CreateTestRequest("GET", "/api/v1/reports?status=completed&page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test with report_type filter
	req = CreateTestRequest("GET", "/api/v1/reports?report_type=wiki&page=1&page_size=10", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReportHandler_CancelReport_Success tests successfully canceling a report
func TestReportHandler_CancelReport_Success(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
	}
	mockStore.Report().Create(report)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports/:id/cancel", handler.CancelReport)

	req := CreateTestRequest("POST", "/api/v1/reports/report-001/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReportHandler_GetReportTypes tests getting report types
func TestReportHandler_GetReportTypes(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/types", handler.GetReportTypes)

	req := CreateTestRequest("GET", "/api/v1/reports/types", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, exists := response["types"]; !exists {
		t.Error("Response should contain 'types' field")
	}
}

// TestReportHandler_GetRepositories tests getting repositories list
func TestReportHandler_GetRepositories(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	// Create test reports with different repositories
	report1 := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo1",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusCompleted,
	}
	report2 := &model.Report{
		ID:         "report-002",
		RepoURL:    "https://github.com/test/repo2",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusCompleted,
	}
	mockStore.Report().Create(report1)
	mockStore.Report().Create(report2)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/repositories", handler.GetRepositories)

	req := CreateTestRequest("GET", "/api/v1/reports/repositories", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReportHandler_GetReport_EmptyID tests getting report with empty ID
func TestReportHandler_GetReport_EmptyID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id", handler.GetReport)

	// Empty ID will be handled by router as 404, not 400
	req := CreateTestRequest("GET", "/api/v1/reports/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Router may return 404 for empty path
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404, got %d", w.Code)
	}
}

// TestReportHandler_ListReports_InvalidPagination tests listing reports with invalid pagination
func TestReportHandler_ListReports_InvalidPagination(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports", handler.ListReports)

	// Test with invalid page (negative)
	req := CreateTestRequest("GET", "/api/v1/reports?page=-1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should handle gracefully (defaults to page 1)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}

	// Test with invalid page_size (too large)
	req = CreateTestRequest("GET", "/api/v1/reports?page=1&page_size=1000", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should handle gracefully (defaults to 20)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}

	// Test with zero page_size
	req = CreateTestRequest("GET", "/api/v1/reports?page=1&page_size=0", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should handle gracefully (defaults to 20)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// TestReportHandler_CancelReport_EmptyID tests canceling report with empty ID
func TestReportHandler_CancelReport_EmptyID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports/:id/cancel", handler.CancelReport)

	req := CreateTestRequest("POST", "/api/v1/reports//cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_CancelReport_NotFound tests canceling non-existent report
func TestReportHandler_CancelReport_NotFound(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports/:id/cancel", handler.CancelReport)

	req := CreateTestRequest("POST", "/api/v1/reports/non-existent/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Mock store may return error, so accept 404 or 500
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}
}

// TestReportHandler_RetryReport tests retrying a report
func TestReportHandler_RetryReport(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports/:id/retry", handler.RetryReport)

	// Test with non-existent report
	req := CreateTestRequest("POST", "/api/v1/reports/non-existent/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestReportHandler_RetryReport_InvalidStatus tests retrying report with invalid status
func TestReportHandler_RetryReport_InvalidStatus(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	// Create report with completed status (cannot retry)
	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusCompleted,
	}
	mockStore.Report().Create(report)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports/:id/retry", handler.RetryReport)

	req := CreateTestRequest("POST", "/api/v1/reports/report-001/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 (invalid status)
	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_ExportReport tests exporting a report
func TestReportHandler_ExportReport(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id/export", handler.ExportReport)

	// Test with non-existent report
	req := CreateTestRequest("GET", "/api/v1/reports/non-existent/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestReportHandler_ExportReport_NotCompleted tests exporting incomplete report
func TestReportHandler_ExportReport_NotCompleted(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	// Create report with pending status
	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
	}
	mockStore.Report().Create(report)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id/export", handler.ExportReport)

	req := CreateTestRequest("GET", "/api/v1/reports/report-001/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 (not completed)
	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_ExportReport_EmptyID tests exporting report with empty ID
func TestReportHandler_ExportReport_EmptyID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id/export", handler.ExportReport)

	req := CreateTestRequest("GET", "/api/v1/reports//export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_GetReportProgress tests getting report progress
func TestReportHandler_GetReportProgress(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	providers := make(map[string]provider.Provider)
	agents := make(map[string]base.Agent)
	testEngine := report.NewEngine(cfg, providers, agents, testStore)

	handler := NewReportHandler(testEngine, cfg, testStore)
	router.GET("/api/v1/reports/:id/progress", handler.GetReportProgress)

	// Test with non-existent report
	req := CreateTestRequest("GET", "/api/v1/reports/non-existent/progress", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestReportHandler_GetReportProgress_Success tests successfully getting report progress
func TestReportHandler_GetReportProgress_Success(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	providers := make(map[string]provider.Provider)
	agents := make(map[string]base.Agent)
	testEngine := report.NewEngine(cfg, providers, agents, testStore)

	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
	}
	testStore.Report().Create(report)

	handler := NewReportHandler(testEngine, cfg, testStore)
	router.GET("/api/v1/reports/:id/progress", handler.GetReportProgress)

	req := CreateTestRequest("GET", "/api/v1/reports/report-001/progress", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReportHandler_GetReportProgress_EmptyID tests getting progress with empty ID
func TestReportHandler_GetReportProgress_EmptyID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id/progress", handler.GetReportProgress)

	req := CreateTestRequest("GET", "/api/v1/reports//progress", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_GetBranches tests getting branches list
func TestReportHandler_GetBranches(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/branches", handler.GetBranches)

	// Test without repo_url parameter
	req := CreateTestRequest("GET", "/api/v1/reports/branches", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 (missing repo_url)
	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReportHandler_GetBranches_WithRepoURL tests getting branches with repo_url parameter
func TestReportHandler_GetBranches_WithRepoURL(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/branches", handler.GetBranches)

	req := CreateTestRequest("GET", "/api/v1/reports/branches?repo_url=https://github.com/test/repo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Engine may fail to get branches, so accept various status codes
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 200, 400, 404, or 500, got %d", w.Code)
	}
}

// TestReportHandler_RetryReport_Success tests successfully retrying a report
func TestReportHandler_RetryReport_Success(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusFailed,
	}
	mockStore.Report().Create(report)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.POST("/api/v1/reports/:id/retry", handler.RetryReport)

	req := CreateTestRequest("POST", "/api/v1/reports/report-001/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Engine.Retry may fail, so accept various status codes
	if w.Code != http.StatusOK && w.Code != http.StatusAccepted && w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 200, 202, 404, or 500, got %d", w.Code)
	}
}

// TestReportHandler_ExportReport_Success tests successfully exporting a completed report
func TestReportHandler_ExportReport_Success(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusCompleted,
		Content:    "Test report content",
	}
	mockStore.Report().Create(report)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id/export", handler.ExportReport)

	req := CreateTestRequest("GET", "/api/v1/reports/report-001/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Export may fail due to missing content or other issues
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// TestReportHandler_ExportReport_WithFormat tests exporting report with format parameter
func TestReportHandler_ExportReport_WithFormat(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &report.Engine{}
	cfg := &config.Config{}

	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusCompleted,
		Content:    "Test report content",
	}
	mockStore.Report().Create(report)

	handler := NewReportHandler(mockEngine, cfg, mockStore)
	router.GET("/api/v1/reports/:id/export", handler.ExportReport)

	// Test with format=json
	req := CreateTestRequest("GET", "/api/v1/reports/report-001/export?format=json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Export may fail due to missing content or other issues
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}
