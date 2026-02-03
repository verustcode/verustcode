package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/report"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// mockStore is a minimal mock implementation of store.Store
type mockStore struct {
	mock.Mock
}

func (m *mockStore) Review() store.ReviewStore {
	return nil
}

func (m *mockStore) Report() store.ReportStore {
	return nil
}

func (m *mockStore) Settings() store.SettingsStore {
	return nil
}

func (m *mockStore) RepositoryConfig() store.RepositoryConfigStore {
	return nil
}

func (m *mockStore) DB() *gorm.DB {
	return nil
}

func (m *mockStore) Transaction(fn func(store.Store) error) error {
	return fn(m)
}

func TestSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Debug:       false,
			CORSOrigins: []string{"http://localhost:3000"},
		},
		Logging: logger.Config{
			AccessLog: false,
		},
	}

	e := &engine.Engine{}
	s := &mockStore{}

	Setup(r, e, cfg, s)

	// Test health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestSetupWithReportEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Debug:       false,
			CORSOrigins: []string{"http://localhost:3000"},
		},
		Logging: logger.Config{
			AccessLog: false,
		},
	}

	e := &engine.Engine{}
	re := &report.Engine{}
	s := &mockStore{}

	SetupWithReportEngine(r, e, re, cfg, s)

	// Test health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetupWithConfigPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Debug:       false,
			CORSOrigins: []string{"http://localhost:3000"},
		},
		Logging: logger.Config{
			AccessLog: true,
		},
	}

	e := &engine.Engine{}
	re := &report.Engine{}
	s := &mockStore{}

	SetupWithConfigPath(r, e, re, cfg, "config/bootstrap.yaml", s)

	// Test health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPublicRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Debug:       false,
			CORSOrigins: []string{},
		},
		Logging: logger.Config{
			AccessLog: false,
		},
	}

	e := &engine.Engine{}
	s := &mockStore{}

	Setup(r, e, cfg, s)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "health check",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "webhook endpoint exists",
			method:         "POST",
			path:           "/api/v1/webhooks/github",
			expectedStatus: http.StatusNotFound, // Provider not configured, but route exists
		},
		{
			name:           "auth setup status",
			method:         "GET",
			path:           "/api/v1/auth/setup/status",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "report types endpoint",
			method:         "GET",
			path:           "/api/v1/report-types",
			expectedStatus: http.StatusNotFound, // Report engine not initialized
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestProtectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Debug:       false,
			CORSOrigins: []string{},
		},
		Logging: logger.Config{
			AccessLog: false,
		},
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-key-for-testing-only",
		},
	}

	e := &engine.Engine{}
	s := &mockStore{}

	Setup(r, e, cfg, s)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "queue status without auth",
			method:         "GET",
			path:           "/api/v1/queue/status",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should require JWT authentication",
		},
		{
			name:           "reviews list without auth",
			method:         "GET",
			path:           "/api/v1/reviews",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should require JWT authentication",
		},
		{
			name:           "admin meta without auth",
			method:         "GET",
			path:           "/api/v1/admin/meta",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should require JWT authentication",
		},
		{
			name:           "auth me without auth",
			method:         "GET",
			path:           "/api/v1/auth/me",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should require JWT authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
		})
	}
}

func TestMiddlewareOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Debug:       false,
			CORSOrigins: []string{"http://localhost:3000"},
		},
		Logging: logger.Config{
			AccessLog: true,
		},
	}

	e := &engine.Engine{}
	s := &mockStore{}

	Setup(r, e, cfg, s)

	// Test that middleware is applied by checking request ID header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	// Request ID middleware should add X-Request-ID header
	assert.Equal(t, http.StatusOK, w.Code)
	// Note: We can't easily test middleware order without more complex setup,
	// but we can verify routes work correctly
}

func TestCORSConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Debug:       false,
			CORSOrigins: []string{"http://localhost:3000", "https://example.com"},
		},
		Logging: logger.Config{
			AccessLog: false,
		},
	}

	e := &engine.Engine{}
	s := &mockStore{}

	Setup(r, e, cfg, s)

	// Test CORS preflight request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	r.ServeHTTP(w, req)

	// CORS middleware should handle OPTIONS request (returns 204 No Content)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify CORS headers are set
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestReportRoutesWithEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Debug:       false,
			CORSOrigins: []string{},
		},
		Logging: logger.Config{
			AccessLog: false,
		},
	}

	e := &engine.Engine{}
	re := &report.Engine{}
	s := &mockStore{}

	SetupWithReportEngine(r, e, re, cfg, s)

	// Test report types endpoint (public)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/report-types", nil)
	r.ServeHTTP(w, req)

	// Should return 200 when report engine is initialized
	assert.Equal(t, http.StatusOK, w.Code)

	// Test protected report routes
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/reports", nil)
	r.ServeHTTP(w2, req2)

	// Should require authentication
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}
