package output

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
)

// mockStore is a mock implementation of store.Store
type mockStore struct {
	mock.Mock
	reviewStore *mockReviewStore
}

func (m *mockStore) Review() store.ReviewStore {
	return m.reviewStore
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

// mockReviewStore is a minimal mock implementation of store.ReviewStore
// Only implements methods needed for webhook tests
type mockReviewStore struct {
	mock.Mock
}

func (m *mockReviewStore) CreateWebhookLog(log *model.ReviewResultWebhookLog) error {
	args := m.Called(log)
	return args.Error(0)
}

// Implement all other ReviewStore methods as no-ops to satisfy the interface
func (m *mockReviewStore) Create(review *model.Review) error                       { return nil }
func (m *mockReviewStore) GetByID(id string) (*model.Review, error)                { return nil, nil }
func (m *mockReviewStore) GetByIDWithDetails(id string) (*model.Review, error)     { return nil, nil }
func (m *mockReviewStore) GetByIDWithRules(id string) (*model.Review, error)       { return nil, nil }
func (m *mockReviewStore) Update(review *model.Review) error                       { return nil }
func (m *mockReviewStore) Save(review *model.Review) error                         { return nil }
func (m *mockReviewStore) Delete(id string) error                                  { return nil }
func (m *mockReviewStore) UpdateStatus(id string, status model.ReviewStatus) error { return nil }
func (m *mockReviewStore) UpdateStatusWithError(id string, status model.ReviewStatus, errMsg string) error {
	return nil
}
func (m *mockReviewStore) UpdateStatusWithErrorAndCompletedAt(id string, status model.ReviewStatus, errMsg string) error {
	return nil
}
func (m *mockReviewStore) UpdateStatusToRunningIfPending(id string, startedAt time.Time) (bool, error) {
	return false, nil
}
func (m *mockReviewStore) UpdateStatusIfAllowed(id string, newStatus model.ReviewStatus, allowedStatuses []model.ReviewStatus) (int64, error) {
	return 0, nil
}
func (m *mockReviewStore) UpdateProgress(id string, currentRuleIndex int) error    { return nil }
func (m *mockReviewStore) UpdateCurrentRuleIndex(reviewID string, index int) error { return nil }
func (m *mockReviewStore) UpdateRepoPath(reviewID, repoPath string) error          { return nil }
func (m *mockReviewStore) UpdateMetadata(reviewID string, updates map[string]interface{}) error {
	return nil
}
func (m *mockReviewStore) IncrementRetryCount(id string) error { return nil }
func (m *mockReviewStore) List(statusFilter string, limit, offset int) ([]model.Review, int64, error) {
	return nil, 0, nil
}
func (m *mockReviewStore) ListByRepository(repoURL string, limit, offset int) ([]model.Review, int64, error) {
	return nil, 0, nil
}
func (m *mockReviewStore) ListByStatus(status model.ReviewStatus) ([]model.Review, error) {
	return nil, nil
}
func (m *mockReviewStore) ListPendingOrRunning() ([]model.Review, error) { return nil, nil }
func (m *mockReviewStore) GetByPRURLAndCommit(prURL, commitSHA string) (*model.Review, error) {
	return nil, nil
}
func (m *mockReviewStore) CreateRule(rule *model.ReviewRule) error         { return nil }
func (m *mockReviewStore) BatchCreateRules(rules []model.ReviewRule) error { return nil }
func (m *mockReviewStore) GetRuleByID(id uint) (*model.ReviewRule, error)  { return nil, nil }
func (m *mockReviewStore) GetRulesByReviewID(reviewID string) ([]model.ReviewRule, error) {
	return nil, nil
}
func (m *mockReviewStore) UpdateRule(rule *model.ReviewRule) error                 { return nil }
func (m *mockReviewStore) UpdateRuleStatus(id uint, status model.RuleStatus) error { return nil }
func (m *mockReviewStore) UpdateRuleStatusWithError(id uint, status model.RuleStatus, errMsg string) error {
	return nil
}
func (m *mockReviewStore) CreateRun(run *model.ReviewRuleRun) error         { return nil }
func (m *mockReviewStore) GetRunByID(id uint) (*model.ReviewRuleRun, error) { return nil, nil }
func (m *mockReviewStore) GetRunsByRuleID(ruleID uint) ([]model.ReviewRuleRun, error) {
	return nil, nil
}
func (m *mockReviewStore) DeleteReviewRuleRunsByRuleID(ruleID uint) error        { return nil }
func (m *mockReviewStore) UpdateRun(run *model.ReviewRuleRun) error              { return nil }
func (m *mockReviewStore) UpdateRunStatus(id uint, status model.RunStatus) error { return nil }
func (m *mockReviewStore) CreateResult(result *model.ReviewResult) error         { return nil }
func (m *mockReviewStore) DeleteReviewResultsByRuleID(ruleID uint) error         { return nil }
func (m *mockReviewStore) GetResultsByRuleID(ruleID uint) ([]model.ReviewResult, error) {
	return nil, nil
}
func (m *mockReviewStore) GetResultsByReviewID(reviewID string) ([]model.ReviewResult, error) {
	return nil, nil
}
func (m *mockReviewStore) UpdateWebhookLog(log *model.ReviewResultWebhookLog) error { return nil }
func (m *mockReviewStore) GetPendingWebhookLogs() ([]model.ReviewResultWebhookLog, error) {
	return nil, nil
}
func (m *mockReviewStore) CountByStatusAndDateRange(status model.ReviewStatus, start, end time.Time) (int64, error) {
	return 0, nil
}
func (m *mockReviewStore) GetReviewsWithResultsByRepository(repoURL string, limit, offset int) ([]model.Review, error) {
	return nil, nil
}
func (m *mockReviewStore) CountAll() (int64, error)                                   { return 0, nil }
func (m *mockReviewStore) CountCreatedAfter(start time.Time) (int64, error)           { return 0, nil }
func (m *mockReviewStore) CountByStatusOnly(status model.ReviewStatus) (int64, error) { return 0, nil }
func (m *mockReviewStore) CountByStatusAndCompletedAfter(status model.ReviewStatus, start time.Time) (int64, error) {
	return 0, nil
}
func (m *mockReviewStore) CountCompletedOrFailedAfter(start time.Time) (int64, error) { return 0, nil }
func (m *mockReviewStore) CountCompletedAfter(start time.Time) (int64, error)         { return 0, nil }
func (m *mockReviewStore) GetAverageDurationAfter(start time.Time) (float64, error)   { return 0, nil }
func (m *mockReviewStore) ListCompletedByRepoAndDateRange(repoURL string, start time.Time) ([]model.Review, error) {
	return nil, nil
}
func (m *mockReviewStore) GetReviewResultsByReviewIDs(reviewIDs []string) ([]model.ReviewResult, error) {
	return nil, nil
}
func (m *mockReviewStore) GetAllFindingsWithRepoInfo(repoURL string) ([]store.FindingWithRepoInfo, error) {
	return nil, nil
}
func (m *mockReviewStore) GetMaxRevisionByPRURL(prURL string) (int, error) { return 0, nil }
func (m *mockReviewStore) UpdateMergedAtByPRURL(prURL string, mergedAt time.Time) (int64, error) {
	return 0, nil
}
func (m *mockReviewStore) FindPreviousReviewResult(prURL, ruleID, currentReviewID string) (string, bool, error) {
	return "", false, nil
}
func (m *mockReviewStore) ResetReviewState(reviewID string, retryCount int) error { return nil }
func (m *mockReviewStore) ResetRuleState(ruleID string, reviewID string, ruleRetryCount, reviewRetryCount int) error {
	return nil
}

func TestNewWebhookChannel(t *testing.T) {
	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	channel := NewWebhookChannel(mockS)

	assert.Equal(t, "webhook", channel.Name())
	assert.Equal(t, DefaultWebhookTimeout, channel.timeout)
	assert.Equal(t, DefaultWebhookMaxRetries, channel.maxRetries)
	assert.Equal(t, "json", channel.format)
	assert.Equal(t, mockS, channel.store)
}

func TestNewWebhookChannelWithConfig(t *testing.T) {
	mockS := &mockStore{reviewStore: &mockReviewStore{}}

	tests := []struct {
		name         string
		url          string
		headerSecret string
		timeout      int
		maxRetries   int
		format       string
		expected     *WebhookChannel
	}{
		{
			name:         "full config",
			url:          "https://example.com/webhook",
			headerSecret: "secret123",
			timeout:      120,
			maxRetries:   10,
			format:       "markdown",
			expected: &WebhookChannel{
				url:          "https://example.com/webhook",
				headerSecret: "secret123",
				timeout:      120,
				maxRetries:   10,
				format:       "markdown",
				store:        mockS,
			},
		},
		{
			name:         "zero timeout uses default",
			url:          "https://example.com/webhook",
			headerSecret: "secret123",
			timeout:      0,
			maxRetries:   5,
			format:       "json",
			expected: &WebhookChannel{
				url:          "https://example.com/webhook",
				headerSecret: "secret123",
				timeout:      DefaultWebhookTimeout,
				maxRetries:   5,
				format:       "json",
				store:        mockS,
			},
		},
		{
			name:         "zero maxRetries uses default",
			url:          "https://example.com/webhook",
			headerSecret: "secret123",
			timeout:      60,
			maxRetries:   0,
			format:       "json",
			expected: &WebhookChannel{
				url:          "https://example.com/webhook",
				headerSecret: "secret123",
				timeout:      60,
				maxRetries:   DefaultWebhookMaxRetries,
				format:       "json",
				store:        mockS,
			},
		},
		{
			name:         "empty format uses default",
			url:          "https://example.com/webhook",
			headerSecret: "secret123",
			timeout:      60,
			maxRetries:   6,
			format:       "",
			expected: &WebhookChannel{
				url:          "https://example.com/webhook",
				headerSecret: "secret123",
				timeout:      60,
				maxRetries:   6,
				format:       "json",
				store:        mockS,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := NewWebhookChannelWithConfig(tt.url, tt.headerSecret, tt.timeout, tt.maxRetries, tt.format, mockS)
			assert.Equal(t, tt.expected.url, channel.url)
			assert.Equal(t, tt.expected.headerSecret, channel.headerSecret)
			assert.Equal(t, tt.expected.timeout, channel.timeout)
			assert.Equal(t, tt.expected.maxRetries, channel.maxRetries)
			assert.Equal(t, tt.expected.format, channel.format)
		})
	}
}

func TestWebhookChannel_Publish_NoURL(t *testing.T) {
	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	channel := NewWebhookChannel(mockS)
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test"},
	}
	opts := &PublishOptions{}

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
}

