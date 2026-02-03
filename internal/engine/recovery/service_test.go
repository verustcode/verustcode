package recovery

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine/task"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

func init() {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
}

// mockTaskEnqueuer implements TaskEnqueuer for testing
type mockTaskEnqueuer struct {
	enqueuedTasks          []*task.Task
	enqueuedAsRunning      []*task.Task
	enqueueReturn          bool
	enqueueAsRunningReturn bool
}

func (m *mockTaskEnqueuer) Enqueue(t *task.Task) bool {
	m.enqueuedTasks = append(m.enqueuedTasks, t)
	return m.enqueueReturn
}

func (m *mockTaskEnqueuer) EnqueueAsRunning(t *task.Task) bool {
	m.enqueuedAsRunning = append(m.enqueuedAsRunning, t)
	return m.enqueueAsRunningReturn
}

// mockProviderResolver implements ProviderResolver for testing
type mockProviderResolver struct {
	providers map[string]provider.Provider
	detectMap map[string]string
}

func (m *mockProviderResolver) Get(name string) provider.Provider {
	return m.providers[name]
}

func (m *mockProviderResolver) DetectFromURL(url string) string {
	if m.detectMap != nil {
		if provider, ok := m.detectMap[url]; ok {
			return provider
		}
	}
	// Default detection logic
	if len(url) > 0 {
		if url[:19] == "https://github.com" {
			return "github"
		}
		if url[:19] == "https://gitlab.com" {
			return "gitlab"
		}
	}
	return ""
}

// mockProvider implements provider.Provider for testing
type mockProvider struct {
	name     string
	parseErr error
	owner    string
	repo     string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) GetBaseURL() string {
	return "https://" + m.name + ".com"
}

func (m *mockProvider) Clone(ctx context.Context, owner, repo, destPath string, opts *provider.CloneOptions) error {
	return nil
}

func (m *mockProvider) ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *provider.CloneOptions) error {
	return nil
}

func (m *mockProvider) GetPRRef(prNumber int) string {
	return ""
}

func (m *mockProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*provider.PullRequest, error) {
	return nil, nil
}

func (m *mockProvider) ListPullRequests(ctx context.Context, owner, repo string) ([]*provider.PullRequest, error) {
	return nil, nil
}

func (m *mockProvider) PostComment(ctx context.Context, owner, repo string, opts *provider.CommentOptions, body string) error {
	return nil
}

func (m *mockProvider) ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*provider.Comment, error) {
	return nil, nil
}

func (m *mockProvider) DeleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	return nil
}

func (m *mockProvider) UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error {
	return nil
}

func (m *mockProvider) ParseWebhook(r *http.Request, secret string) (*provider.WebhookEvent, error) {
	return nil, nil
}

func (m *mockProvider) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	return "", nil
}

func (m *mockProvider) DeleteWebhook(ctx context.Context, owner, repo, webhookID string) error {
	return nil
}

func (m *mockProvider) ParseRepoPath(repoURL string) (owner, repo string, err error) {
	if m.parseErr != nil {
		return "", "", m.parseErr
	}
	return m.owner, m.repo, nil
}

func (m *mockProvider) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	return nil, nil
}

func (m *mockProvider) MatchesURL(repoURL string) bool {
	return true
}

func (m *mockProvider) ValidateToken(ctx context.Context) error {
	return nil
}

func TestNewService(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	require.NotNil(t, service)
	assert.Equal(t, cfg, service.cfg)
	assert.Equal(t, testStore, service.store)
	assert.Equal(t, mockResolver, service.providerResolver)
	assert.Equal(t, mockEnqueuer, service.taskEnqueuer)
}

func TestRecoverToQueue_NoReviews(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          true,
		enqueueAsRunningReturn: true,
	}
	mockResolver := &mockProviderResolver{}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Should not enqueue anything
	assert.Empty(t, mockEnqueuer.enqueuedTasks)
	assert.Empty(t, mockEnqueuer.enqueuedAsRunning)
}

