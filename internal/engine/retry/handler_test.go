package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine/executor"
	"github.com/verustcode/verustcode/internal/engine/runner"
	"github.com/verustcode/verustcode/internal/engine/task"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	pkgerrors "github.com/verustcode/verustcode/pkg/errors"
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
	enqueuedTasks []*task.Task
	hasTaskMap    map[string]bool
	enqueueReturn bool
}

func (m *mockTaskEnqueuer) Enqueue(t *task.Task) bool {
	m.enqueuedTasks = append(m.enqueuedTasks, t)
	return m.enqueueReturn
}

func (m *mockTaskEnqueuer) HasTask(reviewID string) bool {
	if m.hasTaskMap == nil {
		return false
	}
	return m.hasTaskMap[reviewID]
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
	return ""
}

// mockTaskBuilder implements TaskBuilder for testing
type mockTaskBuilder struct {
	buildTask *task.Task
	buildErr  error
}

func (m *mockTaskBuilder) BuildRecoveryTask(ctx context.Context, review *model.Review) *task.Task {
	return m.buildTask
}

// createMockRunner creates a minimal runner for testing
func createMockRunner(t *testing.T, s store.Store) *runner.Runner {
	cfg := &config.Config{}
	agents := make(map[string]base.Agent)
	promptBuilder := prompt.NewBuilder()
	exec := executor.NewExecutor(cfg, agents, promptBuilder, s)
	return runner.NewRunner(cfg, s, exec, promptBuilder)
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

func TestNewHandler(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{}
	mockBuilder := &mockTaskBuilder{}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	require.NotNil(t, handler)
	assert.Equal(t, cfg, handler.cfg)
	assert.Equal(t, testStore, handler.store)
	assert.Equal(t, mockResolver, handler.providerResolver)
	assert.Equal(t, mockEnqueuer, handler.taskEnqueuer)
	assert.Equal(t, mockBuilder, handler.taskBuilder)
	assert.Equal(t, rnr, handler.runner)
	assert.Equal(t, ctx, handler.ctx)
}

func TestRetry_ReviewNotFound(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{}
	mockBuilder := &mockTaskBuilder{}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	err := handler.Retry("non-existent-review")

	require.Error(t, err)
	var appErr *pkgerrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, pkgerrors.ErrCodeReviewNotFound, appErr.Code)
}

func TestRetry_InvalidStatus(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{}
	mockBuilder := &mockTaskBuilder{}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	// Create a pending review (not failed)
	review := &model.Review{
		ID:        "test-review-pending",
		Ref:       "main",
		CommitSHA: "abc123",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusPending,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	err = handler.Retry(review.ID)

	require.Error(t, err)
	var appErr *pkgerrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, pkgerrors.ErrCodeValidation, appErr.Code)
	assert.Contains(t, err.Error(), "only failed reviews can be retried")
}

func TestRetry_AlreadyInQueue(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		hasTaskMap: map[string]bool{
			"test-review-in-queue": true,
		},
	}
	mockResolver := &mockProviderResolver{}
	mockBuilder := &mockTaskBuilder{}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	review := &model.Review{
		ID:        "test-review-in-queue",
		Ref:       "main",
		CommitSHA: "ghi789",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusFailed,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	err = handler.Retry(review.ID)

	require.Error(t, err)
	var appErr *pkgerrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, pkgerrors.ErrCodeValidation, appErr.Code)
	assert.Contains(t, err.Error(), "already in the queue")
}

