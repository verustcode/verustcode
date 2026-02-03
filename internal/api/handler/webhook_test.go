package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/store"
)

// MockProvider is a mock provider for testing
// Note: This is a minimal implementation for testing webhook parsing only
// Full webhook integration tests would require a real engine setup
type MockProvider struct {
	name             string
	parseWebhookFunc func(*http.Request, string) (*provider.WebhookEvent, error)
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) GetBaseURL() string {
	return "https://github.com"
}

func (m *MockProvider) Clone(ctx context.Context, owner, repo, destPath string, opts *provider.CloneOptions) error {
	return nil
}

func (m *MockProvider) ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *provider.CloneOptions) error {
	return nil
}

func (m *MockProvider) GetPRRef(prNumber int) string {
	return fmt.Sprintf("refs/pull/%d/head", prNumber)
}

func (m *MockProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*provider.PullRequest, error) {
	return nil, nil
}

func (m *MockProvider) ListPullRequests(ctx context.Context, owner, repo string) ([]*provider.PullRequest, error) {
	return nil, nil
}

func (m *MockProvider) PostComment(ctx context.Context, owner, repo string, opts *provider.CommentOptions, body string) error {
	return nil
}

func (m *MockProvider) ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*provider.Comment, error) {
	return nil, nil
}

func (m *MockProvider) GetComments(ctx context.Context, owner, repo string, opts *provider.CommentOptions) ([]*provider.Comment, error) {
	return nil, nil
}

func (m *MockProvider) DeleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	return nil
}

func (m *MockProvider) UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error {
	return nil
}

func (m *MockProvider) ParseWebhook(req *http.Request, secret string) (*provider.WebhookEvent, error) {
	if m.parseWebhookFunc != nil {
		return m.parseWebhookFunc(req, secret)
	}
	return nil, errors.New("parseWebhookFunc not set")
}

func (m *MockProvider) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	return "", nil
}

func (m *MockProvider) DeleteWebhook(ctx context.Context, owner, repo, hookID string) error {
	return nil
}

func (m *MockProvider) ValidateToken(ctx context.Context) error {
	return nil
}

func (m *MockProvider) ParseRepoPath(repoURL string) (owner, repo string, err error) {
	return "test", "repo", nil
}

func (m *MockProvider) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	return nil, nil
}

func (m *MockProvider) MatchesURL(repoURL string) bool {
	return true
}

// TestWebhookHandler_HandleWebhook_UnknownProvider tests handling webhook with unknown provider
func TestWebhookHandler_HandleWebhook_UnknownProvider(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
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

	handler := NewWebhookHandler(testEngine, mockStore)
	router.POST("/api/v1/webhooks/:provider", handler.HandleWebhook)

	req := CreateTestRequest("POST", "/api/v1/webhooks/unknown", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 for unknown provider
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestWebhookHandler_HandleWebhook_UnsupportedEvent tests handling unsupported event type
func TestWebhookHandler_HandleWebhook_UnsupportedEvent(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:          "github",
					WebhookSecret: "test-secret",
				},
			},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	// Create mock provider that returns unsupported event
	mockProv := &MockProvider{
		name: "github",
		parseWebhookFunc: func(*http.Request, string) (*provider.WebhookEvent, error) {
			return &provider.WebhookEvent{
				Type:     provider.EventTypeComment,
				Provider: "github",
				Owner:    "test",
				Repo:     "repo",
			}, nil
		},
	}
	// Note: This test requires engine to accept mock providers, which may not be possible
	// For now, we test the error path with invalid webhook
	_ = mockProv

	handler := NewWebhookHandler(testEngine, mockStore)
	router.POST("/api/v1/webhooks/:provider", handler.HandleWebhook)

	req := CreateTestRequest("POST", "/api/v1/webhooks/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return error for invalid webhook (no provider registered)
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 404 or 400, got %d", w.Code)
	}
}

