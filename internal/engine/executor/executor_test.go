package executor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/engine/task"
	"github.com/verustcode/verustcode/internal/llm"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
)

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

// mockAgent implements base.Agent for testing
type mockAgent struct {
	name       string
	responses  map[string]*base.ReviewResult
	execCount  int
	shouldFail bool
	failCount  int
}

func newMockAgent(name string) *mockAgent {
	return &mockAgent{
		name:      name,
		responses: make(map[string]*base.ReviewResult),
	}
}

func (m *mockAgent) Name() string {
	return m.name
}

func (m *mockAgent) Version() string {
	return "test-1.0.0"
}

func (m *mockAgent) Available() bool {
	return true
}

func (m *mockAgent) SetStore(s store.Store) {
	// Mock agent doesn't need database configuration
}

func (m *mockAgent) ExecuteWithPrompt(ctx context.Context, req *base.ReviewRequest, prompt string) (*base.ReviewResult, error) {
	m.execCount++

	// Simulate failure if configured - use retryable error
	if m.shouldFail && m.execCount <= m.failCount {
		return nil, llm.NewRetryableError("mock", "execute", "mock agent execution failed", nil)
	}

	// Return predefined response or create a default one
	result := &base.ReviewResult{
		RequestID:   req.RequestID,
		AgentName:   m.name,
		ModelName:   req.Model,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		Duration:    10 * time.Millisecond,
		Success:     true,
		Text:        "Mock review summary for " + req.RuleID,
		Data: map[string]any{
			"summary": "Mock review summary",
			"findings": []interface{}{
				map[string]interface{}{"file": "test.go", "line": 1, "message": "test issue"},
			},
		},
	}

	if custom, ok := m.responses[req.RuleID]; ok {
		return custom, nil
	}

	return result, nil
}

// setResponse sets a custom response for a specific rule ID
func (m *mockAgent) setResponse(ruleID string, response *base.ReviewResult) {
	m.responses[ruleID] = response
}

// setFailure configures the mock agent to fail for the first N executions
func (m *mockAgent) setFailure(failCount int) {
	m.shouldFail = true
	m.failCount = failCount
}

func TestNewExecutor(t *testing.T) {
	cfg := &config.Config{}
	agents := map[string]base.Agent{
		"mock": newMockAgent("mock"),
	}
	promptBuilder := prompt.NewBuilder()
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	executor := NewExecutor(cfg, agents, promptBuilder, testStore)

	assert.NotNil(t, executor)
	assert.Equal(t, cfg, executor.cfg)
	assert.Equal(t, agents, executor.agents)
	assert.Equal(t, promptBuilder, executor.promptBuilder)
	assert.Equal(t, testStore, executor.store)
}

func TestExecuteRule_SingleRun_Success(t *testing.T) {
	// Setup
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	mockAgent := newMockAgent("mock")
	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxRetries: 3,
			RetryDelay: 1,
		},
	}
	agents := map[string]base.Agent{"mock": mockAgent}
	promptBuilder := prompt.NewBuilder()

	executor := NewExecutor(cfg, agents, promptBuilder, testStore)

	// Create test rule
	rule := &dsl.ReviewRuleConfig{
		ID:          "test-rule-1",
		Description: "Test Rule",
		Agent: dsl.AgentConfig{
			Type: "mock",
		},
		Goals: dsl.GoalsConfig{
			Areas: []string{"business-logic"},
		},
	}

	buildCtx := &prompt.BuildContext{
		RepoPath:  "/tmp/test-repo",
		RepoURL:   "https://github.com/test/repo",
		Ref:       "main",
		CommitSHA: "abc123",
	}

	// Execute
	result, err := executor.ExecuteRule(context.Background(), rule, buildCtx, nil, 0)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, rule.ID, result.ReviewerID)
	assert.NotEmpty(t, result.Text)
	assert.Equal(t, 1, mockAgent.execCount)
}

func TestExecuteRule_SingleRun_WithRetry(t *testing.T) {
	// Setup
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	mockAgent := newMockAgent("mock")
	mockAgent.setFailure(1) // Fail first attempt, succeed on 2nd
	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxRetries: 3,
			RetryDelay: 0, // No delay in tests
		},
	}
	agents := map[string]base.Agent{"mock": mockAgent}
	promptBuilder := prompt.NewBuilder()

	executor := NewExecutor(cfg, agents, promptBuilder, testStore)

	rule := &dsl.ReviewRuleConfig{
		ID:          "test-rule-retry",
		Description: "Test Rule with Retry",
		Agent: dsl.AgentConfig{
			Type: "mock",
		},
		Goals: dsl.GoalsConfig{
			Areas: []string{"business-logic"},
		},
	}

	buildCtx := &prompt.BuildContext{
		RepoPath:  "/tmp/test-repo",
		RepoURL:   "https://github.com/test/repo",
		Ref:       "main",
		CommitSHA: "abc123",
	}

	// Execute
	result, err := executor.ExecuteRule(context.Background(), rule, buildCtx, nil, 0)

	// Assert - should succeed after retries
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, mockAgent.execCount) // Failed once, succeeded on 2nd
}