func TestWebhookChannel_Publish_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, WebhookContentType, r.Header.Get("Content-Type"))
		assert.Equal(t, "secret123", r.Header.Get(WebhookHeaderKey))

		var payload WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		assert.Equal(t, "test-reviewer", payload.RuleID)
		assert.NotEmpty(t, payload.Data)
		assert.NotEmpty(t, payload.Timestamp)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	mockS.reviewStore.On("CreateWebhookLog", mock.AnythingOfType("*model.ReviewResultWebhookLog")).Return(nil)

	channel := NewWebhookChannelWithConfig(server.URL, "secret123", 10, 3, "json", mockS)
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{
		ReviewID: "review-123",
		RepoURL:  "https://github.com/test/repo",
		PRNumber: 456,
	}

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
	mockS.reviewStore.AssertExpectations(t)
}

func TestWebhookChannel_Publish_RetryOnFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	mockS.reviewStore.On("CreateWebhookLog", mock.AnythingOfType("*model.ReviewResultWebhookLog")).Return(nil)

	channel := NewWebhookChannelWithConfig(server.URL, "secret123", 10, 5, "json", mockS)
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{}

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	mockS.reviewStore.AssertExpectations(t)
}

func TestWebhookChannel_Publish_MaxRetriesExhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	mockS.reviewStore.On("CreateWebhookLog", mock.AnythingOfType("*model.ReviewResultWebhookLog")).Return(nil)

	channel := NewWebhookChannelWithConfig(server.URL, "secret123", 1, 3, "json", mockS)
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{}

	err := channel.Publish(context.Background(), result, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook delivery failed after 3 attempts")
	mockS.reviewStore.AssertExpectations(t)
}

