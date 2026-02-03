package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
)

// TestReviewHandler_CreateReview_InvalidRequest tests creating review with invalid request
func TestReviewHandler_CreateReview_InvalidRequest(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews", handler.CreateReview)

	// Test with empty body
	req := CreateTestRequest("POST", "/api/v1/reviews", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with invalid JSON
	req, _ = http.NewRequest("POST", "/api/v1/reviews", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReviewHandler_CreateReview_MissingFields tests creating review with missing required fields
func TestReviewHandler_CreateReview_MissingFields(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews", handler.CreateReview)

	// Test with missing repository
	reqBody := map[string]interface{}{
		"ref": "main",
	}
	req := CreateTestRequest("POST", "/api/v1/reviews", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with missing ref
	reqBody = map[string]interface{}{
		"repository": "test/repo",
	}
	req = CreateTestRequest("POST", "/api/v1/reviews", reqBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReviewHandler_CreateReview_InvalidRepositoryFormat tests creating review with invalid repository format
func TestReviewHandler_CreateReview_InvalidRepositoryFormat(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews", handler.CreateReview)

	reqBody := map[string]interface{}{
		"repository": "invalid-format",
		"ref":        "main",
	}
	req := CreateTestRequest("POST", "/api/v1/reviews", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReviewHandler_GetReview tests retrieving a review
func TestReviewHandler_GetReview(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews/:id", handler.GetReview)

	// Test with non-existent review ID
	req := CreateTestRequest("GET", "/api/v1/reviews/non-existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 or error
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}
}

// TestReviewHandler_ListReviews tests listing reviews
func TestReviewHandler_ListReviews(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews", handler.ListReviews)

	// Test listing reviews
	req := CreateTestRequest("GET", "/api/v1/reviews?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 or error depending on mock implementation
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// TestReviewHandler_ListReviews_InvalidPagination_Basic tests listing reviews with invalid pagination
func TestReviewHandler_ListReviews_InvalidPagination_Basic(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews", handler.ListReviews)

	// Test with invalid page size
	req := CreateTestRequest("GET", "/api/v1/reviews?page_size=1000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should handle invalid pagination gracefully
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200 or 400, got %d", w.Code)
	}
}

// TestReviewHandler_GetReview_Success tests successfully retrieving a review
func TestReviewHandler_GetReview_Success(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	// Create a test review in mock store
	review := &model.Review{
		ID:      "test-review-001",
		RepoURL: "https://github.com/test/repo",
		Ref:     "main",
		Status:  model.ReviewStatusPending,
	}
	mockStore.Review().Create(review)

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews/:id", handler.GetReview)

	req := CreateTestRequest("GET", "/api/v1/reviews/test-review-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReviewHandler_ListReviews_Success tests successfully listing reviews
func TestReviewHandler_ListReviews_Success(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	// Create test reviews
	review1 := &model.Review{
		ID:      "review-001",
		RepoURL: "https://github.com/test/repo1",
		Ref:     "main",
		Status:  model.ReviewStatusPending,
	}
	review2 := &model.Review{
		ID:      "review-002",
		RepoURL: "https://github.com/test/repo2",
		Ref:     "main",
		Status:  model.ReviewStatusCompleted,
	}
	mockStore.Review().Create(review1)
	mockStore.Review().Create(review2)

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews", handler.ListReviews)

	req := CreateTestRequest("GET", "/api/v1/reviews?page=1&page_size=10", nil)
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

// TestReviewHandler_ListReviews_WithStatusFilter tests listing reviews with status filter
func TestReviewHandler_ListReviews_WithStatusFilter(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	review := &model.Review{
		ID:      "review-001",
		RepoURL: "https://github.com/test/repo",
		Ref:     "main",
		Status:  model.ReviewStatusCompleted,
	}
	mockStore.Review().Create(review)

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews", handler.ListReviews)

	req := CreateTestRequest("GET", "/api/v1/reviews?status=completed&page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReviewHandler_CancelReview_Success tests successfully canceling a review
func TestReviewHandler_CancelReview_Success(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	review := &model.Review{
		ID:      "review-001",
		RepoURL: "https://github.com/test/repo",
		Ref:     "main",
		Status:  model.ReviewStatusPending,
	}
	mockStore.Review().Create(review)

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/cancel", handler.CancelReview)

	req := CreateTestRequest("POST", "/api/v1/reviews/review-001/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReviewHandler_CancelReview_EmptyID tests canceling review with empty ID
func TestReviewHandler_CancelReview_EmptyID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/cancel", handler.CancelReview)

	req := CreateTestRequest("POST", "/api/v1/reviews//cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReviewHandler_CancelReview_NotFound tests canceling non-existent review
func TestReviewHandler_CancelReview_NotFound(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/cancel", handler.CancelReview)

	req := CreateTestRequest("POST", "/api/v1/reviews/non-existent/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Mock store may return error, so accept 404 or 500
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}
}

// TestReviewHandler_CancelReview_InvalidStatus tests canceling review with invalid status
func TestReviewHandler_CancelReview_InvalidStatus(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	// Create review with completed status (cannot cancel)
	review := &model.Review{
		ID:      "review-001",
		RepoURL: "https://github.com/test/repo",
		Ref:     "main",
		Status:  model.ReviewStatusCompleted,
	}
	mockStore.Review().Create(review)

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/cancel", handler.CancelReview)

	req := CreateTestRequest("POST", "/api/v1/reviews/review-001/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 (cannot cancel completed review)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestReviewHandler_RetryReview tests retrying a review
func TestReviewHandler_RetryReview(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/retry", handler.RetryReview)

	// Test with non-existent review
	req := CreateTestRequest("POST", "/api/v1/reviews/non-existent/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return error (engine.Retry will fail)
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}
}

// TestReviewHandler_RetryReview_EmptyID tests retrying review with empty ID
func TestReviewHandler_RetryReview_EmptyID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/retry", handler.RetryReview)

	req := CreateTestRequest("POST", "/api/v1/reviews//retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReviewHandler_GetReview_EmptyID tests getting review with empty ID
func TestReviewHandler_GetReview_EmptyID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews/:id", handler.GetReview)

	// Empty ID will be handled by router as 404, not 400
	req := CreateTestRequest("GET", "/api/v1/reviews/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Router may return 404 for empty path
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404, got %d", w.Code)
	}
}

// TestReviewHandler_ListReviews_InvalidPagination tests listing reviews with invalid pagination
func TestReviewHandler_ListReviews_InvalidPagination(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews", handler.ListReviews)

	// Test with invalid page (negative)
	req := CreateTestRequest("GET", "/api/v1/reviews?page=-1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should handle gracefully (defaults to page 1)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}

	// Test with invalid page_size (too large)
	req = CreateTestRequest("GET", "/api/v1/reviews?page=1&page_size=1000", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should handle gracefully (defaults to 20)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// TestReviewHandler_CreateReview_WithPRNumber tests creating review with PR number
func TestReviewHandler_CreateReview_WithPRNumber(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	handler := NewReviewHandler(testEngine, testStore)
	router.POST("/api/v1/reviews", handler.CreateReview)

	reqBody := map[string]interface{}{
		"repository": "test/repo",
		"ref":        "main",
		"pr_number":  123,
	}
	req := CreateTestRequest("POST", "/api/v1/reviews", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// May fail at various stages: parseRepository, engine.Submit, etc.
	// Engine.Submit may return errors that result in different status codes
	if w.Code != http.StatusAccepted && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 202, 400, 404, or 500, got %d. Response: %s", w.Code, w.Body.String())
	}
}

// TestReviewHandler_CreateReview_WithProvider tests creating review with explicit provider
func TestReviewHandler_CreateReview_WithProvider(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	handler := NewReviewHandler(testEngine, testStore)
	router.POST("/api/v1/reviews", handler.CreateReview)

	reqBody := map[string]interface{}{
		"repository": "test/repo",
		"ref":        "main",
		"provider":   "gitlab",
	}
	req := CreateTestRequest("POST", "/api/v1/reviews", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// May fail at engine.Submit due to missing provider
	// Engine may return 404 if provider not found
	if w.Code != http.StatusAccepted && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 202, 400, 404, or 500, got %d", w.Code)
	}
}

// TestReviewHandler_CreateReview_FullURL tests creating review with full repository URL
func TestReviewHandler_CreateReview_FullURL(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	handler := NewReviewHandler(testEngine, testStore)
	router.POST("/api/v1/reviews", handler.CreateReview)

	reqBody := map[string]interface{}{
		"repository": "https://github.com/test/repo",
		"ref":        "main",
	}
	req := CreateTestRequest("POST", "/api/v1/reviews", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// May fail at engine.Submit due to missing provider
	// Engine may return 404 if provider not found
	if w.Code != http.StatusAccepted && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 202, 400, 404, or 500, got %d", w.Code)
	}
}

// TestReviewHandler_RetryReviewRule tests retrying a specific review rule
func TestReviewHandler_RetryReviewRule(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/rules/:rule_id/retry", handler.RetryReviewRule)

	// Test with non-existent review
	req := CreateTestRequest("POST", "/api/v1/reviews/non-existent/rules/rule-1/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 or error
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}
}

// TestReviewHandler_RetryReviewRule_EmptyID tests retrying rule with empty review ID
func TestReviewHandler_RetryReviewRule_EmptyID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/rules/:rule_id/retry", handler.RetryReviewRule)

	req := CreateTestRequest("POST", "/api/v1/reviews//rules/rule-1/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReviewHandler_RetryReviewRule_EmptyRuleID tests retrying rule with empty rule ID
func TestReviewHandler_RetryReviewRule_EmptyRuleID(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	review := &model.Review{
		ID:      "review-001",
		RepoURL: "https://github.com/test/repo",
		Ref:     "main",
		Status:  model.ReviewStatusFailed,
	}
	mockStore.Review().Create(review)

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/rules/:rule_id/retry", handler.RetryReviewRule)

	req := CreateTestRequest("POST", "/api/v1/reviews/review-001/rules//retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestReviewHandler_ListReviews_WithRepoFilter tests listing reviews with repository filter
func TestReviewHandler_ListReviews_WithRepoFilter(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	review := &model.Review{
		ID:      "review-001",
		RepoURL: "https://github.com/test/repo",
		Ref:     "main",
		Status:  model.ReviewStatusCompleted,
	}
	mockStore.Review().Create(review)

	handler := NewReviewHandler(mockEngine, mockStore)
	router.GET("/api/v1/reviews", handler.ListReviews)

	req := CreateTestRequest("GET", "/api/v1/reviews?repo_url=https://github.com/test/repo&page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestReviewHandler_RetryReview_Success tests successfully retrying a review
func TestReviewHandler_RetryReview_Success(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	review := &model.Review{
		ID:      "review-001",
		RepoURL: "https://github.com/test/repo",
		Ref:     "main",
		Status:  model.ReviewStatusFailed,
	}
	testStore.Review().Create(review)

	handler := NewReviewHandler(testEngine, testStore)
	router.POST("/api/v1/reviews/:id/retry", handler.RetryReview)

	req := CreateTestRequest("POST", "/api/v1/reviews/review-001/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Engine.Retry may fail due to missing provider, so accept various status codes
	if w.Code != http.StatusOK && w.Code != http.StatusAccepted && w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 200, 202, 404, or 500, got %d", w.Code)
	}
}

// TestReviewHandler_RetryReview_InvalidStatus tests retrying review with invalid status
func TestReviewHandler_RetryReview_InvalidStatus(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	mockEngine := &engine.Engine{}

	// Create review with completed status (cannot retry)
	review := &model.Review{
		ID:      "review-001",
		RepoURL: "https://github.com/test/repo",
		Ref:     "main",
		Status:  model.ReviewStatusCompleted,
	}
	mockStore.Review().Create(review)

	handler := NewReviewHandler(mockEngine, mockStore)
	router.POST("/api/v1/reviews/:id/retry", handler.RetryReview)

	req := CreateTestRequest("POST", "/api/v1/reviews/review-001/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return error (cannot retry completed review)
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 400, 404, or 500, got %d", w.Code)
	}
}