func TestExecuteRule_SingleRun_WithReviewRule(t *testing.T) {
	// Setup
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create test review
	review := store.CreateTestReview(t, testStore)

	mockAgent := newMockAgent("mock")
	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxRetries: 3,
			RetryDelay: 0,
		},
	}
	agents := map[string]base.Agent{"mock": mockAgent}
	promptBuilder := prompt.NewBuilder()

	executor := NewExecutor(cfg, agents, promptBuilder, testStore)

	rule := &dsl.ReviewRuleConfig{
		ID:          "test-rule-with-record",
		Description: "Test Rule with Record",
		Agent: dsl.AgentConfig{
			Type: "mock",
		},
		Goals: dsl.GoalsConfig{
			Areas: []string{"business-logic"},
		},
	}

	buildCtx := &prompt.BuildContext{
		RepoPath:  "/tmp/test-repo",
		RepoURL:   "https://github.com/test/repo",
		Ref:       "main",
		CommitSHA: "abc123",
	}

	// Create review rule
	reviewRule := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    rule.ID,
		Status:    model.RuleStatusPending,
		RuleIndex: 0,
	}
	err := testStore.Review().CreateRule(reviewRule)
	require.NoError(t, err)

	// Execute
	result, err := executor.ExecuteRule(context.Background(), rule, buildCtx, reviewRule, 0)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify review rule was updated
	updatedRule, err := testStore.Review().GetRuleByID(reviewRule.ID)
	require.NoError(t, err)
	assert.Equal(t, model.RuleStatusCompleted, updatedRule.Status)
	assert.NotNil(t, updatedRule.StartedAt)
	assert.NotNil(t, updatedRule.CompletedAt)
	assert.GreaterOrEqual(t, updatedRule.Duration, int64(0)) // May be 0 if execution is very fast
}

func TestExecuteRule_AgentNotFound(t *testing.T) {
	// Setup
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{}
	agents := map[string]base.Agent{}
	promptBuilder := prompt.NewBuilder()

	executor := NewExecutor(cfg, agents, promptBuilder, testStore)

	rule := &dsl.ReviewRuleConfig{
		ID:          "test-rule-no-agent",
		Description: "Test Rule No Agent",
		Agent: dsl.AgentConfig{
			Type: "nonexistent",
		},
		Goals: dsl.GoalsConfig{
			Areas: []string{"business-logic"},
		},
	}

	buildCtx := &prompt.BuildContext{
		RepoPath:  "/tmp/test-repo",
		RepoURL:   "https://github.com/test/repo",
		Ref:       "main",
		CommitSHA: "abc123",
	}

	// Execute
	_, err := executor.ExecuteRule(context.Background(), rule, buildCtx, nil, 0)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent nonexistent not available")
}

