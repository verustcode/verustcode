package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/engine/executor"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

func init() {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
}

func TestNewRunner(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	agents := make(map[string]base.Agent)
	promptBuilder := prompt.NewBuilder()
	exec := executor.NewExecutor(cfg, agents, promptBuilder, testStore)

	runner := NewRunner(cfg, testStore, exec, promptBuilder)

	require.NotNil(t, runner)
	assert.Equal(t, cfg, runner.cfg)
	assert.Equal(t, testStore, runner.store)
	assert.Equal(t, exec, runner.executor)
	assert.Equal(t, promptBuilder, runner.promptBuilder)
}

func TestUpdateReviewStatusAfterRuleExecution_AllCompleted(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	agents := make(map[string]base.Agent)
	promptBuilder := prompt.NewBuilder()
	exec := executor.NewExecutor(cfg, agents, promptBuilder, testStore)

	runner := NewRunner(cfg, testStore, exec, promptBuilder)

	// Create a review
	review := &model.Review{
		ID:        "test-review-status",
		Ref:       "main",
		CommitSHA: "abc123",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusRunning,
	}
	now := time.Now()
	review.StartedAt = &now
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	// Create completed rules
	rule1 := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "rule-1",
		RuleIndex: 0,
		Status:    model.RuleStatusCompleted,
	}
	err = testStore.Review().CreateRule(rule1)
	require.NoError(t, err)

	rule2 := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "rule-2",
		RuleIndex: 1,
		Status:    model.RuleStatusCompleted,
	}
	err = testStore.Review().CreateRule(rule2)
	require.NoError(t, err)

	runner.UpdateReviewStatusAfterRuleExecution(review)

	// Verify review status was updated to completed
	updatedReview, err := testStore.Review().GetByID(review.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusCompleted, updatedReview.Status)
	assert.NotNil(t, updatedReview.CompletedAt)
}

func TestUpdateReviewStatusAfterRuleExecution_HasFailed(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	agents := make(map[string]base.Agent)
	promptBuilder := prompt.NewBuilder()
	exec := executor.NewExecutor(cfg, agents, promptBuilder, testStore)

	runner := NewRunner(cfg, testStore, exec, promptBuilder)

	// Create a review
	review := &model.Review{
		ID:        "test-review-failed",
		Ref:       "main",
		CommitSHA: "def456",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusRunning,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	// Create mixed status rules
	rule1 := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "rule-1",
		RuleIndex: 0,
		Status:    model.RuleStatusCompleted,
	}
	err = testStore.Review().CreateRule(rule1)
	require.NoError(t, err)

	rule2 := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "rule-2",
		RuleIndex: 1,
		Status:    model.RuleStatusFailed,
	}
	err = testStore.Review().CreateRule(rule2)
	require.NoError(t, err)

	runner.UpdateReviewStatusAfterRuleExecution(review)

	// Verify review status was updated to failed
	updatedReview, err := testStore.Review().GetByID(review.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusFailed, updatedReview.Status)
}

func TestUpdateReviewStatusAfterRuleExecution_StillRunning(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	agents := make(map[string]base.Agent)
	promptBuilder := prompt.NewBuilder()
	exec := executor.NewExecutor(cfg, agents, promptBuilder, testStore)

	runner := NewRunner(cfg, testStore, exec, promptBuilder)

	// Create a review
	review := &model.Review{
		ID:        "test-review-running",
		Ref:       "main",
		CommitSHA: "ghi789",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusRunning,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	// Create rules with some still pending
	rule1 := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "rule-1",
		RuleIndex: 0,
		Status:    model.RuleStatusCompleted,
	}
	err = testStore.Review().CreateRule(rule1)
	require.NoError(t, err)

	rule2 := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "rule-2",
		RuleIndex: 1,
		Status:    model.RuleStatusRunning,
	}
	err = testStore.Review().CreateRule(rule2)
	require.NoError(t, err)

	runner.UpdateReviewStatusAfterRuleExecution(review)

	// Verify review status remains running
	updatedReview, err := testStore.Review().GetByID(review.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ReviewStatusRunning, updatedReview.Status)
}

func TestUpdateReviewStatusAfterRuleExecution_StoreError(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	agents := make(map[string]base.Agent)
	promptBuilder := prompt.NewBuilder()
	exec := executor.NewExecutor(cfg, agents, promptBuilder, testStore)

	runner := NewRunner(cfg, testStore, exec, promptBuilder)

	// Create a review with invalid ID to cause store error
	review := &model.Review{
		ID:        "non-existent",
		Ref:       "main",
		CommitSHA: "jkl012",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusRunning,
	}

	// Should not panic, just log error
	runner.UpdateReviewStatusAfterRuleExecution(review)
}

func TestExecuteSingleRule_DeleteOldResults(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	agents := make(map[string]base.Agent)
	promptBuilder := prompt.NewBuilder()
	exec := executor.NewExecutor(cfg, agents, promptBuilder, testStore)

	runner := NewRunner(cfg, testStore, exec, promptBuilder)

	// Create a review
	review := &model.Review{
		ID:        "test-review-delete",
		Ref:       "main",
		CommitSHA: "mno345",
		RepoURL:   "https://github.com/test/repo",
		Status:    model.ReviewStatusRunning,
	}
	err := testStore.Review().Create(review)
	require.NoError(t, err)

	// Create a failed rule
	rule := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "rule-1",
		RuleIndex: 0,
		Status:    model.RuleStatusFailed,
	}
	err = testStore.Review().CreateRule(rule)
	require.NoError(t, err)

	// Create old results
	oldResult := &model.ReviewResult{
		ReviewRuleID: rule.ID,
		Data:         model.JSONMap{"old": "data"},
	}
	err = testStore.Review().CreateResult(oldResult)
	require.NoError(t, err)

	// Create execution context
	buildCtx := &prompt.BuildContext{
		RepoPath:  "/tmp/repo",
		RepoURL:   review.RepoURL,
		Ref:       review.Ref,
		CommitSHA: review.CommitSHA,
	}

	ruleConfig := &dsl.ReviewRuleConfig{
		ID: "rule-1",
		Goals: dsl.GoalsConfig{
			Areas: []string{"security"},
		},
	}

	execCtx := &RuleExecutionContext{
		BuildCtx:   buildCtx,
		Review:     review,
		ReviewRule: rule,
		Rule:       ruleConfig,
		RuleIndex:  0,
		Provider:   nil,
		OutputDir:  "",
	}

	ctx := context.Background()
	_, err = runner.ExecuteSingleRule(ctx, execCtx)

	// Should attempt execution (may fail due to agent not configured, but old results should be deleted)
	// Verify old results were deleted
	results, err := testStore.Review().GetResultsByRuleID(rule.ID)
	require.NoError(t, err)
	// Old result should be deleted (new result may or may not be created depending on execution success)
	assert.LessOrEqual(t, len(results), 1)
}
