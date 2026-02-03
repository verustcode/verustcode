// Package telemetry provides OpenTelemetry integration for the application.
package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/pkg/logger"
)

const (
	// MeterName is the default meter name for the application
	MeterName = "github.com/verustcode/verustcode"
)

// Metrics holds all application metrics
type Metrics struct {
	// Review metrics
	ReviewsTotal      metric.Int64Counter
	ReviewDuration    metric.Float64Histogram
	ActiveReviews     metric.Int64UpDownCounter
	ReviewsByStatus   metric.Int64Counter
	ReviewsByCategory metric.Int64Counter

	// HTTP metrics
	HTTPRequestsTotal   metric.Int64Counter
	HTTPRequestDuration metric.Float64Histogram

	// Agent metrics
	AgentExecutionsTotal metric.Int64Counter
	AgentExecutionErrors metric.Int64Counter

	// Git metrics
	GitCloneTotal    metric.Int64Counter
	GitCloneDuration metric.Float64Histogram
}

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// GetMetrics returns the global metrics instance, initializing it if necessary
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		var err error
		globalMetrics, err = initMetrics()
		if err != nil {
			logger.Error("Failed to initialize metrics", zap.Error(err))
			// Return empty metrics to avoid nil pointer
			globalMetrics = &Metrics{}
		}
	})
	return globalMetrics
}

// initMetrics initializes all application metrics
func initMetrics() (*Metrics, error) {
	meter := otel.Meter(MeterName)
	m := &Metrics{}

	var err error

	// Review metrics
	m.ReviewsTotal, err = meter.Int64Counter(
		"scopeview_reviews_total",
		metric.WithDescription("Total number of code reviews"),
		metric.WithUnit("{review}"),
	)
	if err != nil {
		return nil, err
	}

	m.ReviewDuration, err = meter.Float64Histogram(
		"scopeview_review_duration_seconds",
		metric.WithDescription("Duration of code reviews in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 30, 60, 120, 300, 600, 1800),
	)
	if err != nil {
		return nil, err
	}

	m.ActiveReviews, err = meter.Int64UpDownCounter(
		"scopeview_active_reviews",
		metric.WithDescription("Number of currently active reviews"),
		metric.WithUnit("{review}"),
	)
	if err != nil {
		return nil, err
	}

	m.ReviewsByStatus, err = meter.Int64Counter(
		"scopeview_reviews_by_status_total",
		metric.WithDescription("Total number of reviews by status"),
		metric.WithUnit("{review}"),
	)
	if err != nil {
		return nil, err
	}

	m.ReviewsByCategory, err = meter.Int64Counter(
		"scopeview_reviews_by_category_total",
		metric.WithDescription("Total number of review findings by category"),
		metric.WithUnit("{finding}"),
	)
	if err != nil {
		return nil, err
	}

	// HTTP metrics
	m.HTTPRequestsTotal, err = meter.Int64Counter(
		"scopeview_http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	m.HTTPRequestDuration, err = meter.Float64Histogram(
		"scopeview_http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, err
	}

	// Agent metrics
	m.AgentExecutionsTotal, err = meter.Int64Counter(
		"scopeview_agent_executions_total",
		metric.WithDescription("Total number of agent executions"),
		metric.WithUnit("{execution}"),
	)
	if err != nil {
		return nil, err
	}

	m.AgentExecutionErrors, err = meter.Int64Counter(
		"scopeview_agent_execution_errors_total",
		metric.WithDescription("Total number of agent execution errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	// Git metrics
	m.GitCloneTotal, err = meter.Int64Counter(
		"scopeview_git_clone_total",
		metric.WithDescription("Total number of git clone operations"),
		metric.WithUnit("{clone}"),
	)
	if err != nil {
		return nil, err
	}

	m.GitCloneDuration, err = meter.Float64Histogram(
		"scopeview_git_clone_duration_seconds",
		metric.WithDescription("Duration of git clone operations in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 30, 60, 120, 300),
	)
	if err != nil {
		return nil, err
	}

	logger.Info("Metrics initialized successfully")
	return m, nil
}

// RecordReviewStarted records that a review has started
func (m *Metrics) RecordReviewStarted(ctx context.Context, agent, provider string) {
	if m.ReviewsTotal == nil {
		return
	}
	m.ReviewsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("agent", agent),
			attribute.String("provider", provider),
		),
	)
	if m.ActiveReviews != nil {
		m.ActiveReviews.Add(ctx, 1)
	}
}

// RecordReviewCompleted records that a review has completed
func (m *Metrics) RecordReviewCompleted(ctx context.Context, status string, durationSeconds float64) {
	if m.ActiveReviews != nil {
		m.ActiveReviews.Add(ctx, -1)
	}
	if m.ReviewsByStatus != nil {
		m.ReviewsByStatus.Add(ctx, 1,
			metric.WithAttributes(attribute.String("status", status)),
		)
	}
	if m.ReviewDuration != nil {
		m.ReviewDuration.Record(ctx, durationSeconds,
			metric.WithAttributes(attribute.String("status", status)),
		)
	}
}

// RecordFindings records review findings by category
func (m *Metrics) RecordFindings(ctx context.Context, category string, count int64) {
	if m.ReviewsByCategory == nil {
		return
	}
	m.ReviewsByCategory.Add(ctx, count,
		metric.WithAttributes(attribute.String("category", category)),
	)
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, durationSeconds float64) {
	if m.HTTPRequestsTotal != nil {
		m.HTTPRequestsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("method", method),
				attribute.String("path", path),
				attribute.Int("status_code", statusCode),
			),
		)
	}
	if m.HTTPRequestDuration != nil {
		m.HTTPRequestDuration.Record(ctx, durationSeconds,
			metric.WithAttributes(
				attribute.String("method", method),
				attribute.String("path", path),
			),
		)
	}
}

// RecordAgentExecution records an agent execution
func (m *Metrics) RecordAgentExecution(ctx context.Context, agentName string, success bool) {
	if m.AgentExecutionsTotal != nil {
		m.AgentExecutionsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("agent.name", agentName),
				attribute.Bool("success", success),
			),
		)
	}
	if !success && m.AgentExecutionErrors != nil {
		m.AgentExecutionErrors.Add(ctx, 1,
			metric.WithAttributes(attribute.String("agent.name", agentName)),
		)
	}
}

// RecordGitClone records a git clone operation
func (m *Metrics) RecordGitClone(ctx context.Context, provider string, success bool, durationSeconds float64) {
	if m.GitCloneTotal != nil {
		m.GitCloneTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.Bool("success", success),
			),
		)
	}
	if m.GitCloneDuration != nil {
		m.GitCloneDuration.Record(ctx, durationSeconds,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.Bool("success", success),
			),
		)
	}
}
