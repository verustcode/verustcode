// Package mock implements the Agent interface for mock responses.
// Mock agent is used for testing and development, returning hardcoded responses
// with timestamps and random IDs to verify execution without consuming tokens.
package mock

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/llm"
	"github.com/verustcode/verustcode/internal/store"

	// Import mock client to register it
	_ "github.com/verustcode/verustcode/internal/llm/mock"
	"github.com/verustcode/verustcode/pkg/logger"
)

// AgentName is the identifier for the Mock agent
const AgentName = "mock"

// Version is the current version of the Mock agent
const Version = "1.0.0"

func init() {
	// Register Mock agent factory
	base.Register(AgentName, NewAgent)
}

// MockAgent implements the Agent interface for mock responses
// It uses the mock llm.Client for actual execution
type MockAgent struct {
	client  llm.Client
	timeout time.Duration
	version string
}

// NewAgent creates a new Mock agent instance
func NewAgent() (base.Agent, error) {
	// Create LLM client configuration
	config := llm.NewClientConfig("mock")

	// Create the mock LLM client
	client, err := llm.Create("mock", config)
	if err != nil {
		return nil, &base.AgentError{
			Agent:   AgentName,
			Message: "failed to create mock LLM client",
			Err:     err,
		}
	}

	return &MockAgent{
		client:  client,
		timeout: llm.DefaultTimeout,
		version: Version,
	}, nil
}

// Name returns the agent identifier
func (a *MockAgent) Name() string {
	return AgentName
}

// Version returns the agent version
func (a *MockAgent) Version() string {
	return a.version
}

// Available checks if Mock agent is available (always true)
func (a *MockAgent) Available() bool {
	return a.client.Available()
}

// SetStore sets the database store (mock agent doesn't use it)
func (a *MockAgent) SetStore(s store.Store) {
	// Mock agent doesn't need database configuration
}

// ExecuteWithPrompt performs code review using a custom prompt (DSL mode)
// In DSL mode, the prompt is rendered from templates and expects markdown output,
// so we don't parse the output as JSON - just use it directly as summary.
func (a *MockAgent) ExecuteWithPrompt(ctx context.Context, req *base.ReviewRequest, prompt string) (*base.ReviewResult, error) {
	startTime := time.Now()
	result := base.NewResult(req.RequestID, a.Name())
	result.StartedAt = startTime
	result.AgentVersion = a.version

	logger.Info("Starting Mock agent review with custom prompt",
		zap.String("request_id", req.RequestID),
		zap.String("repo_path", req.RepoPath),
		zap.String("rule_id", req.RuleID),
	)

	// Execute using mock LLM client
	output, err := a.executeWithClient(ctx, req, prompt)
	if err != nil {
		logger.Error("Mock agent execution failed",
			zap.Error(err),
			zap.String("request_id", req.RequestID),
			zap.String("rule_id", req.RuleID),
		)
		result.Success = false
		result.Error = err.Error()
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result, err
	}

	// DSL mode uses markdown output, no JSON parsing needed
	// Use raw output directly as text
	result.Text = output

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	logger.Info("Mock agent review completed",
		zap.String("request_id", req.RequestID),
		zap.String("rule_id", req.RuleID),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// executeWithClient executes the prompt using the mock LLM client
func (a *MockAgent) executeWithClient(ctx context.Context, req *base.ReviewRequest, prompt string) (string, error) {
	// Build request for DSL mode (always markdown output, no JSON schema)
	// Include agent metadata and rule_id for mock response generation
	metadata := map[string]string{
		"agent_name":    a.Name(),
		"agent_version": a.Version(),
	}
	if req.RuleID != "" {
		metadata["rule_id"] = req.RuleID
	}

	llmReq := llm.NewRequest(prompt).
		WithWorkDir(req.RepoPath).
		WithOptions(&llm.RequestOptions{
			Timeout:  a.timeout,
			Metadata: metadata,
		})

	// Execute
	resp, err := a.client.Execute(ctx, llmReq)
	if err != nil {
		return "", &base.AgentError{
			Agent:   AgentName,
			Message: "mock LLM client execution failed",
			Err:     err,
		}
	}

	return resp.Content, nil
}
