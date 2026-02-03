package llm

import (
	"time"
)

// Request represents a request to the LLM client
type Request struct {
	// Prompt is the input prompt/message to send to the LLM
	Prompt string

	// Model specifies which model to use (e.g., "sonnet-4.5", "opus-4.5", "composer-1")
	Model string

	// SessionID is the conversation session ID (optional)
	// - Empty: single-turn conversation, no session created/used
	// - Non-empty: multi-turn conversation, uses the specified session (call CreateSession first)
	SessionID string

	// WorkDir is the working directory for the CLI tool
	WorkDir string

	// ResponseSchema defines the expected response structure (optional)
	// When provided, the client will add JSON format instructions to the prompt
	// and attempt to parse the response into the specified structure
	ResponseSchema *ResponseSchema

	// Options contains optional configuration
	Options *RequestOptions
}

// ResponseSchema defines the expected response structure for structured output
type ResponseSchema struct {
	// Name is the schema name (e.g., "code_review_result")
	Name string

	// Description describes what the schema represents
	Description string

	// Schema is the JSON Schema definition
	// Can be a map[string]interface{} or a Go struct (will be converted to JSON Schema)
	Schema interface{}

	// Strict indicates whether the LLM must strictly follow the schema
	Strict bool
}

// RequestOptions contains optional request configuration
type RequestOptions struct {
	// Timeout is the maximum duration for the request
	Timeout time.Duration

	// FallbackModels is a list of backup models to try if the primary model fails
	FallbackModels []string

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration

	// Metadata contains additional information (e.g., task_id, name)
	Metadata map[string]string
}

// Response represents the response from the LLM client
type Response struct {
	// Content is the raw response content
	Content string

	// SessionID is the session ID used (if any)
	SessionID string

	// Model is the actual model used for the request
	Model string

	// Metadata contains additional response information
	Metadata map[string]string

	// Usage contains token usage statistics (optional)
	Usage *Usage

	// Parsed is the parsed structured data (if ResponseSchema was provided)
	Parsed interface{}

	// ParseErr is the error from parsing (if parsing failed)
	ParseErr error
}

// Usage represents token usage statistics
type Usage struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int

	// TotalTokens is the total number of tokens used
	TotalTokens int
}

// NewRequest creates a new Request with default values
func NewRequest(prompt string) *Request {
	return &Request{
		Prompt: prompt,
	}
}

// WithModel sets the model for the request
func (r *Request) WithModel(model string) *Request {
	r.Model = model
	return r
}

// WithSessionID sets the session ID for multi-turn conversation
func (r *Request) WithSessionID(sessionID string) *Request {
	r.SessionID = sessionID
	return r
}

// WithWorkDir sets the working directory
func (r *Request) WithWorkDir(workDir string) *Request {
	r.WorkDir = workDir
	return r
}

// WithSchema sets the response schema for structured output
func (r *Request) WithSchema(schema *ResponseSchema) *Request {
	r.ResponseSchema = schema
	return r
}

// WithOptions sets the request options
func (r *Request) WithOptions(opts *RequestOptions) *Request {
	r.Options = opts
	return r
}

// GetTimeout returns the timeout from options, or the default value
func (r *Request) GetTimeout(defaultTimeout time.Duration) time.Duration {
	if r.Options != nil && r.Options.Timeout > 0 {
		return r.Options.Timeout
	}
	return defaultTimeout
}

// GetMaxRetries returns the max retries from options, or the default value
func (r *Request) GetMaxRetries(defaultRetries int) int {
	if r.Options != nil && r.Options.MaxRetries > 0 {
		return r.Options.MaxRetries
	}
	return defaultRetries
}

// GetRetryDelay returns the retry delay from options, or the default value
func (r *Request) GetRetryDelay(defaultDelay time.Duration) time.Duration {
	if r.Options != nil && r.Options.RetryDelay > 0 {
		return r.Options.RetryDelay
	}
	return defaultDelay
}

// GetFallbackModels returns the fallback models from options
func (r *Request) GetFallbackModels() []string {
	if r.Options != nil {
		return r.Options.FallbackModels
	}
	return nil
}

// GetMetadata returns a metadata value, or empty string if not found
func (r *Request) GetMetadata(key string) string {
	if r.Options != nil && r.Options.Metadata != nil {
		return r.Options.Metadata[key]
	}
	return ""
}

