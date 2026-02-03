package errors

import (
	"errors"
	"net/http"
	"testing"
)

// TestNew tests creating a new AppError
func TestNew(t *testing.T) {
	err := New(ErrCodeValidation, "validation failed")

	if err == nil {
		t.Fatal("New() returned nil")
	}

	if err.Code != ErrCodeValidation {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeValidation)
	}

	if err.Message != "validation failed" {
		t.Errorf("Message = %s, want 'validation failed'", err.Message)
	}

	if err.Err != nil {
		t.Error("Err should be nil for New()")
	}
}

// TestWrap tests wrapping an existing error
func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := Wrap(ErrCodeInternal, "wrapped error", originalErr)

	if err == nil {
		t.Fatal("Wrap() returned nil")
	}

	if err.Code != ErrCodeInternal {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeInternal)
	}

	if err.Message != "wrapped error" {
		t.Errorf("Message = %s, want 'wrapped error'", err.Message)
	}

	if err.Err != originalErr {
		t.Error("Err should be the original error")
	}
}

// TestAppError_Error tests the Error method
func TestAppError_Error(t *testing.T) {
	t.Run("without underlying error", func(t *testing.T) {
		err := New(ErrCodeValidation, "invalid input")
		errStr := err.Error()

		expectedPrefix := "[E1001]"
		if len(errStr) < len(expectedPrefix) {
			t.Errorf("Error() = %s, too short", errStr)
		}

		if errStr != "[E1001] invalid input" {
			t.Errorf("Error() = %s, want '[E1001] invalid input'", errStr)
		}
	})

	t.Run("with underlying error", func(t *testing.T) {
		originalErr := errors.New("file not found")
		err := Wrap(ErrCodeConfigNotFound, "config error", originalErr)
		errStr := err.Error()

		if errStr != "[E6001] config error: file not found" {
			t.Errorf("Error() = %s, want '[E6001] config error: file not found'", errStr)
		}
	})
}

// TestAppError_Unwrap tests the Unwrap method
func TestAppError_Unwrap(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		originalErr := errors.New("original")
		err := Wrap(ErrCodeInternal, "message", originalErr)

		unwrapped := err.Unwrap()
		if unwrapped != originalErr {
			t.Error("Unwrap() should return the original error")
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := New(ErrCodeValidation, "message")

		unwrapped := err.Unwrap()
		if unwrapped != nil {
			t.Error("Unwrap() should return nil when no underlying error")
		}
	})

	t.Run("errors.Unwrap compatibility", func(t *testing.T) {
		originalErr := errors.New("original")
		err := Wrap(ErrCodeInternal, "message", originalErr)

		unwrapped := errors.Unwrap(err)
		if unwrapped != originalErr {
			t.Error("errors.Unwrap() should return the original error")
		}
	})
}

// TestAppError_HTTPStatus tests the HTTPStatus method
func TestAppError_HTTPStatus(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected int
	}{
		// Not Found errors
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeGitNotFound, http.StatusNotFound},
		{ErrCodeAgentNotFound, http.StatusNotFound},
		{ErrCodeReviewNotFound, http.StatusNotFound},

		// Bad Request
		{ErrCodeValidation, http.StatusBadRequest},

		// Unauthorized
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeGitAuth, http.StatusUnauthorized},

		// Forbidden
		{ErrCodeForbidden, http.StatusForbidden},

		// Conflict
		{ErrCodeConflict, http.StatusConflict},

		// Gateway Timeout
		{ErrCodeAgentTimeout, http.StatusGatewayTimeout},

		// Service Unavailable
		{ErrCodeAgentUnavailable, http.StatusServiceUnavailable},

		// Internal Server Error (default)
		{ErrCodeInternal, http.StatusInternalServerError},
		{ErrCodeGitClone, http.StatusInternalServerError},
		{ErrCodeDBConnection, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := New(tt.code, "test error")
			status := err.HTTPStatus()

			if status != tt.expected {
				t.Errorf("HTTPStatus() = %d, want %d", status, tt.expected)
			}
		})
	}
}

// TestAppError_WithDetails tests the WithDetails method
func TestAppError_WithDetails(t *testing.T) {
	err := New(ErrCodeValidation, "validation error")

	details := map[string]string{
		"field": "email",
		"error": "invalid format",
	}

	result := err.WithDetails(details)

	// Should return the same error (chainable)
	if result != err {
		t.Error("WithDetails() should return the same error")
	}

	if err.Details == nil {
		t.Fatal("Details should not be nil after WithDetails()")
	}

	detailsMap, ok := err.Details.(map[string]string)
	if !ok {
		t.Fatal("Details should be map[string]string")
	}

	if detailsMap["field"] != "email" {
		t.Errorf("Details[field] = %s, want 'email'", detailsMap["field"])
	}
}