func TestBuildFormatInstructions(t *testing.T) {
	tests := []struct {
		name     string
		rule     *dsl.ReviewRuleConfig
		buildCtx *prompt.BuildContext
		wantLen  bool // true if we expect non-empty result
	}{
		{
			name: "nil rule",
			rule: nil,
			buildCtx: &prompt.BuildContext{
				OutputLanguage: "zh-CN",
			},
			wantLen: false,
		},
		{
			name: "nil output",
			rule: &dsl.ReviewRuleConfig{
				ID:     "test-1",
				Output: nil,
			},
			buildCtx: &prompt.BuildContext{},
			wantLen:  false,
		},
		{
			name: "with extra fields in schema",
			rule: &dsl.ReviewRuleConfig{
				ID: "test-2",
				Output: &dsl.OutputConfig{
					Schema: &dsl.OutputSchemaConfig{
						ExtraFields: []dsl.ExtraFieldConfig{
							{
								Name:        "vulnerability_type",
								Type:        "string",
								Description: "Type of security vulnerability",
								Required:    true,
							},
						},
					},
				},
			},
			buildCtx: &prompt.BuildContext{
				OutputLanguage: "zh-CN",
			},
			wantLen: true,
		},
		{
			name: "with language",
			rule: &dsl.ReviewRuleConfig{
				ID: "test-3",
				Output: &dsl.OutputConfig{
					Style: &dsl.OutputStyleConfig{
						Language: "en",
					},
				},
			},
			buildCtx: &prompt.BuildContext{},
			wantLen:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildFormatInstructions(tt.rule, tt.buildCtx)
			if tt.wantLen {
				assert.NotEmpty(t, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestUpdateReviewRuleAfterExecution(t *testing.T) {
	// Setup
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	review := store.CreateTestReview(t, testStore)

	cfg := &config.Config{
		Review: config.ReviewConfig{
			OutputMetadata: config.OutputMetadataConfig{
				ShowAgent:  boolPtr(true),
				ShowModel:  boolPtr(true),
				CustomText: "Generated by {agent} ({model})",
			},
		},
	}
	agents := map[string]base.Agent{}
	promptBuilder := prompt.NewBuilder()

	executor := NewExecutor(cfg, agents, promptBuilder, testStore)

	t.Run("success", func(t *testing.T) {
		reviewRule := &model.ReviewRule{
			ReviewID:  review.ID,
			RuleID:    "test-update-success",
			Status:    model.RuleStatusRunning,
			RuleIndex: 0,
		}
		err := testStore.Review().CreateRule(reviewRule)
		require.NoError(t, err)

		result := &prompt.ReviewResult{
			ReviewerID: reviewRule.RuleID,
			Text:       "Test summary",
			AgentName:  "mock",
			ModelName:  "test-model",
			Data: map[string]any{
				"findings": []interface{}{
					map[string]interface{}{"issue": "test"},
					map[string]interface{}{"issue": "test2"},
				},
			},
		}

		past := time.Now().Add(-100 * time.Millisecond)
		reviewRule.StartedAt = &past

		executor.UpdateReviewRuleAfterExecution(reviewRule, result, nil, &cfg.Review.OutputMetadata)

		// Verify
		assert.Equal(t, model.RuleStatusCompleted, reviewRule.Status)
		assert.Equal(t, 2, reviewRule.FindingsCount)
		assert.NotNil(t, reviewRule.CompletedAt)
		assert.GreaterOrEqual(t, reviewRule.Duration, int64(0))
	})

	t.Run("failure", func(t *testing.T) {
		reviewRule := &model.ReviewRule{
			ReviewID:  review.ID,
			RuleID:    "test-update-failure",
			Status:    model.RuleStatusRunning,
			RuleIndex: 1,
		}
		err := testStore.Review().CreateRule(reviewRule)
		require.NoError(t, err)

		now := time.Now()
		reviewRule.StartedAt = &now

		testErr := &base.AgentError{
			Agent:   "mock",
			Message: "test error",
		}

		executor.UpdateReviewRuleAfterExecution(reviewRule, nil, testErr, nil)

		// Verify
		assert.Equal(t, model.RuleStatusFailed, reviewRule.Status)
		assert.Contains(t, reviewRule.ErrorMessage, "test error")
		assert.NotNil(t, reviewRule.CompletedAt)
	})
}

func TestLoadExistingReviewRuleRuns(t *testing.T) {
	// Setup
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	review := store.CreateTestReview(t, testStore)

	cfg := &config.Config{}
	agents := map[string]base.Agent{}
	promptBuilder := prompt.NewBuilder()

	executor := NewExecutor(cfg, agents, promptBuilder, testStore)

	// Create review rule
	reviewRule := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleID:    "test-load-runs",
		Status:    model.RuleStatusPending,
		RuleIndex: 0,
	}
	err := testStore.Review().CreateRule(reviewRule)
	require.NoError(t, err)

	// Create some runs
	run1 := &model.ReviewRuleRun{
		ReviewRuleID: reviewRule.ID,
		RunIndex:     0,
		Model:        "model-1",
		Status:       model.RunStatusCompleted,
	}
	run2 := &model.ReviewRuleRun{
		ReviewRuleID: reviewRule.ID,
		RunIndex:     1,
		Model:        "model-2",
		Status:       model.RunStatusFailed,
	}

	err = testStore.Review().CreateRun(run1)
	require.NoError(t, err)
	err = testStore.Review().CreateRun(run2)
	require.NoError(t, err)

	// Load runs
	runsMap, err := executor.LoadExistingReviewRuleRuns(reviewRule.ID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, runsMap, 2)
	assert.Equal(t, model.RunStatusCompleted, runsMap[0].Status)
	assert.Equal(t, model.RunStatusFailed, runsMap[1].Status)
}

func TestMergeReviewResults(t *testing.T) {
	// Setup
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	mockAgent := newMockAgent("mock")
	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxRetries: 3,
			RetryDelay: 0,
		},
	}
	agents := map[string]base.Agent{"mock": mockAgent}
	promptBuilder := prompt.NewBuilder()

	executor := NewExecutor(cfg, agents, promptBuilder, testStore)

	rule := &dsl.ReviewRuleConfig{
		ID: "test-merge",
		MultiRun: &dsl.MultiRunConfig{
			Runs:       2,
			MergeModel: "",
		},
	}

	t.Run("no results", func(t *testing.T) {
		results := []task.RunResult{}
		_, err := executor.mergeReviewResults(context.Background(), rule, mockAgent, results)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no results to merge")
	})

	t.Run("single result", func(t *testing.T) {
		results := []task.RunResult{
			{Index: 1, Model: "model-1", Text: "Test summary 1"},
		}
		merged, err := executor.mergeReviewResults(context.Background(), rule, mockAgent, results)
		require.NoError(t, err)
		assert.Equal(t, "Test summary 1", merged)
	})

	t.Run("multiple results", func(t *testing.T) {
		results := []task.RunResult{
			{Index: 1, Model: "model-1", Text: "Test summary 1"},
			{Index: 2, Model: "model-2", Text: "Test summary 2"},
		}
		merged, err := executor.mergeReviewResults(context.Background(), rule, mockAgent, results)
		require.NoError(t, err)
		assert.NotEmpty(t, merged)
		// Mock agent will return a response containing the merge prompt
	})
}
