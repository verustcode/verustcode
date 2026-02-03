// Package handler provides test utilities for HTTP handler testing.
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/store"
)

// SetupTestRouter creates a Gin router for testing.
// It sets Gin to test mode and applies basic middleware.
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	return r
}

// CreateTestContext creates a test Gin context with a recorder.
// Returns the context and recorder for assertions.
func CreateTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// CreateTestRequest creates an HTTP request for testing.
func CreateTestRequest(method, url string, body interface{}) *http.Request {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	return req
}

// AssertJSONResponse asserts that the response has the expected JSON structure.
func AssertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedBody interface{}) {
	t.Helper()

	// Check status code
	if recorder.Code != expectedStatus {
		t.Errorf("Status code mismatch: got %d, want %d", recorder.Code, expectedStatus)
	}

	// Check content type
	contentType := recorder.Header().Get("Content-Type")
	if contentType != "" && contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type should be application/json, got %s", contentType)
	}

	// If expectedBody is provided, check response body
	if expectedBody != nil {
		var actual map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &actual); err != nil {
			t.Fatalf("Response should be valid JSON: %v", err)
		}

		expectedJSON, err := json.Marshal(expectedBody)
		if err != nil {
			t.Fatalf("Failed to marshal expected body: %v", err)
		}

		var expected map[string]interface{}
		if err := json.Unmarshal(expectedJSON, &expected); err != nil {
			t.Fatalf("Failed to unmarshal expected JSON: %v", err)
		}

		// Compare JSON structures (allowing for additional fields in actual)
		for key, expectedValue := range expected {
			actualValue, exists := actual[key]
			if !exists {
				t.Errorf("Response should contain key: %s", key)
				continue
			}
			if actualValue != expectedValue {
				t.Errorf("Value mismatch for key %s: got %v, want %v", key, actualValue, expectedValue)
			}
		}
	}
}

// AssertErrorResponse asserts that the response is an error response.
// The API uses a standard error format with 'code' and 'message' fields.
func AssertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()
	if recorder.Code != expectedStatus {
		t.Errorf("Status code mismatch: got %d, want %d", recorder.Code, expectedStatus)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "" && contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type should be application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}

	// Check for standard error response format (code + message)
	// or legacy format (error field)
	_, hasCode := response["code"]
	_, hasMessage := response["message"]
	_, hasError := response["error"]

	if !hasError && !(hasCode && hasMessage) {
		t.Error("Error response should contain either 'error' field or 'code' and 'message' fields")
	}
}

// MockStore creates a mock store for testing with in-memory storage.
// This implementation provides basic CRUD operations for testing purposes.
type MockStore struct {
	store.Store
	reviewStore           *MockReviewStore
	reportStore           *MockReportStore
	settingsStore         *MockSettingsStore
	repositoryConfigStore *MockRepositoryConfigStore
}

// NewMockStore creates a new mock store with in-memory storage.
func NewMockStore() *MockStore {
	return &MockStore{
		reviewStore:           NewMockReviewStore(),
		reportStore:           NewMockReportStore(),
		settingsStore:         NewMockSettingsStore(),
		repositoryConfigStore: NewMockRepositoryConfigStore(),
	}
}

// Review returns the mock review store.
func (m *MockStore) Review() store.ReviewStore {
	return m.reviewStore
}

// Report returns the mock report store.
func (m *MockStore) Report() store.ReportStore {
	return m.reportStore
}

// Settings returns the mock settings store.
func (m *MockStore) Settings() store.SettingsStore {
	return m.settingsStore
}

// RepositoryConfig returns the mock repository config store.
func (m *MockStore) RepositoryConfig() store.RepositoryConfigStore {
	return m.repositoryConfigStore
}

// DB returns nil for mock store (not used in tests).
func (m *MockStore) DB() *gorm.DB {
	return nil
}

// Transaction executes operations within a transaction (no-op for mock).
func (m *MockStore) Transaction(fn func(store.Store) error) error {
	return fn(m)
}