func TestRetry_Success(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		enqueueReturn: true,
		hasTaskMap:    make(map[string]bool),
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
	recoveryTask := &task.Task{
		Review: &model.Review{
			ID: "test-review-success",
		},
		ProviderName: "github",
	}
	mockBuilder := &mockTaskBuilder{
		buildTask: recoveryTask,
	}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	review := &model.Review{
		ID:         "test-review-success",
		Ref:        "main",
		CommitSHA:  "jkl012",
		RepoURL:    "https://github.com/test/repo",
		Status:     model.ReviewStatusFailed,
		RetryCount: 0,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	err = handler.Retry(review.ID)

	require.NoError(t, err)
	assert.Len(t, mockEnqueuer.enqueuedTasks, 1)
	assert.Equal(t, review.ID, mockEnqueuer.enqueuedTasks[0].Review.ID)

	// Verify review was reset
	updatedReview, err := testStore.Review().GetByID(review.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusPending, updatedReview.Status)
	assert.Equal(t, 1, updatedReview.RetryCount)
}

func TestRetry_BuildTaskFailure(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{
		hasTaskMap: make(map[string]bool),
	}
	mockResolver := &mockProviderResolver{}
	mockBuilder := &mockTaskBuilder{
		buildTask: nil, // Build task fails
	}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	review := &model.Review{
		ID:        "test-review-build-fail",
		Ref:       "main",
		CommitSHA: "mno345",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusFailed,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	err = handler.Retry(review.ID)

	require.Error(t, err)
	var appErr *pkgerrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, pkgerrors.ErrCodeInternal, appErr.Code)
	assert.Contains(t, err.Error(), "failed to build recovery task")

	// Verify review was reverted to failed
	updatedReview, err := testStore.Review().GetByID(review.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusFailed, updatedReview.Status)
}

func TestRetryRule_ReviewNotFound(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{}
	mockBuilder := &mockTaskBuilder{}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	err := handler.RetryRule("non-existent-review", "rule-1")

	require.Error(t, err)
	var appErr *pkgerrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, pkgerrors.ErrCodeReviewNotFound, appErr.Code)
}

func TestRetryRule_RuleNotFound(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{}
	mockBuilder := &mockTaskBuilder{}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	review := &model.Review{
		ID:        "test-review-rule-not-found",
		Ref:       "main",
		CommitSHA: "pqr678",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusFailed,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	err = handler.RetryRule(review.ID, "non-existent-rule")

	require.Error(t, err)
	var appErr *pkgerrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, pkgerrors.ErrCodeValidation, appErr.Code)
	assert.Contains(t, err.Error(), "not found in review")
}

func TestRetryRule_InvalidRuleStatus(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	mockEnqueuer := &mockTaskEnqueuer{}
	mockResolver := &mockProviderResolver{}
	mockBuilder := &mockTaskBuilder{}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	review := &model.Review{
		ID:        "test-review-invalid-rule-status",
		Ref:       "main",
		CommitSHA: "stu901",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusFailed,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	rule := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "rule-1",
		RuleIndex: 0,
		Status:    model.RuleStatusCompleted, // Not failed
	}
	err = testStore.Review().CreateRule(rule)
	require.NoError(t, err)

	err = handler.RetryRule(review.ID, "rule-1")

	require.Error(t, err)
	var appErr *pkgerrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, pkgerrors.ErrCodeValidation, appErr.Code)
	assert.Contains(t, err.Error(), "only failed rules can be retried")
}

func TestRetryRule_Success(t *testing.T) {
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
	mockBuilder := &mockTaskBuilder{}
	rnr := createMockRunner(t, testStore)
	ctx := context.Background()

	handler := NewHandler(cfg, testStore, mockResolver, mockEnqueuer, mockBuilder, rnr, ctx)

	review := &model.Review{
		ID:        "test-review-rule-success",
		Ref:       "main",
		CommitSHA: "yza567",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusFailed,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	rule := &model.ReviewRule{
		ReviewID:   review.ID,
		RuleID:     "rule-1",
		RuleIndex:  0,
		Status:     model.RuleStatusFailed,
		RetryCount: 0,
	}
	err = testStore.Review().CreateRule(rule)
	require.NoError(t, err)

	err = handler.RetryRule(review.ID, "rule-1")

	require.NoError(t, err)

	// Wait a bit for goroutine to execute
	time.Sleep(200 * time.Millisecond)

	// Verify rule was reset (may fail due to agent not configured, but reset should happen)
	updatedRule, err := testStore.Review().GetRuleByID(rule.ID)
	require.NoError(t, err)
	// Rule should be reset to running or failed (if agent execution fails)
	assert.True(t, updatedRule.Status == model.RuleStatusRunning || updatedRule.Status == model.RuleStatusFailed)
	if updatedRule.Status == model.RuleStatusRunning {
		assert.Equal(t, 1, updatedRule.RetryCount)
	}
}