func TestWebhookChannel_Publish_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	mockS.reviewStore.On("CreateWebhookLog", mock.AnythingOfType("*model.ReviewResultWebhookLog")).Return(nil)

	channel := NewWebhookChannelWithConfig(server.URL, "secret123", 1, 3, "json", mockS)
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := channel.Publish(ctx, result, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestWebhookChannel_buildPayload(t *testing.T) {
	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	channel := NewWebhookChannel(mockS)

	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary", "findings": []string{"issue1"}},
	}
	opts := &PublishOptions{
		ReviewID: "review-123",
		RepoURL:  "https://github.com/test/repo",
		PRNumber: 456,
	}

	payload, err := channel.buildPayload(result, opts)
	assert.NoError(t, err)
	assert.Equal(t, "review-123", payload.ReviewID)
	assert.Equal(t, "test-reviewer", payload.RuleID)
	assert.Equal(t, "https://github.com/test/repo", payload.RepoURL)
	assert.Equal(t, 456, payload.PRNumber)
	assert.NotEmpty(t, payload.Timestamp)
	assert.NotEmpty(t, payload.Data)

	// Decode and verify data
	decodedData, err := base64.StdEncoding.DecodeString(payload.Data)
	assert.NoError(t, err)

	var data map[string]any
	err = json.Unmarshal(decodedData, &data)
	assert.NoError(t, err)
	assert.Equal(t, "Test summary", data["summary"])
}

