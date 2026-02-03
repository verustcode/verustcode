// Package telemetry provides OpenTelemetry integration for the application.
// This file contains unit tests for the metrics.
package telemetry

import (
	"context"
	"testing"
)

// TestGetMetrics tests the GetMetrics function
func TestGetMetrics(t *testing.T) {
	metrics := GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics() returned nil")
	}

	// Second call should return same instance
	metrics2 := GetMetrics()
	if metrics != metrics2 {
		t.Error("GetMetrics() returned different instances on subsequent calls")
	}
}

// TestMetricsRecordReviewStarted tests RecordReviewStarted
func TestMetricsRecordReviewStarted(t *testing.T) {
	metrics := GetMetrics()
	ctx := context.Background()

	// Should not panic even if metrics are nil/empty
	metrics.RecordReviewStarted(ctx, "cursor", "github")
}

// TestMetricsRecordReviewCompleted tests RecordReviewCompleted
func TestMetricsRecordReviewCompleted(t *testing.T) {
	metrics := GetMetrics()
	ctx := context.Background()

	// Should not panic
	metrics.RecordReviewCompleted(ctx, "completed", 10.5)
}

// TestMetricsRecordFindings tests RecordFindings
func TestMetricsRecordFindings(t *testing.T) {
	metrics := GetMetrics()
	ctx := context.Background()

	// Should not panic
	metrics.RecordFindings(ctx, "security", 5)
}

// TestMetricsRecordHTTPRequest tests RecordHTTPRequest
func TestMetricsRecordHTTPRequest(t *testing.T) {
	metrics := GetMetrics()
	ctx := context.Background()

	// Should not panic
	metrics.RecordHTTPRequest(ctx, "GET", "/api/v1/reviews", 200, 0.05)
	metrics.RecordHTTPRequest(ctx, "POST", "/api/v1/webhooks", 201, 0.1)
	metrics.RecordHTTPRequest(ctx, "GET", "/api/v1/reviews/123", 404, 0.01)
}

// TestMetricsRecordAgentExecution tests RecordAgentExecution
func TestMetricsRecordAgentExecution(t *testing.T) {
	metrics := GetMetrics()
	ctx := context.Background()

	// Should not panic
	metrics.RecordAgentExecution(ctx, "cursor", true)
	metrics.RecordAgentExecution(ctx, "cursor", false)
	metrics.RecordAgentExecution(ctx, "gemini", true)
}

// TestMetricsRecordGitClone tests RecordGitClone
func TestMetricsRecordGitClone(t *testing.T) {
	metrics := GetMetrics()
	ctx := context.Background()

	// Should not panic
	metrics.RecordGitClone(ctx, "github", true, 5.5)
	metrics.RecordGitClone(ctx, "gitlab", false, 30.0)
}

// TestMetricsNilSafe tests that metrics methods are nil-safe
func TestMetricsNilSafe(t *testing.T) {
	// Create empty metrics struct (simulating initialization failure)
	emptyMetrics := &Metrics{}
	ctx := context.Background()

	// None of these should panic
	t.Run("RecordReviewStarted", func(t *testing.T) {
		emptyMetrics.RecordReviewStarted(ctx, "test", "test")
	})

	t.Run("RecordReviewCompleted", func(t *testing.T) {
		emptyMetrics.RecordReviewCompleted(ctx, "completed", 1.0)
	})

	t.Run("RecordFindings", func(t *testing.T) {
		emptyMetrics.RecordFindings(ctx, "test", 1)
	})

	t.Run("RecordHTTPRequest", func(t *testing.T) {
		emptyMetrics.RecordHTTPRequest(ctx, "GET", "/test", 200, 0.1)
	})

	t.Run("RecordAgentExecution", func(t *testing.T) {
		emptyMetrics.RecordAgentExecution(ctx, "test", true)
	})

	t.Run("RecordGitClone", func(t *testing.T) {
		emptyMetrics.RecordGitClone(ctx, "test", true, 1.0)
	})
}