func TestRecoverToQueue_WithPendingReviews(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          true,
		enqueueAsRunningReturn: true,
	}
	mockProvider := &mockProvider{
		name:  "github",
		owner: "test",
		repo:  "repo",
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	// Create a pending review (DSL config will be loaded dynamically in processTask)
	review := &model.Review{
		ID:        "test-review-001",
		Ref:       "main",
		CommitSHA: "abc123",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusPending,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Should enqueue the pending review
	assert.Len(t, mockEnqueuer.enqueuedTasks, 1)
	assert.Empty(t, mockEnqueuer.enqueuedAsRunning)
	assert.Equal(t, review.ID, mockEnqueuer.enqueuedTasks[0].Review.ID)
}

func TestRecoverToQueue_WithRunningReviews(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          true,
		enqueueAsRunningReturn: true,
	}
	mockProvider := &mockProvider{
		name:  "github",
		owner: "test",
		repo:  "repo",
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	// Create a running review (DSL config will be loaded dynamically in processTask)
	review := &model.Review{
		ID:        "test-review-002",
		Ref:       "main",
		CommitSHA: "def456",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusRunning,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Should enqueue as running
	assert.Empty(t, mockEnqueuer.enqueuedTasks)
	assert.Len(t, mockEnqueuer.enqueuedAsRunning, 1)
	assert.Equal(t, review.ID, mockEnqueuer.enqueuedAsRunning[0].Review.ID)
}

func TestRecoverToQueue_StoreError(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	// Close the database to cause an error
	cleanup()

	ctx := context.Background()
	// Should not panic, just log error
	service.RecoverToQueue(ctx)

	assert.Empty(t, mockEnqueuer.enqueuedTasks)
	assert.Empty(t, mockEnqueuer.enqueuedAsRunning)
}

func TestRecoverToQueue_ProviderNotFound(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          true,
		enqueueAsRunningReturn: true,
	}
	mockResolver := &mockProviderResolver{
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
		// No provider registered
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	review := &model.Review{
		ID:        "test-review-004",
		Ref:       "main",
		CommitSHA: "jkl012",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusPending,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Should not enqueue, should mark as failed
	assert.Empty(t, mockEnqueuer.enqueuedTasks)

	// Verify review was marked as failed
	updatedReview, err := testStore.Review().GetByID(review.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusFailed, updatedReview.Status)
}

func TestBuildRecoveryTask_Success(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockProvider := &mockProvider{
		name:  "github",
		owner: "test",
		repo:  "repo",
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	// Review without DSL config - config will be loaded dynamically in processTask
	review := &model.Review{
		ID:        "test-review-005",
		Ref:       "main",
		CommitSHA: "mno345",
		RepoURL:   "https://github.com/test/repo",
		PRNumber:  123,
	}

	ctx := context.Background()
	task := service.BuildRecoveryTask(ctx, review)

	require.NotNil(t, task)
	assert.Equal(t, review, task.Review)
	assert.Equal(t, "github", task.ProviderName)
	assert.Nil(t, task.ReviewRulesConfig) // Config is nil - loaded dynamically later
	assert.NotNil(t, task.Request)
	assert.Equal(t, "test", task.Request.Owner)
	assert.Equal(t, "repo", task.Request.RepoName)
	assert.Equal(t, review.Ref, task.Request.Ref)
	assert.Equal(t, review.CommitSHA, task.Request.CommitSHA)
	assert.Equal(t, review.PRNumber, task.Request.PRNumber)
}

func TestBuildRecoveryTask_ProviderNotFound(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
		// No provider registered
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	review := &model.Review{
		ID:        "test-review-008",
		Ref:       "main",
		CommitSHA: "vwx234",
		RepoURL:   "https://github.com/test/repo",
	}

	ctx := context.Background()
	task := service.BuildRecoveryTask(ctx, review)

	assert.Nil(t, task)
}

func TestBuildRecoveryTask_ParseRepoPathError(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockProvider := &mockProvider{
		name:     "github",
		parseErr: errors.New("parse error"),
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	review := &model.Review{
		ID:        "test-review-009",
		Ref:       "main",
		CommitSHA: "yza567",
		RepoURL:   "https://github.com/test/repo",
	}

	ctx := context.Background()
	task := service.BuildRecoveryTask(ctx, review)

	assert.Nil(t, task)
}

func TestRecoverToQueue_MixedStatus(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          true,
		enqueueAsRunningReturn: true,
	}
	mockProvider := &mockProvider{
		name:  "github",
		owner: "test",
		repo:  "repo",
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	// Create pending review (DSL config loaded dynamically in processTask)
	pendingReview := &model.Review{
		ID:        "test-review-pending",
		Ref:       "main",
		CommitSHA: "pending123",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusPending,
	}
	err := testStore.Review().Create(pendingReview)
	require.NoError(t, err)

	// Create running review
	runningReview := &model.Review{
		ID:        "test-review-running",
		Ref:       "main",
		CommitSHA: "running456",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusRunning,
	}
	err = testStore.Review().Create(runningReview)
	require.NoError(t, err)

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Should enqueue both
	assert.Len(t, mockEnqueuer.enqueuedTasks, 1)
	assert.Len(t, mockEnqueuer.enqueuedAsRunning, 1)
	assert.Equal(t, pendingReview.ID, mockEnqueuer.enqueuedTasks[0].Review.ID)
	assert.Equal(t, runningReview.ID, mockEnqueuer.enqueuedAsRunning[0].Review.ID)
}

func TestRecoverToQueue_EnqueueFailure(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          false, // Simulate enqueue failure
		enqueueAsRunningReturn: false,
	}
	mockProvider := &mockProvider{
		name:  "github",
		owner: "test",
		repo:  "repo",
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	review := &model.Review{
		ID:        "test-review-enqueue-fail",
		Ref:       "main",
		CommitSHA: "fail789",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusPending,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Task was attempted but enqueue returned false
	assert.Len(t, mockEnqueuer.enqueuedTasks, 1)
	// But since enqueue returned false, it wasn't actually queued
}

// TestRecoverToQueue_TimeoutReview tests that timed-out reviews are marked as failed
func TestRecoverToQueue_TimeoutReview(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          true,
		enqueueAsRunningReturn: true,
	}
	mockProvider := &mockProvider{
		name:  "github",
		owner: "test",
		repo:  "repo",
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	// Create a running review that started 25 hours ago (exceeds 24h timeout)
	startedAt := time.Now().Add(-25 * time.Hour)
	review := &model.Review{
		ID:        "test-review-timeout",
		Ref:       "main",
		CommitSHA: "timeout123",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusRunning,
		StartedAt: &startedAt,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Should NOT enqueue the review
	assert.Empty(t, mockEnqueuer.enqueuedTasks)
	assert.Empty(t, mockEnqueuer.enqueuedAsRunning)

	// Should mark as failed
	recovered, err := testStore.Review().GetByID("test-review-timeout")
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusFailed, recovered.Status)
	assert.Contains(t, recovered.ErrorMessage, "task timeout")
}

// TestRecoverToQueue_MaxRetryExceeded tests that reviews with too many retries are marked as failed
func TestRecoverToQueue_MaxRetryExceeded(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          true,
		enqueueAsRunningReturn: true,
	}
	mockProvider := &mockProvider{
		name:  "github",
		owner: "test",
		repo:  "repo",
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	// Create a review with retry count >= max
	review := &model.Review{
		ID:         "test-review-retry",
		Ref:        "main",
		CommitSHA:  "retry456",
		RepoURL:    "https://github.com/test/repo",
		Status:     model.ReviewStatusPending,
		RetryCount: 3,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Should NOT enqueue the review
	assert.Empty(t, mockEnqueuer.enqueuedTasks)
	assert.Empty(t, mockEnqueuer.enqueuedAsRunning)

	// Should mark as failed
	recovered, err := testStore.Review().GetByID("test-review-retry")
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusFailed, recovered.Status)
	assert.Contains(t, recovered.ErrorMessage, "max retry count exceeded")
}

// TestRecoverToQueue_MixedRecoverable tests recovery with mix of recoverable and non-recoverable reviews
func TestRecoverToQueue_MixedRecoverable(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn:          true,
		enqueueAsRunningReturn: true,
	}
	mockProvider := &mockProvider{
		name:  "github",
		owner: "test",
		repo:  "repo",
	}
	mockResolver := &mockProviderResolver{
		providers: map[string]provider.Provider{
			"github": mockProvider,
		},
		detectMap: map[string]string{
			"https://github.com/test/repo": "github",
		},
	}

	service := NewService(cfg, testStore, mockResolver, mockEnqueuer)

	// Create multiple reviews with different conditions
	startedAt := time.Now().Add(-1 * time.Hour)
	timeoutStartedAt := time.Now().Add(-25 * time.Hour)

	reviews := []*model.Review{
		{
			ID:        "review-1-ok",
			Ref:       "main",
			CommitSHA: "ok1",
			RepoURL:   "https://github.com/test/repo",
			Status:    model.ReviewStatusPending,
		},
		{
			ID:        "review-2-ok",
			Ref:       "main",
			CommitSHA: "ok2",
			RepoURL:   "https://github.com/test/repo",
			Status:    model.ReviewStatusRunning,
			StartedAt: &startedAt,
		},
		{
			ID:        "review-3-timeout",
			Ref:       "main",
			CommitSHA: "timeout",
			RepoURL:   "https://github.com/test/repo",
			Status:    model.ReviewStatusRunning,
			StartedAt: &timeoutStartedAt,
		},
		{
			ID:         "review-4-retry",
			Ref:        "main",
			CommitSHA:  "retry",
			RepoURL:    "https://github.com/test/repo",
			Status:     model.ReviewStatusPending,
			RetryCount: 5,
		},
	}

	for _, r := range reviews {
		require.NoError(t, testStore.Review().Create(r))
	}

	ctx := context.Background()
	service.RecoverToQueue(ctx)

	// Should enqueue 2 reviews (review-1-ok and review-2-ok)
	assert.Len(t, mockEnqueuer.enqueuedTasks, 1)
	assert.Len(t, mockEnqueuer.enqueuedAsRunning, 1)

	// Check that timeout and retry reviews are marked as failed
	review3, err := testStore.Review().GetByID("review-3-timeout")
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusFailed, review3.Status)
	assert.Contains(t, review3.ErrorMessage, "task timeout")

	review4, err := testStore.Review().GetByID("review-4-retry")
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusFailed, review4.Status)
	assert.Contains(t, review4.ErrorMessage, "max retry count exceeded")
}

// TestShouldRecover tests the shouldRecover logic
func TestShouldRecover(t *testing.T) {
	cfg := &config.Config{
		Recovery: config.RecoveryConfig{
			TaskTimeoutHours: 24,
			MaxRetryCount:    3,
		},
	}

	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	service := &Service{
		cfg:   cfg,
		store: testStore,
	}

	tests := []struct {
		name          string
		review        *model.Review
		shouldRecover bool
		reasonSubstr  string
	}{
		{
			name: "pending review with no retries",
			review: &model.Review{
				Status:     model.ReviewStatusPending,
				RetryCount: 0,
			},
			shouldRecover: true,
		},
		{
			name: "running review within timeout",
			review: &model.Review{
				Status:     model.ReviewStatusRunning,
				StartedAt:  timePtr(time.Now().Add(-1 * time.Hour)),
				RetryCount: 0,
			},
			shouldRecover: true,
		},
		{
			name: "pending review with no started_at (should recover)",
			review: &model.Review{
				Status:     model.ReviewStatusPending,
				StartedAt:  nil,
				RetryCount: 0,
			},
			shouldRecover: true,
		},
		{
			name: "review with max retries",
			review: &model.Review{
				Status:     model.ReviewStatusPending,
				RetryCount: 3,
			},
			shouldRecover: false,
			reasonSubstr:  "max retry count exceeded",
		},
		{
			name: "running review exceeding timeout",
			review: &model.Review{
				Status:     model.ReviewStatusRunning,
				StartedAt:  timePtr(time.Now().Add(-25 * time.Hour)),
				RetryCount: 0,
			},
			shouldRecover: false,
			reasonSubstr:  "task timeout",
		},
		{
			name: "review with both timeout and retry issues",
			review: &model.Review{
				Status:     model.ReviewStatusRunning,
				StartedAt:  timePtr(time.Now().Add(-30 * time.Hour)),
				RetryCount: 5,
			},
			shouldRecover: false,
			reasonSubstr:  "max retry count exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRecover, reason := service.shouldRecover(tt.review)
			assert.Equal(t, tt.shouldRecover, shouldRecover)
			if !tt.shouldRecover {
				assert.Contains(t, reason, tt.reasonSubstr)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
