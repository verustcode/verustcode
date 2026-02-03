// Package cursor implements the Agent interface for Cursor CLI.
// Cursor is the default AI agent for code review in VerustCode.
// This implementation uses the llm.Client interface for actual CLI execution.
package cursor

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/llm"
	"github.com/verustcode/verustcode/internal/store"

	// Import cursor client to register it
	_ "github.com/verustcode/verustcode/internal/llm/cursor"
	"github.com/verustcode/verustcode/pkg/logger"
)

// AgentName is the identifier for the Cursor agent
const AgentName = "cursor"

// Version is the current version of the Cursor agent
const Version = "2.0.0"

func init() {
	// Register Cursor agent factory
	base.Register(AgentName, NewAgent)
}

// CursorAgent implements the Agent interface for Cursor CLI
// It uses the llm.Client interface for actual execution
type CursorAgent struct {
	client  llm.Client
	store   store.Store // Database store for reading runtime configuration
	timeout time.Duration
	version string
}

// NewAgent creates a new Cursor agent instance
func NewAgent() (base.Agent, error) {
	// Create LLM client configuration
	config := llm.NewClientConfig("cursor")

	// Create the LLM client
	client, err := llm.Create("cursor", config)
	if err != nil {
		return nil, &base.AgentError{
			Agent:   AgentName,
			Message: "failed to create LLM client",
			Err:     err,
		}
	}

	return &CursorAgent{
		client:  client,
		timeout: llm.DefaultTimeout,
		version: Version,
	}, nil
}

// Name returns the agent identifier
func (a *CursorAgent) Name() string {
	return AgentName
}

// Version returns the agent version
func (a *CursorAgent) Version() string {
	return a.version
}

// Available checks if Cursor CLI is available
func (a *CursorAgent) Available() bool {
	return a.client.Available()
}

// SetStore sets the database store for reading runtime configuration
func (a *CursorAgent) SetStore(s store.Store) {
	a.store = s
}

// loadConfigFromDB loads agent configuration from database and applies to LLM client
func (a *CursorAgent) loadConfigFromDB() {
	if a.store == nil {
		return
	}

	agentCfg, err := config.GetAgentConfig(a.store, AgentName)
	if err != nil {
		logger.Warn("Failed to load agent config from database",
			zap.String("agent", AgentName),
			zap.Error(err),
		)
		return
	}
	if agentCfg == nil {
		return
	}

	clientConfig := a.client.GetConfig()
	if clientConfig == nil {
		return
	}

	// Apply configuration from database
	if agentCfg.DefaultModel != "" {
		clientConfig.DefaultModel = agentCfg.DefaultModel
	} else {
		// Use default model for cursor agent when not configured
		clientConfig.DefaultModel = "composer-1"
	}
	if agentCfg.APIKey != "" {
		clientConfig.APIKey = agentCfg.APIKey
	}
	if agentCfg.CLIPath != "" {
		clientConfig.CLIPath = agentCfg.CLIPath
	}
	if agentCfg.Timeout > 0 {
		a.timeout = time.Duration(agentCfg.Timeout) * time.Second
	}
}

// ExecuteWithPrompt performs code review using a custom prompt (DSL mode)
// In DSL mode, the prompt is rendered from templates and expects markdown output,
// so we don't parse the output as JSON - just use it directly as summary.
func (a *CursorAgent) ExecuteWithPrompt(ctx context.Context, req *base.ReviewRequest, prompt string) (*base.ReviewResult, error) {
	startTime := time.Now()
	result := base.NewResult(req.RequestID, a.Name())
	result.StartedAt = startTime
	result.AgentVersion = a.version

	logger.Info("Starting Cursor agent review with custom prompt",
		zap.String("review_id", req.ReviewID),
		zap.String("request_id", req.RequestID),
		zap.String("repo_path", req.RepoPath),
	)

	// Execute using LLM client
	output, modelName, err := a.executeWithClient(ctx, req, prompt)
	if err != nil {
		logger.Error("Cursor agent execution failed",
			zap.Error(err),
			zap.String("request_id", req.RequestID),
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
	result.ModelName = modelName

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	logger.Info("Cursor agent review completed",
		zap.String("request_id", req.RequestID),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// executeWithClient executes the prompt using the LLM client
// Returns: (output content, model name, error)
func (a *CursorAgent) executeWithClient(ctx context.Context, req *base.ReviewRequest, prompt string) (string, string, error) {
	// Load latest configuration from database before execution
	a.loadConfigFromDB()

	// Build request with metadata including rule_id and review_id
	metadata := make(map[string]string)
	if req.RuleID != "" {
		metadata["rule_id"] = req.RuleID
	}
	if req.ReviewID != "" {
		metadata["review_id"] = req.ReviewID
	}

	llmReq := llm.NewRequest(prompt).
		WithWorkDir(req.RepoPath).
		WithOptions(&llm.RequestOptions{
			Timeout:  a.timeout,
			Metadata: metadata,
		})

	// Set model if specified in request (DSL override)
	if req.Model != "" {
		llmReq = llmReq.WithModel(req.Model)
	}

	// Execute (LLM client will use DefaultModel if request model is empty)
	resp, err := a.client.Execute(ctx, llmReq)
	if err != nil {
		return "", "", &base.AgentError{
			Agent:   AgentName,
			Message: "LLM client execution failed",
			Err:     err,
		}
	}

	return resp.Content, resp.Model, nil
}