func TestWebhookChannel_buildPayload_EmptyReviewerID(t *testing.T) {
	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	channel := NewWebhookChannel(mockS)

	result := &prompt.ReviewResult{
		ReviewerID: "",
		Data:       map[string]any{"summary": "Test"},
	}
	opts := &PublishOptions{}

	payload, err := channel.buildPayload(result, opts)
	assert.NoError(t, err)
	assert.Equal(t, "unknown", payload.RuleID)
}

func TestWebhookChannel_sendRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, WebhookContentType, r.Header.Get("Content-Type"))
		assert.Equal(t, "secret123", r.Header.Get(WebhookHeaderKey))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	channel := NewWebhookChannelWithConfig(server.URL, "secret123", 10, 3, "json", mockS)

	payload := []byte(`{"test": "data"}`)
	err := channel.sendRequest(context.Background(), payload)
	assert.NoError(t, err)
}

func TestWebhookChannel_sendRequest_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	channel := NewWebhookChannelWithConfig(server.URL, "secret123", 10, 3, "json", mockS)

	payload := []byte(`{"test": "data"}`)
	err := channel.sendRequest(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code 400")
}

func TestWebhookChannel_createDeliveryLog(t *testing.T) {
	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	channel := NewWebhookChannelWithConfig("https://example.com/webhook", "secret", 60, 6, "json", mockS)

	payload := &WebhookPayload{
		RuleID: "test-rule",
	}
	requestBody := `{"test": "data"}`

	log := channel.createDeliveryLog(payload, requestBody)
	assert.Equal(t, "test-rule", log.RuleID)
	assert.Equal(t, "https://example.com/webhook", log.WebhookURL)
	assert.Equal(t, requestBody, log.RequestBody)
	assert.Equal(t, model.WebhookStatusPending, log.Status)
	assert.Equal(t, 0, log.RetryCount)
}

func TestWebhookChannel_saveDeliveryLog(t *testing.T) {
	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	mockS.reviewStore.On("CreateWebhookLog", mock.AnythingOfType("*model.ReviewResultWebhookLog")).Return(nil)

	channel := NewWebhookChannel(mockS)
	log := &model.ReviewResultWebhookLog{
		RuleID:     "test-rule",
		WebhookURL: "https://example.com/webhook",
		Status:     model.WebhookStatusSuccess,
		RetryCount: 1,
	}

	channel.saveDeliveryLog(log)
	mockS.reviewStore.AssertExpectations(t)
}

func TestWebhookChannel_saveDeliveryLog_NilStore(t *testing.T) {
	channel := &WebhookChannel{
		store: nil,
	}
	log := &model.ReviewResultWebhookLog{
		RuleID: "test-rule",
	}

	// Should not panic
	channel.saveDeliveryLog(log)
}

func TestWebhookChannel_saveDeliveryLog_Error(t *testing.T) {
	mockS := &mockStore{reviewStore: &mockReviewStore{}}
	mockS.reviewStore.On("CreateWebhookLog", mock.AnythingOfType("*model.ReviewResultWebhookLog")).Return(assert.AnError)

	channel := NewWebhookChannel(mockS)
	log := &model.ReviewResultWebhookLog{
		RuleID: "test-rule",
	}

	// Should not return error, just log it
	channel.saveDeliveryLog(log)
	mockS.reviewStore.AssertExpectations(t)
}
