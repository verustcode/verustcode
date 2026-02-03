package mock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

func init() {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
}

func TestNewAgent(t *testing.T) {
	agent, err := NewAgent()
	require.NoError(t, err)
	require.NotNil(t, agent)
	assert.Equal(t, AgentName, agent.Name())
	assert.Equal(t, Version, agent.Version())
	assert.True(t, agent.Available())
}

func TestMockAgent_Name(t *testing.T) {
	agent, err := NewAgent()
	require.NoError(t, err)
	assert.Equal(t, AgentName, agent.Name())
}

func TestMockAgent_Version(t *testing.T) {
	agent, err := NewAgent()
	require.NoError(t, err)
	assert.Equal(t, Version, agent.Version())
}

func TestMockAgent_Available(t *testing.T) {
	agent, err := NewAgent()
	require.NoError(t, err)
	assert.True(t, agent.Available())
}

func TestMockAgent_SetStore(t *testing.T) {
	agent, err := NewAgent()
	require.NoError(t, err)

	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Should not panic
	agent.SetStore(testStore)
}

func TestMockAgent_ExecuteWithPrompt(t *testing.T) {
	agent, err := NewAgent()
	require.NoError(t, err)

	req := &base.ReviewRequest{
		RequestID: "test-request-001",
		RepoPath:  "/tmp/test-repo",
		RuleID:    "rule-1",
	}

	ctx := context.Background()
	result, err := agent.ExecuteWithPrompt(ctx, req, "Review this code for security issues")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "test-request-001", result.RequestID)
	assert.Equal(t, AgentName, result.AgentName)
	assert.Equal(t, Version, result.AgentVersion)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Text)
	assert.Contains(t, result.Text, "Mock Code Review Result")
	assert.NotNil(t, result.StartedAt)
	assert.NotNil(t, result.CompletedAt)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

func TestMockAgent_ExecuteWithPrompt_WithRuleID(t *testing.T) {
	agent, err := NewAgent()
	require.NoError(t, err)

	req := &base.ReviewRequest{
		RequestID: "test-request-002",
		RepoPath:  "/tmp/test-repo",
		RuleID:    "security-rule",
	}

	ctx := context.Background()
	result, err := agent.ExecuteWithPrompt(ctx, req, "Review code")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Text, "security-rule")
}

func TestMockAgent_ExecuteWithPrompt_Error(t *testing.T) {
	// This test would require mocking the LLM client to return an error
	// For now, we test the happy path since mock client always succeeds
	agent, err := NewAgent()
	require.NoError(t, err)

	req := &base.ReviewRequest{
		RequestID: "test-request-003",
		RepoPath:  "/tmp/test-repo",
	}

	ctx := context.Background()
	result, err := agent.ExecuteWithPrompt(ctx, req, "test prompt")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}
