package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
)

// TestAdminHandler_GetStats tests the GetStats endpoint
func TestAdminHandler_GetStats(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewAdminHandler(cfg, "", testStore)
	router.GET("/api/v1/admin/stats", handler.GetStats)

	req := CreateTestRequest("GET", "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}
}

// TestAdminHandler_GetStatus tests the GetStatus endpoint
func TestAdminHandler_GetStatus(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewAdminHandler(cfg, "", testStore)
	router.GET("/api/v1/admin/status", handler.GetStatus)

	req := CreateTestRequest("GET", "/api/v1/admin/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var status ServerStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if status.Version == "" {
		t.Error("Expected version to be set")
	}
}

// TestAdminHandler_GetAppMeta tests the GetAppMeta endpoint
func TestAdminHandler_GetAppMeta(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewAdminHandler(cfg, "", testStore)
	router.GET("/api/v1/meta", handler.GetAppMeta)

	req := CreateTestRequest("GET", "/api/v1/meta", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var meta AppMetaResponse
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if meta.Name != "VerustCode" {
		t.Errorf("Expected name 'VerustCode', got '%s'", meta.Name)
	}
}

// TestRulesHandler_ValidateRule tests the ValidateRule endpoint with valid content
func TestRulesHandler_ValidateRule(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.POST("/api/v1/admin/rules/validate", handler.ValidateRule)

	validYAML := `version: "1.0"
rules:
  - id: test-rule
    description: Test rule
`
	body := map[string]string{"content": validYAML}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/rules/validate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestRulesHandler_ValidateRule_Invalid tests the ValidateRule endpoint with invalid content
func TestRulesHandler_ValidateRule_Invalid(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.POST("/api/v1/admin/rules/validate", handler.ValidateRule)

	invalidYAML := `invalid: yaml: content`
	body := map[string]string{"content": invalidYAML}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/rules/validate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	valid, ok := resp["valid"].(bool)
	if !ok || valid {
		t.Error("Expected valid to be false for invalid YAML")
	}
}

// TestRulesHandler_ValidateRule_MissingContent tests validation with missing content
func TestRulesHandler_ValidateRule_MissingContent(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.POST("/api/v1/admin/rules/validate", handler.ValidateRule)

	req, _ := http.NewRequest("POST", "/api/v1/admin/rules/validate", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestRulesHandler_ListRules tests the ListRules endpoint
func TestRulesHandler_ListRules(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.GET("/api/v1/admin/rules", handler.ListRules)

	req := CreateTestRequest("GET", "/api/v1/admin/rules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestRulesHandler_GetRule_NotFound tests getting a non-existent rule file
func TestRulesHandler_GetRule_NotFound(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.GET("/api/v1/admin/rules/:name", handler.GetRule)

	req := CreateTestRequest("GET", "/api/v1/admin/rules/non-existent.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestRulesHandler_GetRule_PathTraversal tests path traversal protection
func TestRulesHandler_GetRule_PathTraversal(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.GET("/api/v1/admin/rules/:name", handler.GetRule)

	// Test with a simple path traversal pattern that bypasses gin's built-in protection
	// Gin framework itself rejects URL-encoded path traversal with 404
	req := CreateTestRequest("GET", "/api/v1/admin/rules/..secret.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should reject path containing ".." pattern with 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for path containing '..', got %d", w.Code)
	}

	// Also test with URL-encoded path traversal (gin rejects this with 404)
	req2 := CreateTestRequest("GET", "/api/v1/admin/rules/..%2F..%2Fetc%2Fpasswd", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Gin framework rejects URL-encoded path traversal patterns itself (returns 404)
	// Either 400 (our handler) or 404 (gin's security) is acceptable
	if w2.Code != http.StatusBadRequest && w2.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404 for URL-encoded path traversal, got %d", w2.Code)
	}
}

// TestRepositoryHandler_ListRepositories tests the ListRepositories endpoint
func TestRepositoryHandler_ListRepositories(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewRepositoryHandler(testStore)
	router.GET("/api/v1/admin/repositories", handler.ListRepositories)

	req := CreateTestRequest("GET", "/api/v1/admin/repositories", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestRepositoryHandler_ParseRepoUrl tests the ParseRepoUrl endpoint
func TestRepositoryHandler_ParseRepoUrl(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantCode int
		provider string
		owner    string
		repo     string
	}{
		{
			name:     "github https url",
			url:      "https://github.com/owner/repo",
			wantCode: http.StatusOK,
			provider: "github",
			owner:    "owner",
			repo:     "repo",
		},
		{
			name:     "gitlab https url",
			url:      "https://gitlab.com/owner/project",
			wantCode: http.StatusOK,
			provider: "gitlab",
			owner:    "owner",
			repo:     "project",
		},
		{
			name:     "simple owner/repo format",
			url:      "owner/repo",
			wantCode: http.StatusOK,
			provider: "github",
			owner:    "owner",
			repo:     "repo",
		},
		{
			name:     "invalid format",
			url:      "invalid",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := SetupTestRouter()
			testStore, cleanup := store.SetupTestDB(t)
			defer cleanup()

			handler := NewRepositoryHandler(testStore)
			router.POST("/api/v1/admin/parse-repo-url", handler.ParseRepoUrl)

			body := map[string]string{"url": tt.url}
			jsonBody, _ := json.Marshal(body)

			req, _ := http.NewRequest("POST", "/api/v1/admin/parse-repo-url", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}

			if tt.wantCode == http.StatusOK {
				var resp ParseRepoUrlResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
				}
				if resp.Provider != tt.provider {
					t.Errorf("Expected provider '%s', got '%s'", tt.provider, resp.Provider)
				}
				if resp.Owner != tt.owner {
					t.Errorf("Expected owner '%s', got '%s'", tt.owner, resp.Owner)
				}
				if resp.Repo != tt.repo {
					t.Errorf("Expected repo '%s', got '%s'", tt.repo, resp.Repo)
				}
			}
		})
	}
}

// TestRulesHandler_ListReviewFiles tests the ListReviewFiles endpoint
func TestRulesHandler_ListReviewFiles(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.GET("/api/v1/admin/review-files", handler.ListReviewFiles)

	req := CreateTestRequest("GET", "/api/v1/admin/review-files", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestRulesHandler_GetRule_Success tests successfully getting a rule file
func TestRulesHandler_GetRule_Success(t *testing.T) {
	// Use actual config/reviews directory if it exists, otherwise skip
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		// Create the directory for testing
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	// Create a test rule file
	testFileName := "test-rule-get.yaml"
	testContent := `version: "1.0"
rules:
  - id: test-rule
    description: Test rule
`
	testFilePath := filepath.Join(testReviewsDir, testFileName)
	if err := os.WriteFile(testFilePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.GET("/api/v1/admin/rules/:name", handler.GetRule)

	req := CreateTestRequest("GET", "/api/v1/admin/rules/"+testFileName, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if content, ok := response["content"].(string); !ok || content != testContent {
		t.Error("Response should contain correct content")
	}
	if _, ok := response["hash"].(string); !ok {
		t.Error("Response should contain hash")
	}
}

// TestRulesHandler_SaveRule_Success tests successfully saving a rule file
func TestRulesHandler_SaveRule_Success(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	testFileName := "test-rule-save.yaml"
	testContent := `version: "1.0"
rules:
  - id: test-rule
    description: Test rule
`
	testFilePath := filepath.Join(testReviewsDir, testFileName)
	defer os.Remove(testFilePath)

	// Compute hash for the content (empty hash for new file)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("")))

	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.PUT("/api/v1/admin/rules/:name", handler.SaveRule)

	reqBody := map[string]string{
		"content": testContent,
		"hash":    hash,
	}
	req := CreateTestRequest("PUT", "/api/v1/admin/rules/"+testFileName, reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify file was created
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Error("Rule file should be created")
	}

	// Verify content
	savedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	if string(savedContent) != testContent {
		t.Errorf("Saved content mismatch: got %s, want %s", string(savedContent), testContent)
	}
}

// TestRulesHandler_SaveRule_HashConflict tests optimistic locking conflict
func TestRulesHandler_SaveRule_HashConflict(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	testFileName := "test-rule-conflict.yaml"
	originalContent := `version: "1.0"
rules:
  - id: test-rule
    description: Original rule
`
	testFilePath := filepath.Join(testReviewsDir, testFileName)
	if err := os.WriteFile(testFilePath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	newContent := `version: "1.0"
rules:
  - id: test-rule
    description: Updated rule
`
	// Use wrong hash (simulating file was modified by another user)
	wrongHash := "wrong-hash"

	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.PUT("/api/v1/admin/rules/:name", handler.SaveRule)

	reqBody := map[string]string{
		"content": newContent,
		"hash":    wrongHash,
	}
	req := CreateTestRequest("PUT", "/api/v1/admin/rules/"+testFileName, reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

// TestRulesHandler_SaveRule_InvalidYAML tests saving invalid YAML
func TestRulesHandler_SaveRule_InvalidYAML(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	testFileName := "test-rule-invalid.yaml"
	invalidContent := `invalid: yaml: content: [`
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("")))

	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.PUT("/api/v1/admin/rules/:name", handler.SaveRule)

	reqBody := map[string]string{
		"content": invalidContent,
		"hash":    hash,
	}
	req := CreateTestRequest("PUT", "/api/v1/admin/rules/"+testFileName, reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
		if _, exists := response["code"]; !exists {
			t.Error("Response should contain 'code' field")
		}
	}
}

// TestRulesHandler_SaveRule_MissingFields tests saving with missing fields
func TestRulesHandler_SaveRule_MissingFields(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.PUT("/api/v1/admin/rules/:name", handler.SaveRule)

	// Test with missing content
	reqBody := map[string]string{
		"hash": "some-hash",
	}
	req := CreateTestRequest("PUT", "/api/v1/admin/rules/test.yaml", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Test with missing hash
	reqBody = map[string]string{
		"content": "test content",
	}
	req = CreateTestRequest("PUT", "/api/v1/admin/rules/test.yaml", reqBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestRulesHandler_CreateRuleFile_Success tests successfully creating a rule file
func TestRulesHandler_CreateRuleFile_Success(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	// Create a source file to copy from
	sourceFileName := "default.example.yaml"
	sourceContent := `version: "1.0"
rules:
  - id: default-rule
    description: Default rule
`
	sourceFilePath := filepath.Join(testReviewsDir, sourceFileName)
	if err := os.WriteFile(sourceFilePath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	defer os.Remove(sourceFilePath)

	newFileName := "new-rule-create.yaml"
	newFilePath := filepath.Join(testReviewsDir, newFileName)
	defer os.Remove(newFilePath)

	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.POST("/api/v1/admin/rules", handler.CreateRuleFile)

	reqBody := map[string]string{
		"name":      newFileName,
		"copy_from": sourceFileName,
	}
	req := CreateTestRequest("POST", "/api/v1/admin/rules", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify file was created
	if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
		t.Error("New rule file should be created")
	}

	// Verify content matches source
	newContent, err := os.ReadFile(newFilePath)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}
	if string(newContent) != sourceContent {
		t.Errorf("New file content should match source")
	}
}

// TestRulesHandler_CreateRuleFile_AlreadyExists tests creating a file that already exists
func TestRulesHandler_CreateRuleFile_AlreadyExists(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	testFileName := "existing-rule-create.yaml"
	testFilePath := filepath.Join(testReviewsDir, testFileName)
	if err := os.WriteFile(testFilePath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}
	defer os.Remove(testFilePath)

	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.POST("/api/v1/admin/rules", handler.CreateRuleFile)

	reqBody := map[string]string{
		"name": testFileName,
	}
	req := CreateTestRequest("POST", "/api/v1/admin/rules", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

// TestRulesHandler_CreateRuleFile_InvalidName tests creating with invalid filename
func TestRulesHandler_CreateRuleFile_InvalidName(t *testing.T) {
	router := SetupTestRouter()

	handler := NewRulesHandler()
	router.POST("/api/v1/admin/rules", handler.CreateRuleFile)

	// Test with path traversal attempt
	reqBody := map[string]string{
		"name": "../../etc/passwd",
	}
	req := CreateTestRequest("POST", "/api/v1/admin/rules", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestRepositoryHandler_CreateRepositoryConfig_Success tests successfully creating a repository config
func TestRepositoryHandler_CreateRepositoryConfig_Success(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	// Create a test review file
	testReviewFile := "test-review.yaml"
	testReviewFilePath := filepath.Join(testReviewsDir, testReviewFile)
	if err := os.WriteFile(testReviewFilePath, []byte("version: \"1.0\"\nrules: []"), 0644); err != nil {
		t.Fatalf("Failed to create test review file: %v", err)
	}
	defer os.Remove(testReviewFilePath)

	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewRepositoryHandler(testStore)
	router.POST("/api/v1/admin/repositories", handler.CreateRepositoryConfig)

	reqBody := map[string]string{
		"repo_url":    "https://github.com/test/repo",
		"review_file": testReviewFile,
		"description": "Test repository",
	}
	req := CreateTestRequest("POST", "/api/v1/admin/repositories", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, exists := response["id"]; !exists {
		t.Error("Response should contain 'id' field")
	}
}

// TestRepositoryHandler_CreateRepositoryConfig_AlreadyExists tests creating duplicate repository config
func TestRepositoryHandler_CreateRepositoryConfig_AlreadyExists(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	testReviewFile := "default.yaml"
	testReviewFilePath := filepath.Join(testReviewsDir, testReviewFile)
	if err := os.WriteFile(testReviewFilePath, []byte("version: \"1.0\"\nrules: []"), 0644); err != nil {
		t.Fatalf("Failed to create test review file: %v", err)
	}
	defer os.Remove(testReviewFilePath)

	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create existing config
	existingConfig := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo",
		ReviewFile: testReviewFile,
	}
	testStore.RepositoryConfig().Create(existingConfig)

	handler := NewRepositoryHandler(testStore)
	router.POST("/api/v1/admin/repositories", handler.CreateRepositoryConfig)

	reqBody := map[string]string{
		"repo_url":    "https://github.com/test/repo",
		"review_file": testReviewFile,
	}
	req := CreateTestRequest("POST", "/api/v1/admin/repositories", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestRepositoryHandler_CreateRepositoryConfig_InvalidRequest tests creating with invalid request
func TestRepositoryHandler_CreateRepositoryConfig_InvalidRequest(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewRepositoryHandler(testStore)
	router.POST("/api/v1/admin/repositories", handler.CreateRepositoryConfig)

	// Test with missing repo_url
	reqBody := map[string]string{
		"review_file": "default.yaml",
	}
	req := CreateTestRequest("POST", "/api/v1/admin/repositories", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestRepositoryHandler_UpdateRepositoryConfig_Success tests successfully updating a repository config
func TestRepositoryHandler_UpdateRepositoryConfig_Success(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	// Create test review files
	defaultFile := "default.yaml"
	customFile := "custom.yaml"
	defaultFilePath := filepath.Join(testReviewsDir, defaultFile)
	customFilePath := filepath.Join(testReviewsDir, customFile)
	if err := os.WriteFile(defaultFilePath, []byte("version: \"1.0\"\nrules: []"), 0644); err != nil {
		t.Fatalf("Failed to create default file: %v", err)
	}
	if err := os.WriteFile(customFilePath, []byte("version: \"1.0\"\nrules: []"), 0644); err != nil {
		t.Fatalf("Failed to create custom file: %v", err)
	}
	defer os.Remove(defaultFilePath)
	defer os.Remove(customFilePath)

	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create existing config
	existingConfig := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo",
		ReviewFile: defaultFile,
	}
	testStore.RepositoryConfig().Create(existingConfig)

	handler := NewRepositoryHandler(testStore)
	router.PUT("/api/v1/admin/repositories/:id", handler.UpdateRepositoryConfig)

	reqBody := map[string]string{
		"review_file": customFile,
		"description": "Updated description",
	}
	req := CreateTestRequest("PUT", fmt.Sprintf("/api/v1/admin/repositories/%d", existingConfig.ID), reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify update
	updated, err := testStore.RepositoryConfig().GetByID(existingConfig.ID)
	if err != nil {
		t.Fatalf("Failed to get updated config: %v", err)
	}
	if updated.ReviewFile != customFile {
		t.Errorf("ReviewFile should be updated to '%s', got '%s'", customFile, updated.ReviewFile)
	}
	if updated.Description != "Updated description" {
		t.Errorf("Description should be updated, got '%s'", updated.Description)
	}
}

// TestRepositoryHandler_UpdateRepositoryConfig_NotFound tests updating non-existent config
func TestRepositoryHandler_UpdateRepositoryConfig_NotFound(t *testing.T) {
	testReviewsDir := config.ReviewsDir
	if _, err := os.Stat(testReviewsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testReviewsDir, 0755); err != nil {
			t.Skipf("Cannot create reviews directory: %v", err)
		}
		defer os.RemoveAll(testReviewsDir)
	}

	testReviewFile := "custom.yaml"
	testReviewFilePath := filepath.Join(testReviewsDir, testReviewFile)
	if err := os.WriteFile(testReviewFilePath, []byte("version: \"1.0\"\nrules: []"), 0644); err != nil {
		t.Fatalf("Failed to create test review file: %v", err)
	}
	defer os.Remove(testReviewFilePath)

	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewRepositoryHandler(testStore)
	router.PUT("/api/v1/admin/repositories/:id", handler.UpdateRepositoryConfig)

	reqBody := map[string]string{
		"review_file": testReviewFile,
	}
	req := CreateTestRequest("PUT", "/api/v1/admin/repositories/999", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestRepositoryHandler_UpdateRepositoryConfig_InvalidID tests updating with invalid ID
func TestRepositoryHandler_UpdateRepositoryConfig_InvalidID(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewRepositoryHandler(testStore)
	router.PUT("/api/v1/admin/repositories/:id", handler.UpdateRepositoryConfig)

	reqBody := map[string]string{
		"review_file": "custom.yaml",
	}
	req := CreateTestRequest("PUT", "/api/v1/admin/repositories/invalid", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestRepositoryHandler_DeleteRepositoryConfig_Success tests successfully deleting a repository config
func TestRepositoryHandler_DeleteRepositoryConfig_Success(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create existing config
	existingConfig := &model.RepositoryReviewConfig{
		RepoURL:    "https://github.com/test/repo",
		ReviewFile: "default.yaml",
	}
	testStore.RepositoryConfig().Create(existingConfig)

	handler := NewRepositoryHandler(testStore)
	router.DELETE("/api/v1/admin/repositories/:id", handler.DeleteRepositoryConfig)

	req := CreateTestRequest("DELETE", fmt.Sprintf("/api/v1/admin/repositories/%d", existingConfig.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify deletion
	_, err := testStore.RepositoryConfig().GetByID(existingConfig.ID)
	if err == nil {
		t.Error("Config should be deleted")
	}
}

// TestRepositoryHandler_DeleteRepositoryConfig_NotFound tests deleting non-existent config
func TestRepositoryHandler_DeleteRepositoryConfig_NotFound(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	handler := NewRepositoryHandler(testStore)
	router.DELETE("/api/v1/admin/repositories/:id", handler.DeleteRepositoryConfig)

	req := CreateTestRequest("DELETE", "/api/v1/admin/repositories/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