// TestErrInternal tests the ErrInternal convenience function
func TestErrInternal(t *testing.T) {
	originalErr := errors.New("database connection failed")
	err := ErrInternal("internal error", originalErr)

	if err.Code != ErrCodeInternal {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeInternal)
	}

	if err.Err != originalErr {
		t.Error("Err should be the original error")
	}
}

// TestErrValidation tests the ErrValidation convenience function
func TestErrValidation(t *testing.T) {
	err := ErrValidation("email is required")

	if err.Code != ErrCodeValidation {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeValidation)
	}

	if err.Message != "email is required" {
		t.Errorf("Message = %s, want 'email is required'", err.Message)
	}
}

// TestErrNotFound tests the ErrNotFound convenience function
func TestErrNotFound(t *testing.T) {
	err := ErrNotFound("user")

	if err.Code != ErrCodeNotFound {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeNotFound)
	}

	expectedMsg := "user not found"
	if err.Message != expectedMsg {
		t.Errorf("Message = %s, want %s", err.Message, expectedMsg)
	}
}

// TestErrUnauthorized tests the ErrUnauthorized convenience function
func TestErrUnauthorized(t *testing.T) {
	err := ErrUnauthorized("invalid token")

	if err.Code != ErrCodeUnauthorized {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeUnauthorized)
	}

	if err.Message != "invalid token" {
		t.Errorf("Message = %s, want 'invalid token'", err.Message)
	}
}

// TestErrForbidden tests the ErrForbidden convenience function
func TestErrForbidden(t *testing.T) {
	err := ErrForbidden("access denied")

	if err.Code != ErrCodeForbidden {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeForbidden)
	}

	if err.Message != "access denied" {
		t.Errorf("Message = %s, want 'access denied'", err.Message)
	}
}

// TestIsAppError tests the IsAppError function
func TestIsAppError(t *testing.T) {
	t.Run("AppError", func(t *testing.T) {
		err := New(ErrCodeValidation, "test")
		if !IsAppError(err) {
			t.Error("IsAppError() should return true for AppError")
		}
	})

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("regular error")
		if IsAppError(err) {
			t.Error("IsAppError() should return false for regular error")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		if IsAppError(nil) {
			t.Error("IsAppError() should return false for nil")
		}
	})
}

// TestAsAppError tests the AsAppError function
func TestAsAppError(t *testing.T) {
	t.Run("AppError", func(t *testing.T) {
		original := New(ErrCodeValidation, "test")
		appErr, ok := AsAppError(original)

		if !ok {
			t.Error("AsAppError() should return true for AppError")
		}

		if appErr != original {
			t.Error("AsAppError() should return the same error")
		}
	})

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("regular error")
		_, ok := AsAppError(err)

		if ok {
			t.Error("AsAppError() should return false for regular error")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		_, ok := AsAppError(nil)
		if ok {
			t.Error("AsAppError() should return false for nil")
		}
	})
}

// TestErrorCodes tests that all error codes are unique
func TestErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrCodeInternal,
		ErrCodeValidation,
		ErrCodeNotFound,
		ErrCodeConflict,
		ErrCodeForbidden,
		ErrCodeUnauthorized,
		ErrCodeGitClone,
		ErrCodeGitAuth,
		ErrCodeGitNotFound,
		ErrCodeGitWebhook,
		ErrCodeAgentNotFound,
		ErrCodeAgentUnavailable,
		ErrCodeAgentTimeout,
		ErrCodeAgentExecution,
		ErrCodeReviewNotFound,
		ErrCodeReviewFailed,
		ErrCodeReviewPending,
		ErrCodeDBConnection,
		ErrCodeDBQuery,
		ErrCodeDBMigration,
		ErrCodeConfigNotFound,
		ErrCodeConfigInvalid,
		ErrCodeConfigParse,
	}

	seen := make(map[ErrorCode]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("Duplicate error code: %s", code)
		}
		seen[code] = true

		// Verify code format
		if len(code) == 0 {
			t.Error("Error code should not be empty")
		}
	}
}

// TestAppErrorImplementsError tests that AppError implements the error interface
func TestAppErrorImplementsError(t *testing.T) {
	var err error = New(ErrCodeValidation, "test")

	if err == nil {
		t.Error("AppError should implement error interface")
	}

	// Should be usable as a regular error
	_ = err.Error()
}


