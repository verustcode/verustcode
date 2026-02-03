package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/verustcode/verustcode/internal/config"
)

// TestAuthHandler_Login_AdminDisabled tests login when admin is disabled
func TestAuthHandler_Login_AdminDisabled(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled: false,
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/login", handler.Login)

	reqBody := map[string]interface{}{
		"username": "admin",
		"password": "password",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusUnauthorized)
}

// TestAuthHandler_Login_InvalidRequest tests login with invalid request
func TestAuthHandler_Login_InvalidRequest(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled: true,
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/login", handler.Login)

	// Test with empty body
	req := CreateTestRequest("POST", "/api/v1/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with missing username
	reqBody := map[string]interface{}{
		"password": "password",
	}
	req = CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with missing password
	reqBody = map[string]interface{}{
		"username": "admin",
	}
	req = CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestAuthHandler_Login_InvalidUsername tests login with invalid username
func TestAuthHandler_Login_InvalidUsername(t *testing.T) {
	router := SetupTestRouter()
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:      true,
			Username:     "admin",
			PasswordHash: string(passwordHash),
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/login", handler.Login)

	reqBody := map[string]interface{}{
		"username": "wrong_user",
		"password": "password",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusUnauthorized)
}

// TestAuthHandler_Login_InvalidPassword tests login with invalid password
func TestAuthHandler_Login_InvalidPassword(t *testing.T) {
	router := SetupTestRouter()
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("correct_password"), bcrypt.DefaultCost)
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:      true,
			Username:     "admin",
			PasswordHash: string(passwordHash),
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/login", handler.Login)

	reqBody := map[string]interface{}{
		"username": "admin",
		"password": "wrong_password",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusUnauthorized)
}

// TestAuthHandler_Login_InvalidJSON tests login with invalid JSON
func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled: true,
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/login", handler.Login)

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestAuthHandler_ValidateToken tests ValidateToken method
func TestAuthHandler_ValidateToken_Invalid(t *testing.T) {
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled: true,
		},
	}

	handler := NewAuthHandler(cfg)

	// Test with invalid token
	_, err := handler.ValidateToken("invalid_token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

// TestAuthHandler_GetSetupStatus tests the GetSetupStatus endpoint
func TestAuthHandler_GetSetupStatus(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled: true,
		},
	}

	handler := NewAuthHandler(cfg)
	router.GET("/api/v1/auth/setup-status", handler.GetSetupStatus)

	req := CreateTestRequest("GET", "/api/v1/auth/setup-status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAuthHandler_Me_Unauthorized tests Me endpoint without token
func TestAuthHandler_Me_Unauthorized(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled: true,
		},
	}

	handler := NewAuthHandler(cfg)
	router.GET("/api/v1/auth/me", handler.Me)

	req := CreateTestRequest("GET", "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Without token, should return error
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 401 or 400, got %d", w.Code)
	}
}

// TestAuthHandler_Login_Success tests successful login
func TestAuthHandler_Login_Success(t *testing.T) {
	router := SetupTestRouter()
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:         true,
			Username:        "admin",
			PasswordHash:    string(passwordHash),
			JWTSecret:       "test-secret-key-for-jwt-token-generation",
			TokenExpiration: 24,
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/login", handler.Login)

	reqBody := map[string]interface{}{
		"username": "admin",
		"password": "password",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Token == "" {
		t.Error("Response should contain token")
	}
	if response.ExpiresAt == "" {
		t.Error("Response should contain expires_at")
	}
}

// TestAuthHandler_Login_RememberMe tests login with remember me enabled
func TestAuthHandler_Login_RememberMe(t *testing.T) {
	router := SetupTestRouter()
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:         true,
			Username:        "admin",
			PasswordHash:    string(passwordHash),
			JWTSecret:       "test-secret-key-for-jwt-token-generation",
			TokenExpiration: 24,
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/login", handler.Login)

	reqBody := map[string]interface{}{
		"username":    "admin",
		"password":    "password",
		"remember_me": true,
	}
	req := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Token == "" {
		t.Error("Response should contain token")
	}

	// Verify expiration is 7 days (168 hours) for remember me
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		t.Fatalf("Failed to parse expires_at: %v", err)
	}
	expectedExpiration := time.Now().Add(168 * time.Hour)
	diff := expiresAt.Sub(expectedExpiration)
	if diff < -time.Hour || diff > time.Hour {
		t.Errorf("Expected expiration around 7 days from now, got %v", expiresAt)
	}
}

// TestAuthHandler_Login_DefaultExpiration tests login with default expiration when not configured
func TestAuthHandler_Login_DefaultExpiration(t *testing.T) {
	router := SetupTestRouter()
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:      true,
			Username:     "admin",
			PasswordHash: string(passwordHash),
			JWTSecret:    "test-secret-key-for-jwt-token-generation",
			// TokenExpiration not set, should default to 24 hours
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/login", handler.Login)

	reqBody := map[string]interface{}{
		"username": "admin",
		"password": "password",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/login", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		t.Fatalf("Failed to parse expires_at: %v", err)
	}
	expectedExpiration := time.Now().Add(24 * time.Hour)
	diff := expiresAt.Sub(expectedExpiration)
	if diff < -time.Hour || diff > time.Hour {
		t.Errorf("Expected expiration around 24 hours from now, got %v", expiresAt)
	}
}

