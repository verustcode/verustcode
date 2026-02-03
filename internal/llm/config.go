package llm

import (
	"time"
)

// Default configuration values
const (
	DefaultTimeout    = 10 * time.Minute
	DefaultMaxRetries = 3
	DefaultRetryDelay = 5 * time.Second
)

// ClientConfig contains configuration for an LLM client
type ClientConfig struct {
	// Name is the client identifier (e.g., "cursor", "claude", "gemini")
	Name string

	// CLIPath is the path to the CLI tool executable
	CLIPath string

	// APIKey is the API key for authentication (if required)
	APIKey string

	// DefaultModel is the default model to use if not specified in request
	DefaultModel string

	// DefaultTimeout is the default request timeout
	DefaultTimeout time.Duration

	// MaxRetries is the default maximum number of retry attempts
	MaxRetries int

	// RetryDelay is the default delay between retry attempts
	RetryDelay time.Duration

	// ExtraArgs contains additional command line arguments to append when executing the CLI
	// These arguments will be added after the default arguments (space-separated string)
	ExtraArgs string
}

// NewClientConfig creates a new ClientConfig with default values
func NewClientConfig(name string) *ClientConfig {
	return &ClientConfig{
		Name:           name,
		DefaultTimeout: DefaultTimeout,
		MaxRetries:     DefaultMaxRetries,
		RetryDelay:     DefaultRetryDelay,
	}
}

// WithCLIPath sets the CLI path
func (c *ClientConfig) WithCLIPath(path string) *ClientConfig {
	c.CLIPath = path
	return c
}

// WithAPIKey sets the API key
func (c *ClientConfig) WithAPIKey(key string) *ClientConfig {
	c.APIKey = key
	return c
}

// WithDefaultModel sets the default model
func (c *ClientConfig) WithDefaultModel(model string) *ClientConfig {
	c.DefaultModel = model
	return c
}

// WithDefaultTimeout sets the default timeout
func (c *ClientConfig) WithDefaultTimeout(timeout time.Duration) *ClientConfig {
	c.DefaultTimeout = timeout
	return c
}

// WithMaxRetries sets the max retries
func (c *ClientConfig) WithMaxRetries(retries int) *ClientConfig {
	c.MaxRetries = retries
	return c
}

// WithRetryDelay sets the retry delay
func (c *ClientConfig) WithRetryDelay(delay time.Duration) *ClientConfig {
	c.RetryDelay = delay
	return c
}

// WithExtraArgs sets additional command line arguments (space-separated string)
func (c *ClientConfig) WithExtraArgs(args string) *ClientConfig {
	c.ExtraArgs = args
	return c
}

// GetTimeout returns the timeout to use, considering request options
func (c *ClientConfig) GetTimeout(req *Request) time.Duration {
	if req != nil {
		return req.GetTimeout(c.DefaultTimeout)
	}
	return c.DefaultTimeout
}

// GetModel returns the model to use, considering request and default
func (c *ClientConfig) GetModel(req *Request) string {
	if req != nil && req.Model != "" {
		return req.Model
	}
	return c.DefaultModel
}