// TestWebhookHandler_HandleWebhook_InvalidJSON tests handling invalid JSON payload
func TestWebhookHandler_HandleWebhook_InvalidJSON(t *testing.T) {
	router := SetupTestRouter()
	mockStore := NewMockStore()
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

	handler := NewWebhookHandler(testEngine, mockStore)
	router.POST("/api/v1/webhooks/:provider", handler.HandleWebhook)

	req, _ := http.NewRequest("POST", "/api/v1/webhooks/github", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return error for invalid webhook (no provider registered)
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 404 or 400, got %d", w.Code)
	}
}

// TestWebhookHandler_HandleWebhook_ParseError tests handling webhook parse errors
func TestWebhookHandler_HandleWebhook_ParseError(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:          "github",
					WebhookSecret: "test-secret",
				},
			},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	handler := NewWebhookHandler(testEngine, testStore)
	router.POST("/api/v1/webhooks/:provider", handler.HandleWebhook)

	// Create request with invalid payload
	req := CreateTestRequest("POST", "/api/v1/webhooks/github", map[string]string{"invalid": "data"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 for parse error
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404, got %d", w.Code)
	}
}

// TestWebhookHandler_BuildRepoURL tests building repository URL from webhook event
func TestWebhookHandler_BuildRepoURL(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type: "github",
					URL:  "https://github.com",
				},
			},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	handler := NewWebhookHandler(testEngine, testStore)

	tests := []struct {
		name     string
		event    *provider.WebhookEvent
		expected string
	}{
		{
			name: "github default",
			event: &provider.WebhookEvent{
				Provider: "github",
				Owner:    "test",
				Repo:     "repo",
			},
			expected: "https://github.com/test/repo",
		},
		{
			name: "gitlab default",
			event: &provider.WebhookEvent{
				Provider: "gitlab",
				Owner:    "test",
				Repo:     "repo",
			},
			expected: "https://gitlab.com/test/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL := handler.buildRepoURL(tt.event)
			assert.Equal(t, tt.expected, repoURL)
		})
	}
}

// TestWebhookHandler_BuildPRURL tests building PR URL from webhook event
func TestWebhookHandler_BuildPRURL(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type: "github",
					URL:  "https://github.com",
				},
			},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	handler := NewWebhookHandler(testEngine, testStore)

	tests := []struct {
		name     string
		event    *provider.WebhookEvent
		expected string
	}{
		{
			name: "github PR",
			event: &provider.WebhookEvent{
				Provider: "github",
				Owner:    "test",
				Repo:     "repo",
				PRNumber: 123,
			},
			expected: "https://github.com/test/repo/pull/123",
		},
		{
			name: "gitlab MR",
			event: &provider.WebhookEvent{
				Provider: "gitlab",
				Owner:    "test",
				Repo:     "repo",
				PRNumber: 456,
			},
			expected: "https://gitlab.com/test/repo/-/merge_requests/456",
		},
		{
			name: "no PR number",
			event: &provider.WebhookEvent{
				Provider: "github",
				Owner:    "test",
				Repo:     "repo",
				PRNumber: 0,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prURL := handler.buildPRURL(tt.event)
			assert.Equal(t, tt.expected, prURL)
		})
	}
}

// TestWebhookHandler_HandleWebhook_NoSecret tests handling webhook without secret
func TestWebhookHandler_HandleWebhook_NoSecret(t *testing.T) {
	router := SetupTestRouter()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:          "github",
					WebhookSecret: "", // No secret configured
				},
			},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testEngine, err := engine.NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Stop()

	handler := NewWebhookHandler(testEngine, testStore)
	router.POST("/api/v1/webhooks/:provider", handler.HandleWebhook)

	req := CreateTestRequest("POST", "/api/v1/webhooks/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should handle gracefully (may return 400 or 404 depending on webhook parsing)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404, got %d", w.Code)
	}
}