// TestAuthHandler_Me_Success tests successfully getting user info
func TestAuthHandler_Me_Success(t *testing.T) {
	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled: true,
		},
	}

	handler := NewAuthHandler(cfg)
	router.GET("/api/v1/auth/me", handler.Me)

	// Create a test context with username set (simulating JWT middleware)
	c, w := CreateTestContext()
	c.Set("username", "admin")
	handler.Me(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if username, ok := response["username"].(string); !ok || username != "admin" {
		t.Errorf("Expected username 'admin', got %v", response["username"])
	}
}

// TestAuthHandler_ValidateToken_Valid tests validating a valid token
func TestAuthHandler_ValidateToken_Valid(t *testing.T) {
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:   true,
			Username:  "admin",
			JWTSecret: "test-secret-key-for-jwt-token-validation",
		},
	}

	handler := NewAuthHandler(cfg)

	// Create a valid token
	claims := &Claims{
		Username: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "verustcode",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Admin.JWTSecret))
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Validate the token
	username, err := handler.ValidateToken(tokenString)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", username)
	}
}

// TestAuthHandler_ValidateToken_EmptySecret tests validation with empty JWT secret
func TestAuthHandler_ValidateToken_EmptySecret(t *testing.T) {
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:   true,
			JWTSecret: "", // Empty secret
		},
	}

	handler := NewAuthHandler(cfg)

	_, err := handler.ValidateToken("any-token")
	if err == nil {
		t.Error("Expected error for empty JWT secret")
	}
}

// TestAuthHandler_SetupPassword_Success tests successfully setting password for the first time
func TestAuthHandler_SetupPassword_Success(t *testing.T) {
	// Create temporary config file
	tmpDir, err := os.MkdirTemp("", "test_config_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	// Create minimal config file
	configContent := `admin:
  enabled: true
  username: admin
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:      true,
			Username:     "admin",
			PasswordHash: "", // No password set yet
		},
	}

	handler := NewAuthHandlerWithConfigPath(cfg, configPath)
	router.POST("/api/v1/auth/setup", handler.SetupPassword)

	reqBody := map[string]interface{}{
		"password":         "NewSecurePassword123!",
		"confirm_password": "NewSecurePassword123!",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/setup", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify password was saved to config file
	updatedConfig, err := config.LoadBootstrap(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	if updatedConfig.Admin.PasswordHash == "" {
		t.Error("Password hash should be set in config file")
	}

	// Verify password works
	if err := bcrypt.CompareHashAndPassword([]byte(updatedConfig.Admin.PasswordHash), []byte("NewSecurePassword123!")); err != nil {
		t.Error("Password hash should match the provided password")
	}
}

// TestAuthHandler_SetupPassword_PasswordMismatch tests setting password with mismatched passwords
func TestAuthHandler_SetupPassword_PasswordMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_config_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	configContent := `admin:
  enabled: true
  username: admin
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:      true,
			Username:     "admin",
			PasswordHash: "",
		},
	}

	handler := NewAuthHandlerWithConfigPath(cfg, configPath)
	router.POST("/api/v1/auth/setup", handler.SetupPassword)

	reqBody := map[string]interface{}{
		"password":         "Password123!",
		"confirm_password": "DifferentPassword123!",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/setup", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestAuthHandler_SetupPassword_AlreadySet tests setting password when password already exists
func TestAuthHandler_SetupPassword_AlreadySet(t *testing.T) {
	router := SetupTestRouter()
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("existing"), bcrypt.DefaultCost)
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:      true,
			Username:     "admin",
			PasswordHash: string(passwordHash), // Password already set
		},
	}

	handler := NewAuthHandler(cfg)
	router.POST("/api/v1/auth/setup", handler.SetupPassword)

	reqBody := map[string]interface{}{
		"password":         "NewPassword123!",
		"confirm_password": "NewPassword123!",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/setup", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 to hide API existence
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestAuthHandler_SetupPassword_InvalidPassword tests setting password with invalid complexity
func TestAuthHandler_SetupPassword_InvalidPassword(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_config_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	configContent := `admin:
  enabled: true
  username: admin
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	router := SetupTestRouter()
	cfg := &config.Config{
		Admin: &config.AdminConfig{
			Enabled:      true,
			Username:     "admin",
			PasswordHash: "",
		},
	}

	handler := NewAuthHandlerWithConfigPath(cfg, configPath)
	router.POST("/api/v1/auth/setup", handler.SetupPassword)

	// Test with too short password
	reqBody := map[string]interface{}{
		"password":         "short",
		"confirm_password": "short",
	}
	req := CreateTestRequest("POST", "/api/v1/auth/setup", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
