package llm

import (
	"errors"
	"fmt"
)

// Sentinel errors for common error conditions
var (
	// ErrClientNotAvailable indicates the CLI tool is not available
	ErrClientNotAvailable = errors.New("client CLI tool not available")

	// ErrSessionNotSupported indicates the client doesn't support sessions
	ErrSessionNotSupported = errors.New("session not supported by this client")

	// ErrModelNotAvailable indicates the requested model is not available
	ErrModelNotAvailable = errors.New("model not available")

	// ErrTimeout indicates the request timed out
	ErrTimeout = errors.New("request timeout")

	// ErrMaxRetriesExceeded indicates all retry attempts failed
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")

	// ErrPromptInjection indicates a potential prompt injection attack was detected
	ErrPromptInjection = errors.New("potential prompt injection detected")

	// ErrInvalidResponse indicates the response could not be parsed
	ErrInvalidResponse = errors.New("invalid response format")

	// ErrSessionCreateFailed indicates session creation failed
	ErrSessionCreateFailed = errors.New("failed to create session")
)

// ClientError represents an error from an LLM client
type ClientError struct {
	// Client is the name of the client that produced the error
	Client string

	// Operation is the operation that failed (e.g., "execute", "create_session")
	Operation string

	// Message is the error message
	Message string

	// Err is the underlying error (if any)
	Err error

	// Retryable indicates whether the operation can be retried
	Retryable bool
}

// Error implements the error interface
func (e *ClientError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s.%s] %s: %v", e.Client, e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s.%s] %s", e.Client, e.Operation, e.Message)
}

// Unwrap returns the underlying error
func (e *ClientError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is for ClientError
func (e *ClientError) Is(target error) bool {
	if e.Err != nil {
		return errors.Is(e.Err, target)
	}
	return false
}

// NewClientError creates a new ClientError
func NewClientError(client, operation, message string, err error) *ClientError {
	return &ClientError{
		Client:    client,
		Operation: operation,
		Message:   message,
		Err:       err,
		Retryable: false,
	}
}

// NewRetryableError creates a new retryable ClientError
func NewRetryableError(client, operation, message string, err error) *ClientError {
	return &ClientError{
		Client:    client,
		Operation: operation,
		Message:   message,
		Err:       err,
		Retryable: true,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var clientErr *ClientError
	if errors.As(err, &clientErr) {
		return clientErr.Retryable
	}
	return false
}

// IsModelError checks if the error is related to model availability
func IsModelError(err error, output string) bool {
	if err == nil {
		return false
	}

	// Check if it's explicitly a model error
	if errors.Is(err, ErrModelNotAvailable) {
		return true
	}

	// Check error message and output for model-related keywords
	errStr := err.Error()
	keywords := []string{
		"model",
		"invalid model",
		"model not found",
		"model unavailable",
		"model not available",
		"unknown model",
		"unsupported model",
		"model error",
	}

	for _, keyword := range keywords {
		if containsIgnoreCase(errStr, keyword) || containsIgnoreCase(output, keyword) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			findIgnoreCase(s, substr) >= 0)
}

// findIgnoreCase finds substr in s (case-insensitive)
func findIgnoreCase(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}

	// Simple case-insensitive search
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return i
		}
	}
	return -1
}

// toLower converts a string to lowercase (ASCII only for performance)
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
