package base

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/verustcode/verustcode/internal/store"
)

// mockAgent is a test implementation of the Agent interface
type mockAgent struct {
	name      string
	version   string
	available bool
	store     store.Store
}

func (m *mockAgent) Name() string {
	return m.name
}

func (m *mockAgent) Version() string {
	return m.version
}

func (m *mockAgent) Available() bool {
	return m.available
}

func (m *mockAgent) ExecuteWithPrompt(ctx context.Context, req *ReviewRequest, prompt string) (*ReviewResult, error) {
	if !m.available {
		return nil, errors.New("agent not available")
	}
	result := NewResult(req.RequestID, m.name)
	result.CompletedAt = time.Now()
	result.Duration = time.Since(result.StartedAt)
	return result, nil
}

func (m *mockAgent) SetStore(s store.Store) {
	m.store = s
}

func TestRegister(t *testing.T) {
	// Clear registry
	Registry = make(map[string]AgentFactory)

	factory := func() (Agent, error) {
		return &mockAgent{name: "test", version: "1.0", available: true}, nil
	}

	Register("test-agent", factory)

	if Registry["test-agent"] == nil {
		t.Error("Register() failed to register agent factory")
	}
}

func TestCreate(t *testing.T) {
	// Clear registry
	Registry = make(map[string]AgentFactory)

	factory := func() (Agent, error) {
		return &mockAgent{name: "test", version: "1.0", available: true}, nil
	}

	Register("test-agent", factory)

	agent, err := Create("test-agent")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if agent == nil {
		t.Fatal("Create() returned nil agent")
	}
	if agent.Name() != "test" {
		t.Errorf("Create() agent.Name() = %q, want %q", agent.Name(), "test")
	}
}

func TestCreate_NotFound(t *testing.T) {
	// Clear registry
	Registry = make(map[string]AgentFactory)

	_, err := Create("non-existent")
	if err == nil {
		t.Error("Create() error = nil, want error")
	}

	var agentErr *AgentError
	if !errors.As(err, &agentErr) {
		t.Errorf("Create() error type = %T, want *AgentError", err)
	}
	if agentErr.Agent != "non-existent" {
		t.Errorf("AgentError.Agent = %q, want %q", agentErr.Agent, "non-existent")
	}
}

func TestList(t *testing.T) {
	// Clear registry
	Registry = make(map[string]AgentFactory)

	factory := func() (Agent, error) {
		return &mockAgent{name: "test", version: "1.0", available: true}, nil
	}

	Register("agent1", factory)
	Register("agent2", factory)
	Register("agent3", factory)

	names := List()
	if len(names) != 3 {
		t.Errorf("List() returned %d names, want 3", len(names))
	}

	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["agent1"] || !nameMap["agent2"] || !nameMap["agent3"] {
		t.Error("List() missing expected agent names")
	}
}

func TestAgentError(t *testing.T) {
	err := &AgentError{
		Agent:   "test-agent",
		Message: "test error",
	}

	if err.Error() != "[agent:test-agent] test error" {
		t.Errorf("AgentError.Error() = %q, want %q", err.Error(), "[agent:test-agent] test error")
	}

	wrappedErr := errors.New("wrapped error")
	err.Err = wrappedErr
	if err.Unwrap() != wrappedErr {
		t.Errorf("AgentError.Unwrap() = %v, want %v", err.Unwrap(), wrappedErr)
	}
	if err.Error() == "[agent:test-agent] test error" {
		t.Error("AgentError.Error() should include wrapped error message")
	}
}

func TestNewResult(t *testing.T) {
	requestID := "test-request-123"
	agentName := "test-agent"

	result := NewResult(requestID, agentName)

	if result.RequestID != requestID {
		t.Errorf("NewResult().RequestID = %q, want %q", result.RequestID, requestID)
	}
	if result.AgentName != agentName {
		t.Errorf("NewResult().AgentName = %q, want %q", result.AgentName, agentName)
	}
	if result.Data == nil {
		t.Error("NewResult().Data = nil, want initialized map")
	}
	if !result.Success {
		t.Error("NewResult().Success = false, want true")
	}
}

func TestReviewRequest(t *testing.T) {
	req := &ReviewRequest{
		RepoPath:     "/path/to/repo",
		RepoURL:      "https://github.com/owner/repo",
		Owner:        "owner",
		RepoName:     "repo",
		Ref:          "main",
		CommitSHA:    "abc123",
		PRNumber:     42,
		PRTitle:      "Test PR",
		PRBody:       "Test description",
		ChangedFiles: []string{"file1.go", "file2.go"},
		RequestID:    "req-123",
		RuleID:       "rule-1",
		ReviewID:     "review-1",
		Model:        "gpt-4",
	}

	if req.RepoPath != "/path/to/repo" {
		t.Errorf("ReviewRequest.RepoPath = %q, want %q", req.RepoPath, "/path/to/repo")
	}
	if req.PRNumber != 42 {
		t.Errorf("ReviewRequest.PRNumber = %d, want %d", req.PRNumber, 42)
	}
	if len(req.ChangedFiles) != 2 {
		t.Errorf("ReviewRequest.ChangedFiles length = %d, want 2", len(req.ChangedFiles))
	}
}

func TestReviewResult(t *testing.T) {
	now := time.Now()
	result := &ReviewResult{
		RequestID:    "req-123",
		Data:         map[string]any{"key": "value"},
		Text:         "test output",
		StartedAt:    now,
		CompletedAt:  now.Add(time.Second),
		Duration:     time.Second,
		AgentName:    "test-agent",
		AgentVersion: "1.0",
		ModelName:    "gpt-4",
		Success:      true,
		Error:        "",
	}

	if result.RequestID != "req-123" {
		t.Errorf("ReviewResult.RequestID = %q, want %q", result.RequestID, "req-123")
	}
	if result.Duration != time.Second {
		t.Errorf("ReviewResult.Duration = %v, want %v", result.Duration, time.Second)
	}
	if !result.Success {
		t.Error("ReviewResult.Success = false, want true")
	}
}
