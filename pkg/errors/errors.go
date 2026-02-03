// Package errors provides custom error types for the application.
// It defines domain-specific errors with error codes for better error handling and API responses.
package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents application error codes
type ErrorCode string

// Error codes for different error categories
const (
	// General errors (1xxx)
	ErrCodeInternal     ErrorCode = "E1000"
	ErrCodeValidation   ErrorCode = "E1001"
	ErrCodeNotFound     ErrorCode = "E1002"
	ErrCodeConflict     ErrorCode = "E1003"
	ErrCodeForbidden    ErrorCode = "E1004"
	ErrCodeUnauthorized ErrorCode = "E1005"

	// Git provider errors (2xxx)
	ErrCodeGitClone    ErrorCode = "E2001"
	ErrCodeGitAuth     ErrorCode = "E2002"
	ErrCodeGitNotFound ErrorCode = "E2003"
	ErrCodeGitWebhook  ErrorCode = "E2004"

	// Agent errors (3xxx)
	ErrCodeAgentNotFound    ErrorCode = "E3001"
	ErrCodeAgentUnavailable ErrorCode = "E3002"
	ErrCodeAgentTimeout     ErrorCode = "E3003"
	ErrCodeAgentExecution   ErrorCode = "E3004"

	// Review errors (4xxx)
	ErrCodeReviewNotFound ErrorCode = "E4001"
	ErrCodeReviewFailed   ErrorCode = "E4002"
	ErrCodeReviewPending  ErrorCode = "E4003"

	// Database errors (5xxx)
	ErrCodeDBConnection ErrorCode = "E5001"
	ErrCodeDBQuery      ErrorCode = "E5002"
	ErrCodeDBMigration  ErrorCode = "E5003"

	// Configuration errors (6xxx)
	ErrCodeConfigNotFound          ErrorCode = "E6001"
	ErrCodeConfigInvalid           ErrorCode = "E6002"
	ErrCodeConfigParse             ErrorCode = "E6003"
	ErrCodeAdminCredentialsEmpty   ErrorCode = "E6004"
	ErrCodePasswordComplexity      ErrorCode = "E6005"
	ErrCodeJWTSecretInvalid        ErrorCode = "E6006"
)

// Exit codes for application startup failures
const (
	// ExitCodeConfigValidation indicates configuration validation failure (e.g., empty credentials, weak password)
	ExitCodeConfigValidation = 2
)

// AppError represents an application-level error with code and context
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Err     error     `json:"-"`
	Details any       `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus returns the HTTP status code for the error
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeNotFound, ErrCodeGitNotFound, ErrCodeAgentNotFound, ErrCodeReviewNotFound:
		return http.StatusNotFound
	case ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeUnauthorized, ErrCodeGitAuth:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeAgentTimeout:
		return http.StatusGatewayTimeout
	case ErrCodeAgentUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with AppError
func Wrap(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// Common error constructors for convenience

// ErrInternal creates an internal server error
func ErrInternal(message string, err error) *AppError {
	return Wrap(ErrCodeInternal, message, err)
}

// ErrValidation creates a validation error
func ErrValidation(message string) *AppError {
	return New(ErrCodeValidation, message)
}

// ErrNotFound creates a not found error
func ErrNotFound(resource string) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found", resource))
}

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(message string) *AppError {
	return New(ErrCodeUnauthorized, message)
}

// ErrForbidden creates a forbidden error
func ErrForbidden(message string) *AppError {
	return New(ErrCodeForbidden, message)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError attempts to convert an error to AppError
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}







