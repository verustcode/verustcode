package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/verustcode/verustcode/pkg/errors"
)

// TestLogger_AccessLogEnabled tests Logger middleware with accessLog enabled
func TestLogger_AccessLogEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &LoggerConfig{AccessLog: true}
	router.Use(Logger(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestLogger_AccessLogDisabled tests Logger middleware with accessLog disabled
func TestLogger_AccessLogDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &LoggerConfig{AccessLog: false}
	router.Use(Logger(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestLogger_NilConfig tests Logger middleware with nil config
func TestLogger_NilConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Logger(nil))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestLogger_ErrorLogging tests Logger middleware error logging
func TestLogger_ErrorLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Logger(nil))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestRecovery tests Recovery middleware panic recovery
func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery())
	router.GET("/test", func(c *gin.Context) {
		panic("test panic")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if code, ok := response["code"].(string); !ok || code != string(errors.ErrCodeInternal) {
		t.Errorf("Expected error code %s, got %v", errors.ErrCodeInternal, response["code"])
	}
}

// TestCORS_AllowedOrigin tests CORS middleware with allowed origin
func TestCORS_AllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	router.Use(CORS(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:3000" {
		t.Errorf("Expected Access-Control-Allow-Origin header 'http://localhost:3000', got %s", origin)
	}
}

// TestCORS_NotAllowedOrigin tests CORS middleware with not allowed origin
func TestCORS_NotAllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	allowedOrigins := []string{"http://localhost:3000"}
	router.Use(CORS(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("Expected no Access-Control-Allow-Origin header, got %s", origin)
	}
}

// TestCORS_OPTIONSRequest tests CORS middleware with OPTIONS request
func TestCORS_OPTIONSRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	allowedOrigins := []string{"http://localhost:3000"}
	router.Use(CORS(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

// TestCORS_OPTIONSRequest_NotAllowed tests CORS middleware with OPTIONS request from not allowed origin
func TestCORS_OPTIONSRequest_NotAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	allowedOrigins := []string{"http://localhost:3000"}
	router.Use(CORS(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

// TestRequestID_Generated tests RequestID middleware generating new ID
func TestRequestID_Generated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "request_id not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

// TestRequestID_FromHeader tests RequestID middleware using existing header
func TestRequestID_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		requestID, _ := c.Get("request_id")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	requestID := w.Header().Get("X-Request-ID")
	if requestID != "test-request-id-123" {
		t.Errorf("Expected X-Request-ID header 'test-request-id-123', got %s", requestID)
	}
}

// TestErrorHandler_DebugMode tests ErrorHandler middleware in debug mode
func TestErrorHandler_DebugMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler(true))
	router.GET("/test", func(c *gin.Context) {
		c.Error(errors.New(errors.ErrCodeValidation, "test error"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "test"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestErrorHandler_ProductionMode tests ErrorHandler middleware in production mode
func TestErrorHandler_ProductionMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler(false))
	router.GET("/test", func(c *gin.Context) {
		c.Error(errors.New(errors.ErrCodeInternal, "sensitive error details"))
		c.Abort()
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// In production mode, should hide error details
	if msg, ok := response["message"].(string); ok && msg == "sensitive error details" {
		t.Error("Expected error message to be hidden in production mode")
	}
}

// TestErrorHandler_AppError tests ErrorHandler middleware with AppError
func TestErrorHandler_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler(true))
	router.GET("/test", func(c *gin.Context) {
		appErr := errors.New(errors.ErrCodeValidation, "validation error")
		appErr.Details = "field 'name' is required"
		c.Error(appErr)
		c.Abort()
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if code, ok := response["code"].(string); !ok || code != string(errors.ErrCodeValidation) {
		t.Errorf("Expected error code %s, got %v", errors.ErrCodeValidation, response["code"])
	}
}

// MockTokenValidator is a mock implementation of TokenValidator for testing
type MockTokenValidator struct {
	validTokens map[string]string // token -> username
}

// NewMockTokenValidator creates a new mock token validator
func NewMockTokenValidator() *MockTokenValidator {
	return &MockTokenValidator{
		validTokens: make(map[string]string),
	}
}

// AddToken adds a valid token to the mock validator
func (m *MockTokenValidator) AddToken(token, username string) {
	m.validTokens[token] = username
}

// ValidateToken validates a token
func (m *MockTokenValidator) ValidateToken(token string) (string, error) {
	username, ok := m.validTokens[token]
	if !ok {
		return "", errors.New(errors.ErrCodeUnauthorized, "invalid token")
	}
	return username, nil
}

// TestJWTAuth_ValidToken tests JWTAuth middleware with valid token
func TestJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	validator := NewMockTokenValidator()
	validator.AddToken("valid-token-123", "testuser")
	router.Use(JWTAuth(validator))
	router.GET("/test", func(c *gin.Context) {
		username, _ := c.Get("username")
		c.JSON(http.StatusOK, gin.H{"username": username})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if username, ok := response["username"].(string); !ok || username != "testuser" {
		t.Errorf("Expected username 'testuser', got %v", response["username"])
	}
}

// TestJWTAuth_InvalidToken tests JWTAuth middleware with invalid token
func TestJWTAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	validator := NewMockTokenValidator()
	router.Use(JWTAuth(validator))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestJWTAuth_NoHeader tests JWTAuth middleware without Authorization header
func TestJWTAuth_NoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	validator := NewMockTokenValidator()
	router.Use(JWTAuth(validator))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestJWTAuth_InvalidFormat tests JWTAuth middleware with invalid Authorization format
func TestJWTAuth_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	validator := NewMockTokenValidator()
	router.Use(JWTAuth(validator))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
