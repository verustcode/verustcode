package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSettingsHandler_GetAllSettings tests getting all settings
func TestSettingsHandler_GetAllSettings(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.GET("/api/admin/settings", handler.GetAllSettings)

	req := CreateTestRequest("GET", "/api/admin/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, exists := response["settings"]; !exists {
		t.Error("Response should contain 'settings' field")
	}
}

// TestSettingsHandler_GetSettingsByCategory tests getting settings by category
func TestSettingsHandler_GetSettingsByCategory(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.GET("/api/admin/settings/:category", handler.GetSettingsByCategory)

	req := CreateTestRequest("GET", "/api/admin/settings/git", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if category, ok := response["category"].(string); !ok || category != "git" {
		t.Errorf("Expected category 'git', got %v", response["category"])
	}
	if _, exists := response["settings"]; !exists {
		t.Error("Response should contain 'settings' field")
	}
}

// TestSettingsHandler_GetSettingsByCategory_InvalidCategory tests invalid category
func TestSettingsHandler_GetSettingsByCategory_InvalidCategory(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.GET("/api/admin/settings/:category", handler.GetSettingsByCategory)

	req := CreateTestRequest("GET", "/api/admin/settings/invalid_category", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestSettingsHandler_UpdateSettingsByCategory tests updating settings by category
func TestSettingsHandler_UpdateSettingsByCategory(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.PUT("/api/admin/settings/:category", handler.UpdateSettingsByCategory)

	reqBody := map[string]interface{}{
		"settings": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}
	req := CreateTestRequest("PUT", "/api/admin/settings/app", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success=true")
	}
}

// TestSettingsHandler_UpdateSettingsByCategory_InvalidCategory tests updating with invalid category
func TestSettingsHandler_UpdateSettingsByCategory_InvalidCategory(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.PUT("/api/admin/settings/:category", handler.UpdateSettingsByCategory)

	reqBody := map[string]interface{}{
		"settings": map[string]interface{}{
			"key1": "value1",
		},
	}
	req := CreateTestRequest("PUT", "/api/admin/settings/invalid_category", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestSettingsHandler_UpdateSettingsByCategory_InvalidRequest tests invalid request body
func TestSettingsHandler_UpdateSettingsByCategory_InvalidRequest(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.PUT("/api/admin/settings/:category", handler.UpdateSettingsByCategory)

	// Test with empty body
	req := CreateTestRequest("PUT", "/api/admin/settings/app", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with invalid JSON
	req, _ = http.NewRequest("PUT", "/api/admin/settings/app", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestSettingsHandler_ApplySettings tests applying all settings at once
func TestSettingsHandler_ApplySettings(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.POST("/api/admin/settings/apply", handler.ApplySettings)

	reqBody := map[string]interface{}{
		"settings": map[string]map[string]interface{}{
			"app": {
				"key1": "value1",
			},
			"git": {
				"key2": "value2",
			},
		},
	}
	req := CreateTestRequest("POST", "/api/admin/settings/apply", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success=true")
	}
}

// TestSettingsHandler_ApplySettings_InvalidRequest tests invalid request body
func TestSettingsHandler_ApplySettings_InvalidRequest(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.POST("/api/admin/settings/apply", handler.ApplySettings)

	// Test with empty body
	req := CreateTestRequest("POST", "/api/admin/settings/apply", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestSettingsHandler_UpdateSettingsByCategory_GitCategory tests Git settings update
func TestSettingsHandler_UpdateSettingsByCategory_GitCategory(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()

	handler := NewSettingsHandler(mockStore)
	router.PUT("/api/admin/settings/:category", handler.UpdateSettingsByCategory)

	reqBody := map[string]interface{}{
		"settings": map[string]interface{}{
			"providers": []map[string]interface{}{
				{
					"type":  "github",
					"url":   "https://github.com",
					"token": "test-token",
				},
			},
		},
	}
	req := CreateTestRequest("PUT", "/api/admin/settings/git", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

