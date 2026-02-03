// Package llm provides a unified interface for interacting with various LLM CLI tools.
// It abstracts away the differences between cursor-agent, gemini, etc.
package llm

import (
	"context"
)

// Client defines the interface for LLM CLI clients.
// Different implementations (Cursor, Gemini) implement this interface.
type Client interface {
	// Name returns the client identifier (e.g., "cursor", "claude", "gemini")
	Name() string

	// Available checks if the client CLI tool is available for use
	Available() bool

	// GetConfig returns the client configuration for reading or updating
	GetConfig() *ClientConfig

	// Execute performs a synchronous execution and returns the complete response.
	// If req.SessionID is non-empty, it uses multi-turn conversation mode.
	Execute(ctx context.Context, req *Request) (*Response, error)

	// ExecuteStream performs a streaming execution, calling the callback for each chunk.
	// If req.SessionID is non-empty, it uses multi-turn conversation mode.
	ExecuteStream(ctx context.Context, req *Request, callback StreamCallback) (*Response, error)

	// CreateSession creates a new conversation session and returns the session ID.
	// Use this for multi-turn conversations, then pass the session ID in Request.SessionID.
	CreateSession(ctx context.Context) (string, error)

	// Close releases any resources held by the client
	Close() error
}

// StreamCallback is the callback function for streaming output
type StreamCallback func(chunk *StreamChunk)

// ChunkType represents the type of streaming data chunk
type ChunkType string

const (
	// ChunkTypeText represents text output from the LLM
	ChunkTypeText ChunkType = "text"
	// ChunkTypeThinking represents the thinking/reasoning process
	ChunkTypeThinking ChunkType = "thinking"
	// ChunkTypeToolCall represents a tool call initiation
	ChunkTypeToolCall ChunkType = "tool_call"
	// ChunkTypeToolResult represents a tool call result
	ChunkTypeToolResult ChunkType = "tool_result"
	// ChunkTypeError represents an error message
	ChunkTypeError ChunkType = "error"
	// ChunkTypeSystem represents system messages (e.g., initialization)
	ChunkTypeSystem ChunkType = "system"
	// ChunkTypeResult represents the final result
	ChunkTypeResult ChunkType = "result"
)

// StreamChunk represents a chunk of streaming data
type StreamChunk struct {
	// Type indicates the kind of chunk (text, tool_call, etc.)
	Type ChunkType

	// Content is the accumulated content (mainly for text type)
	Content string

	// Delta is the incremental content since the last chunk
	Delta string

	// IsComplete indicates whether the stream is complete
	IsComplete bool

	// Tool call related fields (only for tool_call/tool_result types)
	ToolName   string // Name of the tool being called
	ToolInput  string // Tool input parameters (JSON)
	ToolOutput string // Tool output result

	// Metadata contains additional information
	Metadata map[string]string
}
