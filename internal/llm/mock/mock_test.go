package mock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/llm"
	"github.com/verustcode/verustcode/pkg/logger"
)

func init() {
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
}

func TestNewClient(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.True(t, client.Available())
}

func TestNewClient_WithConfig(t *testing.T) {
	config := llm.NewClientConfig(ClientName)
	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.True(t, client.Available())
}

func TestClient_Available(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)
	assert.True(t, client.Available())
}

func TestClient_Execute(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)

	req := llm.NewRequest("test prompt")
	ctx := context.Background()

	resp, err := client.Execute(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.Content, "Mock Code Review Result")
	assert.Contains(t, resp.Content, "Mock")
}

func TestClient_Execute_WithMetadata(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)

	req := llm.NewRequest("test prompt").
		WithOptions(&llm.RequestOptions{
			Metadata: map[string]string{
				"agent_name":    "test-agent",
				"agent_version": "2.0.0",
				"rule_id":       "rule-123",
			},
		})
	ctx := context.Background()

	resp, err := client.Execute(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.Content, "test-agent")
	assert.Contains(t, resp.Content, "2.0.0")
	assert.Contains(t, resp.Content, "rule-123")
}

func TestClient_ExecuteStream(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)

	req := llm.NewRequest("test prompt")
	ctx := context.Background()

	chunks := []*llm.StreamChunk{}
	callback := func(chunk *llm.StreamChunk) {
		chunks = append(chunks, chunk)
	}

	resp, err := client.ExecuteStream(ctx, req, callback)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, len(chunks), 0)
	assert.Contains(t, resp.Content, "Mock Code Review Result")
}

func TestClient_CreateSession(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)

	ctx := context.Background()
	sessionID, err := client.CreateSession(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, sessionID)
}

func TestClient_Close(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// TestClient_generateRandomID tests random ID generation indirectly
// through Execute method since generateRandomID is private
func TestClient_generateRandomID(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)

	req := llm.NewRequest("test prompt")
	ctx := context.Background()

	resp1, err := client.Execute(ctx, req)
	require.NoError(t, err)

	resp2, err := client.Execute(ctx, req)
	require.NoError(t, err)

	// Responses should contain different request IDs (generated randomly)
	// Extract request IDs from response content
	assert.NotEqual(t, resp1.Content, resp2.Content)
	// Both should contain "Request ID" which indicates random ID was generated
	assert.Contains(t, resp1.Content, "Request ID")
	assert.Contains(t, resp2.Content, "Request ID")
}

// TestClient_generateMarkdownResponse tests the markdown response generation indirectly
// through Execute method since generateMarkdownResponse is private
func TestClient_generateMarkdownResponse(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)

	req := llm.NewRequest("test prompt").
		WithOptions(&llm.RequestOptions{
			Metadata: map[string]string{
				"agent_name":    "test-agent",
				"agent_version": "1.0.0",
				"rule_id":       "rule-123",
			},
		})
	ctx := context.Background()

	resp, err := client.Execute(ctx, req)
	require.NoError(t, err)
	// Verify markdown response contains expected elements
	assert.Contains(t, resp.Content, "Mock Code Review Result")
	assert.Contains(t, resp.Content, "test-agent")
	assert.Contains(t, resp.Content, "1.0.0")
	assert.Contains(t, resp.Content, "rule-123")
	assert.Contains(t, resp.Content, "Findings")
}
