// Package mock implements a mock LLM Client for testing and development.
// It returns hardcoded responses with timestamps and random IDs to verify execution.
package mock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/llm"
)

// ClientName is the identifier for the Mock client
const ClientName = "mock"

func init() {
	// Register the Mock client factory
	llm.Register(ClientName, NewClient)
}

// Client implements the llm.Client interface for mock responses
type Client struct {
	*llm.BaseClient
}

// NewClient creates a new Mock client
func NewClient(config *llm.ClientConfig) (llm.Client, error) {
	if config == nil {
		config = llm.NewClientConfig(ClientName)
	}

	return &Client{
		BaseClient: llm.NewBaseClient(config),
	}, nil
}

// Available always returns true for mock client
func (c *Client) Available() bool {
	return true
}

// Execute performs a synchronous execution and returns a mock response
func (c *Client) Execute(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	startTime := time.Now()

	// Prepare request (应用默认值、schema、安全包装等)
	// 这样 mock 流程就和真实 client 一致了
	prepared, err := c.PrepareRequest(req)
	if err != nil {
		return nil, err
	}

	c.LogRequest(prepared, "execute")

	// Generate timestamp and random ID
	timestamp := time.Now().Format(time.RFC3339)
	requestID := c.generateRandomID()

	// Get metadata from request (agent name, version, etc.)
	agentName := prepared.GetMetadata("agent_name")
	if agentName == "" {
		agentName = "mock"
	}
	agentVersion := prepared.GetMetadata("agent_version")
	if agentVersion == "" {
		agentVersion = "1.0.0"
	}

	// Build response
	model := c.GetConfig().GetModel(prepared)
	if model == "" {
		model = "mock-model"
	}

	// Get rule_id from metadata
	ruleID := prepared.GetMetadata("rule_id")

	// DSL mode always returns markdown format
	content := c.generateMarkdownResponse(timestamp, requestID, agentName, agentVersion, model, ruleID)

	resp := c.BuildResponse(content, model, prepared.SessionID, prepared.ResponseSchema)

	// Copy metadata from request to response (including rule_id)
	if prepared.Options != nil && prepared.Options.Metadata != nil {
		if resp.Metadata == nil {
			resp.Metadata = make(map[string]string)
		}
		for k, v := range prepared.Options.Metadata {
			resp.Metadata[k] = v
		}
	}

	c.LogResponse(resp, time.Since(startTime), nil)
	return resp, nil
}

// ExecuteStream performs a streaming execution with callback
func (c *Client) ExecuteStream(ctx context.Context, req *llm.Request, callback llm.StreamCallback) (*llm.Response, error) {
	startTime := time.Now()

	// Prepare request (应用默认值、schema、安全包装等)
	// 这样 mock 流程就和真实 client 一致了
	prepared, err := c.PrepareRequest(req)
	if err != nil {
		return nil, err
	}

	c.LogRequest(prepared, "execute_stream")

	// Generate timestamp and random ID
	timestamp := time.Now().Format(time.RFC3339)
	requestID := c.generateRandomID()

	// Get metadata from request (agent name, version, etc.)
	agentName := prepared.GetMetadata("agent_name")
	if agentName == "" {
		agentName = "mock"
	}
	agentVersion := prepared.GetMetadata("agent_version")
	if agentVersion == "" {
		agentVersion = "1.0.0"
	}

	// Build response
	model := c.GetConfig().GetModel(prepared)
	if model == "" {
		model = "mock-model"
	}

	// Get rule_id from metadata
	ruleID := prepared.GetMetadata("rule_id")

	// DSL mode always returns markdown format
	content := c.generateMarkdownResponse(timestamp, requestID, agentName, agentVersion, model, ruleID)

	// Simulate streaming by sending chunks
	if callback != nil {
		// Send system chunk
		callback(&llm.StreamChunk{
			Type:    llm.ChunkTypeSystem,
			Content: "Mock client streaming response",
			Delta:   "Mock client streaming response",
		})

		// Send text chunks (split content into smaller pieces)
		chunkSize := 50
		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}
			delta := content[i:end]
			callback(&llm.StreamChunk{
				Type:    llm.ChunkTypeText,
				Content: content[:end],
				Delta:   delta,
			})
		}

		// Send final result chunk
		callback(&llm.StreamChunk{
			Type:       llm.ChunkTypeResult,
			Content:    content,
			Delta:      "",
			IsComplete: true,
		})
	}

	resp := c.BuildResponse(content, model, prepared.SessionID, prepared.ResponseSchema)

	c.LogResponse(resp, time.Since(startTime), nil)
	return resp, nil
}

// CreateSession creates a new conversation session
func (c *Client) CreateSession(ctx context.Context) (string, error) {
	c.Logger().Debug("Creating mock session")
	sessionID := c.generateRandomID()
	c.Logger().Info("Mock session created", zap.String("session_id", sessionID))
	return sessionID, nil
}

// Close releases any resources held by the client
func (c *Client) Close() error {
	// No resources to release for mock client
	return nil
}

// generateRandomID generates a random hexadecimal string of 16 characters
func (c *Client) generateRandomID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based ID if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// generateMarkdownResponse generates a Markdown format response with timestamp, random ID, and metadata
func (c *Client) generateMarkdownResponse(timestamp, requestID, agentName, agentVersion, model, ruleID string) string {
	findingID1 := c.generateRandomID()[:8]
	findingID2 := c.generateRandomID()[:8]

	// Use default values if not provided
	if model == "" {
		model = "mock-model"
	}

	// Build metadata section
	metadataSection := fmt.Sprintf(`## Metadata

- **Agent**: %s
- **Agent Version**: %s
- **Model**: %s
- **Client**: %s`, agentName, agentVersion, model, ClientName)

	if ruleID != "" {
		metadataSection += fmt.Sprintf("\n- **Rule ID**: %s", ruleID)
	}

	return fmt.Sprintf(`# Mock Code Review Result

**Generated At**: %s  
**Request ID**: %s

%s

This is a mock response for testing purposes.

## Findings

- **Style Issue**: Code formatting could be improved [ID: %s]
- **Logic Issue**: Potential edge case not handled [ID: %s]

## Summary

Mock review completed successfully at %s.
`, timestamp, requestID, metadataSection, findingID1, findingID2, timestamp)
}
